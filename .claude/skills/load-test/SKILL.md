---
name: load-test
description: k6 load testing with multiple scenarios
user-invocable: true
---

# /load-test [smoke|load|stress|spike|kind]

Runs k6 load tests and analyzes results.

## Scenarios

| Scenario | Command | Description |
|----------|---------|-------------|
| smoke | `make load-smoke` | Basic validation (1 VU, 30s) |
| load | `make load-test` | Progressive load (up to 50 VUs) |
| stress | `make load-stress` | Find limits (up to 200 VUs) |
| spike | `make load-spike` | Sudden burst (0 → 100 VUs) |
| kind | Run against Kind cluster | Test K8s deployment |

## Execution

1. Ensure target is running (`make dev` or `make kind-deploy`)
2. Run the appropriate scenario
3. Analyze results against thresholds

## Analysis

For each scenario, report:

| Metric | Value | Threshold | Status |
|--------|-------|-----------|--------|
| p95 latency | Xms | <500ms | PASS/FAIL |
| p99 latency | Xms | <1000ms | PASS/FAIL |
| Error rate | X% | <1% | PASS/FAIL |
| Throughput | X req/s | - | INFO |

## Cleanup

After load tests: `make load-clean` to remove test data.
