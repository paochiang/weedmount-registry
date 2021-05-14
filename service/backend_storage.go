package service

import (
	"encoding/json"
	"github.com/prometheus/common/log"
	"gitlab.virtaitech.com/gemini-platform/docker-registry/storage"
	"go.uber.org/zap"
)

var (
	BackendStorageRootPath = "/registry"
	RootPath               = "/var/lib/registry"
	BackendTypeSWFS        = "swfs"
)

func InitStorage() {
	//specified filer mount param
	swsConfig := storage.SwsConfig{
		CacheCapacity:      0,
		Filer:              "filer:8888",
		FilerPath:          BackendStorageRootPath,
		VolumeServerAccess: "",
	}
	swsConfigBytes, err := json.Marshal(swsConfig)
	if err != nil {
		log.Fatalf("marshal SwsConfig error: %s\n", err)
	}

	//new backend storage
	_, err = storage.NewStorage(storage.Config{Type: BackendTypeSWFS, RootPath: RootPath,
		Param: swsConfigBytes})
	if err != nil {
		log.Fatalf("NewStorage error: %s\n", err)
	}
	log.Info("InitStorage, seaweedfs init finished! ",
		zap.String("config", string(swsConfigBytes)))
}
