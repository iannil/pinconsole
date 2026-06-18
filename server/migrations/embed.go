// Package migrations 嵌入 SQL 迁移文件供 server 启动时自动应用（1k P0-14）。
//
// 文件命名约定：000001_init.up.sql、000001_init.down.sql。
// version 为文件名前 6 位数字（前导零填充）。
package migrations

import "embed"

// Files 嵌入所有 .up.sql 与 .down.sql 文件。
//
//go:embed *.up.sql *.down.sql
var Files embed.FS
