#!/usr/bin/env bash
# Run smoke checks before a demo, printing colour-coded results.
# Usage: ./scripts/demo-preflight.sh
#
# Checks:
#   1. App health endpoint responds with "healthy"
#   2. CloudWatch error-rate alarm is in OK state
#   3. Triage agent Lambda function exists
#   4. DynamoDB orders table is ACTIVE
#   5. DynamoDB orders table contains data
#   6. Git PAT in Secrets Manager is valid
#   7. CloudWatch dashboard URL (informational)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../.env
[[ -f "${SCRIPT_DIR}/../.env" ]] && source "${SCRIPT_DIR}/../.env"

REGION="${AWS_REGION:?Set AWS_REGION in .env}"
PROFILE="${AWS_PROFILE:-}"

APP_URL="https://${APP_DOMAIN:?Set APP_DOMAIN in .env}"
PROFILE_FLAG=()
[[ -n "${PROFILE}" ]] && PROFILE_FLAG=(--profile "${PROFILE}")
ALARM_NAME="demo-incident-response-error-rate"
FUNCTION_NAME="demo-triage-agent"
TABLE_NAME="demo-orders"
SECRET_NAME="demo-incident-response/git-pat"
DASHBOARD_NAME="demo-incident-response"
DASHBOARD_URL="https://${REGION}.console.aws.amazon.com/cloudwatch/home?region=${REGION}#dashboards:name=${DASHBOARD_NAME}"

GIT_PROVIDER="${GIT_PROVIDER:-github}"
GITLAB_URL="${GITLAB_URL:-https://gitlab.com}"

# ── Counters ──────────────────────────────────────────────────────────────────
passes=0
failures=0
warnings=0

# ── Colour helpers ────────────────────────────────────────────────────────────
GREEN="\033[0;32m"
RED="\033[0;31m"
YELLOW="\033[0;33m"
RESET="\033[0m"

pass() {
  printf "${GREEN}  ✓ %s${RESET}\n" "$1"
  passes=$(( passes + 1 ))
}

fail() {
  printf "${RED}  ✗ %s${RESET}\n" "$1"
  failures=$(( failures + 1 ))
}

warn() {
  printf "${YELLOW}  ⚠ %s${RESET}\n" "$1"
  warnings=$(( warnings + 1 ))
}

# ── Header ────────────────────────────────────────────────────────────────────
echo ""
echo "========================================="
echo "   Demo Pre-flight Checks"
echo "========================================="
echo ""

# ── 1. App health ─────────────────────────────────────────────────────────────
echo "1. App health"
if health_body=$(curl -sf "${APP_URL}/health" 2>/dev/null) && echo "${health_body}" | grep -q '"healthy"'; then
  pass "Health endpoint returned healthy"
else
  fail "Health endpoint did not return healthy (URL: ${APP_URL}/health)"
fi

# ── 2. CloudWatch alarm state ─────────────────────────────────────────────────
echo ""
echo "2. CloudWatch alarm state"
alarm_state=$(aws cloudwatch describe-alarms \
  --alarm-names "${ALARM_NAME}" \
  --region "${REGION}" \
  "${PROFILE_FLAG[@]}" \
  --query "MetricAlarms[0].StateValue" \
  --output text 2>/dev/null || echo "ERROR")

case "${alarm_state}" in
  OK)
    pass "Alarm '${ALARM_NAME}' is OK"
    ;;
  INSUFFICIENT_DATA)
    warn "Alarm '${ALARM_NAME}' has INSUFFICIENT_DATA — may need a moment to warm up"
    ;;
  ALARM)
    fail "Alarm '${ALARM_NAME}' is in ALARM state — clear before demoing"
    ;;
  None|ERROR)
    fail "Could not retrieve alarm '${ALARM_NAME}' — does it exist?"
    ;;
  *)
    fail "Unexpected alarm state for '${ALARM_NAME}': ${alarm_state}"
    ;;
esac

# ── 3. Lambda function exists ─────────────────────────────────────────────────
echo ""
echo "3. Triage agent Lambda"
if aws lambda get-function \
     --function-name "${FUNCTION_NAME}" \
     --region "${REGION}" \
     "${PROFILE_FLAG[@]}" \
     --output text > /dev/null 2>&1; then
  pass "Lambda function '${FUNCTION_NAME}' exists"
else
  fail "Lambda function '${FUNCTION_NAME}' not found"
fi

# ── 4. DynamoDB table accessible and ACTIVE ───────────────────────────────────
echo ""
echo "4. DynamoDB table status"
table_status=$(aws dynamodb describe-table \
  --table-name "${TABLE_NAME}" \
  --region "${REGION}" \
  "${PROFILE_FLAG[@]}" \
  --query "Table.TableStatus" \
  --output text 2>/dev/null || echo "ERROR")

if [[ "${table_status}" == "ACTIVE" ]]; then
  pass "Table '${TABLE_NAME}' is ACTIVE"
else
  fail "Table '${TABLE_NAME}' is not ACTIVE (got: ${table_status})"
fi

# ── 5. DynamoDB table has data ────────────────────────────────────────────────
echo ""
echo "5. DynamoDB table data"
item_count=$(aws dynamodb scan \
  --table-name "${TABLE_NAME}" \
  --select COUNT \
  --region "${REGION}" \
  "${PROFILE_FLAG[@]}" \
  --query "Count" \
  --output text 2>/dev/null || echo "0")

if [[ "${item_count}" -gt 0 ]]; then
  pass "Table '${TABLE_NAME}' contains ${item_count} item(s)"
else
  warn "Table '${TABLE_NAME}' is empty — run seed script before demoing"
fi

# ── 6. Git PAT valid ─────────────────────────────────────────────────────────
echo ""
echo "6. Git PAT (${GIT_PROVIDER})"
pat=$(aws secretsmanager get-secret-value \
  --secret-id "${SECRET_NAME}" \
  --region "${REGION}" \
  "${PROFILE_FLAG[@]}" \
  --query "SecretString" \
  --output text 2>/dev/null || echo "")

if [[ -z "${pat}" ]]; then
  fail "Could not retrieve PAT from Secrets Manager ('${SECRET_NAME}')"
elif [[ "${GIT_PROVIDER}" == "gitlab" ]]; then
  # Try personal token endpoint first; fall back to project endpoint for project access tokens.
  if curl -sf -H "PRIVATE-TOKEN: ${pat}" "${GITLAB_URL}/api/v4/user" > /dev/null 2>&1; then
    pass "GitLab PAT is valid (personal token)"
  elif curl -sf -H "PRIVATE-TOKEN: ${pat}" "${GITLAB_URL}/api/v4/projects/${GIT_REPO}" > /dev/null 2>&1; then
    pass "GitLab PAT is valid (project token)"
  else
    fail "GitLab PAT is invalid or lacks required permissions"
  fi
else
  if curl -sf -H "Authorization: token ${pat}" https://api.github.com/user > /dev/null 2>&1; then
    pass "GitHub PAT is valid"
  else
    fail "GitHub PAT is invalid or lacks required permissions"
  fi
fi

# ── 7. Dashboard URL (informational) ──────────────────────────────────────────
echo ""
echo "7. CloudWatch dashboard"
printf "${GREEN}  ✓ Dashboard URL:${RESET} %s\n" "${DASHBOARD_URL}"
passes=$(( passes + 1 ))

# ── Summary ───────────────────────────────────────────────────────────────────
echo ""
echo "========================================="
printf "   Summary: ${GREEN}%d passed${RESET}  ${RED}%d failed${RESET}  ${YELLOW}%d warned${RESET}\n" \
  "${passes}" "${failures}" "${warnings}"
echo "========================================="
echo ""

if [[ "${failures}" -gt 0 ]]; then
  exit 1
fi
