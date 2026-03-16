"""Custom tools for the incident triage agent."""

import json
import os
import time
from urllib.parse import quote as url_encode

import boto3
import requests
from strands import tool

# Shared boto3 clients — created once, reused across warm invocations.
_cw_client = boto3.client("cloudwatch")
_logs_client = boto3.client("logs")
_xray_client = boto3.client("xray")
_sm_client = boto3.client("secretsmanager")

LOG_GROUP = os.environ.get("LOG_GROUP_NAME", "/ecs/demo-order-api")
ALARM_NAME = os.environ.get("ALARM_NAME", "demo-incident-response-error-rate")
GIT_PROVIDER = os.environ.get("GIT_PROVIDER", "github")
GIT_REPO = os.environ["GIT_REPO"]
GIT_SECRET_NAME = os.environ.get("GIT_SECRET_NAME", "demo-incident-response/git-pat")
GITLAB_URL = os.environ.get("GITLAB_URL", "https://gitlab.com").rstrip("/")

_git_pat_cache: str | None = None


def _get_git_pat() -> str:
    """Retrieve the Git provider PAT from Secrets Manager, with simple caching."""
    global _git_pat_cache
    if _git_pat_cache is None:
        resp = _sm_client.get_secret_value(SecretId=GIT_SECRET_NAME)
        _git_pat_cache = resp["SecretString"]
    return _git_pat_cache


def _git_auth_headers(pat: str) -> dict:
    """Return the correct auth headers for the configured Git provider."""
    if GIT_PROVIDER == "gitlab":
        return {"PRIVATE-TOKEN": pat}
    return {
        "Authorization": f"Bearer {pat}",
        "Accept": "application/vnd.github+json",
    }


@tool
def describe_alarm() -> dict:
    """Retrieve details of the CloudWatch alarm that triggered this investigation, including its
    configuration, threshold, current state, and state change reason.

    Returns:
        Alarm details including name, threshold, state, and timestamps.
    """
    resp = _cw_client.describe_alarms(AlarmNames=[ALARM_NAME])

    alarms = resp.get("CompositeAlarms", []) + resp.get("MetricAlarms", [])
    if not alarms:
        return {"error": f"Alarm '{ALARM_NAME}' not found"}

    alarm = alarms[0]
    return {
        "alarm_name": alarm.get("AlarmName"),
        "alarm_description": alarm.get("AlarmDescription"),
        "state": alarm.get("StateValue"),
        "state_reason": alarm.get("StateReason"),
        "state_updated": alarm.get("StateUpdatedTimestamp", "").isoformat()
        if alarm.get("StateUpdatedTimestamp")
        else None,
        "threshold": alarm.get("Threshold"),
        "comparison_operator": alarm.get("ComparisonOperator"),
        "evaluation_periods": alarm.get("EvaluationPeriods"),
        "period": alarm.get("Period"),
        "metrics": [
            {
                "id": m.get("Id"),
                "expression": m.get("MetricStat", {}).get("Metric", {}).get("MetricName")
                or m.get("Expression"),
                "label": m.get("Label"),
            }
            for m in alarm.get("Metrics", [])
        ],
    }


@tool
def query_logs(query: str, minutes_ago: int = 15) -> dict:
    """Run a CloudWatch Logs Insights query against the application log group.

    Args:
        query: A CloudWatch Logs Insights query string. For example:
               'fields @timestamp, @message | filter level = "ERROR" | sort @timestamp desc | limit 20'
        minutes_ago: How far back to search, in minutes. Defaults to 15.

    Returns:
        Query results containing matching log entries.
    """
    end_time = int(time.time())
    start_time = end_time - (minutes_ago * 60)

    start_resp = _logs_client.start_query(
        logGroupName=LOG_GROUP,
        startTime=start_time,
        endTime=end_time,
        queryString=query,
    )
    query_id = start_resp["queryId"]

    # Poll for results.
    for _ in range(30):
        result = _logs_client.get_query_results(queryId=query_id)
        if result["status"] in ("Complete", "Failed", "Cancelled"):
            break
        time.sleep(1)

    if result["status"] != "Complete":
        return {"error": f"Query {result['status']}", "query_id": query_id}

    # Flatten results into a list of dicts.
    rows = []
    for entry in result.get("results", []):
        row = {field["field"]: field["value"] for field in entry if not field["field"].startswith("@ptr")}
        rows.append(row)

    return {
        "status": "Complete",
        "match_count": len(rows),
        "results": rows[:50],  # Cap at 50 to avoid token bloat.
        "statistics": result.get("statistics"),
    }


@tool
def get_metric_data(metric_queries: list[dict], minutes_ago: int = 30) -> dict:
    """Fetch CloudWatch metric data for one or more metric queries.

    Args:
        metric_queries: A list of metric query objects. Each must have:
            - id: A short lowercase identifier (e.g. "errors", "requests")
            - metric_name: The CloudWatch metric name (e.g. "ErrorCount", "RequestCount", "Latency")
            - namespace: The metric namespace (e.g. "DemoOrderAPI")
            - stat: The statistic to retrieve (e.g. "Sum", "Average", "p99")
            - dimensions: Optional list of {name, value} pairs
        minutes_ago: How far back to fetch data, in minutes. Defaults to 30.

    Returns:
        Metric data with timestamps and values for each query.
    """
    end_time = int(time.time())
    start_time = end_time - (minutes_ago * 60)

    queries = []
    for q in metric_queries:
        metric_stat = {
            "Metric": {
                "Namespace": q["namespace"],
                "MetricName": q["metric_name"],
            },
            "Period": 60,
            "Stat": q.get("stat", "Sum"),
        }
        if q.get("dimensions"):
            metric_stat["Metric"]["Dimensions"] = [
                {"Name": d["name"], "Value": d["value"]} for d in q["dimensions"]
            ]
        queries.append({"Id": q["id"], "MetricStat": metric_stat})

    resp = _cw_client.get_metric_data(
        MetricDataQueries=queries,
        StartTime=start_time,
        EndTime=end_time,
    )

    results = {}
    for series in resp.get("MetricDataResults", []):
        results[series["Id"]] = {
            "label": series.get("Label"),
            "values": [
                {"timestamp": ts.isoformat(), "value": val}
                for ts, val in zip(series.get("Timestamps", []), series.get("Values", []))
            ],
            "status_code": series.get("StatusCode"),
        }

    return results


@tool
def get_xray_traces(minutes_ago: int = 15) -> dict:
    """Retrieve recent X-Ray traces that have errors or faults, showing the full call chain
    of failing requests.

    Args:
        minutes_ago: How far back to search for traces, in minutes. Defaults to 15.

    Returns:
        A list of failing trace summaries with segment details.
    """
    end_time = time.time()
    start_time = end_time - (minutes_ago * 60)

    summaries_resp = _xray_client.get_trace_summaries(
        StartTime=start_time,
        EndTime=end_time,
        FilterExpression="fault = true OR error = true",
    )

    summaries = summaries_resp.get("TraceSummaries", [])
    if not summaries:
        return {"trace_count": 0, "traces": []}

    # Fetch full traces for the first 5.
    trace_ids = [s["Id"] for s in summaries[:5]]
    traces_resp = _xray_client.batch_get_traces(TraceIds=trace_ids)

    traces = []
    for trace in traces_resp.get("Traces", []):
        segments = []
        for seg in trace.get("Segments", []):
            doc = json.loads(seg.get("Document", "{}"))
            segment_info = {
                "name": doc.get("name"),
                "http": doc.get("http"),
                "fault": doc.get("fault"),
                "error": doc.get("error"),
                "cause": doc.get("cause"),
            }
            # Include subsegments if present.
            subsegments = doc.get("subsegments", [])
            if subsegments:
                segment_info["subsegments"] = [
                    {
                        "name": ss.get("name"),
                        "fault": ss.get("fault"),
                        "error": ss.get("error"),
                        "cause": ss.get("cause"),
                    }
                    for ss in subsegments
                    if ss.get("fault") or ss.get("error")
                ]
            segments.append(segment_info)
        traces.append({"id": trace.get("Id"), "segments": segments})

    return {
        "trace_count": len(summaries),
        "total_available": len(summaries),
        "traces": traces,
    }


@tool
def get_source_file(file_path: str) -> str:
    """Read a source file from the repository. Use this to inspect code when logs or
    traces point to a specific file and line number.

    Args:
        file_path: Path to the file relative to the repository root.
                   For app code, prefix with 'demo-order-api/' (e.g. 'demo-order-api/internal/discount/discount.go').

    Returns:
        The file contents as text.
    """
    pat = _get_git_pat()
    headers = _git_auth_headers(pat)

    if GIT_PROVIDER == "gitlab":
        encoded_path = url_encode(file_path, safe="")
        url = f"{GITLAB_URL}/api/v4/projects/{GIT_REPO}/repository/files/{encoded_path}/raw"
    else:
        headers["Accept"] = "application/vnd.github.raw+json"
        url = f"https://api.github.com/repos/{GIT_REPO}/contents/{file_path}"

    resp = requests.get(url, headers=headers, params={"ref": "main"}, timeout=10)

    if resp.status_code == 404:
        return f"File not found: {file_path}"
    resp.raise_for_status()
    return resp.text


@tool
def create_issue(title: str, body: str, labels: list[str] | None = None) -> dict:
    """Create an issue with the RCA findings. This should be the final step after
    completing the investigation.

    Args:
        title: A concise issue title describing the incident.
        body: The full issue body in Markdown, following the RCA format from the system prompt.
        labels: Optional list of labels to apply (e.g. ["bug", "incident", "P1"]).

    Returns:
        The created issue URL and number.
    """
    pat = _get_git_pat()
    headers = _git_auth_headers(pat)

    if GIT_PROVIDER == "gitlab":
        url = f"{GITLAB_URL}/api/v4/projects/{GIT_REPO}/issues"
        payload = {"title": title, "description": body}
        if labels:
            payload["labels"] = ",".join(labels)
    else:
        url = f"https://api.github.com/repos/{GIT_REPO}/issues"
        payload = {"title": title, "body": body}
        if labels:
            payload["labels"] = labels

    resp = requests.post(url, headers=headers, json=payload, timeout=10)
    resp.raise_for_status()

    issue = resp.json()
    if GIT_PROVIDER == "gitlab":
        return {"issue_number": issue["iid"], "url": issue["web_url"]}
    return {"issue_number": issue["number"], "url": issue["html_url"]}
