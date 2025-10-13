package main

import (
	"log"
	"net/http"

	"go-slim.dev/slim"
)

func main() {
	s := slim.New()
	n := slim.NewNegotiator(10, nil)

	type payload struct {
		Message string `json:"message" xml:"message"`
	}

	s.GET("/data", func(c slim.Context) error {
		accept := c.Request().Header.Get(slim.HeaderAccept)
		mt := n.Accepts(accept, slim.MIMEApplicationJSON, slim.MIMEApplicationXML)
		p := payload{Message: "hello"}
		switch mt {
		case slim.MIMEApplicationXML, slim.MIMETextXML:
			return c.XML(http.StatusOK, p)
		default:
			return c.JSON(http.StatusOK, p)
		}
	})

	log.Fatal(s.Start(":1328"))
}
