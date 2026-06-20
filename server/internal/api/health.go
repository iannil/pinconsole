package api

import (
	"context"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/iannil/pinconsole/internal/logging"
	"github.com/iannil/pinconsole/internal/storage"
)

// healthLive 仅检查进程存活（无依赖检查）。
func healthLive(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":   "alive",
		"trace_id": logging.TraceID(c.Request.Context()),
	})
}

// healthReady 检查所有依赖（PG / Redis / MinIO）。
func healthReady(stores *storage.Stores) gin.HandlerFunc {
	return func(c *gin.Context) {
		ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
		defer cancel()

		components := map[string]string{}
		allOK := true

		if err := stores.PG.Ping(ctx); err != nil {
			components["postgres"] = "fail: " + err.Error()
			allOK = false
		} else {
			components["postgres"] = "ok"
		}

		if err := stores.Redis.Ping(ctx); err != nil {
			components["redis"] = "fail: " + err.Error()
			allOK = false
		} else {
			components["redis"] = "ok"
		}

		if err := stores.MinIO.Ping(ctx); err != nil {
			components["minio"] = "fail: " + err.Error()
			allOK = false
		} else {
			components["minio"] = "ok"
		}

		status := http.StatusOK
		if !allOK {
			status = http.StatusServiceUnavailable
		}

		c.JSON(status, gin.H{
			"status":     ternary(allOK, "ready", "not_ready"),
			"components": components,
			"trace_id":   logging.TraceID(c.Request.Context()),
		})
	}
}

func ternary(b bool, t, f string) string {
	if b {
		return t
	}
	return f
}
