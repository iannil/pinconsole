//go:build release

// Package main embed：release 模式下嵌入 admin/dist + visitor-sdk/dist + landing/ 到二进制。
// 构建步骤（详见 Makefile build-server）：
//   1. pnpm build:admin → admin/dist
//   2. pnpm build:sdk → visitor-sdk/dist
//   3. 拷贝到 server/cmd/server/embedded/{admin,sdk,landing}
//   4. go build -tags release
package main

import "embed"

// embeddedAssets 含 admin/、sdk/、landing/ 三个子树。
//
//go:embed all:embedded
var embeddedAssets embed.FS

// isRelease 报告当前是否为 release 构建。
func isRelease() bool { return true }
