package storage

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"github.com/prometheus/common/log"
	"go.uber.org/zap"
	"golang.org/x/sync/errgroup"
	"math/rand"
	"os"
	"os/exec"
	"path/filepath"
	"strconv"
	"strings"
	"time"
)

type Storage interface {
	Umount() error
}

type swsStorage struct {
	conf     *SwsConfig
	rootPath string
}

type Config struct {
	Type     string          `json:"type"`
	RootPath string          `json:"mountPath"`
	Param    json.RawMessage `json:"param"`
}

func NewStorage(conf Config) (Storage, error) {
	if conf.Type != "" {
		return newSwsStorage(conf, nil)
	}
	return nil, errors.New("unsupported type")
}

type SwsConfig struct {
	CacheCapacity      int64  `json:"cache_capacity"`
	CachePath          string `json:"cache_path"`
	Filer              string `json:"filer"`
	FilerPath          string `json:"filer_path"`
	VolumeServerAccess string `json:"volume_server_access"`
}

type sDriver interface {
	Mount(path string, conf *SwsConfig) error
}

type swsDriver struct {
}

func newSwsStorage(conf Config, driver sDriver) (*swsStorage, error) {
	sc := &SwsConfig{}
	if err := json.Unmarshal(conf.Param, sc); err != nil {
		log.Error("newSwsStorage error: config unmarshal failed", err)
		return nil, err
	}
	if driver == nil {
		driver = &swsDriver{}
	}
	eg := errgroup.Group{}
	eg.Go(func() error {
		return driver.Mount(conf.RootPath, sc)
	})
	succed := make(chan int, 1)
	eg.Go(func() error {
		for range []int{1, 2, 3} {
			time.Sleep(500 * time.Millisecond)
			out, err := ListMount()
			if err == nil {
				p := conf.RootPath
				if strings.HasSuffix(p, "/") {
					p = p[:len(p)-1]
				}
				if strings.Contains(out, "on "+p+" type fuse.seaweedfs") {
					succed <- 0
					return nil
				}
			}
		}
		succed <- 1
		return errors.New("mount not succeed")
	})
	go func() {
		if err := eg.Wait(); err != nil {
			log.Error("sws mount err:", zap.Error(err))
		}
	}()
	i := <-succed
	if i != 0 {
		return nil, errors.New("mount not succeed")
	}
	return &swsStorage{
		conf:     sc,
		rootPath: conf.RootPath,
	}, nil
}

func ListMount() (string, error) {
	command := "mount "

	ctx, cancel := context.WithTimeout(context.Background(), time.Second*10)
	defer cancel()
	fmt.Println("cmd:", command)
	c := exec.CommandContext(ctx, "/bin/sh", "-c", command)
	if out, err := c.Output(); err != nil {
		if ctx.Err() != nil && ctx.Err() == context.DeadlineExceeded {
			fmt.Println("DeadlineExceeded error:", err.Error())
		}
		return "", err
	} else {
		return string(out), nil
	}
}

func newTempDir() string {
	temp := os.TempDir()
	for i := 0; i < 100; i++ {
		path := joinPath(temp, RandStringRunes(10))
		if err := os.Mkdir(path, 0755); err == nil {
			return path
		}
	}
	return ""
}

var letterRunes = []rune("abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")

func RandStringRunes(n int) string {
	rand.Seed(time.Now().UnixNano())
	b := make([]rune, n)
	for i := range b {
		b[i] = letterRunes[rand.Intn(len(letterRunes))]
	}
	return string(b)
}

func joinPath(root, path string) string {
	return filepath.Join(root, path)
}

func (d *swsDriver) Mount(path string, conf *SwsConfig) error {
	command := "/usr/bin/weed mount -filer=" + conf.Filer
	if conf.CacheCapacity < 0 {
		return errors.New("CacheCapacity error")
	}
	cache := strconv.FormatInt(conf.CacheCapacity, 10)
	command += " -cacheCapacityMB=" + cache
	temp := newTempDir()
	if len(temp) < 1 {
		return errors.New("create temp cache dir error")
	}
	command += " -cacheDir=" + temp
	if len(path) == 0 {
		return os.ErrNotExist
	}
	conf.CachePath = temp

	if len(conf.VolumeServerAccess) > 0 {
		command += " -volumeServerAccess=" + conf.VolumeServerAccess
	}

	command += " -dir=" + path
	if len(conf.FilerPath) > 0 {
		command += " -filer.path=" + conf.FilerPath
	}
	fmt.Println("mount command:", command)
	c := exec.Command("/bin/sh", "-c", command)
	if out, err := c.Output(); err != nil {
		log.Error("weed command exec err:", zap.Error(err), zap.String("command", command))
		return err
	} else {
		log.Info("mount succeed:", zap.String("output", string(out)))
	}
	return nil
}

func (s *swsStorage) Umount() error {
	command := "umount -f " + s.rootPath
	c := exec.Command("/bin/sh", "-c", command)
	if out, err := c.Output(); err != nil {
		command := "umount -l" + s.rootPath
		_ = exec.Command("/bin/sh", "-c", command)
		log.Error("umount exec err:", zap.Error(err))
	} else {
		log.Info("umount succeed:", zap.String("output", string(out)), zap.String("path", s.rootPath))
	}

	if _, err := os.Stat(s.conf.CachePath); err == nil || !(os.IsNotExist(err)) {
		rmCmd := "rm -rf " + s.conf.CachePath
		c := exec.Command("/bin/sh", "-c", rmCmd)
		if out, err := c.Output(); err != nil {
			log.Error("rm tmp cache err:", zap.String("path", s.conf.CachePath), zap.Error(err))
			return err
		} else {
			fmt.Println("rm tmp cache dir:", out)
		}
	}
	return nil
}
