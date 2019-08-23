package env

import (
	"os"
	"strconv"
	"time"
)

// Int32 returns parsed int32 value of environment variable
func UInt32(name string, defvalue uint32) uint32 {
	if envVar, ok := os.LookupEnv(name); ok {
		if value, err := strconv.Atoi(envVar); err == nil {
			return uint32(value)
		}
	}
	return defvalue
}

// Duration returns parsed time.Duration value of environment variable
func Duration(name string, defvalue time.Duration) time.Duration {
	if envVar, ok := os.LookupEnv(name); ok {
		if value, err := time.ParseDuration(envVar); err == nil {
			return value
		}
	}
	return defvalue
}
