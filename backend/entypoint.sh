#!/bin/sh
set -e

# Wait for app.env
while [ ! -f /vault/config/app.env ]; do
  echo "Waiting for app.env..."
  sleep 2
done

echo "Found app.env, starting server..."
exec npm start
