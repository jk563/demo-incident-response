"""Lambda handler for the incident triage agent."""

import json
import logging
import os
import time
from pathlib import Path

from strands import Agent
from strands.models.bedrock import BedrockModel

from callback_handler import ObserverCallbackHandler
from tools import (
    set_observer_callback,
    create_issue,
    describe_alarm,
    get_metric_data,
    get_source_file,
    get_xray_traces,
    query_logs,
)

logger = logging.getLogger()
logger.setLevel(logging.INFO)

# Load system prompt from bundled file.
_prompt_path = Path(__file__).parent / "prompts" / "system.md"
SYSTEM_PROMPT = _prompt_path.read_text()

# Initialise model and agent at module level for Lambda warm-start reuse.
model = BedrockModel(
    model_id=os.environ.get("BEDROCK_MODEL", "anthropic.claude-sonnet-4-6"),
    region_name=os.environ.get("BEDROCK_REGION", "eu-west-2"),
    max_tokens=4096,
)

observer_callback = ObserverCallbackHandler()
set_observer_callback(observer_callback)

agent = Agent(
    model=model,
    system_prompt=SYSTEM_PROMPT,
    tools=[describe_alarm, query_logs, get_metric_data, get_xray_traces, get_source_file, create_issue],
    callback_handler=observer_callback,
)


def handler(event, context):
    """Parse SNS event, extract alarm details, and invoke the triage agent."""
    logger.info("Received event: %s", json.dumps(event))

    # Extract the alarm payload from the SNS notification.
    try:
        sns_record = event["Records"][0]["Sns"]
        message = json.loads(sns_record["Message"])
        alarm_name = message.get("AlarmName", "unknown")
        new_state = message.get("NewStateValue", "unknown")
        reason = message.get("NewStateReason", "")
    except (KeyError, IndexError, json.JSONDecodeError) as exc:
        logger.error("Failed to parse SNS event: %s", exc)
        return {"statusCode": 400, "body": f"Invalid event: {exc}"}

    # Only investigate ALARM state transitions, not OK.
    if new_state != "ALARM":
        logger.info("Ignoring state transition to %s for alarm %s", new_state, alarm_name)
        return {"statusCode": 200, "body": f"Ignored {new_state} state for {alarm_name}"}

    logger.info("Investigating alarm: %s (reason: %s)", alarm_name, reason)

    # Set up observer for this invocation.
    incident_id = f"{alarm_name}-{int(time.time())}"
    observer_callback.set_incident_id(incident_id)
    logger.info("Observer incident_id: %s", incident_id)

    prompt = (
        f"A CloudWatch alarm has fired.\n\n"
        f"**Alarm name:** {alarm_name}\n"
        f"**New state:** {new_state}\n"
        f"**Reason:** {reason}\n\n"
        f"Investigate this alarm following the investigation workflow in your instructions. "
        f"Gather evidence from logs, metrics, traces, and source code, then create an issue with "
        f"a complete RCA."
    )

    result = agent(prompt)
    output = str(result)
    observer_callback.emit_complete()

    logger.info("Agent completed. Output length: %d chars", len(output))

    return {"statusCode": 200, "body": output}
