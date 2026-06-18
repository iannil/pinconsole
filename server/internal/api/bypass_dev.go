//go:build !release

// 1k fail-secure：dev build 下的认证绕过实现，便于 e2e 测试。
// release build 下此文件不参与编译（见 bypass_release.go），
// 因此 release 二进制结构上无法走 dev bypass。

package api

import (
	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

// tryDevBypass 在 dev build 下注入 mock user_id 并返回 true。
// 调用方据此跳过 session 校验。
func tryDevBypass(c *gin.Context) bool {
	c.Set("user_id", uuid.Nil)
	c.Set("dev_mode", true)
	c.Next()
	return true
}
