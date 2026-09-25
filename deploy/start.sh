#!/usr/bin/env bash

set -u

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"
cd "$script_dir" || exit 1

if [[ -x "./web3-agent" ]]; then
  app_name="web3-agent"
elif [[ -x "./web3-service-agent" ]]; then
  app_name="web3-service-agent"
else
  echo "启动失败：没有找到可执行文件 web3-agent 或 web3-service-agent。" >&2
  exit 1
fi

running_pid="$(pgrep -f "(^|/)${app_name}([[:space:]]|$)" | head -n 1)"
if [[ -n "$running_pid" ]]; then
  echo "启动失败：$app_name 已经启动（PID: $running_pid），不能重复启动。" >&2
  exit 1
fi

nohup "./$app_name" >/dev/null 2>&1 &
app_pid=$!

# Give the process a moment to report immediate startup failures.
sleep 1
if ! kill -0 "$app_pid" >/dev/null 2>&1; then
  echo "启动失败：$app_name 未能正常运行，请检查配置和系统日志。" >&2
  exit 1
fi

echo "$app_pid" > "./${app_name}.pid"
echo "$app_name 启动成功，PID: $app_pid"
