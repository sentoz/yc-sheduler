package web

import "net/http"

// HealthHandler возвращает "OK" для всех health endpoints
func HealthHandler(w http.ResponseWriter, _ *http.Request) {
	w.WriteHeader(http.StatusOK)
	if _, err := w.Write([]byte("OK")); err != nil {
		// ResponseWriter errors are typically not recoverable
		// and logging here would require importing a logger package
		// which would add unnecessary dependency
		_ = err
	}
}
