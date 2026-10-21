package main

import (
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"strings"
)

type request struct {
	Operation string `json:"operation_id"`
	Asset     string `json:"asset_id"`
	Seller    string `json:"seller_id"`
	Buyer     string `json:"buyer_id"`
	Ask       int    `json:"ask"`
	Budget    int    `json:"budget"`
	Order     string `json:"order_id"`
}

func handler(m *market) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			http.Error(w, "POST required", http.StatusMethodNotAllowed)
			return
		}
		var input request
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 1<<16)).Decode(&input); err != nil {
			http.Error(w, "invalid JSON", http.StatusBadRequest)
			return
		}
		var result asset
		var err error
		switch strings.Trim(r.URL.Path, "/") {
		case "assets":
			result, err = m.register(r.Context(), asset{ID: input.Asset, Seller: input.Seller, Ask: input.Ask}, input.Operation)
		case "match":
			result, err = m.match(r.Context(), input.Asset, input.Buyer, input.Budget, input.Operation)
		case "handoff":
			result, err = m.handoff(r.Context(), input.Asset, input.Buyer, input.Order, input.Operation)
		default:
			http.NotFound(w, r)
			return
		}
		if err != nil {
			status := http.StatusConflict
			var upstream *apiError
			if errors.As(err, &upstream) {
				status = http.StatusBadGateway
				if upstream.Status >= 400 && upstream.Status < 500 {
					status = upstream.Status
				}
			}
			if input.Operation == "" {
				status = http.StatusBadRequest
			}
			http.Error(w, err.Error(), status)
			return
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(result)
	})
}

func main() {
	client, err := newRoomClient()
	if err != nil {
		log.Fatal(err)
	}
	address := os.Getenv("LISTEN_ADDR")
	if address == "" {
		address = ":8080"
	}
	log.Fatal(http.ListenAndServe(address, handler(newMarket(client))))
}
