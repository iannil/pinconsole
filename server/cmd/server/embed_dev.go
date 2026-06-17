//go:build !release

// Package main embed：dev 模式下不嵌入前端产物，由 Vite dev server (5173) 与
// SDK playground (5174) 独立服务。embeddedAssets 为空 embed.FS。
package main

import "embed"

var embeddedAssets embed.FS // dev 模式空

func isRelease() bool { return false }
