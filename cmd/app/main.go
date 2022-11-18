package main

import (
	"flag"
	"log"

	"github.com/s02190058/billing-service/internal/app"
	"github.com/s02190058/billing-service/internal/config"
)

var configPath = flag.String("config", "./configs/main.yml", "path to config file")

func main() {
	cfg, err := config.New(*configPath)
	if err != nil {
		log.Fatalf("unable to read config: %v", err)
	}

	app.Run(cfg)
}
