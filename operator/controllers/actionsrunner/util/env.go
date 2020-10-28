package util

import (
	"os"
)

func EnvVar(variable string, defaultValue string) string {
	if value, ok := os.LookupEnv(variable); ok {
		return value
	}

	return defaultValue
}
