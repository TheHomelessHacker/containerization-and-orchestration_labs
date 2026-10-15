#!/bin/bash

# Переходим в папку лабы (родитель папки scripts/)
cd "$(dirname "$0")/.."

./api/api &
API_PID=$!
echo "API PID: $API_PID"

wait $API_PID
