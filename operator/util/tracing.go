package util

import (
	"log"
	"os"
)

var (
	trace = func() bool {
		_, ok := os.LookupEnv("KUBEACTIONS_TRACE")
		return ok
	}()
)

func Tracef(format string, v ...interface{}) {
	if trace {
		log.Printf(format, v...)
	}
}

func Trace(v ...interface{}) {
	if trace {
		log.Print(v...)
	}
}

func Traceln(v ...interface{}) {
	if trace {
		log.Println(v...)
	}
}
