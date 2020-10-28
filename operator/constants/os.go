package constants

import "runtime"

var (
	os = runtime.GOOS
)

func OS() string {
	return os
}
