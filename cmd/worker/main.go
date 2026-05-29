package main

import (
	"context"
	"log"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/nats-io/nats.go"
	"github.com/pop/erp_meta/internal/config"
	"github.com/pop/erp_meta/internal/db"
	"github.com/pop/erp_meta/internal/delivery"
	"github.com/pop/erp_meta/internal/events"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatalf("config: %v", err)
	}

	ctx := context.Background()
	pool, err := db.NewPool(ctx, cfg.DatabaseURL)
	if err != nil {
		log.Fatalf("database: %v", err)
	}
	defer pool.Close()

	nc, err := nats.Connect(cfg.NATSURL,
		nats.Name("meta-gateway-worker"),
		nats.MaxReconnects(-1),
		nats.ReconnectWait(2*time.Second),
	)
	if err != nil {
		log.Fatalf("nats: %v", err)
	}
	defer nc.Close()

	js, err := events.SetupJetStream(nc)
	if err != nil {
		log.Fatalf("jetstream: %v", err)
	}

	webhookRepo := db.NewWebhookRepo(pool)
	eventRepo := db.NewEventRepo(pool)
	deliverer := delivery.NewDeliverer(webhookRepo, eventRepo, cfg.DeliveryMaxAttempts, cfg.DeliveryBackoffs())

	subscriber := events.NewSubscriber(js)
	sub, err := subscriber.Subscribe(func(env events.Envelope) error {
		return deliverer.Handle(context.Background(), env)
	})
	if err != nil {
		log.Fatalf("subscribe: %v", err)
	}
	defer sub.Unsubscribe()

	log.Println("worker started (JetStream), listening for events...")

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit
	log.Println("worker shutting down...")
}
