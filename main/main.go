package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/facefunk/hnoss"
)

const defaultConfigFile = "./hnoss.yaml"

func main() {
	logger := log.Default()

	configFile := defaultConfigFile
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}
	conf, err := hnoss.ConfigureFromFile(configFile)
	if err != nil {
		logger.Print(err)
		return
	}

	unlock, err := hnoss.Lock(conf.PIDFile)
	if err != nil {
		logger.Print(err)
		return
	}
	defer func() {
		if err = unlock(); err != nil {
			logger.Print(err)
			return
		}
	}()

	// stop does cancel, but also deconstructs.
	ctx, stop := signal.NotifyContext(context.Background(), os.Interrupt, syscall.SIGTERM)
	defer stop()

	ran := hnoss.NewTextFileTimeAdapter(conf.RanFile)
	ipService := hnoss.NewPlainTextIPServiceAdapter(conf.IPServiceURL)
	ipCache := hnoss.NewTextFileIPAdapter(conf.IPCacheFile)
	chat := hnoss.NewDiscordChatAdapter(conf.DiscordBotToken, conf.DiscordDefaultChannelName)
	now := hnoss.NewRealNowAdapter()

	h := hnoss.New(conf, logger, ran, ipService, ipCache, chat, now)
	h.Start(ctx)
}
