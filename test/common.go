package test

import (
	"fmt"
	"os"
	"os/exec"
)

func startEnvoy(configPath string) {
	cmd := exec.Command("envoy", "-c", configPath)
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr
	err := cmd.Start()
	if err != nil {
		panic(fmt.Sprintf("failed to start envoy: %v", err))
	}
	err = cmd.Wait()
	if err != nil {
		panic(fmt.Sprintf("failed to wait envoy: %v", err))
	}
}
