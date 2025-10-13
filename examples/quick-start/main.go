package main

import (
	"log"
	"net/http"

	"go-slim.dev/slim"
)

func main() {
	s := slim.New()
	s.GET("/", func(c slim.Context) error {
		return c.String(http.StatusOK, "Hello, World!")
	})
	log.Fatal(s.Start(":1324"))
}
