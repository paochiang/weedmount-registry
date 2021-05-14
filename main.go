package main

import (
	"github.com/prometheus/common/log"
	"gitlab.virtaitech.com/gemini-platform/docker-registry/service"
	"os"
	"os/exec"
)

func init() {
	//init backend storage
	service.InitStorage()
}

func main() {
	log.Info("docker registry...")
	startDockerRegistry()
}

func startDockerRegistry() {
	command := "/entrypoint.sh /etc/docker/registry/config.yml"
	cmd := exec.Command("/bin/sh", "-c", command)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	if err := cmd.Run(); err != nil {
		log.Fatalf("failed to start docker registry: %v", err)
	}
}
