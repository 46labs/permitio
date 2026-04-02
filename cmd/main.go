package main

import (
	"log"

	"github.com/46labs/permitio/pkg/config"
	"github.com/46labs/permitio/pkg/server"
)

func main() {
	cfg, err := config.Load()
	if err != nil {
		log.Fatal(err)
	}

	srv := server.New(cfg)
	if err := srv.Start(); err != nil {
		log.Fatal(err)
	}
}
