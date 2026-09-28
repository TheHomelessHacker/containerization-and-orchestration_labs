#!/bin/bash
set -euo pipefail

LAB_ROOT="$(cd "$(dirname "$0")/.." && pwd)"
API_BIN="$LAB_ROOT/api/api"
CGROUP_PATH="/sys/fs/cgroup/mydocker"
API_PORT=8080

# === 0. Очистка ===
sudo pkill -f 'unshare.*api' 2>/dev/null || true
sudo pkill -x api 2>/dev/null || true
sleep 1
sudo rmdir "$CGROUP_PATH" 2>/dev/null || true

# === 1. Создание cgroup с лимитами ===
sudo mkdir -p "$CGROUP_PATH"
echo "200M"         | sudo tee "$CGROUP_PATH/memory.max" > /dev/null
echo "50000 100000" | sudo tee "$CGROUP_PATH/cpu.max"    > /dev/null
echo "30"           | sudo tee "$CGROUP_PATH/pids.max"   > /dev/null
echo "0" | sudo tee "$CGROUP_PATH/memory.swap.max" > /dev/null

# === 2. Запуск api ОТ MARK (без sudo!) → --map-root-user даст mark→root ===
unshare --pid --mount --net --uts --ipc --user --map-root-user --fork --mount-proc "$API_BIN" &

sleep 2
API_PID=$(pgrep -x api | head -1)

if [ -z "$API_PID" ]; then
    echo "ERROR: api not found"
    exit 1
fi

# === 3. Добавляем PID в cgroup ОТ ROOT (через sudo tee) ===
echo "$API_PID" | sudo tee "$CGROUP_PATH/cgroup.procs" > /dev/null

echo "API PID: $API_PID"
echo "Owner (host): $(ps -o user= -p $API_PID)"
echo "cgroup.procs: $(cat $CGROUP_PATH/cgroup.procs)"

# === 4. Поднять lo и проверить ===
sudo nsenter --target "$API_PID" --net ip link set lo up

if ! sudo nsenter --target "$API_PID" --net --pid curl -sf "http://127.0.0.1:$API_PORT/health" > /dev/null; then
    echo "ERROR: api not responding"
    exit 1
fi

echo "=== mydocker started ==="
echo "health: $(sudo nsenter --target $API_PID --net --pid curl -s http://127.0.0.1:$API_PORT/health)"
