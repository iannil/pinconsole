//go:build !release

// 1k 测试:dev build 下 isReleaseBuild 应为 false。
package api

// isReleaseBuild 在 dev build 下为 false (见 bypass_release_test.go 的 release 版本)。
const isReleaseBuild = false
