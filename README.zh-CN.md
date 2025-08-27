# Slim

一款快速、可组合的 Go HTTP 框架。提供简洁的 API、强大的路由与务实的默认设置。

官网：https://go-slim.dev

## 特性

- 极简内核，支持可插拔中间件
- 基于前缀树的高性能路由，支持路径参数与通配符
- 统一错误处理机制，并支持按路由收集器与路由器级别覆盖
- 内容协商（类型、编码、字符集、语言）
- JSON / XML / JSONP 渲染与自定义序列化器
- 基于 fs.FS 的静态文件与目录服务
- 内置中间件：日志、异常恢复、CORS、限流
- 虚拟主机（vhost）支持

## 安装

```bash
go get go-slim.dev/slim
```

## 快速开始

```go
package main

import (
    "net/http"
    "go-slim.dev/slim"
)

func main() {
    s := slim.New()

    s.GET("/hello", func(c slim.Context) error {
        return c.String(http.StatusOK, "Hello, Slim!")
    })

    s.Start(":8080")
}
```

## 示例

在 `examples/` 目录中提供了可运行示例：
- `examples/static` —— 静态文件服务
- `examples/cors` —— CORS 中间件
- `examples/logger-recovery` —— 日志与异常恢复
- `examples/nego` —— 内容协商
- `examples/rate-limiter` —— 限流

运行示例：
```bash
cd examples/static
go run .
```

## 文档

完整文档、指南与 API 参考： https://go-slim.dev

## 测试

```bash
go test ./...
```

## 许可证

Apache-2.0，详见 `LICENSE`。
