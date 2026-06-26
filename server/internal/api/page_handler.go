// Package api：Page 编辑器端点（page-editor pe-1）。
//
//	GET    /api/pages                      admin 列表
//	POST   /api/pages                      admin 创建
//	GET    /api/pages/:slug                admin 详情
//	PUT    /api/pages/:slug                admin 更新
//	DELETE /api/pages/:slug                admin 删除
//	POST   /api/pages/:slug/publish        admin 发布/取消发布
//	GET    /api/pages/:slug/leads          admin 表单提交列表
//	POST   /api/pages/:slug/form           公开 表单提交
//	GET    /p/:slug                        公开 SSR 渲染
package api

import (
	"encoding/json"
	"log/slog"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/pinconsole/internal/pages"
	"github.com/iannil/pinconsole/internal/storage"
)

// PageHandler 处理 /api/pages/* 端点。
type PageHandler struct {
	stores   *storage.Stores
	renderer *pages.Renderer
	logger   *slog.Logger
}

// NewPageHandler 创建 handler。
func NewPageHandler(stores *storage.Stores, renderer *pages.Renderer, logger *slog.Logger) *PageHandler {
	return &PageHandler{stores: stores, renderer: renderer, logger: logger}
}

// RegisterPublic 注册公开端点（SSR 渲染 + 表单提交）。
func (h *PageHandler) RegisterPublic(r gin.IRoutes) {
	r.GET("/p/:slug", h.renderPage)
	r.POST("/api/pages/:slug/form", h.submitForm)
}

// RegisterProtected 注册受保护端点（admin CRUD）。
func (h *PageHandler) RegisterProtected(r gin.IRoutes) {
	r.GET("/api/pages", h.listPages)
	r.POST("/api/pages", h.createPage)
	r.GET("/api/pages/:slug", h.getPage)
	r.PUT("/api/pages/:slug", h.updatePage)
	r.DELETE("/api/pages/:slug", h.deletePage)
	r.POST("/api/pages/:slug/publish", h.publishPage)
	r.GET("/api/pages/:slug/leads", h.listLeads)
}

// ── admin CRUD ─────────────────────────────────────────────

func (h *PageHandler) listPages(c *gin.Context) {
	tenantID := storage.DefaultTenantID
	pages, err := h.stores.PG.ListPages(c.Request.Context(), tenantID)
	if err != nil {
		h.logger.Warn("list pages failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}

	items := make([]gin.H, 0, len(pages))
	for _, p := range pages {
		items = append(items, gin.H{
			"id":         p.ID,
			"slug":       p.Slug,
			"title":      p.Title,
			"status":     p.Status,
			"updated_at": p.UpdatedAt,
		})
	}
	c.JSON(http.StatusOK, items)
}

func (h *PageHandler) createPage(c *gin.Context) {
	tenantID := storage.DefaultTenantID

	var body struct {
		Title string `json:"title"`
		Slug  string `json:"slug,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}
	if body.Title == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "title_required"})
		return
	}

	slug := body.Slug
	if slug == "" {
		slug = generateSlug(body.Title)
	}

	page, err := h.stores.PG.CreatePage(c.Request.Context(), tenantID, slug, body.Title)
	if err != nil {
		// 唯一冲突
		if strings.Contains(err.Error(), "duplicate key") {
			c.JSON(http.StatusConflict, gin.H{"error": "slug_exists"})
			return
		}
		h.logger.Warn("create page failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}

	c.JSON(http.StatusCreated, pageToJSON(page))
}

func (h *PageHandler) getPage(c *gin.Context) {
	tenantID := storage.DefaultTenantID
	slug := c.Param("slug")

	page, err := h.stores.PG.GetPageBySlug(c.Request.Context(), tenantID, slug)
	if err != nil {
		h.logger.Warn("get page failed", "slug", slug, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}
	if page == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	c.JSON(http.StatusOK, pageToJSON(page))
}

func (h *PageHandler) updatePage(c *gin.Context) {
	tenantID := storage.DefaultTenantID
	slug := c.Param("slug")

	var body struct {
		Title  *string          `json:"title,omitempty"`
		Schema *json.RawMessage `json:"schema,omitempty"`
		Status *string          `json:"status,omitempty"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	var schemaBytes []byte
	if body.Schema != nil {
		schemaBytes = []byte(*body.Schema)
	}

	updated, err := h.stores.PG.UpdatePage(c.Request.Context(), tenantID, slug, body.Title, schemaBytes, body.Status)
	if err != nil {
		h.logger.Warn("update page failed", "slug", slug, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}
	if updated == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	c.JSON(http.StatusOK, pageToJSON(updated))
}

func (h *PageHandler) deletePage(c *gin.Context) {
	tenantID := storage.DefaultTenantID
	slug := c.Param("slug")

	if err := h.stores.PG.DeletePage(c.Request.Context(), tenantID, slug); err != nil {
		h.logger.Warn("delete page failed", "slug", slug, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}

func (h *PageHandler) publishPage(c *gin.Context) {
	tenantID := storage.DefaultTenantID
	slug := c.Param("slug")

	var body struct {
		Status string `json:"status"`
	}
	if err := c.ShouldBindJSON(&body); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}
	if body.Status != "draft" && body.Status != "published" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_status"})
		return
	}

	updated, err := h.stores.PG.UpdatePage(c.Request.Context(), tenantID, slug, nil, nil, &body.Status)
	if err != nil {
		h.logger.Warn("publish page failed", "slug", slug, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}
	if updated == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	c.JSON(http.StatusOK, pageToJSON(updated))
}

// ── 表单提交 ───────────────────────────────────────────────

func (h *PageHandler) submitForm(c *gin.Context) {
	tenantID := storage.DefaultTenantID
	slug := c.Param("slug")

	// 显式解析表单
	_ = c.Request.ParseForm()

	// honeypot 检查
	if c.Request.Form.Get("_pin") != "" {
		// 机器人填写了隐藏字段，静默成功
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
		return
	}

	// 收集表单字段
	fields := make(map[string]string)
	for key, values := range c.Request.Form {
		if key == "_pin" {
			continue
		}
		if len(values) > 0 {
			fields[key] = values[0]
		}
	}

	fieldsJSON, err := json.Marshal(fields)
	if err != nil {
		h.logger.Warn("marshal form fields failed", "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}

	if _, err := h.stores.PG.InsertPageLead(c.Request.Context(), tenantID, slug, fieldsJSON); err != nil {
		h.logger.Warn("insert page lead failed", "slug", slug, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "ok"})
}

func (h *PageHandler) listLeads(c *gin.Context) {
	tenantID := storage.DefaultTenantID
	slug := c.Param("slug")

	leads, err := h.stores.PG.ListPageLeads(c.Request.Context(), tenantID, slug)
	if err != nil {
		h.logger.Warn("list page leads failed", "slug", slug, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}

	items := make([]gin.H, 0, len(leads))
	for _, l := range leads {
		var fields any
		if err := json.Unmarshal(l.Fields, &fields); err != nil {
			fields = nil
		}
		items = append(items, gin.H{
			"id":         l.ID,
			"page_slug":  l.PageSlug,
			"fields":     fields,
			"created_at": l.CreatedAt,
		})
	}
	c.JSON(http.StatusOK, items)
}

// ── SSR 渲染 ───────────────────────────────────────────────

func (h *PageHandler) renderPage(c *gin.Context) {
	tenantID := storage.DefaultTenantID
	slug := c.Param("slug")

	page, err := h.stores.PG.GetPageBySlug(c.Request.Context(), tenantID, slug)
	if err != nil {
		h.logger.Warn("get page for render failed", "slug", slug, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}
	if page == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "not_found"})
		return
	}

	var schema pages.PageSchema
	if err := json.Unmarshal(page.Schema, &schema); err != nil {
		h.logger.Warn("unmarshal page schema failed", "slug", slug, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "internal"})
		return
	}

	publicURL := ""
	scheme := c.Request.Header.Get("X-Forwarded-Proto")
	if scheme == "" {
		scheme = "http"
	}
	publicURL = scheme + "://" + c.Request.Host

	c.Header("Content-Type", "text/html; charset=utf-8")
	if err := h.renderer.Render(c.Writer, schema, pages.RenderOptions{
		Lang:      "zh",
		Slug:      slug,
		PublicURL: publicURL,
		SDKScript: "/sdk.js",
	}); err != nil {
		h.logger.Warn("render page failed", "slug", slug, "error", err)
		// 写入已经开始，无法再改 status code
	}
}

// ── 辅助函数 ───────────────────────────────────────────────

// pageToJSON 将 *storage.Page 转为 JSON 友好的 map。
func pageToJSON(p *storage.Page) gin.H {
	var schema any
	if err := json.Unmarshal(p.Schema, &schema); err != nil {
		schema = nil
	}
	return gin.H{
		"id":         p.ID,
		"tenant_id":  p.TenantID,
		"slug":       p.Slug,
		"title":      p.Title,
		"status":     p.Status,
		"schema":     schema,
		"created_at": p.CreatedAt,
		"updated_at": p.UpdatedAt,
	}
}

// generateSlug 将标题转为简单的 URL slug。
// 暂简单实现：小写、去空格、取英文/数字片段。
func generateSlug(title string) string {
	slug := strings.ToLower(title)
	slug = strings.TrimSpace(slug)
	// 简单替换：只保留 a-z、0-9、连字符
	var b strings.Builder
	for _, r := range slug {
		if (r >= 'a' && r <= 'z') || (r >= '0' && r <= '9') || r == '-' || r == '_' {
			b.WriteRune(r)
		} else if r == ' ' {
			b.WriteRune('-')
		}
	}
	result := strings.Trim(b.String(), "-")
	if result == "" {
		// fallback: 用 uuid 前 8 位
		return uuid.New().String()[:8]
	}
	// 限制长度
	if len(result) > 60 {
		result = result[:60]
	}
	return result
}
