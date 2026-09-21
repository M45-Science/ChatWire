package webapi

import (
	"embed"
	"net/http"
)

//go:embed assets/*
var assets embed.FS

func (s *Server) asset(w http.ResponseWriter, r *http.Request, name string) {
	b, e := assets.ReadFile("assets/" + name)
	if e != nil {
		http.NotFound(w, r)
		return
	}
	switch name {
	case "app.js", "login.js":
		w.Header().Set("Content-Type", "text/javascript; charset=utf-8")
	case "style.css":
		w.Header().Set("Content-Type", "text/css; charset=utf-8")
	default:
		w.Header().Set("Content-Type", "text/html; charset=utf-8")
	}
	_, _ = w.Write(b)
}
func (s *Server) static(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		s.asset(w, r, "index.html")
	case "/app.js":
		s.asset(w, r, "app.js")
	case "/login.js":
		s.asset(w, r, "login.js")
	case "/style.css":
		s.asset(w, r, "style.css")
	default:
		http.NotFound(w, r)
	}
}
