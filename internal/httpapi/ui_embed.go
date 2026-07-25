package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed ui/*
var uiFS embed.FS

func (s *Server) handleIndex(w http.ResponseWriter, _ *http.Request) {
	content, err := fs.ReadFile(uiFS, "ui/index.html")
	if err != nil {
		http.Error(w, "ui not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	_, _ = w.Write(content)
}

func (s *Server) handleBIP39Wordlist(w http.ResponseWriter, _ *http.Request) {
	content, err := fs.ReadFile(uiFS, "ui/bip39_english.txt")
	if err != nil {
		http.Error(w, "wallet wordlist is not available", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "text/plain; charset=utf-8")
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(content)
}
