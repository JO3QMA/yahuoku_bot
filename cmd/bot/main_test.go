package main

import (
	"context"
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

func Test_runBotMainEntry_invokesRunWithSignal(t *testing.T) {
	prev := runWithSignalHook
	t.Cleanup(func() { runWithSignalHook = prev })
	called := false
	runWithSignalHook = func(parent context.Context, deps *botDeps) error {
		called = true
		return nil
	}
	if err := runBotMainEntry(); err != nil {
		t.Fatal(err)
	}
	if !called {
		t.Fatal("expected hook")
	}
}
