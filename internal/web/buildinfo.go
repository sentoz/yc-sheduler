package web

import (
	"encoding/json"
	"net/http"

	"github.com/rs/zerolog/log"
	"github.com/woozymasta/yc-scheduler/internal/vars"
)

// BuildInfoHandler возвращает JSON с информацией о сборке приложения
func BuildInfoHandler(w http.ResponseWriter, _ *http.Request) {
	buildInfo := vars.Info()
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	if err := json.NewEncoder(w).Encode(buildInfo); err != nil {
		log.Warn().Err(err).Msg("Failed to encode build info")
	}
}
