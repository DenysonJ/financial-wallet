#!/usr/bin/env python3
"""
Kibana Dashboard Generator — Go Financial Wallet

Generates a kibana dashboard NDJSON file with:
- 3 Data Views (otel-traces-*, otel-logs-*, otel-metrics-*)
- 1 Dashboard with ~16 panels organized in 6 sections

Based on the people-service-registry dashboard generator pattern.

Usage:
    python3 generate_dashboard.py > dashboard.ndjson
    python3 generate_dashboard.py --service-name my-service > dashboard.ndjson

Sections (visual order):
    1. SLO Overview        — Availability, error rate, p95 latency, requests/s
    2. HTTP Overview       — Requests by status code, latency percentiles, top endpoints
    3. Cache Redis         — Hit rate, hits vs misses, operation duration
    4. DB Pool             — Open connections, idle connections, pool saturation
    5. Logs                — Log volume by level, recent errors table
    6. (reserved)          — Future expansion

Field Name Notes (OTel Collector -> Elasticsearch):
    - Metrics (otel-metrics-*): OTel Collector writes counter/histogram fields as
      top-level (e.g. http.server.duration, http.server.request.count).
      Attributes: http.response.status_code, http.route, http.request.method.
    - Logs (otel-logs-*): severity_text, body, resource attributes.

Requirements:
    - Python 3.6+ (stdlib only, no pip dependencies)
    - Kibana 8.x
"""

import argparse
import hashlib
import json
import os
import sys
from datetime import datetime, timezone

# ============================================================================
# Configuration
# ============================================================================

METRICS_DV_ID = "financial-metrics-dataview"
LOGS_DV_ID = "financial-logs-dataview"
TRACES_DV_ID = "financial-traces-dataview"

METRICS_INDEX = os.environ.get("METRICS_INDEX", "otel-metrics-*")
LOGS_INDEX = os.environ.get("LOGS_INDEX", "otel-logs-*")
TRACES_INDEX = os.environ.get("TRACES_INDEX", "otel-traces-*")

DASHBOARD_ID = "financial-dashboard"
DASHBOARD_TITLE = "Go financial — Observability Dashboard"

NOW = datetime.now(timezone.utc).strftime("%Y-%m-%dT%H:%M:%S.000Z")


# ============================================================================
# Helpers
# ============================================================================

def uid(prefix):
    """Generate a deterministic UUID-like ID from a prefix string."""
    h = hashlib.md5(prefix.encode()).hexdigest()
    return f"{h[:8]}-{h[8:12]}-{h[12:16]}-{h[16:20]}-{h[20:32]}"


def ndjson_line(obj):
    """Serialize object to compact JSON (one NDJSON line)."""
    return json.dumps(obj, ensure_ascii=False, separators=(",", ":"))


# ============================================================================
# Data View (Index Pattern) Builder
# ============================================================================

def make_data_view(dv_id, name, title):
    """Create a Kibana data view (index pattern) saved object."""
    return {
        "attributes": {
            "allowHidden": False,
            "fieldAttrs": "{}",
            "fieldFormatMap": "{}",
            "fields": "[]",
            "name": name,
            "runtimeFieldMap": "{}",
            "sourceFilters": "[]",
            "timeFieldName": "@timestamp",
            "title": title,
        },
        "coreMigrationVersion": "8.8.0",
        "created_at": NOW,
        "id": dv_id,
        "managed": False,
        "references": [],
        "type": "index-pattern",
        "typeMigrationVersion": "8.0.0",
        "updated_at": NOW,
        "version": "WzEsMV0=",
    }


# ============================================================================
# Column Builders (Lens datasource layer columns)
# ============================================================================

def col_count(label, kql_filter=None):
    col = {
        "label": label,
        "dataType": "number",
        "operationType": "count",
        "sourceField": "___records___",
        "isBucketed": False,
        "scale": "ratio",
        "params": {"emptyAsNull": True},
        "customLabel": True,
    }
    if kql_filter:
        col["filter"] = {"query": kql_filter, "language": "kuery"}
    return col


def col_last_value(label, source_field):
    return {
        "label": label,
        "dataType": "number",
        "operationType": "last_value",
        "sourceField": source_field,
        "isBucketed": False,
        "scale": "ratio",
        "params": {
            "emptyAsNull": True,
            "sortField": "@timestamp",
            "showArrayValues": False,
        },
        "filter": {"query": f"{source_field}: *", "language": "kuery"},
        "customLabel": True,
    }


def col_sum(label, source_field, kql_filter=None):
    col = {
        "label": label,
        "dataType": "number",
        "operationType": "sum",
        "sourceField": source_field,
        "isBucketed": False,
        "scale": "ratio",
        "params": {"emptyAsNull": True},
        "customLabel": True,
    }
    if kql_filter:
        col["filter"] = {"query": kql_filter, "language": "kuery"}
    return col


def col_avg(label, source_field):
    return {
        "label": label,
        "dataType": "number",
        "operationType": "average",
        "sourceField": source_field,
        "isBucketed": False,
        "scale": "ratio",
        "params": {"emptyAsNull": True},
        "customLabel": True,
    }


def col_percentile(label, source_field, percentile):
    return {
        "label": label,
        "dataType": "number",
        "operationType": "percentile",
        "sourceField": source_field,
        "isBucketed": False,
        "scale": "ratio",
        "params": {
            "emptyAsNull": True,
            "percentile": percentile,
        },
        "customLabel": True,
    }


def col_terms(label, source_field, size=5, order_by_col=None, order="desc"):
    order_by = (
        {"type": "column", "columnId": order_by_col}
        if order_by_col
        else {"type": "alphabetical"}
    )
    return {
        "label": label,
        "dataType": "string",
        "operationType": "terms",
        "sourceField": source_field,
        "isBucketed": True,
        "scale": "ordinal",
        "params": {
            "size": size,
            "orderBy": order_by,
            "orderDirection": order,
            "otherBucket": True,
            "missingBucket": False,
        },
        "customLabel": True,
    }


def col_date_histogram(label="@timestamp", source_field="@timestamp"):
    return {
        "label": label,
        "dataType": "date",
        "operationType": "date_histogram",
        "sourceField": source_field,
        "isBucketed": True,
        "scale": "interval",
        "params": {
            "interval": "auto",
            "includeEmptyRows": True,
            "dropPartials": False,
        },
    }


# ============================================================================
# Formula Decomposition Helpers
# ============================================================================
# Kibana 8.x does NOT auto-rehydrate formula columns on import.
# Each formula must be manually decomposed into: leaf metrics -> math -> formula.

def _sub(main_id, index):
    """Sub-column ID: mainIdX0, mainIdX1, ..."""
    return f"{main_id}X{index}"


def _leaf(label, op_type, source_field, kql_filter=None):
    """Leaf metric column for formula decomposition."""
    col = {
        "label": label,
        "dataType": "number",
        "operationType": op_type,
        "sourceField": source_field,
        "isBucketed": False,
        "scale": "ratio",
        "params": {"emptyAsNull": False},
        "customLabel": True,
    }
    if op_type == "last_value":
        col["params"]["sortField"] = "@timestamp"
        col["params"]["showArrayValues"] = False
        exists_kql = f"{source_field}: *"
        if kql_filter:
            col["filter"] = {"query": f"({exists_kql}) AND ({kql_filter})", "language": "kuery"}
        else:
            col["filter"] = {"query": exists_kql, "language": "kuery"}
        return col
    if kql_filter:
        col["filter"] = {"query": kql_filter, "language": "kuery"}
    return col


def _math(label, ast, refs):
    """Math column (intermediate) for formula decomposition."""
    return {
        "label": label,
        "dataType": "number",
        "operationType": "math",
        "isBucketed": False,
        "scale": "ratio",
        "params": {"tinymathAst": ast},
        "references": refs,
        "customLabel": True,
    }


def _froot(label, formula_text, ref_id):
    """Formula root column with single reference."""
    return {
        "label": label,
        "dataType": "number",
        "operationType": "formula",
        "isBucketed": False,
        "scale": "ratio",
        "params": {
            "formula": formula_text,
            "isFormulaBroken": False,
        },
        "references": [ref_id],
        "customLabel": True,
    }


def _ast(name, args, text=""):
    """Build a TinyMath AST node."""
    return {
        "type": "function",
        "name": name,
        "args": args,
        "location": {"min": 0, "max": len(text)},
        "text": text,
    }


def formula_count_ratio(cid, label, formula_text, kql_filter, den_filter=None):
    """count(kql='...') / count(kql='...') -> decomposed columns dict."""
    x0, x1, x2 = _sub(cid, 0), _sub(cid, 1), _sub(cid, 2)
    p = f"Part of {label}"
    return {
        x0: _leaf(p, "count", "___records___", kql_filter),
        x1: _leaf(p, "count", "___records___", den_filter),
        x2: _math(p, _ast("divide", [x0, x1], formula_text), [x0, x1]),
        cid: _froot(label, formula_text, x2),
    }


def formula_last_value_ratio(cid, label, formula_text, num_field, den_field):
    """last_value(A) / last_value(B) -> decomposed. Correct for gauge ratios."""
    x0, x1, x2 = _sub(cid, 0), _sub(cid, 1), _sub(cid, 2)
    p = f"Part of {label}"
    return {
        x0: _leaf(p, "last_value", num_field),
        x1: _leaf(p, "last_value", den_field),
        x2: _math(p, _ast("divide", [x0, x1], formula_text), [x0, x1]),
        cid: _froot(label, formula_text, x2),
    }


def formula_hit_rate(cid, label, formula_text, hits_field, misses_field):
    """(sum(hits) / (sum(hits) + sum(misses))) * 100 -> decomposed hit rate %."""
    x0, x1, x2, x2g, x3, x4 = [_sub(cid, i) for i in range(6)]
    p = f"Part of {label}"
    return {
        x0: _leaf(p, "sum", hits_field),
        x1: _leaf(p, "sum", misses_field),
        x2: _math(p, _ast("add", [x0, x1]), [x0, x1]),
        x2g: _math(p, _ast("max", [x2, 1]), [x2]),
        x3: _math(p, _ast("divide", [x0, x2g]), [x0, x2g]),
        x4: _math(p, _ast("multiply", [x3, 100], formula_text), [x3]),
        cid: _froot(label, formula_text, x4),
    }


# ============================================================================
# Panel Builders
# ============================================================================

COLOR_MAPPING_DEFAULT = {
    "assignments": [],
    "specialAssignments": [
        {"rule": {"type": "other"}, "color": {"type": "loop"}, "touched": False}
    ],
    "paletteId": "eui_amsterdam_color_blind",
    "colorMode": {"type": "categorical"},
}


def _base_state(layer_id, columns, index_pattern_id, query=""):
    """Build the common Lens state object."""
    return {
        "visualization": {},
        "query": {"query": query, "language": "kuery"},
        "filters": [],
        "datasourceStates": {
            "formBased": {
                "layers": {
                    layer_id: {
                        "columns": columns,
                        "columnOrder": list(columns.keys()),
                        "sampling": 1,
                        "ignoreGlobalFilters": False,
                        "incompleteColumns": {},
                        "indexPatternId": index_pattern_id,
                    }
                },
                "currentIndexPatternId": index_pattern_id,
            },
            "indexpattern": {"layers": {}},
            "textBased": {"layers": {}},
        },
        "internalReferences": [],
        "adHocDataViews": {},
    }


def _wrap_panel(title, viz_type, panel_id, layer_id, state, index_pattern_id, grid):
    """Wrap a Lens state into a full panel object."""
    return {
        "type": "lens",
        "title": title,
        "embeddableConfig": {
            "attributes": {
                "title": "",
                "visualizationType": viz_type,
                "type": "lens",
                "references": [
                    {
                        "type": "index-pattern",
                        "id": index_pattern_id,
                        "name": f"indexpattern-datasource-layer-{layer_id}",
                    }
                ],
                "state": state,
            },
            "enhancements": {},
        },
        "panelIndex": panel_id,
        "gridData": {**grid, "i": panel_id},
    }


def panel_metric(title, panel_id, layer_id, columns, ip_id, grid, metric_col_id, query=""):
    state = _base_state(layer_id, columns, ip_id, query)
    state["visualization"] = {
        "layerId": layer_id,
        "layerType": "data",
        "metricAccessor": metric_col_id,
        "titlesTextAlign": "left",
        "valuesTextAlign": "center",
        "valueFontMode": "fit",
    }
    return _wrap_panel(title, "lnsMetric", panel_id, layer_id, state, ip_id, grid)


def panel_xy(title, panel_id, layer_id, columns, ip_id, grid,
             series_type, x_accessor, y_accessors, split_accessor=None,
             query="", y_title=None):
    state = _base_state(layer_id, columns, ip_id, query)
    layer = {
        "seriesType": series_type,
        "layerId": layer_id,
        "layerType": "data",
        "xAccessor": x_accessor,
        "accessors": y_accessors,
        "colorMapping": COLOR_MAPPING_DEFAULT,
    }
    if split_accessor:
        layer["splitAccessor"] = split_accessor

    viz = {
        "legend": {"isVisible": True, "position": "right"},
        "valueLabels": "hide",
        "fittingFunction": "None",
        "axisTitlesVisibilitySettings": {"x": True, "yLeft": True, "yRight": True},
        "tickLabelsVisibilitySettings": {"x": True, "yLeft": True, "yRight": True},
        "labelsOrientation": {"x": 0, "yLeft": 0, "yRight": 0},
        "gridlinesVisibilitySettings": {"x": True, "yLeft": True, "yRight": True},
        "preferredSeriesType": series_type,
        "layers": [layer],
    }
    if y_title:
        viz["yTitle"] = y_title

    state["visualization"] = viz
    return _wrap_panel(title, "lnsXY", panel_id, layer_id, state, ip_id, grid)


def panel_datatable(title, panel_id, layer_id, columns, ip_id, grid, column_configs, query=""):
    state = _base_state(layer_id, columns, ip_id, query)
    state["visualization"] = {
        "layerId": layer_id,
        "layerType": "data",
        "columns": column_configs,
    }
    return _wrap_panel(title, "lnsDatatable", panel_id, layer_id, state, ip_id, grid)


def panel_section_title(text, panel_id, grid):
    """Create a Markdown panel used as a visual section header."""
    return {
        "type": "visualization",
        "panelIndex": panel_id,
        "gridData": {**grid, "i": panel_id},
        "title": "",
        "embeddableConfig": {
            "savedVis": {
                "id": "",
                "title": "",
                "description": "",
                "type": "markdown",
                "params": {
                    "fontSize": 14,
                    "openLinksInNewTab": False,
                    "markdown": text,
                },
                "uiState": {},
                "data": {
                    "aggs": [],
                    "searchSource": {
                        "query": {"query": "", "language": "kuery"},
                        "filter": [],
                    },
                },
            },
            "enhancements": {},
        },
    }


# ============================================================================
# Dashboard Builder
# ============================================================================

class DashboardBuilder:
    def __init__(self, service_name):
        self.panels = []
        self.refs = []
        self.service_name = service_name

    def add(self, panel, ip_id, layer_id):
        self.panels.append(panel)
        ref_name = f"indexpattern-datasource-layer-{layer_id}"
        self.refs.append({
            "id": ip_id,
            "name": f"{panel['panelIndex']}:{ref_name}",
            "type": "index-pattern",
        })

    def add_raw(self, panel):
        """Add a panel without index-pattern reference (e.g., markdown)."""
        self.panels.append(panel)

    def build_dashboard(self):
        return {
            "attributes": {
                "controlGroupInput": {
                    "chainingSystem": "HIERARCHICAL",
                    "controlStyle": "oneLine",
                    "ignoreParentSettingsJSON": json.dumps({
                        "ignoreFilters": False,
                        "ignoreQuery": False,
                        "ignoreTimerange": False,
                        "ignoreValidations": False,
                    }),
                    "panelsJSON": "{}",
                },
                "description": f"Observability dashboard for {self.service_name} — Go Financial Wallet",
                "hits": 0,
                "kibanaSavedObjectMeta": {
                    "searchSourceJSON": json.dumps({
                        "query": {"query": "", "language": "kuery"},
                        "filter": [],
                    })
                },
                "optionsJSON": json.dumps({
                    "useMargins": True,
                    "syncColors": False,
                    "syncCursor": True,
                    "syncTooltips": False,
                    "hidePanelTitles": False,
                }),
                "panelsJSON": json.dumps(self.panels),
                "timeRestore": True,
                "timeTo": "now",
                "timeFrom": "now-1h",
                "refreshInterval": {"pause": False, "value": 30000},
                "title": DASHBOARD_TITLE,
                "version": 3,
            },
            "coreMigrationVersion": "8.8.0",
            "created_at": NOW,
            "id": DASHBOARD_ID,
            "managed": False,
            "references": self.refs,
            "type": "dashboard",
            "typeMigrationVersion": "8.9.0",
            "updated_at": NOW,
            "version": "WzEsMV0=",
        }


# ============================================================================
# Build all panels
# ============================================================================

def build_panels(db):
    """Build all dashboard panels across 6 sections."""
    IP = METRICS_DV_ID
    LP = LOGS_DV_ID
    TITLE_H = 3  # Height of each section title panel

    # Section content base y-positions (after each section's own title)
    S1_Y = TITLE_H                        # SLO Overview
    S2_Y = S1_Y + 8 + TITLE_H             # HTTP Overview
    S3_Y = S2_Y + 16 + TITLE_H            # Cache Redis
    S4_Y = S3_Y + 16 + TITLE_H            # DB Pool
    S5_Y = S4_Y + 16 + TITLE_H            # Logs

    # ================================================================
    # SECTION 1: SLO Overview (4 metric panels in a row)
    # ================================================================

    db.add_raw(panel_section_title(
        "## SLO Overview",
        uid("section-slo"), {"x": 0, "y": 0, "w": 48, "h": TITLE_H},
    ))

    # --- Availability % ---
    # Formula: 1 - (count of 5xx / total count)
    # Denominator filters to HTTP docs only (status_code exists)
    http_exists = "http.response.status_code: *"
    lid, pid = uid("slo-avail-l"), uid("slo-avail-p")
    cid = uid("slo-avail-c")
    db.add(
        panel_metric(
            "Availability %", pid, lid,
            formula_count_ratio(
                cid, "Availability",
                f"count(kql='http.response.status_code < 500') / count(kql='{http_exists}')",
                "http.response.status_code < 500",
                den_filter=http_exists,
            ),
            IP, {"x": 0, "y": S1_Y, "w": 12, "h": 8}, cid,
        ),
        IP, lid,
    )

    # --- Error Rate % ---
    lid, pid = uid("slo-err-l"), uid("slo-err-p")
    cid = uid("slo-err-c")
    db.add(
        panel_metric(
            "Error Rate % (5xx)", pid, lid,
            formula_count_ratio(
                cid, "Error Rate",
                f"count(kql='http.response.status_code >= 500') / count(kql='{http_exists}')",
                "http.response.status_code >= 500",
                den_filter=http_exists,
            ),
            IP, {"x": 12, "y": S1_Y, "w": 12, "h": 8}, cid,
        ),
        IP, lid,
    )

    # --- p95 Latency ---
    lid, pid = uid("slo-p95-l"), uid("slo-p95-p")
    cid = uid("slo-p95-c")
    db.add(
        panel_metric(
            "p95 Latency (ms)", pid, lid,
            {cid: col_percentile("p95", "http.server.duration", 95)},
            IP, {"x": 24, "y": S1_Y, "w": 12, "h": 8}, cid,
        ),
        IP, lid,
    )

    # --- Requests/s ---
    lid, pid = uid("slo-rps-l"), uid("slo-rps-p")
    cid = uid("slo-rps-c")
    db.add(
        panel_metric(
            "Requests/s", pid, lid,
            {cid: col_count("Requests", http_exists)},
            IP, {"x": 36, "y": S1_Y, "w": 12, "h": 8}, cid,
        ),
        IP, lid,
    )

    # ================================================================
    # SECTION 2: HTTP Overview (3 panels)
    # ================================================================

    db.add_raw(panel_section_title(
        "## HTTP Overview",
        uid("section-http"), {"x": 0, "y": S2_Y - TITLE_H, "w": 48, "h": TITLE_H},
    ))

    # --- Requests by Status Code (bar chart over time) ---
    lid, pid = uid("http-by-status-l"), uid("http-by-status-p")
    ts_col = uid("http-by-status-ts")
    count_col = uid("http-by-status-cnt")
    split_col = uid("http-by-status-split")
    db.add(
        panel_xy(
            "Requests by Status Code", pid, lid,
            {
                ts_col: col_date_histogram(),
                count_col: col_count("Count"),
                split_col: col_terms("Status Code", "http.response.status_code", size=10, order_by_col=count_col),
            },
            IP, {"x": 0, "y": S2_Y, "w": 16, "h": 16},
            "bar_stacked", ts_col, [count_col], split_accessor=split_col,
            y_title="Count",
        ),
        IP, lid,
    )

    # --- Latency p50/p95/p99 (line chart) ---
    lid, pid = uid("http-latency-l"), uid("http-latency-p")
    ts_col = uid("http-latency-ts")
    p50_col = uid("http-latency-p50")
    p95_col = uid("http-latency-p95")
    p99_col = uid("http-latency-p99")
    db.add(
        panel_xy(
            "Latency Percentiles (p50/p95/p99)", pid, lid,
            {
                ts_col: col_date_histogram(),
                p50_col: col_percentile("p50", "http.server.duration", 50),
                p95_col: col_percentile("p95", "http.server.duration", 95),
                p99_col: col_percentile("p99", "http.server.duration", 99),
            },
            IP, {"x": 16, "y": S2_Y, "w": 16, "h": 16},
            "line", ts_col, [p50_col, p95_col, p99_col],
            y_title="ms",
        ),
        IP, lid,
    )

    # --- Top Endpoints by Count (table) ---
    lid, pid = uid("http-top-endpoints-l"), uid("http-top-endpoints-p")
    route_col = uid("http-top-endpoints-route")
    count_col = uid("http-top-endpoints-cnt")
    avg_col = uid("http-top-endpoints-avg")
    db.add(
        panel_datatable(
            "Top Endpoints by Request Count", pid, lid,
            {
                route_col: col_terms("Endpoint", "http.route", size=15, order_by_col=count_col),
                count_col: col_count("Count"),
                avg_col: col_avg("Avg Duration (ms)", "http.server.duration"),
            },
            IP, {"x": 32, "y": S2_Y, "w": 16, "h": 16},
            [
                {"columnId": route_col, "isTransposed": False},
                {"columnId": count_col, "isTransposed": False},
                {"columnId": avg_col, "isTransposed": False},
            ],
        ),
        IP, lid,
    )

    # ================================================================
    # SECTION 3: Cache Redis (3 panels)
    # ================================================================

    db.add_raw(panel_section_title(
        "## Cache Redis",
        uid("section-cache"), {"x": 0, "y": S3_Y - TITLE_H, "w": 48, "h": TITLE_H},
    ))

    # --- Cache Hit Rate % ---
    lid, pid = uid("cache-hitrate-l"), uid("cache-hitrate-p")
    cid = uid("cache-hitrate-c")
    db.add(
        panel_metric(
            "Cache Hit Rate %", pid, lid,
            formula_hit_rate(
                cid, "Hit Rate",
                "sum(cache_hits) / (sum(cache_hits) + sum(cache_misses)) * 100",
                "cache_hits", "cache_misses",
            ),
            IP, {"x": 0, "y": S3_Y, "w": 12, "h": 16}, cid,
        ),
        IP, lid,
    )

    # --- Hits vs Misses over time (line) ---
    lid, pid = uid("cache-hitsmiss-l"), uid("cache-hitsmiss-p")
    ts_col = uid("cache-hitsmiss-ts")
    hits_col = uid("cache-hitsmiss-hits")
    misses_col = uid("cache-hitsmiss-misses")
    db.add(
        panel_xy(
            "Cache Hits vs Misses", pid, lid,
            {
                ts_col: col_date_histogram(),
                hits_col: col_sum("Hits", "cache_hits"),
                misses_col: col_sum("Misses", "cache_misses"),
            },
            IP, {"x": 12, "y": S3_Y, "w": 18, "h": 16},
            "line", ts_col, [hits_col, misses_col],
            y_title="Count",
        ),
        IP, lid,
    )

    # --- Cache Operation Duration (line) ---
    lid, pid = uid("cache-duration-l"), uid("cache-duration-p")
    ts_col = uid("cache-duration-ts")
    avg_col = uid("cache-duration-avg")
    p95_col = uid("cache-duration-p95")
    db.add(
        panel_xy(
            "Cache Operation Duration", pid, lid,
            {
                ts_col: col_date_histogram(),
                avg_col: col_avg("Avg", "cache_operation_duration"),
                p95_col: col_percentile("p95", "cache_operation_duration", 95),
            },
            IP, {"x": 30, "y": S3_Y, "w": 18, "h": 16},
            "line", ts_col, [avg_col, p95_col],
            y_title="ms",
        ),
        IP, lid,
    )

    # ================================================================
    # SECTION 4: DB Pool (3 panels)
    # ================================================================

    db.add_raw(panel_section_title(
        "## DB Pool",
        uid("section-dbpool"), {"x": 0, "y": S4_Y - TITLE_H, "w": 48, "h": TITLE_H},
    ))

    # --- Open Connections (line) ---
    lid, pid = uid("db-open-l"), uid("db-open-p")
    ts_col = uid("db-open-ts")
    open_col = uid("db-open-val")
    db.add(
        panel_xy(
            "DB Open Connections", pid, lid,
            {
                ts_col: col_date_histogram(),
                open_col: col_last_value("Open", "db_pool_open_connections"),
            },
            IP, {"x": 0, "y": S4_Y, "w": 16, "h": 16},
            "line", ts_col, [open_col],
            y_title="Connections",
        ),
        IP, lid,
    )

    # --- Idle Connections (line) ---
    lid, pid = uid("db-idle-l"), uid("db-idle-p")
    ts_col = uid("db-idle-ts")
    idle_col = uid("db-idle-val")
    db.add(
        panel_xy(
            "DB Idle Connections", pid, lid,
            {
                ts_col: col_date_histogram(),
                idle_col: col_last_value("Idle", "db_pool_idle_connections"),
            },
            IP, {"x": 16, "y": S4_Y, "w": 16, "h": 16},
            "line", ts_col, [idle_col],
            y_title="Connections",
        ),
        IP, lid,
    )

    # --- Pool Saturation % (metric) ---
    # in_use / max_open * 100
    lid, pid = uid("db-saturation-l"), uid("db-saturation-p")
    cid = uid("db-saturation-c")
    db.add(
        panel_metric(
            "DB Pool Saturation %", pid, lid,
            formula_last_value_ratio(
                cid, "Saturation",
                "last_value(db_pool_in_use_connections) / last_value(db_pool_max_open_connections)",
                "db_pool_in_use_connections", "db_pool_max_open_connections",
            ),
            IP, {"x": 32, "y": S4_Y, "w": 16, "h": 16}, cid,
        ),
        IP, lid,
    )

    # ================================================================
    # SECTION 5: Logs (2 panels)
    # ================================================================

    db.add_raw(panel_section_title(
        "## Logs",
        uid("section-logs"), {"x": 0, "y": S5_Y - TITLE_H, "w": 48, "h": TITLE_H},
    ))

    # --- Log Volume by Level (stacked bar) ---
    lid, pid = uid("log-volume-l"), uid("log-volume-p")
    ts_col = uid("log-volume-ts")
    count_col = uid("log-volume-cnt")
    level_col = uid("log-volume-level")
    db.add(
        panel_xy(
            "Log Volume by Level", pid, lid,
            {
                ts_col: col_date_histogram(),
                count_col: col_count("Count"),
                level_col: col_terms("Level", "severity_text", size=6, order_by_col=count_col),
            },
            LP, {"x": 0, "y": S5_Y, "w": 24, "h": 16},
            "bar_stacked", ts_col, [count_col], split_accessor=level_col,
            y_title="Log Count",
        ),
        LP, lid,
    )

    # --- Recent Errors Table ---
    lid, pid = uid("log-errors-l"), uid("log-errors-p")
    ts_col_err = uid("log-errors-ts")
    count_col_err = uid("log-errors-cnt")
    body_col = uid("log-errors-body")
    db.add(
        panel_datatable(
            "Recent Errors (ERROR/FATAL)", pid, lid,
            {
                body_col: col_terms("Message", "body", size=20, order_by_col=count_col_err),
                count_col_err: col_count("Count"),
            },
            LP, {"x": 24, "y": S5_Y, "w": 24, "h": 16},
            [
                {"columnId": body_col, "isTransposed": False},
                {"columnId": count_col_err, "isTransposed": False},
            ],
            query='severity_text: "ERROR" OR severity_text: "FATAL"',
        ),
        LP, lid,
    )


# ============================================================================
# Main
# ============================================================================

def main():
    parser = argparse.ArgumentParser(
        description="Generate Kibana NDJSON dashboard for Go Financial Wallet"
    )
    parser.add_argument(
        "--service-name",
        default=os.environ.get("OTEL_SERVICE_NAME", "entity-service"),
        help="Service name for dashboard description (default: from OTEL_SERVICE_NAME or 'entity-service')",
    )
    args = parser.parse_args()

    objects = []

    # 1. Data Views
    objects.append(make_data_view(METRICS_DV_ID, "OTel Metrics", METRICS_INDEX))
    objects.append(make_data_view(LOGS_DV_ID, "OTel Logs", LOGS_INDEX))
    objects.append(make_data_view(TRACES_DV_ID, "OTel Traces", TRACES_INDEX))

    # 2. Build all panels into a dashboard
    db = DashboardBuilder(args.service_name)
    build_panels(db)

    # 3. Dashboard object
    objects.append(db.build_dashboard())

    # 4. Write NDJSON to stdout
    for obj in objects:
        print(ndjson_line(obj))


if __name__ == "__main__":
    main()
