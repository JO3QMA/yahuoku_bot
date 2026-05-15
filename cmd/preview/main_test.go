package main

import (
	"os"
	"path/filepath"
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
		t.Fatalf("exit code=%d, want 2 (missing GEMINI_API_KEY etc.)", code)
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

func Test_main_preview_configPathFromEnv(t *testing.T) {
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "fromenv.yaml")
	if err := os.WriteFile(cfgPath, []byte("allowed:\n  guilds: []\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	t.Setenv("CONFIG_PATH", cfgPath)
	t.Cleanup(func() { _ = os.Unsetenv("CONFIG_PATH") })
	prevE := previewExit
	prevA := previewArgv
	t.Cleanup(func() {
		previewExit = prevE
		previewArgv = prevA
	})
	var code int
	previewExit = func(c int) { code = c }
	previewArgv = func() []string { return []string{"id1"} }
	main()
	if code != 2 {
		t.Fatalf("exit=%d want 2 (no GEMINI_API_KEY)", code)
	}
}
