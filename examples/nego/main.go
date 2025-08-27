package main

import (
	"net/http"

	"go-slim.dev/slim"
	"go-slim.dev/slim/nego"
)

func main() {
	s := slim.New()
	n := nego.New(10, nil)

	type payload struct {
		Message string `json:"message" xml:"message"`
	}

	s.GET("/data", func(c slim.Context) error {
		accept := c.Request().Header.Get(nego.HeaderAccept)
		mt := n.Accepts(accept, nego.MIMEApplicationJSON, nego.MIMEApplicationXML)
		p := payload{Message: "hello"}
		switch mt {
		case nego.MIMEApplicationXML, nego.MIMETextXML:
			return c.XML(http.StatusOK, p)
		default:
			return c.JSON(http.StatusOK, p)
		}
	})

	s.Logger.Fatal(s.Start(":1328"))
}
