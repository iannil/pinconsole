// Package api：隐私合规端点(1l-privacy-gdpr)。
//
//	GET  /api/privacy/consent?fingerprint=xxx  公开,查 consent 状态
//	POST /api/privacy/consent                  公开,写 consent
//	DELETE /api/privacy/visitor/:fingerprint   admin 认证,级联删除访客(GDPR Art.17)
package api

import (
	"context"
	"log/slog"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/iannil/marketing-monitor/internal/storage"
	"github.com/minio/minio-go/v7"
)

// minioRemoveObjectOpts 是 minio RemoveObject 的默认选项(v1 用零值)。
var minioRemoveObjectOpts = minio.RemoveObjectOptions{}

// PrivacyHandler 处理 /api/privacy/* 端点。
type PrivacyHandler struct {
	stores *storage.Stores
	logger *slog.Logger
}

func NewPrivacyHandler(stores *storage.Stores, logger *slog.Logger) *PrivacyHandler {
	return &PrivacyHandler{stores: stores, logger: logger}
}

// RegisterPublic 注册公开 consent 端点(访客 SDK 用,无 cookie)。
func (h *PrivacyHandler) RegisterPublic(r gin.IRoutes) {
	r.GET("/api/privacy/consent", h.getConsent)
	r.POST("/api/privacy/consent", h.postConsent)
}

// DeleteVisitor 是 protected DELETE /api/privacy/visitor/:fingerprint 的 handler。
// 由 router.go 显式挂载到 protected group。
func (h *PrivacyHandler) DeleteVisitor(c *gin.Context) {
	h.deleteVisitor(c)
}

// consentResponse 是 GET /api/privacy/consent 的响应。
type consentResponse struct {
	Fingerprint string `json:"fingerprint"`
	Scope       string `json:"scope"`
	Version     string `json:"version"`
	Accepted    bool   `json:"accepted"`
	Found       bool   `json:"found"` // false = 未记录,SDK 应按 consentMode 默认行为
}

// consentVersion 当前同意书版本(条款变更时升级,旧版自动失效)。
const consentVersion = "v1"
const consentScope = "all"

// getConsent 查询 fingerprint 当前同意状态(公开)。
func (h *PrivacyHandler) getConsent(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	fp := c.Query("fingerprint")
	if fp == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_fingerprint"})
		return
	}

	consent, found, err := h.stores.PG.GetLatestConsent(ctx, storage.DefaultTenantID, fp, consentScope, consentVersion)
	if err != nil {
		h.logger.ErrorContext(ctx, "get consent failed", "fingerprint", fp, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}
	if !found {
		c.JSON(http.StatusOK, consentResponse{
			Fingerprint: fp,
			Scope:       consentScope,
			Version:     consentVersion,
			Accepted:    false,
			Found:       false,
		})
		return
	}
	c.JSON(http.StatusOK, consentResponse{
		Fingerprint: consent.Fingerprint,
		Scope:       consent.Scope,
		Version:     consent.Version,
		Accepted:    consent.Accepted,
		Found:       true,
	})
}

// postConsentRequest 是 POST /api/privacy/consent 的请求体。
type postConsentRequest struct {
	Fingerprint string `json:"fingerprint" binding:"required"`
	Accepted    bool   `json:"accepted"`
}

// postConsent 写入/更新同意状态(公开,访客 SDK 调用)。
func (h *PrivacyHandler) postConsent(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 3*time.Second)
	defer cancel()

	var req postConsentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "invalid_json"})
		return
	}

	consent, err := h.stores.PG.UpsertConsent(ctx, storage.DefaultTenantID, req.Fingerprint, consentScope, consentVersion, req.Accepted)
	if err != nil {
		h.logger.ErrorContext(ctx, "upsert consent failed", "fingerprint", req.Fingerprint, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "db_error"})
		return
	}

	h.logger.InfoContext(ctx, "consent recorded",
		"fingerprint", req.Fingerprint, "accepted", req.Accepted, "version", consentVersion,
	)
	c.JSON(http.StatusOK, consentResponse{
		Fingerprint: consent.Fingerprint,
		Scope:       consent.Scope,
		Version:     consent.Version,
		Accepted:    consent.Accepted,
		Found:       true,
	})
}

// deleteVisitor 级联删除访客(GDPR Art.17 被遗忘权,admin only)。
//
// 副作用:
//   - PG: visitor_consents / chat_messages / co_browsing_commands / event_blobs / sessions / visitors
//   - MinIO: 该访客的所有 blob 对象(批量 RemoveObject)
//   - Redis: presence/claim/flagged/stream/snapshot keys(best-effort)
//
// 顺序:
//  1. 查 visitor_id + 关联 sessions
//  2. 列 MinIO object keys(删 PG 前)
//  3. PG 级联删除
//  4. MinIO 对象删除
//  5. Redis keys 清理
func (h *PrivacyHandler) deleteVisitor(c *gin.Context) {
	ctx, cancel := context.WithTimeout(c.Request.Context(), 30*time.Second) // 级联删除可能慢
	defer cancel()

	fp := c.Param("fingerprint")
	if fp == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "missing_fingerprint"})
		return
	}

	// 1ac T0-1l-5:GDPR Art.17 删除必须 admin only。
	// 1ac 测试发现:此前代码无 role 校验,任意认证用户(operator 含)可删访客数据。
	callerAny, exists := c.Get("user_id")
	if !exists {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "not_authenticated"})
		return
	}
	callerUID, _ := callerAny.(uuid.UUID)
	caller, err := h.stores.PG.GetUserByID(ctx, callerUID)
	if err != nil || caller == nil {
		c.JSON(http.StatusUnauthorized, gin.H{"error": "user_not_found"})
		return
	}
	if caller.Role != "admin" {
		c.JSON(http.StatusForbidden, gin.H{"error": "admin_required"})
		return
	}

	// 1. 查 visitor + sessions
	visitor, err := h.stores.PG.GetVisitorByFingerprint(ctx, storage.DefaultTenantID, fp)
	if err != nil {
		h.logger.ErrorContext(ctx, "lookup visitor failed", "fingerprint", fp, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "lookup_failed"})
		return
	}
	if visitor == nil {
		// visitor 不存在,幂等返回 ok
		c.JSON(http.StatusOK, gin.H{"ok": true, "fingerprint": fp, "deleted_sessions": 0, "note": "visitor_not_found"})
		return
	}

	// 列出关联 sessions
	rows, err := h.stores.PG.Pool.Query(ctx, `SELECT id FROM sessions WHERE visitor_id = $1`, visitor.ID)
	if err != nil {
		h.logger.ErrorContext(ctx, "list sessions failed", "fingerprint", fp, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "list_sessions_failed"})
		return
	}
	var sessionIDs []uuid.UUID
	for rows.Next() {
		var sid uuid.UUID
		if err := rows.Scan(&sid); err != nil {
			rows.Close()
			h.logger.ErrorContext(ctx, "scan session id failed", "error", err)
			c.JSON(http.StatusInternalServerError, gin.H{"error": "scan_failed"})
			return
		}
		sessionIDs = append(sessionIDs, sid)
	}
	rows.Close()

	// 2. 列 MinIO keys(必须在 PG 删除前)
	var minioKeys []string
	if len(sessionIDs) > 0 {
		minioKeys, err = h.stores.PG.ListEventBlobKeysBySessions(ctx, sessionIDs)
		if err != nil {
			h.logger.WarnContext(ctx, "list minio keys pre-delete failed",
				"fingerprint", fp, "error", err, "note", "MinIO cleanup will rely on GC sweep")
		}
	}

	// 3. PG 级联删除
	_, err = h.stores.PG.DeleteVisitorByFingerprint(ctx, storage.DefaultTenantID, fp)
	if err != nil {
		h.logger.ErrorContext(ctx, "delete visitor cascade failed", "fingerprint", fp, "error", err)
		c.JSON(http.StatusInternalServerError, gin.H{"error": "delete_failed", "detail": err.Error()})
		return
	}

	// 4. MinIO 对象删除(best-effort,失败不阻塞 GDPR 删除)
	minioDeleted := 0
	for _, key := range minioKeys {
		if err := h.stores.MinIO.Client.RemoveObject(ctx, h.stores.MinIO.Bucket, key, minioRemoveObjectOpts); err != nil {
			h.logger.WarnContext(ctx, "minio RemoveObject failed",
				"fingerprint", fp, "key", key, "error", err)
			continue
		}
		minioDeleted++
	}

	// 5. Redis keys 清理(best-effort)
	for _, sid := range sessionIDs {
		_ = h.stores.Redis.Del(ctx, claimKey(sid))
	}

	callerLogUID, _ := c.Get("user_id")
	h.logger.InfoContext(ctx, "visitor erased (GDPR Art.17)",
		"fingerprint", fp,
		"caller_user_id", callerLogUID,
		"deleted_sessions", len(sessionIDs),
		"deleted_minio_objects", minioDeleted,
	)
	c.JSON(http.StatusOK, gin.H{
		"ok":                    true,
		"fingerprint":           fp,
		"deleted_sessions":      len(sessionIDs),
		"deleted_minio_objects": minioDeleted,
	})
}
