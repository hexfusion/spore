package main

import (
	"context"
	"time"
)

const (
	addrB = "localhost:6121"
)

func main() {
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	go runPeerB(ctx)
	time.Sleep(1 * time.Second)

	runPeerA(ctx)
	time.Sleep(1 * time.Second)
}
