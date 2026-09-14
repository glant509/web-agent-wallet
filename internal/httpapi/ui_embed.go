package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed ui/*
var uiFS embed.FS

var uiStaticFS = func() fs.FS {
	subtree, err := fs.Sub(uiFS, "ui")
	if err != nil {
		panic(err)
	}
	return subtree
}()

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

func (s *Server) handleUIAsset(w http.ResponseWriter, r *http.Request) {
	path := r.PathValue("path")
	if path == "" {
		http.NotFound(w, r)
		return
	}
	content, err := fs.ReadFile(uiStaticFS, path)
	if err != nil {
		http.NotFound(w, r)
		return
	}
	w.Header().Set("Content-Type", http.DetectContentType(content))
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(content)
}
