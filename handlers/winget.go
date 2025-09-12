package handlers

import (
	"encoding/json"
	"net/http"
	"strings"

	"setupforme/utils"
)

type WingetSearchResponse struct {
	ID        string `json:"id"`
	Name      string `json:"name"`
	Publisher string `json:"publisher"`
}

// WingetSearchHandler searches winget.run for a given query and returns top results
func WingetSearchHandler(w http.ResponseWriter, r *http.Request) {
	q := strings.TrimSpace(r.URL.Query().Get("q"))
	if q == "" {
		w.WriteHeader(http.StatusBadRequest)
		json.NewEncoder(w).Encode(map[string]string{"message": "missing q"})
		return
	}

	// For now, resolve only the top id; could be extended to return multiple
	id, err := utils.ResolveWingetID(q)
	if err != nil || id == "" {
		w.WriteHeader(http.StatusNotFound)
		json.NewEncoder(w).Encode(map[string]string{"message": "no package found"})
		return
	}

	json.NewEncoder(w).Encode([]WingetSearchResponse{{ID: id, Name: q}})
}
