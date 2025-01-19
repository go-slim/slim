package serde

import (
	"encoding/json"
	"io"
)

// JSONSerializer 为 JSON 实现序列化接口
type JSONSerializer struct{}

// Serialize 序列化数据到 w 接口
func (JSONSerializer) Serialize(w io.Writer, v any, indent string) error {
	enc := json.NewEncoder(w)
	if indent != "" {
		enc.SetIndent("", indent)
	}
	return enc.Encode(v)
}

// Deserialize 反序列化数据并绑定到 v 上
func (JSONSerializer) Deserialize(r io.Reader, v any) error {
	return json.NewDecoder(r).Decode(v)
}
