"""Observer callback handler — writes agent triage events to DynamoDB for real-time display."""

import json
import logging
import os
import time
from decimal import Decimal

import boto3

logger = logging.getLogger(__name__)

_ddb = boto3.resource("dynamodb")
_table = _ddb.Table(os.environ.get("OBSERVER_TABLE_NAME", "demo-agent-events"))

# TTL: 1 hour from now.
_TTL_SECONDS = 3600


class ObserverCallbackHandler:
    """Strands callback handler that emits structured events to DynamoDB.

    Usage:
        cb = ObserverCallbackHandler()
        cb.set_incident_id("alarm-1234-5678")
        agent = Agent(..., callback_handler=cb)
    """

    def __init__(self):
        self._incident_id: str | None = None
        self._seq: int = 0
        self._active_tool: str | None = None
        self._active_tool_use_id: str | None = None
        self._tool_start_time: float | None = None
        self._tool_input_buffer: str = ""
        self._text_buffer: str = ""

    def set_incident_id(self, incident_id: str) -> None:
        """Reset state for a new invocation and write the _latest sentinel."""
        self._incident_id = incident_id
        self._seq = 0
        self._active_tool = None
        self._active_tool_use_id = None
        self._tool_start_time = None
        self._tool_input_buffer = ""
        self._text_buffer = ""

        # Sentinel so the observer frontend can discover the active incident.
        now = _now_iso()
        try:
            _table.put_item(Item={
                "incident_id": "_latest",
                "seq": 0,
                "incident_ref": incident_id,
                "timestamp": now,
                "ttl": _ttl(),
            })
        except Exception:
            logger.exception("Failed to write _latest sentinel")

        # Index entry so the observer can list past incidents.
        try:
            # Extract alarm name from the incident_id (strip trailing timestamp).
            parts = incident_id.rsplit("-", 1)
            alarm_name = parts[0] if len(parts) == 2 and parts[1].isdigit() else incident_id
            _table.put_item(Item={
                "incident_id": "_incidents",
                "seq": int(time.time()),
                "incident_ref": incident_id,
                "alarm_name": alarm_name,
                "started_at": now,
                "ttl": _ttl(),
            })
        except Exception:
            logger.exception("Failed to write incidents index entry")

    def __call__(self, **kwargs) -> None:
        if self._incident_id is None:
            return

        # Tool start: contentBlockStart with toolUse.
        if "event" in kwargs:
            event = kwargs["event"]
            cbs = event.get("contentBlockStart", {}).get("start", {}).get("toolUse")
            if cbs:
                # If we had a previous tool running, close it first.
                if self._active_tool:
                    self._emit_tool_end()
                self._active_tool = cbs.get("name", "unknown")
                self._active_tool_use_id = cbs.get("toolUseId")
                self._tool_start_time = time.time()
                self._tool_input_buffer = ""
                self._emit("tool_start", {"tool": self._active_tool, "tool_use_id": self._active_tool_use_id})
                return

            # Buffer tool input from contentBlockDelta.
            tool_input_chunk = (
                event.get("contentBlockDelta", {}).get("delta", {}).get("toolUse", {}).get("input")
            )
            if tool_input_chunk and self._active_tool:
                self._tool_input_buffer += tool_input_chunk
                return

            # contentBlockStop while a tool is active means tool finished.
            if "contentBlockStop" in event and self._active_tool:
                self._emit_tool_end()
                return

        # Reasoning text.
        if "reasoningText" in kwargs:
            self._emit("thinking", {"text": kwargs["reasoningText"][:500]})
            return

        # Text output — debounce by buffering.
        if "data" in kwargs and isinstance(kwargs["data"], str):
            self._text_buffer += kwargs["data"]
            if len(self._text_buffer) >= 200 or self._text_buffer.endswith((".", "\n")):
                self._emit("text", {"text": self._text_buffer})
                self._text_buffer = ""
            return

        # Completion.
        if kwargs.get("complete"):
            self._flush_and_complete()
            return

    def emit_complete(self) -> None:
        """Emit a completion event. Called from handler.py after agent() returns."""
        if self._incident_id is None:
            return
        self._flush_and_complete()

    def set_tool_result(self, tool_name: str, result: dict) -> None:
        """Emit a tool_result event immediately. Called from tool functions after execution."""
        if self._incident_id is None:
            return
        self._emit("tool_result", {
            "tool": tool_name,
            "result": _sanitise_for_dynamo(result),
        })

    def _flush_and_complete(self) -> None:
        if self._text_buffer:
            self._emit("text", {"text": self._text_buffer})
            self._text_buffer = ""
        if self._active_tool:
            self._emit_tool_end()
        self._emit("complete", {})

    def _emit_tool_end(self) -> None:
        duration = round(time.time() - self._tool_start_time, 2) if self._tool_start_time else 0

        detail = {
            "tool": self._active_tool,
            "tool_use_id": self._active_tool_use_id,
            "duration_s": Decimal(str(duration)),
        }

        # Parse and attach tool input if we captured any.
        if self._tool_input_buffer:
            try:
                parsed = json.loads(self._tool_input_buffer)
                detail["input"] = _sanitise_for_dynamo(parsed)
            except (json.JSONDecodeError, ValueError):
                detail["input"] = {"_raw": self._tool_input_buffer[:500]}

        self._emit("tool_end", detail)
        self._active_tool = None
        self._active_tool_use_id = None
        self._tool_start_time = None
        self._tool_input_buffer = ""

    def _emit(self, event_type: str, detail: dict) -> None:
        self._seq += 1
        item = {
            "incident_id": self._incident_id,
            "seq": self._seq,
            "event_type": event_type,
            "timestamp": _now_iso(),
            "ttl": _ttl(),
            "detail": detail,
        }
        try:
            _table.put_item(Item=item)
        except Exception:
            logger.exception("Failed to write event seq=%d type=%s", self._seq, event_type)


def _sanitise_for_dynamo(obj):
    """Convert floats to Decimals and strip empty strings (DynamoDB rejects both)."""
    if isinstance(obj, float):
        return Decimal(str(obj))
    if isinstance(obj, dict):
        return {k: _sanitise_for_dynamo(v) for k, v in obj.items() if v != ""}
    if isinstance(obj, list):
        return [_sanitise_for_dynamo(i) for i in obj]
    return obj


def _now_iso() -> str:
    return time.strftime("%Y-%m-%dT%H:%M:%SZ", time.gmtime())


def _ttl() -> int:
    return int(time.time()) + _TTL_SECONDS
