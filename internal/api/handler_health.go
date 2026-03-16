package api

import (
	"net/http"
	"time"
)

// HealthCheckHandler handles health check requests
// @Summary Vérifier l'état de santé de l'application
// @Description Retourne le statut de l'application et de la base de données
// @Tags monitoring
// @Produce json
// @Success 200 {object} map[string]interface{} "Application healthy"
// @Failure 503 {object} map[string]interface{} "Application unhealthy"
// @Router /health [get]
func (h *Handler) HealthCheckHandler(w http.ResponseWriter, r *http.Request) {
	// Check database connection
	if err := h.DB.Ping(); err != nil {
		respondJSON(w, http.StatusServiceUnavailable, map[string]interface{}{
			"status":   "unhealthy",
			"database": "down",
			"error":    err.Error(),
		})
		return
	}

	uptime := time.Since(h.StartTime)
	respondJSON(w, http.StatusOK, map[string]interface{}{
		"status":   "healthy",
		"version":  h.Version,
		"uptime":   uptime.String(),
		"database": "up",
	})
}
