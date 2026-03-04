package server

import (
	"encoding/json"
	"net/http"
	"time"

	_ "github.com/kosc/chessweb/docs"
	httpSwagger "github.com/swaggo/http-swagger/v2"
)

func NewHTTPHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("GET /healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]any{
			"ok":   true,
			"time": time.Now().UTC().Format(time.RFC3339Nano),
		})
	})

	// API v1
	mux.HandleFunc("POST /api/v1/games", handleCreateGame)
	mux.HandleFunc("GET /api/v1/games/{id}", handleGetGame)

	mux.HandleFunc("POST /api/v1/games/{id}/legal-moves", handleLegalMovesRoute)
	mux.HandleFunc("POST /api/v1/games/{id}/move", handleMakeMoveRoute)

	// Swagger
	mux.Handle("GET /swagger/", httpSwagger.WrapHandler)

	// CORS для dev (React Vite)
	return withCORS(mux)
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		// Для dev-режима достаточно. В проде сузим Origin.
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")
		w.Header().Set("Access-Control-Allow-Methods", "GET,POST,OPTIONS")
		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json; charset=utf-8")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func readJSON(r *http.Request, dst any) error {
	defer r.Body.Close()
	dec := json.NewDecoder(r.Body)
	dec.DisallowUnknownFields()
	return dec.Decode(dst)
}
