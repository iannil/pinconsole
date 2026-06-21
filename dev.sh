#!/usr/bin/env bash
# pinconsole 开发环境一键启动脚本。
#
# 解决痛点:之前每次改前端代码都要 `make build-server` 重打包 embed + 重启 server。
# 本脚本启动独立的 vite dev server(HMR),admin 通过 5173 端口开发,API/WS 代理到 8080,
# SDK 通过 5174 端口开发并被 admin 代理。Go 改用 air 热重载,改 .go 文件自动重建。
#
# 用法:
#   ./dev.sh            启动全部(Go + admin + sdk)
#   ./dev.sh --no-go    只起前端(admin + sdk),假设 Go server 已在 8080 跑
#   ./dev.sh --no-build 跳过首次 server 二进制 build(已有 bin/pinconsole-server 时用)
#
# 访问:
#   admin(开发,带 HMR):http://localhost:5173/admin/
#   admin(8080 embed,生产模式):http://localhost:8080/admin/
#   访客 demo:http://localhost:8080/
#
# Ctrl+C 一次性关闭所有子进程。
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

# ===== 颜色 =====
if [[ -t 1 ]]; then
    C_RESET='\033[0m'; C_GREEN='\033[32m'; C_CYAN='\033[36m'; C_YELLOW='\033[33m'; C_RED='\033[31m'; C_MAG='\033[35m'
else
    C_RESET=''; C_GREEN=''; C_CYAN=''; C_YELLOW=''; C_RED=''; C_MAG=''
fi

log()  { printf "${C_CYAN}[dev]${C_RESET} %s\n" "$*"; }
warn() { printf "${C_YELLOW}[warn]${C_RESET} %s\n" "$*" >&2; }
err()  { printf "${C_RED}[err]${C_RESET} %s\n" "$*" >&2; }
ok()   { printf "${C_GREEN}[ok]${C_RESET} %s\n" "$*"; }

# ===== 参数解析 =====
NO_GO=0
NO_BUILD=0
for arg in "$@"; do
    case "$arg" in
        --no-go)    NO_GO=1 ;;
        --no-build) NO_BUILD=1 ;;
        *) err "未知参数: $arg"; exit 2 ;;
    esac
done

# ===== 预检:.env =====
if [[ ! -f .env ]]; then
    err ".env 不存在,请先复制 .env.example 并按需修改"
    exit 1
fi
# shellcheck disable=SC1091
set -a; . ./.env; set +a
: "${SERVER_PORT:?SERVER_PORT 未设置}"
: "${PG_HOST:?PG_HOST 未设置}"

# ===== 预检:docker 基础设施 =====
if [[ "$NO_GO" -eq 0 ]] || [[ "$NO_BUILD" -eq 0 ]]; then
    log "检查 docker 基础设施..."
    if ! docker compose ps postgres redis minio 2>/dev/null | grep -q "Up"; then
        warn "PG/Redis/MinIO 未全部运行,启动中..."
        docker compose up -d postgres redis minio
        sleep 2
    fi
    ok "docker 基础设施就绪"
fi

# ===== 子进程清理 =====
CHILD_PIDS=()
cleanup() {
    printf "\n${C_YELLOW}[dev]${C_RESET} 收到中断,清理子进程...\n"
    for pid in "${CHILD_PIDS[@]}"; do
        if kill -0 "$pid" 2>/dev/null; then
            kill "$pid" 2>/dev/null || true
        fi
    done
    # 二次 SIGKILL 兜底
    sleep 1
    for pid in "${CHILD_PIDS[@]}"; do
        kill -9 "$pid" 2>/dev/null || true
    done
    ok "已退出"
}
trap cleanup INT TERM EXIT

# ===== 启动 Go server(air 热重载)=====
if [[ "$NO_GO" -eq 0 ]]; then
    AIR_BIN="$(go env GOPATH)/bin/air"
    if [[ ! -x "$AIR_BIN" ]]; then
        log "air 未安装,执行 go install github.com/air-verse/air@latest ..."
        go install github.com/air-verse/air@latest
    fi

    # air 默认产物是 bin/server-dev,首次跑前确保 cmd/server 能编译。
    # 如果有 bin/pinconsole-server 旧产物,air 不影响它。
    log "启动 Go server(air 热重载,http://localhost:${SERVER_PORT})..."
    (
        cd server
        exec "$AIR_BIN"
    ) &
    CHILD_PIDS+=("$!")
fi

# ===== 启动 admin vite(HMR)=====
log "启动 admin vite(HMR,http://localhost:5173/admin/)..."
pnpm --filter @pinconsole/admin dev &
CHILD_PIDS+=("$!")

# ===== 启动 visitor-sdk vite(HMR)=====
log "启动 visitor-sdk vite(HMR,http://localhost:5174/)..."
pnpm --filter @pinconsole/visitor-sdk dev &
CHILD_PIDS+=("$!")

# ===== 等待子进程退出 =====
ok "全部启动完毕。Ctrl+C 退出全部。"
wait
