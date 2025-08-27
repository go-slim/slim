package main

import (
	"errors"
	"net/http"

	"go-slim.dev/slim"
	"go-slim.dev/slim/middleware"
)

func main() {
	s := slim.New()

	// Route errors go through ErrorHandler; Logger should log end of request.
	s.ErrorHandler = func(c slim.Context, err error) {
		// ensure logger logs end of request
		middleware.LogEnd(c, err)
		// basic mapping to 500 for non-HTTPError
		var he *slim.HTTPError
		if errors.As(err, &he) {
			http.Error(c.Response(), http.StatusText(he.Code), he.Code)
			return
		}
		http.Error(c.Response(), http.StatusText(http.StatusInternalServerError), http.StatusInternalServerError)
	}

	s.Use(middleware.Logger())
	s.Use(slim.Recovery())

	s.GET("/ok", func(c slim.Context) error {
		return c.String(http.StatusOK, "logged ok")
	})

	s.GET("/panic", func(c slim.Context) error {
		panic("boom")
	})

	s.Logger.Fatal(s.Start(":1327"))
}
