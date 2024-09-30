package main

import (
	"gateway/config"
	"gateway/internal/app/gateway"
	"log"
)

func main() {
	cfg, err := config.NewConfig()
	if err != nil {
		log.Fatalf("Ошибка чтения конфигурации: %v", err)
	}
	gateway.Run(cfg)
}
