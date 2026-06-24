package main

import (
	"errors"
	"fmt"
	"testing"
)

func Test_main_smoke(t *testing.T) {
	prev := botMainRun
	t.Cleanup(func() { botMainRun = prev })
	botMainRun = func() error { return nil }
	main()
}

func Test_main_logsFatalOnRunError(t *testing.T) {
	prevRun := botMainRun
	prevFatal := logFatalf
	t.Cleanup(func() {
		botMainRun = prevRun
		logFatalf = prevFatal
	})
	botMainRun = func() error { return errors.New("boom") }
	var logged string
	logFatalf = func(format string, v ...any) { logged = fmt.Sprintf(format, v...) }
	main()
	if logged == "" {
		t.Fatal("expected logFatalf")
	}
}