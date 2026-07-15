package api

import (
	"encoding/json"
	"net/http"
	"strconv"

	"github.com/go-chi/chi/v5"
)

func writeJSON(w http.ResponseWriter, status int, v interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}

func writeError(w http.ResponseWriter, status int, message string) {
	writeJSON(w, status, map[string]string{"error": message})
}

func urlParamUint(r *http.Request, name string) (uint, error) {
	v, err := strconv.ParseUint(chi.URLParam(r, name), 10, 64)
	if err != nil {
		return 0, err
	}
	return uint(v), nil
}
