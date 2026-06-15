package main

import (
	"context"
	"flag"
	"fmt"
	"os"
	"os/signal"
	"syscall"
	"time"

	"telegram-bot/internal/api"
	"telegram-bot/internal/bot"
	"telegram-bot/internal/config"
	"telegram-bot/internal/i18n"
	"telegram-bot/internal/logger"
	"telegram-bot/internal/notify"
)

// Version 由 -ldflags 注入
var Version = "dev"

func main() {
	cfgPath := flag.String("config", "configs/config.yaml", "path to config file")
	showVersion := flag.Bool("version", false, "print version and exit")
	flag.Parse()

	if *showVersion {
		fmt.Printf("telegram-bot %s\n", Version)
		return
	}

	cfg, err := config.Load(*cfgPath)
	if err != nil {
		fmt.Fprintf(os.Stderr, "load config: %v\n", err)
		os.Exit(1)
	}

	log, err := logger.Init(cfg.Log.Level)
	if err != nil {
		fmt.Fprintf(os.Stderr, "init logger: %v\n", err)
		os.Exit(1)
	}
	defer log.Sync()

	channelKey := config.ChannelKey()
	channelSecret := config.ChannelSecret()
	if channelKey == "" || channelSecret == "" {
		log.Fatalw("TG_CHANNEL_KEY / TG_CHANNEL_SECRET env vars are required")
	}

	apiClient, err := api.NewClient(api.Config{
		BaseURL:            cfg.APIBaseURL(),
		ChannelKey:         channelKey,
		ChannelSecret:      channelSecret,
		TimeoutSeconds:     cfg.API.TimeoutSeconds,
		InsecureSkipVerify: cfg.API.InsecureSkipVerify,
	})
	if err != nil {
		log.Fatalw("create api client", "error", err)
	}

	bundle, err := i18n.NewBundle(cfg.Bot.DefaultLocale)
	if err != nil {
		log.Fatalw("init i18n bundle", "error", err)
	}

	tgBot, err := bot.New(cfg, apiClient, bundle, log)
	if err != nil {
		log.Fatalw("init bot", "error", err)
	}

	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	if err := tgBot.Start(ctx); err != nil {
		log.Fatalw("start bot", "error", err)
	}

	notifySrv := notify.NewServer(cfg, channelKey, channelSecret, apiClient, tgBot.RawTelebot(), bundle, log)
	go func() {
		if err := notifySrv.Start(); err != nil {
			log.Errorw("notify server stopped", "error", err)
		}
	}()

	log.Infow("telegram-bot started",
		"version", Version,
		"api_base", cfg.APIBaseURL(),
		"listen", cfg.Server.Listen,
		"default_locale", cfg.Bot.DefaultLocale,
		"machine_code", tgBot.MachineCode(),
	)

	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	<-sigCh
	log.Infow("shutting down")

	// 优雅退出：给 5 秒处理进行中的请求
	tgBot.Stop()
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer shutdownCancel()
	_ = notifySrv.Stop(shutdownCtx)
	_ = log.Sync()
}
