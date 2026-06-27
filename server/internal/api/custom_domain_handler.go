// Package api:custom_domain handler（cd-1 自定义域名 REST API）。
package api

import (
	"errors"
	"log/slog"
	"net/http"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/iannil/pinconsole/internal/cert"
	"github.com/iannil/pinconsole/internal/storage"
)

// CustomDomainHandler 处理自定义域名 CRUD 操作。
type CustomDomainHandler struct {
	stores      *storage.Stores
	certManager *cert.Manager
	logger      *slog.Logger
}

// NewCustomDomainHandler 创建 CustomDomainHandler。
func NewCustomDomainHandler(stores *storage.Stores, certManager *cert.Manager, logger *slog.Logger) *CustomDomainHandler {
	return &CustomDomainHandler{
		stores:      stores,
		certManager: certManager,
		logger:      logger,
	}
}

// domainRequest 是 POST /api/custom-domains 的请求体。
type domainRequest struct {
	Domain string `json:"domain" binding:"required"`
}

// List 返回当前 tenant 的所有自定义域名。GET /api/custom-domains
func (h *CustomDomainHandler) List(c *gin.Context) {
	domains, err := h.stores.PG.ListCustomDomains(c.Request.Context(), storage.DefaultTenantID)
	if err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to list custom domains"})
		return
	}
	c.JSON(http.StatusOK, domains)
}

// Create 添加新域名并触发证书签发。POST /api/custom-domains
func (h *CustomDomainHandler) Create(c *gin.Context) {
	var req domainRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid request: domain required"})
		return
	}

	domain := strings.TrimSpace(strings.ToLower(req.Domain))
	if domain == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "domain cannot be empty"})
		return
	}

	created, err := h.stores.PG.CreateCustomDomain(c.Request.Context(), storage.DefaultTenantID, domain)
	if err != nil {
		if errors.Is(err, storage.ErrConflict) {
			c.JSON(http.StatusConflict, gin.H{"error": "domain already exists"})
			return
		}
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to create custom domain"})
		return
	}

	// 触发异步证书签发
	if h.certManager != nil {
		go h.certManager.AddDomain(c.Request.Context(), domain, created.ID)
	}

	c.JSON(http.StatusCreated, created)
}

// Delete 删除自定义域名。DELETE /api/custom-domains/:id
func (h *CustomDomainHandler) Delete(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
		return
	}

	// 先查域名，用于从 certmagic 移除
	if h.certManager != nil {
		domains, _ := h.stores.PG.ListCustomDomains(c.Request.Context(), storage.DefaultTenantID)
		for _, d := range domains {
			if d.ID == id {
				h.certManager.RemoveDomain(d.Domain)
				break
			}
		}
	}

	if err := h.stores.PG.DeleteCustomDomain(c.Request.Context(), storage.DefaultTenantID, id); err != nil {
		_ = c.Error(err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "failed to delete custom domain"})
		return
	}

	c.JSON(http.StatusOK, gin.H{"status": "deleted"})
}
