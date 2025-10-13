package slim

import (
	"encoding/json"
	"encoding/xml"
	"io"
)

// Codec 定义了数据的编码和解码接口
type Codec interface {
	// Encode 将数据编码为字节流
	Encode(w io.Writer, v any, indent string) error
	// Decode 从字节流解码数据
	Decode(r io.Reader, v any) error
}

// JSONCodec 为 JSON 实现序列化接口
type JSONCodec struct{}

// Encode 序列化数据到 w 接口
func (JSONCodec) Encode(w io.Writer, v any, indent string) error {
	enc := json.NewEncoder(w)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	return enc.Encode(v)
}

// Decode 反序列化数据并绑定到 v 上
func (JSONCodec) Decode(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}

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
