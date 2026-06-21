#!/usr/bin/env bash
# pinconsole 运维脚本
# 用法：./ops.sh {start|stop|restart|status|logs|build|migrate|clean}
set -euo pipefail

# ===== 路径与配置 =====
SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

SERVER_BIN="${SERVER_BIN:-server/bin/pinconsole-server}"
PID_FILE=".server.pid"
LOG_FILE="/tmp/pinconsole-server.log"
COMPOSE="docker compose"
HEALTH_URL="http://localhost:${SERVER_PORT:-8080}/healthz"

# 1k fail-secure 后 server 启动需要显式 env（ADMIN_PASSWORD 等）；
# 自动 source 根目录 .env（若存在），让 ./ops.sh start 开箱即用。
if [ -f "$SCRIPT_DIR/.env" ]; then
    set -a
    # shellcheck disable=SC1091
    . "$SCRIPT_DIR/.env"
    set +a
fi

# ===== 颜色 =====
RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[1;33m'
CYAN='\033[0;36m'
NC='\033[0m'

info()  { echo -e "${GREEN}[INFO]${NC}  $*"; }
warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
error() { echo -e "${RED}[ERROR]${NC} $*"; }

# ===== 子命令 =====

cmd_infra_up() {
    info "启动基础设施（PG + Redis + MinIO）..."
    $COMPOSE up -d postgres redis minio
    info "等待容器 healthy..."
    for i in $(seq 1 30); do
        local status
        status=$($COMPOSE ps --format json 2>/dev/null | python3 -c "
import sys,json
all_ok=True
for line in sys.stdin:
    try:
        d=json.loads(line)
        h=d.get('Health','')
        if h and h!='healthy': all_ok=False
    except: pass
print('healthy' if all_ok else 'waiting')
" 2>/dev/null || echo "waiting")
        if [ "$status" = "healthy" ]; then
            info "基础设施已就绪"
            return 0
        fi
        sleep 2
    done
    error "基础设施未就绪，超时"
    $COMPOSE ps
    return 1
}

cmd_infra_down() {
    info "停止基础设施..."
    $COMPOSE down
}

cmd_migrate() {
    # 1v:server 启动时自动跑内嵌 migrator（migrations.go + schema_migrations + advisory lock）。
    # 此处不再用 psql -f 手动跑（曾导致 schema_migrations 表缺失 + 与 server migrator 冲突）。
    # 详见 docs/audits/2026-06-18-1k-1u-regression.md §4 新-1 + 新-2。
    warn "migrations 由 server 启动时自动执行（cmd_migrate 已废弃，保留为兼容入口）。"
    warn "如需手动应用：./ops.sh restart（重启 server 即触发自动迁移）"
    warn "如需重置 dev DB：./ops.sh reset"
}

cmd_migrate_reset() {
    warn "重置数据库（DROP ALL）..."
    # 1v:加 visitor_consents（1l 表）+ schema_migrations（防 golang-migrate CLI 残留脏表）。
    # 不再调 cmd_migrate；启动 server 时 server 自家 migrator 会在干净 DB 上跑全套。
    $COMPOSE exec -T postgres psql -U "${PG_USER:-mm}" -d "${PG_DB:-pinconsole}" \
        -c "DROP TABLE IF EXISTS visitor_consents, chat_messages, co_browsing_commands, event_blobs, users, sessions, visitors, schema_migrations CASCADE;" || true
    info "数据库已清空。下次 ./ops.sh start 时 server 会自动跑全部 migrations。"
}

cmd_build() {
    info "构建前端..."
    pnpm --filter @pinconsole/admin build 2>&1 | tail -3
    pnpm --filter @pinconsole/visitor-sdk build 2>&1 | tail -3

    info "暂存前端产物到 embed 目录..."
    rm -rf server/cmd/server/embedded/admin server/cmd/server/embedded/sdk server/cmd/server/embedded/landing
    mkdir -p server/cmd/server/embedded/admin server/cmd/server/embedded/sdk server/cmd/server/embedded/landing
    cp -r admin/dist/. server/cmd/server/embedded/admin/
    cp -r visitor-sdk/dist/. server/cmd/server/embedded/sdk/
    cp -r landing/. server/cmd/server/embedded/landing/

    info "构建 Go release 二进制..."
    cd server && CGO_ENABLED=0 go build -tags release -o bin/pinconsole-server ./cmd/server && cd ..
    info "构建完成：$(ls -lh "$SERVER_BIN" | awk '{print $5, $9}')"
}

cmd_server_start() {
    if [ -f "$PID_FILE" ] && kill -0 "$(cat "$PID_FILE")" 2>/dev/null; then
        warn "server 已在运行（PID $(cat "$PID_FILE")）"
        return 0
    fi
    if [ ! -f "$SERVER_BIN" ]; then
        error "二进制不存在：$SERVER_BIN，请先 ./ops.sh build"
        return 1
    fi
    info "启动 server..."
    "${SERVER_BIN}" > "${LOG_FILE}" 2>&1 &
    local bg_pid=$!
    echo "${bg_pid}" > "${PID_FILE}"
    info "server PID=${bg_pid}  log: ${LOG_FILE}"

    # 等待健康检查
    info "等待健康检查..."
    for i in $(seq 1 15); do
        if curl -sf "${HEALTH_URL}" >/dev/null 2>&1; then
            info "server 已就绪"
            return 0
        fi
        sleep 1
    done
    error "server 健康检查超时  tail -50 ${LOG_FILE}"
    return 1
}

cmd_server_stop() {
    if [ ! -f "${PID_FILE}" ]; then
        info "server 未运行"
        return 0
    fi
    local stop_pid
    stop_pid=$(cat "${PID_FILE}")
    if kill -0 "${stop_pid}" 2>/dev/null; then
        info "停止 server (PID ${stop_pid})..."
        kill "${stop_pid}" 2>/dev/null || true
        for i in $(seq 1 10); do
            kill -0 "${stop_pid}" 2>/dev/null || break
            sleep 0.5
        done
        kill -9 "${stop_pid}" 2>/dev/null || true
        info "server 已停止"
    else
        info "server 进程不存在，清理 PID 文件"
    fi
    rm -f "${PID_FILE}"
}

cmd_start() {
    cmd_infra_up
    # 1v:不再调 cmd_migrate（已废弃，且与 server 内嵌 migrator 冲突）。
    # server 启动时自动跑 migrations（migrations.go + advisory lock + schema_migrations）。
    if [ ! -f "$SERVER_BIN" ]; then
        warn "二进制不存在，自动构建..."
        cmd_build
    fi
    cmd_server_start
    echo ""
    info "=== 全部服务已启动 ==="
    cmd_status
}

cmd_stop() {
    cmd_server_stop
    cmd_infra_down
    info "=== 全部服务已停止 ==="
}

cmd_restart() {
    info "重启中..."
    cmd_server_stop
    sleep 1
    cmd_server_start
}

cmd_status() {
    echo ""
    echo -e "${CYAN}=== 服务状态 ===${NC}"
    echo ""

    # Docker 服务
    echo -e "${CYAN}[Docker]${NC}"
    $COMPOSE ps --format "table {{.Name}}\t{{.Status}}\t{{.Ports}}" 2>/dev/null || echo "  (docker compose 未运行)"
    echo ""

    # Server
    echo -e "${CYAN}[Server]${NC}"
    if [ -f "${PID_FILE}" ] && kill -0 "$(cat "${PID_FILE}")" 2>/dev/null; then
        local status_pid
        status_pid=$(cat "${PID_FILE}")
        echo "  PID: ${status_pid} (running)"
        if curl -sf "${HEALTH_URL}" >/dev/null 2>&1; then
            local health
            health=$(curl -s "${HEALTH_URL}" 2>/dev/null)
            echo "  Health: OK (${health})"
        else
            echo "  Health: ✗ (无响应)"
        fi
    else
        echo "  状态: 未运行"
    fi
    echo ""

    # 端口
    echo -e "${CYAN}[端口]${NC}"
    echo "  Server:     http://localhost:${SERVER_PORT:-8080}/"
    echo "  Admin:      http://localhost:${SERVER_PORT:-8080}/admin"
    echo "  Landing:    http://localhost:${SERVER_PORT:-8080}/"
    echo "  MinIO:      http://localhost:9001 (console)"
    echo "  Server Log: $LOG_FILE"
    echo ""
}

cmd_logs() {
    if [ -f "$LOG_FILE" ]; then
        info "tail -f $LOG_FILE (Ctrl+C 退出)"
        tail -f "$LOG_FILE"
    else
        error "日志文件不存在：$LOG_FILE"
    fi
}

cmd_dev() {
    info "启动开发模式（热重载）..."
    cmd_infra_up
    # 1v:不再调 cmd_migrate；air 重启 server 时自动跑 migrations。
    if [ ! -f "server/.air.toml" ]; then
        error "缺少 .air.toml，请先 make install-tools"
        return 1
    fi
    info "启动 pnpm dev（Go air + admin Vite + SDK playground）..."
    pnpm dev
}

cmd_clean() {
    warn "清理构建产物 + 停止全部服务..."
    cmd_stop
    rm -rf server/bin admin/dist visitor-sdk/dist server/cmd/server/embedded/admin server/cmd/server/embedded/sdk server/cmd/server/embedded/landing
    info "清理完成"
}

# ===== 入口 =====

show_help() {
    echo -e "${CYAN}pinconsole 运维脚本${NC}"
    echo ""
    echo "用法: ./ops.sh <命令>"
    echo ""
    echo "命令:"
    echo -e "  ${GREEN}start${NC}     启动全部服务（infra + migrate + server）"
    echo -e "  ${GREEN}stop${NC}      停止全部服务"
    echo -e "  ${GREEN}restart${NC}   重启 server（不动 infra）"
    echo -e "  ${GREEN}status${NC}    查看全部服务状态"
    echo -e "  ${GREEN}logs${NC}      跟踪 server 日志"
    echo -e "  ${GREEN}build${NC}     构建前端 + Go 二进制"
    echo -e "  ${GREEN}migrate${NC}   应用数据库 migration"
    echo -e "  ${GREEN}reset${NC}     重置数据库（DROP + migrate）"
    echo -e "  ${GREEN}dev${NC}       开发模式（air + Vite HMR）"
    echo -e "  ${GREEN}clean${NC}     停止全部 + 清理构建产物"
    echo ""
    echo "环境变量:"
    echo "  SERVER_PORT     server 端口（默认 8080）"
    echo "  SERVER_BIN      二进制路径（默认 server/bin/pinconsole-server）"
    echo "  PG_USER         PG 用户（默认 mm）"
    echo "  PG_DB           PG 库名（默认 pinconsole）"
    echo ""
    echo "示例:"
    echo "  ./ops.sh start       # 一键启动"
    echo "  ./ops.sh status      # 查看状态"
    echo "  ./ops.sh restart     # 重启 server"
    echo "  ./ops.sh logs        # 看日志"
    echo "  ./ops.sh stop        # 一键关闭"
}

case "${1:-}" in
    start)   cmd_start ;;
    stop)    cmd_stop ;;
    restart) cmd_restart ;;
    status)  cmd_status ;;
    logs)    cmd_logs ;;
    build)   cmd_build ;;
    migrate) cmd_migrate ;;
    reset)   cmd_migrate_reset ;;
    dev)     cmd_dev ;;
    clean)   cmd_clean ;;
    *)       show_help ;;
esac
