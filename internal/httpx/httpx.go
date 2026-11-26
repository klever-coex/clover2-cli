package httpx

import (
	"encoding/json"
	"net/http"

	"github.com/klever-coex/clover2-cli/internal/agentapi"
)

func WriteError(w http.ResponseWriter, status int, code, msg string) {
	WriteJSON(w, status, map[string]agentapi.Error{"error": {Code: code, Message: msg}})
}

func WriteJSON(w http.ResponseWriter, status int, v any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(v)
}
