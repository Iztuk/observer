#!/usr/bin/env bash

BASE_URL="${BASE_URL:-http://localhost:8080}"
HOST_HEADER="${HOST_HEADER:-api.local}"
CONCURRENCY="${CONCURRENCY:-3}"
SLEEP="${SLEEP:-0.25}"
CURL_MAX_TIME="${CURL_MAX_TIME:-15}"

ENDPOINTS=(
  "GET /health"
  "GET /users"
  "GET /users/1"
  "GET /users/2"
  "GET /users/999"
  "GET /timeout"
  "GET /reset"
  "GET /unknown-route"
)

send_request() {
  worker_id="$1"

  while true; do
    selected="${ENDPOINTS[$RANDOM % ${#ENDPOINTS[@]}]}"

    method="$(echo "$selected" | awk '{print $1}')"
    path="$(echo "$selected" | awk '{print $2}')"

    timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"

    echo "[$timestamp] worker=$worker_id sending method=$method host=$HOST_HEADER path=$path"

    status_code="$(
      curl -sS \
        --max-time "$CURL_MAX_TIME" \
        -o /dev/null \
        -w "%{http_code}" \
        -X "$method" \
        -H "Host: $HOST_HEADER" \
        "$BASE_URL$path" 2>/dev/null
    )"

    timestamp="$(date -u +"%Y-%m-%dT%H:%M:%SZ")"
    echo "[$timestamp] worker=$worker_id completed method=$method path=$path status=$status_code"

    sleep "$SLEEP"
  done
}

echo "Random traffic simulator"
echo "Base URL:      $BASE_URL"
echo "Host header:   $HOST_HEADER"
echo "Concurrency:   $CONCURRENCY"
echo "Sleep:         $SLEEP"
echo "Curl max time: $CURL_MAX_TIME"
echo "Stop with Ctrl+C"
echo

for i in $(seq 1 "$CONCURRENCY"); do
  send_request "$i" &
done

wait
