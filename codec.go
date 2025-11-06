package slim

import (
	"encoding/json"
	"encoding/xml"
	"io"
)

// Codec defines the encoding and decoding interface for data
type Codec interface {
	// Encode encodes data into byte stream
	Encode(w io.Writer, v any, indent string) error
	// Decode decodes data from byte stream
	Decode(r io.Reader, v any) error
}

// JSONCodec implements encoding and decoding interface for JSON
type JSONCodec struct{}

// Encode serializes data to w interface
func (JSONCodec) Encode(w io.Writer, v any, indent string) error {
	enc := json.NewEncoder(w)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	return enc.Encode(v)
}

// Decode deserializes data and binds it to v
func (JSONCodec) Decode(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

// XMLCodec implements encoding and decoding interface for XML
type XMLCodec struct{}

func (XMLCodec) Encode(w io.Writer, v any, indent string) error {
	enc := xml.NewEncoder(w)
	if indent != "" {
		enc.Indent("", indent)
	}
	return enc.Encode(v)
}

func (XMLCodec) Decode(r io.Reader, v any) error {
	return xml.NewDecoder(r).Decode(v)
}
