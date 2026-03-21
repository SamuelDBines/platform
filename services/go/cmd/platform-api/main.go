package main

import (
	"encoding/json"
	"log"
	"net/http"
	"os"
)

type managedApp struct {
	ID       string `json:"id"`
	Domain   string `json:"domain"`
	WebPort  int    `json:"webPort"`
	DocsPath string `json:"docsPath"`
	Service  string `json:"service"`
}

var apps = []managedApp{
	{
		ID:       "relayemail",
		Domain:   "relayemail.net",
		WebPort:  4173,
		DocsPath: "relayemail.net/docs",
		Service:  "services/go/cmd/platform-api",
	},
	{
		ID:       "statuslater",
		Domain:   "statuslater.co.uk",
		WebPort:  4174,
		DocsPath: "statuslater.co.uk/docs",
		Service:  "services/go/cmd/platform-api",
	},
}

func main() {
	mux := http.NewServeMux()
	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
	})
	mux.HandleFunc("/api/apps", func(w http.ResponseWriter, r *http.Request) {
		writeJSON(w, http.StatusOK, apps)
	})

	addr := ":" + envOr("PLATFORM_API_PORT", "8080")
	log.Printf("platform api listening on %s", addr)

	if err := http.ListenAndServe(addr, withCORS(mux)); err != nil {
		log.Fatal(err)
	}
}

func withCORS(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", envOr("PLATFORM_WEB_ORIGIN", "*"))
		w.Header().Set("Access-Control-Allow-Methods", "GET, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type")

		if r.Method == http.MethodOptions {
			w.WriteHeader(http.StatusNoContent)
			return
		}

		next.ServeHTTP(w, r)
	})
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)

	if err := json.NewEncoder(w).Encode(value); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func envOr(key string, fallback string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}

	return fallback
}
