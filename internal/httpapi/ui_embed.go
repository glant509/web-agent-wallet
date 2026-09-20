package httpapi

import (
	"embed"
	"io/fs"
	"mime"
	"net/http"
	"path/filepath"
)

//go:embed ui/index.html ui/app.css ui/bootstrap.js ui/app.js ui/shared.js ui/bip39_english.txt ui/wallet_derivation.js ui/evm_signer.js ui/platform_runtime.js
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
	s.serveUIAsset(w, path)
}

func (s *Server) serveUIAsset(w http.ResponseWriter, path string) {
	content, err := fs.ReadFile(uiStaticFS, path)
	if err != nil {
		http.Error(w, "404 page not found", http.StatusNotFound)
		return
	}
	contentType := mime.TypeByExtension(filepath.Ext(path))
	if contentType == "" {
		contentType = http.DetectContentType(content)
	}
	w.Header().Set("Content-Type", contentType)
	w.Header().Set("Cache-Control", "public, max-age=31536000, immutable")
	_, _ = w.Write(content)
}
