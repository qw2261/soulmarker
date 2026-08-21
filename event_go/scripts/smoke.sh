#!/usr/bin/env sh
# 容器/部署 smoke test：验证一个正在运行的 event-go 实例对外提供 liveness 与 readiness 探针。
# 用法：scripts/smoke.sh [base_url]   （默认 http://127.0.0.1:8080）
set -eu

base="${1:-http://127.0.0.1:8080}"

wait_for_status() {
  url="$1"
  want="$2"
  last=""
  for _ in $(seq 1 60); do
    last=$(curl -s -o /dev/null -w '%{http_code}' "$url" 2>/dev/null || true)
    if [ "$last" = "$want" ]; then
      return 0
    fi
    sleep 1
  done
  echo "ERROR: $url 未在 60s 内返回 $want（最后状态=$last）" >&2
  return 1
}

assert_contains() {
  url="$1"
  needle="$2"
  body=$(curl -s --max-time 10 "$url" 2>/dev/null || true)
  case "$body" in
    *"$needle"*) return 0 ;;
  esac
  echo "ERROR: $url 响应缺少 \"$needle\"：$body" >&2
  return 1
}

# liveness：进程存活即返回 200。
wait_for_status "$base/healthz" 200
assert_contains "$base/healthz" '"status":"ok"'

# readiness：数据库就绪返回 200。
wait_for_status "$base/readyz" 200
assert_contains "$base/readyz" '"db":"connected"'

echo "smoke ok: $base"
