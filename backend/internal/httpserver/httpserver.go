package httpserver

import (
	"encoding/json"
	"fmt"
	"net/http"
	"time"
)

type HTTPResponse struct {
	Status  int         `json:"status",omitempty`
	Message string      `json:"message"`
	Data    interface{} `json:"data",omitempty`
}

type Config struct {
	Port int
	Name string
}

func NewServer(cfg Config, handler http.Handler) *http.Server {
	addr := fmt.Sprintf(":%d", cfg.Port)
	return &http.Server{
		Addr:    addr,
		Handler: handler,
	}
}

func JSON(w http.ResponseWriter, code int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(code)
	_ = json.NewEncoder(w).Encode(v)
}

func OK(w http.ResponseWriter, data any) {
	JSON(w, http.StatusOK, HTTPResponse{Status: http.StatusOK, Message: "Success", Data: data})
}
func BadRequest(w http.ResponseWriter, msg string, data any) {
	JSON(w, http.StatusBadRequest, HTTPResponse{Status: http.StatusBadRequest, Message: msg, Data: data})
}
func Unauthorized(w http.ResponseWriter, msg string) {
	JSON(w, http.StatusUnauthorized, HTTPResponse{Status: http.StatusUnauthorized, Message: msg})
}

func SendJsonRequest(w http.ResponseWriter, j interface{}) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(j)
}

func LoggerMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		startTime := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("[%s] %s %s %v\n", r.Method, r.URL.Path, r.RemoteAddr, time.Since(startTime))
	})
}

// Simple logging may use the logger pkg later
func Logger(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		start := time.Now()
		next.ServeHTTP(w, r)
		fmt.Printf("[%s] %s %s %v\n", r.Method, r.URL.Path, r.RemoteAddr, time.Since(start))
	})
}

func AuthMiddleware(next http.HandlerFunc) http.HandlerFunc {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		authenticated := true
		if !authenticated {
			http.Error(w, "Unauthorized", http.StatusUnauthorized)
			return
		}
		next.ServeHTTP(w, r)
	})
}

func With(mux *http.ServeMux, path string, h http.Handler) {
	mux.Handle(path, Logger(h))
}

func HandleMiddleWare(mux *http.ServeMux, path string, next http.HandlerFunc) {
	mux.HandleFunc(path, LoggerMiddleware(AuthMiddleware(next)))
}
