#!/usr/bin/env bash

set -u

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
cd "$script_dir" || exit 1

if [[ -x "./web3-agent" ]]; then
  app_name="web3-agent"
elif [[ -x "./web3-service-agent" ]]; then
  app_name="web3-service-agent"
else
  echo "停止失败：没有找到可执行文件 web3-agent 或 web3-service-agent。" >&2
  exit 1
fi

pid_file="./${app_name}.pid"
app_pid=""

if [[ -f "$pid_file" ]]; then
  read -r saved_pid < "$pid_file" || true
  if [[ "${saved_pid:-}" =~ ^[0-9]+$ ]] && kill -0 "$saved_pid" >/dev/null 2>&1; then
    saved_command="$(ps -p "$saved_pid" -o args= 2>/dev/null || true)"
    if [[ "$saved_command" =~ (^|/)${app_name}([[:space:]]|$) ]]; then
      app_pid="$saved_pid"
    fi
  fi
fi

# Also support a process started by an older script that did not write a PID file.
if [[ -z "$app_pid" ]]; then
  app_pid="$(pgrep -f "(^|/)${app_name}([[:space:]]|$)" | head -n 1)"
fi

if [[ -z "$app_pid" ]]; then
  rm -f "$pid_file"
  echo "$app_name 当前没有运行。"
  exit 0
fi

echo "正在停止 $app_name（PID: $app_pid）..."
if ! kill "$app_pid" >/dev/null 2>&1; then
  echo "停止失败：无法向 PID $app_pid 发送退出信号。" >&2
  exit 1
fi

for _ in {1..15}; do
  if ! kill -0 "$app_pid" >/dev/null 2>&1; then
    rm -f "$pid_file"
    echo "$app_name 已停止。"
    exit 0
  fi
  sleep 1
done

echo "$app_name 未在 15 秒内退出，正在强制停止..." >&2
if ! kill -9 "$app_pid" >/dev/null 2>&1; then
  echo "停止失败：无法强制停止 PID $app_pid。" >&2
  exit 1
fi

rm -f "$pid_file"
echo "$app_name 已强制停止。"
