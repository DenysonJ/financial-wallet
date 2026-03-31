#!/usr/bin/env python3
"""
Kibana Alerting Rules Generator — Go Financial Wallet

Creates 6 basic Kibana alerting rules via the Kibana Alerting API.
Uses ES|QL queries against OTel metrics data. Stdlib only (no pip deps).

Usage:
    python3 generate_rules.py                          # dry-run (default)
    python3 generate_rules.py --apply                  # create rules in Kibana
    python3 generate_rules.py --delete                 # delete all managed rules
    python3 generate_rules.py --kibana-url http://kibana:5601  # custom URL

Environment Variables:
    KIBANA_URL          Kibana base URL (default: http://localhost:5601)
    METRICS_INDEX       ES index pattern for metrics (default: otel-metrics-*)

Requirements:
    - Kibana 8.13+ with encryptedSavedObjects key configured
    - Elasticsearch with metrics data stream populated by OTel Collector
    - Python 3.6+
"""

import argparse
import json
import sys
import urllib.request
import urllib.error

# --------------------------------------------------------------------------- #
# Configuration
# --------------------------------------------------------------------------- #

RULE_PREFIX = "FINANCIAL"
INDEX_PATTERN = "otel-metrics-*"
TIME_FIELD = "@timestamp"
TAGS = ["financial", "slo", "auto-generated"]


# --------------------------------------------------------------------------- #
# ES|QL Query Builders
# --------------------------------------------------------------------------- #

def esql_burn_rate(window_minutes, threshold):
    """Burn rate = error_rate / error_budget. Fires if burn_rate > threshold.

    Error budget assumes 99.9% availability target (budget = 0.001).
    """
    error_budget = 0.001
    return (
        "FROM {idx} "
        "| WHERE {ts} > NOW() - {win} MINUTES "
        "| WHERE http_server_request_count IS NOT NULL "
        "| STATS total = SUM(http_server_request_count), "
        "errors_5xx = SUM(CASE(http_response_status_code >= 500, http_server_request_count, 0.0)) "
        "| WHERE total > 0 "
        "| EVAL error_rate = TO_DOUBLE(errors_5xx) / TO_DOUBLE(total) "
        "| EVAL burn_rate = error_rate / {budget} "
        "| WHERE burn_rate > {thresh}"
    ).format(
        idx=INDEX_PATTERN,
        ts=TIME_FIELD,
        win=window_minutes,
        budget=error_budget,
        thresh=threshold,
    )


def esql_latency_percentile(window_minutes, percentile, threshold_ms):
    """Fires if latency percentile exceeds threshold (in milliseconds)."""
    return (
        "FROM {idx} "
        "| WHERE {ts} > NOW() - {win} MINUTES "
        "| WHERE http.server.duration IS NOT NULL "
        "| STATS pval = PERCENTILE(http.server.duration, {pct}) "
        "| WHERE pval > {thresh}"
    ).format(
        idx=INDEX_PATTERN,
        ts=TIME_FIELD,
        win=window_minutes,
        pct=percentile,
        thresh=threshold_ms,
    )


def esql_error_5xx_rate(window_minutes, threshold_fraction):
    """Fires if 5xx error rate exceeds threshold (as fraction, e.g. 0.01 = 1%)."""
    return (
        "FROM {idx} "
        "| WHERE {ts} > NOW() - {win} MINUTES "
        "| WHERE http_server_request_count IS NOT NULL "
        "| STATS total = SUM(http_server_request_count), "
        "errors_5xx = SUM(CASE(http_response_status_code >= 500, http_server_request_count, 0.0)) "
        "| WHERE total > 0 "
        "| EVAL error_rate = TO_DOUBLE(errors_5xx) / TO_DOUBLE(total) "
        "| WHERE error_rate > {thresh}"
    ).format(
        idx=INDEX_PATTERN,
        ts=TIME_FIELD,
        win=window_minutes,
        thresh=threshold_fraction,
    )


def esql_db_pool_saturation(window_minutes, threshold_pct):
    """Fires if DB pool saturation exceeds threshold (as percentage, e.g. 95.0)."""
    return (
        "FROM {idx} "
        "| WHERE {ts} > NOW() - {win} MINUTES "
        "| STATS in_use = MAX(db_pool_connections_in_use), "
        "max_open = MAX(db_pool_connections_max_open) "
        "| WHERE max_open IS NOT NULL AND max_open > 0 "
        "| EVAL saturation_pct = TO_DOUBLE(in_use) / TO_DOUBLE(max_open) * 100.0 "
        "| WHERE saturation_pct > {thresh}"
    ).format(
        idx=INDEX_PATTERN,
        ts=TIME_FIELD,
        win=window_minutes,
        thresh=threshold_pct,
    )


# --------------------------------------------------------------------------- #
# Rule Definitions
# --------------------------------------------------------------------------- #

RULES = [
    # 1. Burn Rate Critical - error budget burn rate > 10 in 5m window
    {
        "name": "[{prefix}] Burn Rate Critical".format(prefix=RULE_PREFIX),
        "tags": TAGS + ["severity:critical", "sli:burn-rate"],
        "schedule_interval": "1m",
        "esql": esql_burn_rate(window_minutes=5, threshold=10),
        "severity": "critical",
        "description": (
            "Error budget burn rate > 10x in 5-minute window. "
            "Risk of exhausting monthly budget in ~12h. Immediate action required."
        ),
    },
    # 2. Latency p95 Critical - 95th percentile > 500ms
    {
        "name": "[{prefix}] Latency p95 Critical".format(prefix=RULE_PREFIX),
        "tags": TAGS + ["severity:critical", "sli:latency-p95"],
        "schedule_interval": "1m",
        "esql": esql_latency_percentile(window_minutes=5, percentile=95, threshold_ms=500),
        "severity": "critical",
        "description": (
            "HTTP latency p95 > 500ms in last 5 minutes. "
            "Severe performance degradation detected."
        ),
    },
    # 3. Latency p99 Warning - 99th percentile > 1000ms
    {
        "name": "[{prefix}] Latency p99 Warning".format(prefix=RULE_PREFIX),
        "tags": TAGS + ["severity:warning", "sli:latency-p99"],
        "schedule_interval": "2m",
        "esql": esql_latency_percentile(window_minutes=5, percentile=99, threshold_ms=1000),
        "severity": "warning",
        "description": (
            "HTTP latency p99 > 1000ms in last 5 minutes. "
            "Tail latency is high. Investigate slow endpoints."
        ),
    },
    # 4. Error 5xx Rate Critical - 5xx error rate > 1%
    {
        "name": "[{prefix}] Error 5xx Rate Critical".format(prefix=RULE_PREFIX),
        "tags": TAGS + ["severity:critical", "sli:error-5xx"],
        "schedule_interval": "1m",
        "esql": esql_error_5xx_rate(window_minutes=5, threshold_fraction=0.01),
        "severity": "critical",
        "description": (
            "5xx error rate > 1% in last 5 minutes. "
            "Indicates systemic server failure. Check logs and dependencies."
        ),
    },
    # 5. DB Pool Saturation Critical - connection pool > 95% utilized
    {
        "name": "[{prefix}] DB Pool Saturation Critical".format(prefix=RULE_PREFIX),
        "tags": TAGS + ["severity:critical", "sli:db-saturation"],
        "schedule_interval": "1m",
        "esql": esql_db_pool_saturation(window_minutes=5, threshold_pct=95.0),
        "severity": "critical",
        "description": (
            "DB connection pool > 95% utilized in last 5 minutes. "
            "Imminent risk of connection exhaustion. Scale pool or optimize queries."
        ),
    },
    # 6. DB Pool Saturation Warning - connection pool > 80% utilized
    {
        "name": "[{prefix}] DB Pool Saturation Warning".format(prefix=RULE_PREFIX),
        "tags": TAGS + ["severity:warning", "sli:db-saturation"],
        "schedule_interval": "2m",
        "esql": esql_db_pool_saturation(window_minutes=10, threshold_pct=80.0),
        "severity": "warning",
        "description": (
            "DB connection pool > 80% utilized in last 10 minutes. "
            "Pool approaching capacity. Consider scaling or optimizing queries."
        ),
    },
]


# --------------------------------------------------------------------------- #
# Kibana API Helpers
# --------------------------------------------------------------------------- #

def kibana_request(kibana_url, method, path, body=None):
    """Make a request to Kibana API. Returns (status_code, response_body)."""
    url = "{base}{path}".format(base=kibana_url.rstrip("/"), path=path)
    headers = {
        "kbn-xsrf": "true",
        "Content-Type": "application/json",
    }
    data = json.dumps(body).encode("utf-8") if body else None
    req = urllib.request.Request(url, data=data, headers=headers, method=method)
    try:
        resp = urllib.request.urlopen(req, timeout=30)
        raw = resp.read().decode("utf-8")
        response_body = json.loads(raw) if raw.strip() else {}
        return resp.status, response_body
    except urllib.error.HTTPError as exc:
        raw = exc.read().decode("utf-8") if exc.fp else ""
        error_body = json.loads(raw) if raw.strip() else {}
        return exc.code, error_body
    except urllib.error.URLError as exc:
        print("  ERROR: Cannot connect to Kibana at {url}: {err}".format(
            url=kibana_url, err=exc.reason,
        ))
        return 0, {}


def ensure_server_log_connector(kibana_url):
    """Create or find the Server Log connector. Returns connector ID or None."""
    connector_name = "[{prefix}] Server Log".format(prefix=RULE_PREFIX)

    # Check if connector already exists
    status, body = kibana_request(kibana_url, "GET", "/api/actions/connectors")
    if status == 200 and isinstance(body, list):
        for c in body:
            if c.get("name") == connector_name:
                print("  Connector already exists: {id} ({name})".format(
                    id=c["id"], name=connector_name,
                ))
                return c["id"]

    # Create connector
    payload = {
        "name": connector_name,
        "connector_type_id": ".server-log",
        "config": {},
        "secrets": {},
    }
    status, body = kibana_request(
        kibana_url, "POST", "/api/actions/connector", payload,
    )
    if status in (200, 201):
        connector_id = body.get("id", "unknown")
        print("  Connector created: {id} ({name})".format(
            id=connector_id, name=connector_name,
        ))
        return connector_id
    else:
        print("  ERROR creating connector (HTTP {status}): {msg}".format(
            status=status, msg=json.dumps(body)[:200],
        ))
        return None


def build_rule_body(rule, connector_id=None):
    """Build the Kibana Rule API payload for an ES|QL rule."""
    actions = []
    if connector_id:
        alert_msg = (
            "ALERT: {name}\n"
            "Severity: {sev}\n"
            "{desc}"
        ).format(name=rule["name"], sev=rule["severity"], desc=rule["description"])

        actions.append({
            "group": "query matched",
            "id": connector_id,
            "params": {
                "message": alert_msg + "\n{{context.message}}",
                "level": "error" if rule["severity"] == "critical" else "warn",
            },
        })
        actions.append({
            "group": "recovered",
            "id": connector_id,
            "params": {
                "message": "RECOVERED: {name}".format(name=rule["name"]),
                "level": "info",
            },
        })

    return {
        "name": rule["name"],
        "rule_type_id": ".es-query",
        "consumer": "stackAlerts",
        "tags": rule["tags"],
        "schedule": {"interval": rule["schedule_interval"]},
        "notify_when": "onActionGroupChange",
        "params": {
            "searchType": "esqlQuery",
            "esqlQuery": {"esql": rule["esql"]},
            "timeField": TIME_FIELD,
            "threshold": [0],
            "thresholdComparator": ">",
            "timeWindowSize": 5,
            "timeWindowUnit": "m",
            "size": 1,
        },
        "actions": actions,
    }


def find_existing_rules(kibana_url):
    """Find existing auto-generated financial rules. Returns {name: rule_id}."""
    prefix_marker = "[{prefix}]".format(prefix=RULE_PREFIX)
    existing = {}
    page = 1
    while True:
        status, body = kibana_request(
            kibana_url,
            "GET",
            "/api/alerting/rules/_find?per_page=100&page={p}"
            "&search_fields=tags&search=financial".format(p=page),
        )
        if status != 200:
            break
        for rule in body.get("data", []):
            if (
                "auto-generated" in rule.get("tags", [])
                and rule["name"].startswith(prefix_marker)
            ):
                existing[rule["name"]] = rule["id"]
        if page * 100 >= body.get("total", 0):
            break
        page += 1
    return existing


# --------------------------------------------------------------------------- #
# Commands
# --------------------------------------------------------------------------- #

def cmd_dry_run():
    """Print all rules that would be created (no API calls)."""
    print("\n--- Dry Run: {count} rules would be created ---\n".format(
        count=len(RULES),
    ))
    for i, rule in enumerate(RULES, 1):
        body = build_rule_body(rule)
        print("{i}. {name}".format(i=i, name=rule["name"]))
        print("   Severity:  {sev}".format(sev=rule["severity"]))
        print("   Schedule:  every {interval}".format(interval=rule["schedule_interval"]))
        print("   Tags:      {tags}".format(tags=", ".join(rule["tags"])))
        print("   ES|QL:     {esql}".format(esql=rule["esql"][:120]))
        if len(rule["esql"]) > 120:
            print("              ...")
        print("   Description: {desc}".format(desc=rule["description"]))
        print()
    print("Run with --apply to create these rules in Kibana.")


def cmd_apply(kibana_url):
    """Create all alerting rules in Kibana."""
    print("\n--- Creating Alerting Rules ---")
    print("  Kibana URL: {url}".format(url=kibana_url))
    print("  Rules:      {count}".format(count=len(RULES)))
    print()

    # Ensure Server Log connector exists
    print("--- Setting up Server Log Connector ---")
    connector_id = ensure_server_log_connector(kibana_url)
    if not connector_id:
        print("\nWARNING: No connector created. Rules will have no actions.")
    print()

    # Find existing rules to avoid duplicates
    existing = find_existing_rules(kibana_url)
    if existing:
        print("  Found {count} existing financial rules".format(count=len(existing)))

    created = 0
    updated = 0
    errors = 0

    for rule in RULES:
        name = rule["name"]
        body = build_rule_body(rule, connector_id)

        if name in existing:
            # Delete and recreate
            rule_id = existing[name]
            kibana_request(kibana_url, "DELETE", "/api/alerting/rule/{id}".format(id=rule_id))
            status, resp = kibana_request(kibana_url, "POST", "/api/alerting/rule", body)
            if status in (200, 201):
                print("  UPDATED: {name}".format(name=name))
                updated += 1
            else:
                msg = resp.get("message", json.dumps(resp)[:200])
                print("  ERROR updating {name} (HTTP {status}): {msg}".format(
                    name=name, status=status, msg=msg,
                ))
                errors += 1
        else:
            status, resp = kibana_request(kibana_url, "POST", "/api/alerting/rule", body)
            if status in (200, 201):
                print("  CREATED: {name}".format(name=name))
                created += 1
            else:
                msg = resp.get("message", json.dumps(resp)[:200])
                print("  ERROR creating {name} (HTTP {status}): {msg}".format(
                    name=name, status=status, msg=msg,
                ))
                errors += 1

    print("\n  Summary: {c} created, {u} updated, {e} errors".format(
        c=created, u=updated, e=errors,
    ))
    if errors > 0:
        sys.exit(1)


def cmd_delete(kibana_url):
    """Delete all auto-generated financial rules."""
    print("\n--- Deleting All financial Rules ---")
    print("  Kibana URL: {url}".format(url=kibana_url))
    print()

    existing = find_existing_rules(kibana_url)
    if not existing:
        print("  No financial rules found.")
        return

    deleted = 0
    for name, rule_id in existing.items():
        status, _ = kibana_request(
            kibana_url, "DELETE", "/api/alerting/rule/{id}".format(id=rule_id),
        )
        if status in (200, 204):
            print("  DELETED: {name}".format(name=name))
            deleted += 1
        else:
            print("  ERROR deleting {name} (HTTP {status})".format(
                name=name, status=status,
            ))

    print("\n  Deleted {d}/{t} rules".format(d=deleted, t=len(existing)))

    # Also clean up the Server Log connector
    connector_name = "[{prefix}] Server Log".format(prefix=RULE_PREFIX)
    status, body = kibana_request(kibana_url, "GET", "/api/actions/connectors")
    if status == 200 and isinstance(body, list):
        for c in body:
            if c.get("name") == connector_name:
                del_status, _ = kibana_request(
                    kibana_url, "DELETE",
                    "/api/actions/connector/{id}".format(id=c["id"]),
                )
                if del_status in (200, 204):
                    print("  DELETED connector: {name} ({id})".format(
                        name=c["name"], id=c["id"],
                    ))


# --------------------------------------------------------------------------- #
# Main
# --------------------------------------------------------------------------- #

def main():
    parser = argparse.ArgumentParser(
        description="Generate Kibana alerting rules for financial-wallet.",
        formatter_class=argparse.RawDescriptionHelpFormatter,
        epilog=(
            "Examples:\n"
            "  python3 generate_rules.py                          # dry-run (default)\n"
            "  python3 generate_rules.py --apply                  # create rules\n"
            "  python3 generate_rules.py --delete                 # remove all rules\n"
            "  python3 generate_rules.py --apply --kibana-url http://kibana:5601\n"
        ),
    )

    group = parser.add_mutually_exclusive_group()
    group.add_argument(
        "--dry-run",
        action="store_true",
        default=True,
        help="Print rules that would be created without calling the API (default)",
    )
    group.add_argument(
        "--apply",
        action="store_true",
        help="Create rules via the Kibana Alerting API",
    )
    group.add_argument(
        "--delete",
        action="store_true",
        help="Delete all rules created by this script",
    )

    parser.add_argument(
        "--kibana-url",
        default="http://localhost:5601",
        help="Kibana base URL (default: http://localhost:5601)",
    )

    args = parser.parse_args()

    if args.delete:
        cmd_delete(args.kibana_url)
    elif args.apply:
        cmd_apply(args.kibana_url)
    else:
        cmd_dry_run()


if __name__ == "__main__":
    main()
