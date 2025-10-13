package main

import (
	"log"
	"net/http"
	"path/filepath"

	"go-slim.dev/slim"
)

func main() {
	s := slim.New()

	// Serve files from ./public (create it and add index.html to try)
	root := filepath.Clean("public")
	s.Use(slim.Static(root))

	// Fallback API route
	s.GET("/ping", func(c slim.Context) error { return c.String(http.StatusOK, "pong") })

	log.Fatal(s.Start(":1329"))
}
