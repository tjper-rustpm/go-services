package main

import (
	"log"
	"os"

	"github.com/tjper/rustcron/cmd/payment/config"
	"github.com/tjper/rustcron/cmd/payment/controller"

	"github.com/stripe/stripe-go/v72/client"
	"go.uber.org/zap"
)

func main() {
	os.Exit(run())
}

const (
	ecExit = iota
	_
	ecDatabaseConnection
	ecMigration
)

func run() int {
	cfg := config.Load()

	logger, err := zap.NewDevelopment()
	if err != nil {
		log.Fatal(err)
	}
	defer func() { _ = logger.Sync() }()

	stripe := &client.API{}
	stripe.Init(cfg.StripeKey(), nil)

	ctrl := controller.New(
		logger,
		stripe.CheckoutSessions,
		stripe.BillingPortalSessions,
	)

	return ecExit
}
