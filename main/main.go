package main

import (
	"context"
	"os"
	"os/signal"
	"syscall"

	"github.com/facefunk/hnoss"
)

const defaultConfigFile = "/etc/hnoss.yaml"

func main() {

	configFile := defaultConfigFile
	if len(os.Args) > 1 {
		configFile = os.Args[1]
	}
	conf, err := hnoss.ConfigureFromFile(configFile)
	if err != nil {
		panic(err)
	}

	unlock, err := hnoss.Lock(conf.PIDFile)
	if err != nil {
		panic(err)
	}
	defer hnoss.PanicOnError(unlock)

	logger, err := hnoss.NewLogger(conf.LogFile)
	if err != nil {
		panic(err)
	}
	defer hnoss.PanicOnError(logger.Close)

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
