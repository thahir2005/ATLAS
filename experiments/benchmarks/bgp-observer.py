#!/usr/bin/env python3

import argparse
import json
import subprocess
import time
from datetime import datetime, timezone


def timestamp():
    return datetime.now(timezone.utc).isoformat(
        timespec="microseconds"
    ).replace("+00:00", "Z")


def run_command(command):
    result = subprocess.run(
        command,
        capture_output=True,
        text=True,
        timeout=5
    )

    if result.returncode != 0:
        raise RuntimeError(result.stderr.strip() or "Command failed")

    return result.stdout.strip()


def get_json(command):
    output = run_command(command)
    return json.loads(output)


def get_peer_states(namespace):
    data = get_json([
        "vtysh", "-N", namespace,
        "-c", "show ip bgp summary json"
    ])

    peers = data.get("ipv4Unicast", {}).get("peers", {})

    return {
        peer: details.get("state", "Unknown")
        for peer, details in peers.items()
    }


def get_bgp_bestpath(namespace, prefix):
    data = get_json([
        "vtysh", "-N", namespace,
        "-c", f"show ip bgp {prefix} json"
    ])

    for path in data.get("paths", []):
        best = path.get("bestpath", {})

        if best.get("overall"):
            next_hops = [
                item.get("ip")
                for item in path.get("nexthops", [])
                if item.get("ip")
            ]

            return {
                "prefix": prefix,
                "peer": path.get("peer", {}).get("peerId"),
                "as_path": path.get("aspath", {}).get("string"),
                "next_hops": next_hops
            }

    return None


def get_kernel_route(namespace, target):
    try:
        output = run_command([
            "ip", "-n", namespace,
            "-o", "route", "get", target
        ])

        fields = output.split()
        route = {
            "target": target,
            "next_hop": None,
            "interface": None,
            "source": None
        }

        for index, field in enumerate(fields[:-1]):
            if field == "via":
                route["next_hop"] = fields[index + 1]
            elif field == "dev":
                route["interface"] = fields[index + 1]
            elif field == "src":
                route["source"] = fields[index + 1]

        return route

    except RuntimeError:
        return None


def emit(event, details):
    record = {
        "timestamp": timestamp(),
        "event": event,
        "details": details
    }

    print(json.dumps(record), flush=True)


def observe(namespace, prefix, target, interval):
    previous_peers = None
    previous_bestpath = object()
    previous_route = object()

    print(
        f"ATLAS BGP Observer: namespace={namespace}, "
        f"prefix={prefix}, target={target}, "
        f"poll_target={interval}s",
        flush=True
    )

    while True:
        try:
            peers = get_peer_states(namespace)
            bestpath = get_bgp_bestpath(namespace, prefix)
            route = get_kernel_route(namespace, target)

            if previous_peers is None:
                emit("INITIAL_STATE", {
                    "peers": peers,
                    "bestpath": bestpath,
                    "kernel_route": route
                })
            else:
                for peer in sorted(set(previous_peers) | set(peers)):
                    old_state = previous_peers.get(peer, "Missing")
                    new_state = peers.get(peer, "Missing")

                    if old_state != new_state:
                        emit("PEER_STATE_CHANGE", {
                            "peer": peer,
                            "previous_state": old_state,
                            "new_state": new_state
                        })

                if bestpath != previous_bestpath:
                    emit("BGP_BESTPATH_CHANGE", {
                        "previous": previous_bestpath,
                        "current": bestpath
                    })

                if route != previous_route:
                    emit("KERNEL_ROUTE_CHANGE", {
                        "previous": previous_route,
                        "current": route
                    })

            previous_peers = peers
            previous_bestpath = bestpath
            previous_route = route

        except Exception as error:
            emit("OBSERVER_ERROR", {"message": str(error)})

        time.sleep(interval)


if __name__ == "__main__":
    parser = argparse.ArgumentParser(
        description="ATLAS FRR BGP and kernel route observer"
    )

    parser.add_argument("--namespace", default="atlas-mumbai")
    parser.add_argument("--prefix", default="10.10.2.1/32")
    parser.add_argument("--target", default="10.10.2.1")
    parser.add_argument("--interval", type=float, default=0.1)

    args = parser.parse_args()

    if args.interval <= 0:
        parser.error("interval must be positive")

    observe(
        args.namespace,
        args.prefix,
        args.target,
        args.interval
    )
