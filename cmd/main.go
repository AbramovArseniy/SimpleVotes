package main

import (
	"log"
	"net/http"

	"github.com/AbramovArseniy/SimpleVotes/internal/config"
)

func main() {
	cfg := config.NewConfig()
	srv := http.Server{
		Addr: cfg.Address,
	}
	log.Println("server starting at", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("failed to start server:", err)
	}
}
