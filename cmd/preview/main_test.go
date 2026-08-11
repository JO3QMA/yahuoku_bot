package main

import (
	"os"
	"testing"
)

func Test_main_preview_exitCode(t *testing.T) {
	prevE := previewExit
	prevA := previewArgv
	t.Cleanup(func() {
		previewExit = prevE
		previewArgv = prevA
	})
	var code int
	previewExit = func(c int) { code = c }
	previewArgv = func() []string { return []string{"anyid"} }
	main()
	if code != 2 {
		t.Fatalf("exit code=%d, want 2 (missing OPENAI_API_KEY etc.)", code)
	}
}

func Test_previewArgv_readsOsArgs(t *testing.T) {
	old := os.Args
	t.Cleanup(func() { os.Args = old })
	os.Args = []string{"binary", "only-one"}
	got := previewArgv()
	if len(got) != 1 || got[0] != "only-one" {
		t.Fatalf("%#v", got)
	}
}

func Test_previewArgvDefault_impl(t *testing.T) {
	old := os.Args
	t.Cleanup(func() { os.Args = old })
	os.Args = []string{"bin", "a", "b"}
	got := previewArgvDefault()
	if len(got) != 2 || got[0] != "a" || got[1] != "b" {
		t.Fatalf("%#v", got)
	}
}
