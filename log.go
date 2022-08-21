package main

import (
	"io"
	"log"
	"os"
)

var logger = New(os.Stderr, &app.Verbose, "", log.Lshortfile|log.LstdFlags)

func New(out io.Writer, verbose *bool, prefix string, flag int) *log.Logger {
	w := &logWriter{
		w:       out,
		verbose: verbose,
	}
	return log.New(w, prefix, flag)
}

type logWriter struct {
	w       io.Writer
	verbose *bool
}

func (l logWriter) Write(p []byte) (n int, err error) {
	if *l.verbose {
		return l.w.Write(p)
	}
	return len(p), nil
}
