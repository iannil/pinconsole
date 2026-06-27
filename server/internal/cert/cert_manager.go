// Package cert 封装 certmagic 自动证书管理（cd-1 自定义域名）。
package cert

import (
	"context"
	"crypto/tls"
	"log/slog"
	"net/http"
	"sync"

	"github.com/caddyserver/certmagic"
	"github.com/iannil/pinconsole/internal/storage"
)

// Manager 管理 certmagic 实例与域名列表。
type Manager struct {
	cfg     *certmagic.Config
	storage *storage.Postgres
	logger  *slog.Logger
	mu      sync.Mutex
	domains map[string]int64 // domain → custom_domain.id
}

// New 创建 certmagic Manager。
// 启动时自动加载数据库中所有 active 域名。
func New(ctx context.Context, postgres *storage.Postgres, logger *slog.Logger, acmeEmail, dataDir string, staging bool) (*Manager, error) {
	// 配置全局 ACME 设置
	certmagic.DefaultACME.Email = acmeEmail
	certmagic.DefaultACME.Agreed = true
	if staging {
		certmagic.DefaultACME.CA = certmagic.LetsEncryptStagingCA
	}
	certmagic.Default.Storage = &certmagic.FileStorage{Path: dataDir}

	cfg := certmagic.NewDefault()

	m := &Manager{
		cfg:     cfg,
		storage: postgres,
		logger:  logger,
		domains: make(map[string]int64),
	}

	// 启动时加载已激活域名
	if err := m.loadActiveDomains(ctx); err != nil {
		return nil, err
	}

	return m, nil
}

// loadActiveDomains 从 DB 加载所有 active 域名并注册到 certmagic。
func (m *Manager) loadActiveDomains(ctx context.Context) error {
	active, err := m.storage.ListActiveCustomDomains(ctx)
	if err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	domainNames := make([]string, 0, len(active))
	for _, d := range active {
		m.domains[d.Domain] = d.ID
		domainNames = append(domainNames, d.Domain)
	}

	if len(domainNames) > 0 {
		m.logger.Info("loading existing custom domains for cert management",
			"count", len(domainNames),
			"domains", domainNames,
		)
		// 预签发证书
		for _, domain := range domainNames {
			if err := m.obtainCert(ctx, domain); err != nil {
				m.logger.Warn("initial cert obtain failed (will retry)", "domain", domain, "error", err)
			}
		}
	}
	return nil
}

// AddDomain 添加一个新域名到 certmagic 管理并签发证书。
// 异步签发：成功 → "active"，失败 → "failed"。
func (m *Manager) AddDomain(ctx context.Context, domain string, domainID int64) {
	m.mu.Lock()
	m.domains[domain] = domainID
	m.mu.Unlock()

	m.logger.Info("obtaining cert for custom domain", "domain", domain)
	if err := m.obtainCert(ctx, domain); err != nil {
		m.logger.Error("cert obtain failed", "domain", domain, "error", err)
		if err := m.storage.UpdateCustomDomainStatus(ctx, domainID, "failed", err.Error()); err != nil {
			m.logger.Error("failed to update cert status", "domain", domain, "error", err)
		}
		return
	}

	if err := m.storage.UpdateCustomDomainStatus(ctx, domainID, "active", ""); err != nil {
		m.logger.Error("failed to update cert status to active", "domain", domain, "error", err)
	}
}

// obtainCert 为单个域名签发证书（异步，已存在时不会重复签发）。
func (m *Manager) obtainCert(ctx context.Context, domain string) error {
	return m.cfg.ObtainCertAsync(ctx, domain)
}

// RemoveDomain 从管理列表中移除域名（不做 revoked）。
func (m *Manager) RemoveDomain(domain string) {
	m.mu.Lock()
	delete(m.domains, domain)
	m.mu.Unlock()
}

// TLSConfig 返回配置好 certmagic GetCertificate 的 *tls.Config。
func (m *Manager) TLSConfig() *tls.Config {
	return m.cfg.TLSConfig()
}

// HTTPChallengeHandler wraps an http.Handler to respond to ACME HTTP-01 challenges.
func (m *Manager) HTTPChallengeHandler(next http.Handler) http.Handler {
	return certmagic.DefaultACME.HTTPChallengeHandler(next)
}

// HasDomain 检查域名是否在管理列表中。
func (m *Manager) HasDomain(domain string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	_, ok := m.domains[domain]
	return ok
}

// ManagedDomains 返回当前管理的域名列表。
func (m *Manager) ManagedDomains() []string {
	m.mu.Lock()
	defer m.mu.Unlock()
	domains := make([]string, 0, len(m.domains))
	for d := range m.domains {
		domains = append(domains, d)
	}
	return domains
}
