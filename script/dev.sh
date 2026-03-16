#!/usr/bin/env bash
# ==============================================================
# Mall 本地开发服务管理脚本
#
# 用法:
#   script/dev.sh start [service]   启动服务 (不指定则启动全部)
#   script/dev.sh stop  [service]   停止服务 (不指定则停止全部)
#   script/dev.sh status            查看运行状态
#   script/dev.sh logs  [service]   tail 日志 (不指定则全部)
#   script/dev.sh restart [service] 重启
# ==============================================================

set -eo pipefail

ROOT_DIR="$(cd "$(dirname "$0")/.." && pwd)"
DEV_DIR="$ROOT_DIR/.dev"
PID_DIR="$DEV_DIR/pids"
LOG_DIR="$DEV_DIR/logs"

ALL_SERVICES="user tenant product inventory order payment cart search marketing logistics notification consumer-bff merchant-bff admin-bff"

# Service → port mapping
get_port() {
  case "$1" in
    user)           echo 8081 ;;
    tenant)         echo 8082 ;;
    product)        echo 8083 ;;
    inventory)      echo 8084 ;;
    order)          echo 8085 ;;
    payment)        echo 8086 ;;
    cart)           echo 8087 ;;
    search)         echo 8088 ;;
    marketing)      echo 8089 ;;
    logistics)      echo 8090 ;;
    notification)   echo 8091 ;;
    consumer-bff)   echo 8080 ;;
    merchant-bff)   echo 8180 ;;
    admin-bff)      echo 8280 ;;
    *)              echo "?" ;;
  esac
}

# --------------- helpers ---------------

RED='\033[0;31m'
GREEN='\033[0;32m'
YELLOW='\033[0;33m'
CYAN='\033[0;36m'
BOLD='\033[1m'
NC='\033[0m'

log_info()  { echo -e "${CYAN}[INFO]${NC}  $*"; }
log_ok()    { echo -e "${GREEN}[OK]${NC}    $*"; }
log_warn()  { echo -e "${YELLOW}[WARN]${NC}  $*"; }
log_err()   { echo -e "${RED}[ERR]${NC}   $*"; }

ensure_dirs() {
  mkdir -p "$PID_DIR" "$LOG_DIR"
}

is_valid_service() {
  echo "$ALL_SERVICES" | tr ' ' '\n' | grep -qx "$1"
}

pid_file()  { echo "$PID_DIR/$1.pid"; }
log_file()  { echo "$LOG_DIR/$1.log"; }

is_running() {
  local pf
  pf="$(pid_file "$1")"
  [[ -f "$pf" ]] && kill -0 "$(cat "$pf")" 2>/dev/null
}

# --------------- start ---------------

start_service() {
  local svc="$1"

  if is_running "$svc"; then
    log_warn "$svc 已在运行 (PID $(cat "$(pid_file "$svc")"))"
    return 0
  fi

  local svc_dir="$ROOT_DIR/$svc"
  if [[ ! -f "$svc_dir/main.go" ]]; then
    log_err "$svc: main.go 不存在"
    return 1
  fi

  local lf pf
  lf="$(log_file "$svc")"
  pf="$(pid_file "$svc")"

  : > "$lf"  # truncate log

  cd "$ROOT_DIR"
  go run "./$svc/" >> "$lf" 2>&1 &
  local pid=$!
  echo "$pid" > "$pf"
  log_ok "$svc 已启动 (PID $pid, port $(get_port "$svc"))"
}

start_services() {
  local services="$*"
  [[ -z "$services" ]] && services="$ALL_SERVICES"

  local count
  count=$(echo "$services" | wc -w | tr -d ' ')
  log_info "启动 ${count} 个服务..."
  echo ""

  for svc in $services; do
    if ! is_valid_service "$svc"; then
      log_err "未知服务: $svc"
      echo "  可用: $ALL_SERVICES"
      return 1
    fi
  done

  for svc in $services; do
    start_service "$svc"
  done

  echo ""
  log_info "日志目录: $LOG_DIR/"
  log_info "停止服务: script/dev.sh stop"
  log_info "查看状态: script/dev.sh status"
}

# --------------- stop ---------------

stop_service() {
  local svc="$1"
  local pf
  pf="$(pid_file "$svc")"

  if [[ ! -f "$pf" ]]; then
    return 0
  fi

  local pid
  pid="$(cat "$pf")"

  if kill -0 "$pid" 2>/dev/null; then
    kill "$pid" 2>/dev/null || true
    # Wait up to 3 seconds for graceful shutdown
    local i=0
    while kill -0 "$pid" 2>/dev/null && [[ $i -lt 30 ]]; do
      sleep 0.1
      i=$((i + 1))
    done
    if kill -0 "$pid" 2>/dev/null; then
      kill -9 "$pid" 2>/dev/null || true
    fi
    log_ok "$svc 已停止 (PID $pid)"
  fi

  rm -f "$pf"
}

stop_services() {
  local services="$*"
  [[ -z "$services" ]] && services="$ALL_SERVICES"

  log_info "停止服务..."
  for svc in $services; do
    stop_service "$svc"
  done
  log_ok "完成"
}

# --------------- status ---------------

show_status() {
  echo ""
  printf "${BOLD}%-20s %-8s %-8s %-8s${NC}\n" "SERVICE" "PORT" "PID" "STATUS"
  printf "%-20s %-8s %-8s %-8s\n" "-------" "----" "---" "------"

  local running=0 stopped=0

  for svc in $ALL_SERVICES; do
    local port pid status pf

    port="$(get_port "$svc")"
    pf="$(pid_file "$svc")"

    if [[ -f "$pf" ]]; then
      pid="$(cat "$pf")"
      if kill -0 "$pid" 2>/dev/null; then
        status="${GREEN}running${NC}"
        running=$((running + 1))
      else
        status="${RED}dead${NC}"
        stopped=$((stopped + 1))
        rm -f "$pf"
        pid="-"
      fi
    else
      pid="-"
      status="${YELLOW}stopped${NC}"
      stopped=$((stopped + 1))
    fi

    printf "%-20s %-8s %-8s " "$svc" "$port" "$pid"
    echo -e "$status"
  done

  echo ""
  echo -e "  运行: ${GREEN}${running}${NC}  停止: ${YELLOW}${stopped}${NC}  共: $(echo "$ALL_SERVICES" | wc -w | tr -d ' ')"
  echo ""
}

# --------------- logs ---------------

tail_logs() {
  local services="$*"
  [[ -z "$services" ]] && services="$ALL_SERVICES"

  local log_files=""
  for svc in $services; do
    local lf
    lf="$(log_file "$svc")"
    [[ -f "$lf" ]] && log_files="$log_files $lf"
  done

  if [[ -z "$log_files" ]]; then
    log_warn "没有找到日志文件"
    return 0
  fi

  log_info "Ctrl+C 退出日志查看"
  # shellcheck disable=SC2086
  tail -f $log_files
}

# --------------- main ---------------

main() {
  ensure_dirs

  local cmd="${1:-help}"
  shift || true

  case "$cmd" in
    start)
      start_services "$@"
      ;;
    stop)
      stop_services "$@"
      ;;
    restart)
      stop_services "$@"
      sleep 1
      start_services "$@"
      ;;
    status)
      show_status
      ;;
    logs)
      tail_logs "$@"
      ;;
    help|--help|-h)
      echo "用法: script/dev.sh <command> [service...]"
      echo ""
      echo "Commands:"
      echo "  start [service]   启动服务 (不指定=全部)"
      echo "  stop  [service]   停止服务 (不指定=全部)"
      echo "  restart [service] 重启服务"
      echo "  status            查看运行状态"
      echo "  logs [service]    tail 日志"
      echo ""
      echo "Services:"
      echo "  $ALL_SERVICES"
      echo ""
      echo "Examples:"
      echo "  script/dev.sh start              # 启动全部"
      echo "  script/dev.sh start order        # 只启动 order"
      echo "  script/dev.sh start order payment # 启动 order + payment"
      echo "  script/dev.sh stop               # 停止全部"
      echo "  script/dev.sh logs order          # 查看 order 日志"
      ;;
    *)
      log_err "未知命令: $cmd"
      echo "  用法: script/dev.sh {start|stop|restart|status|logs} [service...]"
      exit 1
      ;;
  esac
}

main "$@"
