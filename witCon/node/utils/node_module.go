package utils

import (
	"fmt"
	"io"
	"os"
	"runtime"
)

func Fatalf(format string, args ...interface{}) {
	w := io.MultiWriter(os.Stdout, os.Stderr)
	if runtime.GOOS == "windows" {
		w = os.Stdout
	} else {
		outF, _ := os.Stdout.Stat()
		errF, _ := os.Stderr.Stat()
		if outF != nil && errF != nil && os.SameFile(outF, errF) {
			w = os.Stderr
		}
	}
	fmt.Fprintf(w, "Fatal: "+format+"\n", args...)
	os.Exit(1)
}
