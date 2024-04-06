package utils

import (
	"os"
	"os/signal"
	"syscall"

	ipfsLog "github.com/ipfs/go-log/v2"
)

func WaitForTermSignal(log *ipfsLog.ZapEventLogger) {
	// wait for a SIGINT or SIGTERM signal
	ch := make(chan os.Signal, 1)
	signal.Notify(ch, syscall.SIGINT, syscall.SIGTERM)
	<-ch

	log.Debugln("Received signal, shutting down...")

}
