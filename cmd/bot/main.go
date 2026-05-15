package main

import (
	"context"
	"log"
)

func runBotMainEntry() error {
	return runWithSignal(context.Background(), nil)
}

// botMainRun は main から呼ばれる実体（テストで差し替え可能）。
var botMainRun = runBotMainEntry

// logFatalf は log.Fatalf の差し替え（テストで終了を避ける）。
var logFatalf = log.Fatalf

func main() {
	if err := botMainRun(); err != nil {
		logFatalf("[yahoo_auctions_bot] %v", err)
	}
}
