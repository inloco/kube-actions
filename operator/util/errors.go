package util

import (
	"fmt"
	"runtime"
)

func WrapErr(err error) error {
	pc, file, line, ok := runtime.Caller(1)
	if !ok {
		return err
	}

	prefix := runtime.FuncForPC(pc).Name()
	if trace {
		prefix += fmt.Sprintf("@%s:%d", file, line)
	}

	return fmt.Errorf("%s: %w", prefix, err)
}
