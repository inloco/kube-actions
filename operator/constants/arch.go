package constants

import "runtime"

var (
	arch = runtime.GOARCH
)

func Arch() string {
	return arch
}
