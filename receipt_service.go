package main

import (
	"context"
	"encoding/json"
	"errors"
	"log"
	"net/http"
	"os"
	"time"
)

func main() {
	key, bucket := os.Getenv("INFRAI_API_KEY"), os.Getenv("INFRAI_BUCKET")
	if key == "" || bucket == "" {
		log.Fatal("set INFRAI_API_KEY and INFRAI_BUCKET")
	}
	api := API{key: key, http: &http.Client{Timeout: 30 * time.Second}}
	if err := api.ensureBucket(context.Background(), bucket); err != nil {
		log.Fatal(err)
	}
	http.HandleFunc("POST /events", func(w http.ResponseWriter, r *http.Request) {
		var event PaymentEvent
		if err := json.NewDecoder(http.MaxBytesReader(w, r.Body, 12<<20)).Decode(&event); err != nil || !validEvent(event) {
			http.Error(w, "invalid payment event", 400)
			return
		}
		decision := decide(event)
		response := struct {
			EventID     string `json:"event_id"`
			Action      string `json:"action"`
			AuditNotice string `json:"audit_notice"`
			AssetURL    string `json:"asset_url,omitempty"`
		}{EventID: event.ID, Action: decision.Action, AuditNotice: decision.Notice}
		if decision.Action == "publish" {
			image, err := api.compress(r.Context(), event.Image)
			if err == nil {
				err = api.store(r.Context(), bucket, objectKey(event.ID), image)
			}
			if err == nil {
				response.AssetURL, err = api.signedRead(r.Context(), bucket, objectKey(event.ID))
			}
			if err != nil {
				var remote remoteError
				if errors.As(err, &remote) && remote.Status >= 400 && remote.Status < 500 {
					http.Error(w, remote.Error(), remote.Status)
					return
				}
				log.Printf("event %s: %v", event.ID, err)
				http.Error(w, "asset processing failed", 502)
				return
			}
		}
		w.Header().Set("Content-Type", "application/json")
		if err := json.NewEncoder(w).Encode(response); err != nil {
			log.Printf("encode response: %v", err)
		}
	})
	log.Fatal(http.ListenAndServe(":8080", nil))
}
