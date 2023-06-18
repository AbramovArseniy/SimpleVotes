package main

import (
	"log"
	"net/http"

	"github.com/AbramovArseniy/SimpleVotes/internal/config"
	"github.com/AbramovArseniy/SimpleVotes/internal/handlers"
)

func main() {
	cfg := config.NewConfig()
	handler := handlers.NewHandler(*cfg)
	srv := http.Server{
		Addr:    cfg.Address,
		Handler: handler.Route(),
	}
	log.Println("server starting at", srv.Addr)
	if err := srv.ListenAndServe(); err != nil {
		log.Fatal("failed to start server:", err)
	}
}
