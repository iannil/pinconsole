#!/usr/bin/env bash
# pinconsole 开发环境管理脚本。
#
# 解决痛点:之前每次改前端代码都要 `make build-server` 重打包 embed + 重启 server。
# 本脚本启动独立的 vite dev server(HMR),admin 通过 7073 端口开发,API/WS 代理到 7080,
# SDK 通过 7074 端口开发并被 admin 代理。Go 改用 air 热重载,改 .go 文件自动重建。
#
# 服务在后台运行,PID 写入 .dev/*.pid,日志写入 .dev/*.log。
#
# 用法:
#   ./dev.sh start             启动全部(Go + admin + sdk + marketing)
#   ./dev.sh stop              关闭全部
#   ./dev.sh restart           重启全部
#   ./dev.sh status            查看服务状态
#   ./dev.sh logs [name]       跟踪日志(name: go|admin|sdk|marketing,省略则全部)
#
# 启动选项(仅 start / restart 有效):
#   --no-go      只起前端(admin + sdk),假设 Go server 已在 7080 跑
#   --no-build   跳过首次 server 二进制 build(已有 bin/pinconsole-server 时用)
#
# 访问:
#   admin(开发,带 HMR):http://localhost:7073/admin/
#   admin(7080 embed,生产模式):http://localhost:7080/admin/
#   访客 demo:http://localhost:7080/
#   marketing(开发,带 HMR):http://localhost:7075/
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
cd "$SCRIPT_DIR"

RUN_DIR="$SCRIPT_DIR/.dev"
mkdir -p "$RUN_DIR"

# 服务定义:名称用于 pid/log 文件命名与状态展示。
SERVICES=(go admin sdk marketing)

# 端口兜底清理映射:与 SERVICES 数组索引对齐
SERVICE_PORTS=(7080 7073 7074 7075)

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

pid_file() { printf '%s/%s.pid' "$RUN_DIR" "$1"; }
log_file() { printf '%s/%s.log' "$RUN_DIR" "$1"; }

# 读取指定服务的 PID(进程存活才回显,否则清理过期 pid 文件)。
running_pid() {
    local name="$1" pf
    pf="$(pid_file "$name")"
    [[ -f "$pf" ]] || return 1
    local pid
    pid="$(cat "$pf" 2>/dev/null || true)"
    [[ -n "$pid" ]] || { rm -f "$pf"; return 1; }
    if kill -0 "$pid" 2>/dev/null; then
        printf '%s' "$pid"
        return 0
    fi
    rm -f "$pf"
    return 1
}

# ===== 预检:.env =====
load_env() {
    if [[ ! -f .env ]]; then
        err ".env 不存在,请先复制 .env.example 并按需修改"
        exit 1
    fi
    # shellcheck disable=SC1091
    set -a; . ./.env; set +a
    : "${SERVER_PORT:?SERVER_PORT 未设置}"
    : "${PG_HOST:?PG_HOST 未设置}"
}

# ===== 预检:docker 基础设施 =====
ensure_docker() {
    log "检查 docker 基础设施..."
    if ! docker compose ps postgres redis minio 2>/dev/null | grep -q "Up"; then
        warn "PG/Redis/MinIO 未全部运行,启动中..."
        docker compose up -d postgres redis minio
        sleep 2
    fi
    ok "docker 基础设施就绪"
}

# 后台启动一个服务并记录 PID。
# 用法:spawn <name> <command...>
spawn() {
    local name="$1" port="$2"; shift 2
    # 端口兜底:先清理该端口上的僵尸进程(pid 文件失效但端口被占)
    if port_is_listening "$port"; then
        warn "$name 端口 $port 被僵尸进程占用,正在清理..."
        kill_port_if_occupied "$port" "$name"
    fi
    if running_pid "$name" >/dev/null; then
        warn "$name 已在运行(PID $(running_pid "$name")),跳过"
        return 0
    fi
    local lf; lf="$(log_file "$name")"
    # 开启 monitor 模式(job control),让 backgrounded job 成为独立进程组的组长,
    # $! 即为 PGID,便于 cmd_stop 用 `kill -- -$pid` 整组 kill(air / vite 会派生子进程)。
    # 用 monitor 模式替代 setsid:macOS 不自带 setsid。
    set -m
    # 注意:命令串自身已带 `cd ... && exec <bin>`,这里不能再前缀 exec
    #(`exec cd` 非法,cd 是 builtin);让内层 bash 跑 cd 再 exec 真正的进程,PID 即被替换保留。
    bash -c "$*" >"$lf" 2>&1 &
    local pid=$!
    set +m
    disown
    echo "$pid" >"$(pid_file "$name")"
    ok "$name 已启动(PID $pid,日志 ${lf#$SCRIPT_DIR/})"
}

# 端口兜底清理:当进程组方式失效(Vite/node 脱离进程组)时,
# 用 lsof 找到并杀死占用已知端口的进程。
kill_port_if_occupied() {
    local port="$1" name="$2"
    local pids
    # macOS:lsof -ti :PORT 输出 PID 列表(一行一个)
    pids="$(lsof -ti ":$port" 2>/dev/null || true)"
    if [[ -n "$pids" ]]; then
        warn "$name 端口 $port 仍被进程占用(PID $(echo "$pids" | tr '\n' ' ')),正在清理..."
        echo "$pids" | xargs kill -TERM 2>/dev/null || true
        sleep 1
        pids="$(lsof -ti ":$port" 2>/dev/null || true)"
        if [[ -n "$pids" ]]; then
            warn "$name 端口 $port 未优雅退出,强制结束"
            echo "$pids" | xargs kill -9 2>/dev/null || true
        fi
    fi
}

# ===== start =====
cmd_start() {
    local no_go="$1" no_build="$2"

    load_env
    if [[ "$no_go" -eq 0 ]] || [[ "$no_build" -eq 0 ]]; then
        ensure_docker
    fi

    if [[ "$no_go" -eq 0 ]]; then
        local air_bin
        air_bin="$(go env GOPATH)/bin/air"
        if [[ ! -x "$air_bin" ]]; then
            log "air 未安装,执行 go install github.com/air-verse/air@latest ..."
            go install github.com/air-verse/air@latest
        fi
        log "启动 Go server(air 热重载,http://localhost:${SERVER_PORT})..."
        spawn go "${SERVICE_PORTS[0]}" "cd '$SCRIPT_DIR/server' && exec '$air_bin'"
    else
        warn "--no-go 已指定,跳过 Go server"
    fi

    log "启动 admin vite(HMR,http://localhost:7073/admin/)..."
    spawn admin "${SERVICE_PORTS[1]}" "cd '$SCRIPT_DIR' && exec pnpm --filter @pinconsole/admin dev"

    log "启动 visitor-sdk vite(HMR,http://localhost:7074/)..."
    spawn sdk "${SERVICE_PORTS[2]}" "cd '$SCRIPT_DIR' && exec pnpm --filter @pinconsole/visitor-sdk dev"

    log "启动 marketing Astro(HMR,http://localhost:7075/)..."
    spawn marketing "${SERVICE_PORTS[3]}" "cd '$SCRIPT_DIR/marketing' && exec pnpm astro dev --port 7075 --host"

    echo
    # 等待服务端口就绪(最多 8 秒),避免刚启动时误判为 hang
    log "等待服务就绪..."
    local idx=0
    for name in "${SERVICES[@]}"; do
        local port="${SERVICE_PORTS[$idx]}"
        local tries=0
        while [[ $tries -lt 8 ]]; do
            if port_is_listening "$port"; then
                break
            fi
            sleep 1
            tries=$((tries + 1))
        done
        idx=$((idx + 1))
    done
    cmd_status
    echo
    ok "全部启动完毕。查看日志:./dev.sh logs   关闭:./dev.sh stop"
}

# ===== stop =====
cmd_stop() {
    local any=0
    # 倒序关闭,先停依赖方。
    for ((i=${#SERVICES[@]}-1; i>=0; i--)); do
        local name="${SERVICES[i]}" pid
        if pid="$(running_pid "$name")"; then
            any=1
            log "关闭 $name(PID $pid)..."
            # 杀整个进程组(air / vite 会派生子进程)。
            kill -TERM "-$pid" 2>/dev/null || kill -TERM "$pid" 2>/dev/null || true
        fi
    done

    if [[ "$any" -eq 1 ]]; then
        # 等待最多 5s 优雅退出,否则 SIGKILL 兜底。
        for _ in 1 2 3 4 5; do
            local alive=0
            for name in "${SERVICES[@]}"; do
                running_pid "$name" >/dev/null && alive=1
            done
            [[ "$alive" -eq 0 ]] && break
            sleep 1
        done

        for name in "${SERVICES[@]}"; do
            local pid
            if pid="$(running_pid "$name")"; then
                warn "$name 未优雅退出,强制结束(PID $pid)"
                kill -9 "-$pid" 2>/dev/null || kill -9 "$pid" 2>/dev/null || true
            fi
            rm -f "$(pid_file "$name")"
        done
    else
        warn "没有运行中的服务(pid 文件)"
    fi

    # 端口兜底:清理脱离进程组的僵尸进程
    local idx=0
    for name in "${SERVICES[@]}"; do
        kill_port_if_occupied "${SERVICE_PORTS[$idx]}" "$name"
        rm -f "$(pid_file "$name")"
        idx=$((idx + 1))
    done
    ok "已全部关闭"
}

# 检查指定端口是否在监听(用 lsof -ti,兼容 macOS/Linux)。
# 返回 0(true)表示端口上有进程监听。
port_is_listening() {
    local port="$1"
    lsof -ti ":$port" 2>/dev/null | grep -q .
}

# ===== status =====
cmd_status() {
    printf "${C_MAG}%-8s %-10s %-8s %s${C_RESET}\n" "服务" "状态" "PID" "日志"
    local idx=0
    for name in "${SERVICES[@]}"; do
        local port="${SERVICE_PORTS[$idx]}"
        local pid lf state_label pid_display
        lf="$(log_file "$name")"
        lf="${lf#$SCRIPT_DIR/}"
        pid_display="-"
        state_label="${C_RED}stopped${C_RESET}"

        if pid="$(running_pid "$name")"; then
            # PID 文件有效且进程存活
            if port_is_listening "$port"; then
                # 优先显示实际监听端口的进程 PID(air→server 场景)
                local actual_pid
                actual_pid="$(lsof -ti ":$port" 2>/dev/null | head -1)"
                pid_display="${actual_pid:-$pid}"
                state_label="${C_GREEN}running${C_RESET}"
            else
                # PID 存活但端口没在监听——进程可能卡住或 non-server 状态
                pid_display="$pid"
                state_label="${C_YELLOW}hang${C_RESET}"
            fi
        elif port_is_listening "$port"; then
            # 端口上有进程但 pid 文件失效(僵尸/脱离进程组)
            local orphan_pid
            orphan_pid="$(lsof -ti ":$port" 2>/dev/null | head -1)"
            pid_display="$orphan_pid"
            state_label="${C_YELLOW}zombie${C_RESET}"
        fi

        printf "%-8s ${state_label} %-8s %s\n" "$name" "$pid_display" "$lf"
        idx=$((idx + 1))
    done
}

# ===== logs =====
cmd_logs() {
    local target="${1:-}"
    if [[ -n "$target" ]]; then
        local lf; lf="$(log_file "$target")"
        [[ -f "$lf" ]] || { err "无日志文件: $lf"; exit 1; }
        exec tail -f "$lf"
    fi
    # 全部:tail -f 多文件
    local files=()
    for name in "${SERVICES[@]}"; do
        local lf; lf="$(log_file "$name")"
        [[ -f "$lf" ]] && files+=("$lf")
    done
    [[ ${#files[@]} -gt 0 ]] || { err "暂无任何日志文件"; exit 1; }
    exec tail -f "${files[@]}"
}

# ===== 参数解析 =====
COMMAND="${1:-start}"
[[ $# -gt 0 ]] && shift || true

NO_GO=0
NO_BUILD=0
LOG_TARGET=""
for arg in "$@"; do
    case "$arg" in
        --no-go)    NO_GO=1 ;;
        --no-build) NO_BUILD=1 ;;
        go|admin|sdk|marketing) LOG_TARGET="$arg" ;;
        *) err "未知参数: $arg"; exit 2 ;;
    esac
done

case "$COMMAND" in
    start)
        cmd_start "$NO_GO" "$NO_BUILD"
        ;;
    stop)
        cmd_stop
        ;;
    restart)
        cmd_stop
        echo
        cmd_start "$NO_GO" "$NO_BUILD"
        ;;
    status)
        cmd_status
        ;;
    logs)
        cmd_logs "$LOG_TARGET"
        ;;
    -h|--help|help)
        sed -n '2,30p' "$0" | sed 's/^# \{0,1\}//'
        ;;
    *)
        err "未知命令: $COMMAND(可用:start|stop|restart|status|logs)"
        exit 2
        ;;
esac
