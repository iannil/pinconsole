#!/usr/bin/env bash
# pinconsole marketing — 一键部署到 Cloudflare Pages 生产环境
#
# 用法：./scripts/deploy.sh [check|db|secrets|deploy|--help]
#
# 完整流程（默认 deploy）：
#   1. 预检：node/pnpm/wrangler 登录/wrangler.toml database_id/D1 远程可达
#   2. D1 schema：远程 leads 表存在则跳过，不存在则应用 migration
#   3. secrets：仅提示（不强制），列出已配置项 + 给出设置命令
#   4. build：pnpm build（产出 dist/）
#   5. deploy：wrangler pages deploy ./dist
#   6. 收尾：打印 dashboard URL / leads 查询命令 / 自定义域名提示
set -euo pipefail

# ===== 路径与配置 =====
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
MARKETING_DIR="$(cd "$SCRIPT_DIR/.." && pwd)"
cd "$MARKETING_DIR"

PROJECT_NAME="pinconsole"
D1_DB_NAME="pinconsole-leads"
WRANGLER_TOML="wrangler.toml"
MIGRATIONS_DIR="migrations"
LEADS_MIGRATION="0001-create-leads.sql"

# Secrets that affect /api/leads behavior in prod.
REQUIRED_SECRETS=("RESEND_API_KEY" "LEAD_NOTIFY_EMAIL" "TURNSTILE_SECRET")

# ===== 颜色 =====
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*" >&2; }
step()  { echo -e "\n${CYAN}${BOLD}▶ $*${NC}"; }

# Run wrangler through pnpm (respects local install + lockfile).
wr() { pnpm exec wrangler "$@"; }

# ===== 步骤 =====

cmd_check() {
    step "预检（preflight）"

    # 1) node + pnpm
    command -v node >/dev/null 2>&1 || { error "缺少 node"; exit 1; }
    command -v pnpm >/dev/null 2>&1 || { error "缺少 pnpm"; exit 1; }
    info "node $(node --version)   pnpm $(pnpm --version)"

    # 2) node_modules
    if [ ! -d node_modules ]; then
        warn "node_modules 缺失，运行 pnpm install --frozen-lockfile..."
        pnpm install --frozen-lockfile
    fi
    info "deps OK"

    # 3) wrangler.toml 存在
    [ -f "$WRANGLER_TOML" ] || { error "缺少 $WRANGLER_TOML（应在 $MARKETING_DIR 下）"; exit 1; }

    # 4) Cloudflare 登录
    step "验证 Cloudflare 登录"
    local who
    if ! who=$(wr whoami 2>&1); then
        error "未登录 Cloudflare。请先运行: pnpm exec wrangler login"
        echo "$who" | grep -vE '^\s*$' | head -10 | sed 's/^/    /'
        exit 1
    fi
    # 提取账号 / 邮箱行
    local account_line
    account_line=$(echo "$who" | grep -iE 'logged in|account|email|┌|│' | head -8 | sed 's/^/    /') || true
    if [ -n "$account_line" ]; then
        echo "$account_line"
    else
        info "$(echo "$who" | grep -vE '^\s*$' | head -3)"
    fi

    # 5) database_id 已配置
    step "验证 wrangler.toml database_id"
    if ! grep -qE '^database_id[[:space:]]*=[[:space:]]*"[^"]+"' "$WRANGLER_TOML" \
       || grep -qE '^database_id[[:space:]]*=[[:space:]]*""' "$WRANGLER_TOML"; then
        error "wrangler.toml 缺少有效的 database_id"
        echo "    首次设置：pnpm exec wrangler d1 create $D1_DB_NAME"
        echo "    然后把返回的 database_id 填入 $WRANGLER_TOML"
        exit 1
    fi
    local dbid
    dbid=$(grep -E '^database_id' "$WRANGLER_TOML" | head -1 | sed -E 's/.*"([^"]+)".*/\1/')
    info "database_id = ${dbid:0:8}…${dbid: -4}（已配置）"

    # 6) D1 远程可达
    step "验证 D1 远程连接"
    if wr d1 execute "$D1_DB_NAME" --remote --command "SELECT 1" >/dev/null 2>&1; then
        info "D1 远程连接 OK"
    else
        error "D1 远程连接失败。可能原因：database_id 与当前账号不匹配 / 网络问题"
        echo "    手动验证：pnpm exec wrangler d1 execute $D1_DB_NAME --remote --command 'SELECT 1'"
        exit 1
    fi

    info "✅ 预检通过"
}

cmd_db() {
    cmd_check

    step "检查远程 leads 表"
    local check_output
    if check_output=$(wr d1 execute "$D1_DB_NAME" --remote \
        --command "SELECT name FROM sqlite_master WHERE type='table' AND name='leads'" 2>&1); then

        if echo "$check_output" | grep -q 'leads'; then
            info "leads 表已存在，跳过迁移（migration 幂等，重跑也无副作用）"
            return 0
        fi
    else
        warn "leads 表存在性查询失败，将尝试直接应用 migration（CREATE TABLE IF NOT EXISTS 是安全的）"
    fi

    info "应用 $LEADS_MIGRATION 到远程 D1 ..."
    wr d1 execute "$D1_DB_NAME" --remote --file "$MIGRATIONS_DIR/$LEADS_MIGRATION"
    info "✅ schema 已应用"
}

cmd_secrets() {
    cmd_check

    step "Pages secrets 状态"
    local list_output
    if list_output=$(wr pages secret list --project-name "$PROJECT_NAME" 2>&1); then
        echo "$list_output" | grep -vE '^\s*$' | sed 's/^/    /'
    else
        warn "无法列出 secrets（可能首次部署，project 尚未创建 — deploy 时会自动创建）"
        list_output=""
    fi

    echo ""
    echo "  /api/leads 完整工作需要的 secrets："
    echo "    RESEND_API_KEY    — Resend.com API key（外发邮件通知）"
    echo "    LEAD_NOTIFY_EMAIL — 接收新 lead 通知的邮箱"
    echo "    TURNSTILE_SECRET  — Cloudflare Turnstile server secret（与 TURNSTILE_SITE_KEY 配对）"
    echo ""

    for secret in "${REQUIRED_SECRETS[@]}"; do
        if echo "$list_output" | grep -q "\"$secret\"\|'^$secret'\|$secret"; then
            info "$secret ✓ 已配置"
            continue
        fi
        read -rp "$(echo -e "${YELLOW}设置 $secret ？[y/N]${NC} ")" ans
        if [[ "${ans:-}" =~ ^[Yy]$ ]]; then
            wr pages secret put "$secret" --project-name "$PROJECT_NAME"
            info "$secret ✓ 已写入"
        else
            warn "跳过 $secret（/api/leads 相关功能将不工作）"
        fi
    done

    info "✅ secrets 检查完成"
}

cmd_build() {
    step "构建静态站点（astro build）"
    pnpm build
    info "✅ build 完成：$(ls -1 dist/ | head -5 | tr '\n' ' ')"
}

cmd_deploy() {
    cmd_check
    cmd_db
    # secrets 不强制（部署本身不依赖），只提示
    step "secrets 提示"
    echo "  若未配置 RESEND_API_KEY/LEAD_NOTIFY_EMAIL/TURNSTILE_SECRET，/api/leads 仍会入库但不会发邮件。"
    echo "  事后配置：./scripts/deploy.sh secrets"
    cmd_build

    step "部署到 Cloudflare Pages（project: $PROJECT_NAME）"
    local deploy_out
    deploy_out=$(wr pages deploy ./dist --project-name "$PROJECT_NAME" 2>&1 | tee /dev/stderr)

    step "✅ 部署完成"
    # 从输出里抓 deployment URL
    local url
    url=$(echo "$deploy_out" | grep -oE 'https://[a-z0-9.-]*pages\.dev[a-zA-Z0-9/_-]*' | head -1) || true
    if [ -n "$url" ]; then
        info "Deployment URL: $url"
    fi
    info "Dashboard:    https://dash.cloudflare.com/?to=/:account/pages/views/$PROJECT_NAME"
    echo ""
    echo "  下一步："
    echo "    1) 自定义域名（pinconsole.com）："
    echo "       Pages dashboard → $PROJECT_NAME → Custom domains → Set up a custom domain"
    echo "       （首次需在 Cloudflare DNS 加 CNAME）"
    echo "    2) 查看新 leads（远程 D1）："
    echo "       pnpm exec wrangler d1 execute $D1_DB_NAME --remote --command \\"
    echo "         \"SELECT id,name,company,contact,purpose,created_at FROM leads WHERE status='new' ORDER BY created_at DESC LIMIT 50;\""
    echo "    3) 配置邮件通知（可选）："
    echo "       ./scripts/deploy.sh secrets"
}

# ===== 入口 =====

show_help() {
    local text
    text=$(cat <<EOF
${CYAN}${BOLD}pinconsole marketing — Cloudflare 生产部署脚本${NC}

用法: ./scripts/deploy.sh <命令>

命令:
  ${GREEN}deploy${NC}     默认。完整流程：预检 → D1 远程 schema → build → 部署
  ${GREEN}check${NC}      仅预检（不部署）：登录 / database_id / D1 远程可达
  ${GREEN}db${NC}         仅应用 D1 schema 到远程（幂等，leads 表存在则跳过）
  ${GREEN}secrets${NC}    交互式检查 / 设置 RESEND_API_KEY/LEAD_NOTIFY_EMAIL/TURNSTILE_SECRET
  ${GREEN}build${NC}      仅 astro build（不部署）
  ${GREEN}--help${NC}     显示本帮助

前置条件:
  1. 已 wrangler login 并选定正确的 Cloudflare 账号
  2. wrangler.toml 的 database_id 已填（首次: pnpm exec wrangler d1 create ${D1_DB_NAME}）
  3. (可选) secrets 已通过 ./scripts/deploy.sh secrets 配置

环境变量:
  暂无（脚本不读取 .env；所有敏感配置走 Cloudflare Pages secrets）

示例:
  ./scripts/deploy.sh                     # 一键部署生产
  ./scripts/deploy.sh check               # 只跑预检
  ./scripts/deploy.sh secrets             # 设置邮件 / Turnstile secrets
  ./scripts/deploy.sh db                  # 仅把 D1 schema 应用到远程
EOF
)
    echo -e "$text"
}

case "${1:-deploy}" in
    deploy)        cmd_deploy ;;
    check)         cmd_check ;;
    db)            cmd_db ;;
    secrets)       cmd_secrets ;;
    build)         cmd_build ;;
    -h|--help|help) show_help ;;
    *)             error "未知命令: $1"; echo ""; show_help; exit 1 ;;
esac
