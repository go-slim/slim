package main

import (
	"net/http"

	"go-slim.dev/slim"
)

func main() {
	s := slim.New()
	s.GET("/", func(c slim.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	s.Logger.Fatal(s.Start(":1324"))
}
