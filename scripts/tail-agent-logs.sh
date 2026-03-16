#!/usr/bin/env bash
# Tail the triage agent Lambda's CloudWatch log group.
# Usage: ./scripts/tail-agent-logs.sh [--since DURATION]
#
# DURATION accepts any value accepted by `aws logs tail --since`, e.g.:
#   1h (default), 30m, 2h, 1d
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../.env
[[ -f "${SCRIPT_DIR}/../.env" ]] && source "${SCRIPT_DIR}/../.env"

REGION="${AWS_REGION:?Set AWS_REGION in .env}"
PROFILE="${AWS_PROFILE:-}"
PROFILE_FLAG=()
[[ -n "${PROFILE}" ]] && PROFILE_FLAG=(--profile "${PROFILE}")
LOG_GROUP="/aws/lambda/demo-triage-agent"
SINCE="1h"

# Parse optional arguments.
while [[ $# -gt 0 ]]; do
  case "$1" in
    --since)
      SINCE="$2"
      shift 2
      ;;
    *)
      echo "Unknown argument: $1" >&2
      echo "Usage: $0 [--since DURATION]" >&2
      exit 1
      ;;
  esac
done

echo "Tailing agent logs (since ${SINCE})..."

aws logs tail "${LOG_GROUP}" \
  --follow \
  --since "${SINCE}" \
  --format short \
  --region "${REGION}" \
  "${PROFILE_FLAG[@]}"
