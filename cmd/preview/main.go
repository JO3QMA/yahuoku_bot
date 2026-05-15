package main

import (
	"os"
)

// previewExit は RunPreview の終了コード処理（テストで差し替え可能）。
var previewExit = os.Exit

// previewArgvDefault は RunPreview に渡す引数の既定実装。
func previewArgvDefault() []string {
	return os.Args[1:]
}

// previewArgv は RunPreview に渡す引数（テストで差し替え可能）。
var previewArgv = previewArgvDefault

func main() {
	cfgPath := "config.yaml"
	if p := os.Getenv("CONFIG_PATH"); p != "" {
		cfgPath = p
	}
	previewExit(RunPreview(os.Stdout, previewArgv(), cfgPath, nil))
}
