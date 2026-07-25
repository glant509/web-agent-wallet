package httpapi

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"
	"time"
	"web3-service-agent/internal/agent"
	"web3-service-agent/internal/llm"

	"web3-service-agent/internal/session"
)

type Server struct {
	runtime  *agent.Runtime
	sessions *session.Manager
}

type createSessionRequest struct {
	SessionID string `json:"session_id"`
}

type createSessionResponse struct {
	SessionID string `json:"session_id"`
}

func New(runtimeEngine *agent.Runtime, sessions *session.Manager) http.Handler {
	server := &Server{
		runtime:  runtimeEngine,
		sessions: sessions,
	}

	mux := http.NewServeMux()
	mux.HandleFunc("GET /{$}", server.handleIndex)
	mux.HandleFunc("GET /wallet/bip39-english", server.handleBIP39Wordlist)
	mux.HandleFunc("GET /healthz", server.handleHealthz)
	mux.HandleFunc("GET /v1/market/top", server.handleMarketTop)
	mux.HandleFunc("GET /v1/trade/token", server.handleTradeToken)
	mux.HandleFunc("GET /v1/trade/klines", server.handleTradeKlines)
	mux.HandleFunc("POST /v1/sessions", server.handleCreateSession)
	mux.HandleFunc("POST /v1/agent/runs", server.handleRun)
	mux.HandleFunc("POST /v1/agent/runs/stream", server.handleStream)
	return mux
}

func (s *Server) handleHealthz(w http.ResponseWriter, _ *http.Request) {
	writeJSON(w, http.StatusOK, map[string]string{"status": "ok"})
}

func (s *Server) handleCreateSession(w http.ResponseWriter, r *http.Request) {
	var request createSessionRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	snapshot := s.sessions.Create(request.SessionID)
	writeJSON(w, http.StatusCreated, createSessionResponse{
		SessionID: snapshot.ID,
	})
}

func (s *Server) handleRun(w http.ResponseWriter, r *http.Request) {
	var request agent.RunRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	ctx, cancel := context.WithTimeout(r.Context(), 900*time.Second)
	defer cancel()

	response, err := s.runtime.Run(ctx, request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	writeJSON(w, http.StatusOK, response)
}

func (s *Server) handleStream(w http.ResponseWriter, r *http.Request) {
	var request agent.RunRequest
	if err := decodeJSON(r, &request); err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	stream, sessionID, err := s.runtime.Stream(r.Context(), request)
	if err != nil {
		writeError(w, http.StatusBadRequest, err)
		return
	}

	w.Header().Set("Content-Type", "text/event-stream")
	w.Header().Set("Cache-Control", "no-cache")
	w.Header().Set("Connection", "keep-alive")

	flusher, ok := w.(http.Flusher)
	if !ok {
		writeError(w, http.StatusInternalServerError, errors.New("streaming is not supported by this server"))
		return
	}

	writeSSE(w, "session", map[string]string{"session_id": sessionID})
	flusher.Flush()

	for event := range stream {
		if event.Err != nil {
			writeSSE(w, "error", map[string]string{"message": event.Err.Error()})
			flusher.Flush()
			return
		}

		writeSSE(w, "delta", event)
		flusher.Flush()

		if event.Done {
			return
		}
	}
}

func decodeJSON(r *http.Request, target any) error {
	defer r.Body.Close()

	if r.Body == nil {
		return errors.New("request body is required")
	}

	decoder := json.NewDecoder(r.Body)
	decoder.DisallowUnknownFields()
	if err := decoder.Decode(target); err != nil {
		return err
	}
	return nil
}

func writeJSON(w http.ResponseWriter, status int, value any) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(value)
}

func writeError(w http.ResponseWriter, status int, err error) {
	writeJSON(w, status, map[string]string{"error": err.Error()})
}

func writeSSE(w http.ResponseWriter, event string, payload any) {
	body, err := json.Marshal(payload)
	if err != nil {
		body = []byte(`{"error":"marshal sse payload failed"}`)
	}

	_, _ = w.Write([]byte("event: " + event + "\n"))
	_, _ = w.Write([]byte("data: "))
	_, _ = w.Write(body)
	_, _ = w.Write([]byte("\n\n"))
}

var _ = llm.StreamEvent{}
