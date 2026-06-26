// Package api：Widget 配置端点（page-editor pe-1）。
//
//	GET  /api/widget-config          公开,SDK 拉取全部配置
//	PUT  /api/widget-config/:type   admin 保护,更新某类 widget 配置
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/iannil/pinconsole/internal/storage"
)

// jsonRaw is a raw JSON string that implements json.Unmarshaler for gin binding.
type jsonRaw string

func (r *jsonRaw) UnmarshalJSON(b []byte) error {
	*r = jsonRaw(b)
	return nil
}

// jsonUnmarshal is a shorthand for json.Unmarshal into a pointer.
func jsonUnmarshal(data []byte, v any) error {
	return json.Unmarshal(data, v)
}

// WidgetConfigHandler 处理 /api/widget-config/* 端点。
type WidgetConfigHandler struct {
	stores *storage.Stores
	logger *slog.Logger
}

// NewWidgetConfigHandler 创建 handler。
func NewWidgetConfigHandler(stores *storage.Stores, logger *slog.Logger) *WidgetConfigHandler {
	return &WidgetConfigHandler{stores: stores, logger: logger}
}

// RegisterPublic 注册公开端点（SDK 拉取，无 cookie 要求）。
func (h *WidgetConfigHandler) RegisterPublic(r gin.IRoutes) {
	r.GET("/api/widget-config", h.getConfigs)
}

// RegisterProtected 注册受保护端点（admin 编辑，AuthMiddleware 保护）。
func (h *WidgetConfigHandler) RegisterProtected(r gin.IRoutes) {
	r.PUT("/api/widget-config/:type", h.upsertConfig)
}

// getConfigs 返回 default tenant 的所有 widget 配置（简化：v1 单租户直接用 uuid.Nil）。
func (h *WidgetConfigHandler) getConfigs(c *gin.Context) {
	tenantID := storage.DefaultTenantID
	configs, err := h.stores.PG.ListWidgetConfigs(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Warn("list widget configs failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}

	// 构建 JSON 响应（与 proto WidgetConfigMap 结构对应）
	result := make(map[string]any, len(configs))
	for _, cfg := range configs {
		var parsed any
		if err := jsonUnmarshal(cfg.Config, &parsed); err != nil {
			h.logger.Warn("parse widget config failed", "widget_type", cfg.WidgetType, "error", err)
			continue
		}
		result[cfg.WidgetType] = parsed
	}

	c.JSON(http.StatusOK, gin.H{
		"tenant_id": tenantID,
		"configs":   result,
	})
}

// upsertConfig 更新指定 type 的 widget 配置（admin only）。
func (h *WidgetConfigHandler) upsertConfig(c *gin.Context) {
	tenantID := storage.DefaultTenantID
	widgetType := c.Param("type")

	var body struct {
		Config jsonRaw `json:"config"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	updated, err := h.stores.PG.UpsertWidgetConfig(c.Request.Context(), tenantID, widgetType, []byte(body.Config))
	if err != nil {
		h.logger.Warn("upsert widget config failed", "widget_type", widgetType, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}

	var parsed any
	if err := jsonUnmarshal(updated.Config, &parsed); err != nil {
		parsed = nil
	}

	c.JSON(http.StatusOK, gin.H{
		"widget_type": updated.WidgetType,
		"config":      parsed,
	})
}
