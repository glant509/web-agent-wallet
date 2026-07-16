package httpapi

import (
	"embed"
	"io/fs"
	"net/http"
)

//go:embed ui/index.html
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
