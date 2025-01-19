package serde

import (
	"encoding/xml"
	"io"
)

type XMLSerializer struct{}

func (XMLSerializer) Serialize(w io.Writer, v any, indent string) error {
	enc := xml.NewEncoder(w)
	if indent != "" {
		enc.Indent("", indent)
	}
	return enc.Encode(v)
}

func (XMLSerializer) Deserialize(r io.Reader, v any) error {
	return xml.NewDecoder(r).Decode(v)
}
