#!/usr/bin/env bash

set -u

NS="${1:-atlas-mumbai}"
TARGET="${2:-10.10.2.1}"
INTERVAL="${3:-0.02}"

previous=""

echo "ATLAS BGP Route Observer"
echo "Namespace: $NS"
echo "Target: $TARGET"
echo "Polling interval: ${INTERVAL}s"
echo "timestamp,event,route"

while true; do
    current=$(
        ip -n "$NS" -o route get "$TARGET" 2>/dev/null |
        awk '{
            via="-"
            dev="-"

            for (i=1; i<=NF; i++) {
                if ($i=="via") via=$(i+1)
                if ($i=="dev") dev=$(i+1)
            }

            print "via=" via " dev=" dev
        }'
    )

    if [ -z "$current" ]; then
        current="UNREACHABLE"
    fi

    if [ "$current" != "$previous" ]; then
        timestamp=$(date -u +"%Y-%m-%dT%H:%M:%S.%NZ")

        if [ -z "$previous" ]; then
            event="INITIAL"
        elif [ "$current" = "UNREACHABLE" ]; then
            event="ROUTE_LOST"
        elif [ "$previous" = "UNREACHABLE" ]; then
            event="ROUTE_RESTORED"
        else
            event="ROUTE_CHANGED"
        fi

        echo "$timestamp,$event,$current"
        previous="$current"
    fi

    sleep "$INTERVAL"
done
