package main

import (
	"log"
	"net/http"
	"os"
	"time"

	"github.com/kosc/chessweb/internal/server"
)

// @title           Chess
// @version         0.1
// @description     Шахматы человек vs компьютер (учебный проект)
// @BasePath        /
// @schemes         http
//
// @contact.name    @hotkosc
// @license.name    MIT
func main() {
	addr := getenv("ADDR", ":8080")

	srv := &http.Server{
		Addr:              addr,
		Handler:           server.NewHTTPHandler(),
		ReadHeaderTimeout: 5 * time.Second,
	}

	log.Printf("listening on %s", addr)
	log.Fatal(srv.ListenAndServe())
}

func getenv(k, def string) string {
	if v := os.Getenv(k); v != "" {
		return v
	}
	return def
}
