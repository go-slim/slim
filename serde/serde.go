package serde

import "io"

// Serializer 序列化 json 或 xml
type Serializer interface {
	// Serialize 序列化数据
	Serialize(w io.Writer, v any, indent string) error
	// Deserialize 反序列化
	Deserialize(r io.Reader, v any) error
}
