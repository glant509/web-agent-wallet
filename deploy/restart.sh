#!/usr/bin/env bash

set -u

script_dir="$(CDPATH= cd -- "$(dirname -- "$0")" && pwd)"

if ! "$script_dir/stop.sh"; then
  echo "重启失败：服务没有成功停止。" >&2
  exit 1
fi

if ! "$script_dir/start.sh"; then
  echo "重启失败：服务没有成功启动。" >&2
  exit 1
fi
