package main

import (
	"log"
	"os"
	"os/signal"
	"syscall"

	"github.com/pop/erp_meta/internal/config"
	"github.com/pop/erp_meta/internal/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	app, err := server.New(cfg)
	if err != nil {
		log.Fatalf("server: %v", err)
	}

	go func() {
		if err := app.Listen(cfg.Addr()); err != nil {
			log.Fatalf("listen: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("shutting down...")
	if err := app.Shutdown(); err != nil {
		log.Fatalf("shutdown: %v", err)
	}
}
