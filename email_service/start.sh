#!/bin/sh
# Start worker in background, redirecting output to container's stdout/stderr
python worker.py >> /proc/1/fd/1 2>> /proc/1/fd/2 &

# Start API server (foreground)
exec uvicorn main:app --host 0.0.0.0 --port 8000
