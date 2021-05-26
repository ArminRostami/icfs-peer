package env

import (
	"os"
	"strings"
)

const (
	Bootstrap = "bootstrap"
)

func DockerEnabled() bool {
	val, exists := os.LookupEnv("DOCKER_ENABLED")
	if !exists {
		return false
	}
	return strings.EqualFold(val, "1")
}
