//go:build release

// 1k fail-secure：release build 下的认证绕过实现 —— 永不绕过。
// 此函数在 release 二进制中函数体恒返回 false，
// 即使运维误配 SERVER_ENV=dev，AuthMiddleware 仍走完整 session 校验。

package api

import "github.com/gin-gonic/gin"

// tryDevBypass 在 release build 下永不绕过。
func tryDevBypass(c *gin.Context) bool {
	_ = c // 显式忽略，避免 unused 警告
	return false
}
