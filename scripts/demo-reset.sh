#!/usr/bin/env bash
# Full environment reset: destroys and rebuilds all infrastructure, then seeds
# data and runs the pre-flight checks.
# Usage: ./scripts/demo-reset.sh [--yes|-y]
#   --yes / -y  Skip the confirmation prompt (useful in CI or scripted resets).
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
PROJECT_ROOT="$(dirname "$SCRIPT_DIR")"

# Parse flags
SKIP_CONFIRM=false
for arg in "$@"; do
  case "$arg" in
    --yes|-y) SKIP_CONFIRM=true ;;
    *)
      echo "Unknown argument: $arg" >&2
      echo "Usage: $0 [--yes|-y]" >&2
      exit 1
      ;;
  esac
done

# Confirmation prompt
if [[ "$SKIP_CONFIRM" == false ]]; then
  printf "This will destroy and rebuild all infrastructure. Continue? [y/N] "
  read -r reply
  case "$reply" in
    [Yy]*) ;;
    *)
      echo "Aborted." >&2
      exit 1
      ;;
  esac
fi

# ---------------------------------------------------------------------------
# Step 1: Destroy existing infrastructure
# ---------------------------------------------------------------------------
echo ""
echo "==> Step 1/4: Destroying existing infrastructure..."
cd "$PROJECT_ROOT" && just tf-destroy -auto-approve

# ---------------------------------------------------------------------------
# Step 2: Rebuild infrastructure
# ---------------------------------------------------------------------------
echo ""
echo "==> Step 2/4: Rebuilding infrastructure..."
cd "$PROJECT_ROOT" && just tf-apply -auto-approve

# ---------------------------------------------------------------------------
# Step 3: Seed data
# ---------------------------------------------------------------------------
echo ""
echo "==> Step 3/4: Seeding data..."
cd "$PROJECT_ROOT" && just seed

# ---------------------------------------------------------------------------
# Step 4: Pre-flight checks
# ---------------------------------------------------------------------------
echo ""
echo "==> Step 4/4: Running pre-flight checks..."
cd "$PROJECT_ROOT" && just preflight

# ---------------------------------------------------------------------------
echo ""
echo "Environment reset complete. The demo is ready."
