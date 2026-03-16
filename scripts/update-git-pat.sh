#!/usr/bin/env bash
# Update the Git provider PAT in Secrets Manager.
# Usage: ./scripts/update-git-pat.sh [TOKEN]
# If TOKEN is omitted, reads from stdin (pipe-friendly).
#
# For GitHub, create a fine-grained PAT at: https://github.com/settings/personal-access-tokens/new
#   Repository: <your GitHub repo>
#   Permissions:
#     - Issues:        Read and write  (triage agent creates issues)
#     - Contents:      Read and write  (triage agent reads source files)
#
# For GitLab, create a project access token at: Settings > Access Tokens
#   Scopes: api (or read_api + read_repository for read-only agent use)
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../.env
[[ -f "${SCRIPT_DIR}/../.env" ]] && source "${SCRIPT_DIR}/../.env"

SECRET_NAME="demo-incident-response/git-pat"
REGION="${AWS_REGION:?Set AWS_REGION in .env}"
PROFILE="${AWS_PROFILE:-}"
PROFILE_FLAG=()
[[ -n "${PROFILE}" ]] && PROFILE_FLAG=(--profile "${PROFILE}")

if [[ $# -ge 1 ]]; then
  token="$1"
elif [[ ! -t 0 ]]; then
  read -r token
else
  printf "PAT: " >&2
  read -rs token
  echo >&2
fi

if [[ -z "${token}" ]]; then
  echo "Error: no token provided" >&2
  exit 1
fi

aws secretsmanager put-secret-value \
  --secret-id "${SECRET_NAME}" \
  --secret-string "${token}" \
  --region "${REGION}" \
  "${PROFILE_FLAG[@]}"

echo "Secret '${SECRET_NAME}' updated successfully."
