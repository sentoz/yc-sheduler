package web

import "net/http"

// HealthHandler возвращает "OK" для всех health endpoints
func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	w.Write([]byte("OK"))
}
