#!/bin/sh
set -e

mkdir -p /data
chown -R appuser:appuser /data
exec su-exec appuser /app/study-plan-backend
