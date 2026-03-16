#!/usr/bin/env bash
# Seed DynamoDB with 25 sample orders for the demo.
# Usage: ./scripts/seed-data.sh
#
# Requires: aws CLI, uuidgen, bc
# The orders are staggered over the past 24 hours and use a mix of
# statuses, discount codes, and randomly selected catalogue items.
set -euo pipefail

SCRIPT_DIR="$(cd "$(dirname "${BASH_SOURCE[0]}")" && pwd)"
# shellcheck source=../.env
[[ -f "${SCRIPT_DIR}/../.env" ]] && source "${SCRIPT_DIR}/../.env"

TABLE_NAME="demo-orders"
REGION="${AWS_REGION:?Set AWS_REGION in .env}"
PROFILE="${AWS_PROFILE:-}"
PROFILE_FLAG=()
[[ -n "${PROFILE}" ]] && PROFILE_FLAG=(--profile "${PROFILE}")

# Product catalogue: "name|unit_price" pairs.
PRODUCTS=(
  "Mechanical Keyboard|89.99"
  "USB-C Cable|9.99"
  "Monitor Stand|45.00"
  "Webcam HD|59.99"
  "Mouse Pad XL|19.99"
  "Desk Lamp|34.99"
  "Headphone Hook|14.99"
  "Notebook|4.99"
  "Pen Set|12.50"
  "Cable Tidy|7.99"
)

# Discount codes: index 0 = none, 1 = SAVE5 (5%), 2 = SAVE10 (10%), 3 = SAVE15 (15%).
DISCOUNT_CODES=("" "SAVE5" "SAVE10" "SAVE15")
DISCOUNT_RATES=("0" "0.05" "0.10" "0.15")

TOTAL_ORDERS=25
NOW_EPOCH=$(date -u +%s)
SECONDS_IN_DAY=86400

echo "Seeding ${TOTAL_ORDERS} orders into '${TABLE_NAME}' (${REGION})..."
echo ""

for i in $(seq 1 "${TOTAL_ORDERS}"); do
  ORDER_ID=$(uuidgen | tr '[:upper:]' '[:lower:]')

  # Stagger timestamps evenly across the past 24 hours, oldest first.
  OFFSET=$(( SECONDS_IN_DAY - (i * SECONDS_IN_DAY / TOTAL_ORDERS) ))
  ORDER_EPOCH=$(( NOW_EPOCH - OFFSET ))
  CREATED_AT=$(date -u -r "${ORDER_EPOCH}" +"%Y-%m-%dT%H:%M:%SZ" 2>/dev/null \
    || date -u -d "@${ORDER_EPOCH}" +"%Y-%m-%dT%H:%M:%SZ")
  UPDATED_AT="${CREATED_AT}"

  # ~80% confirmed, ~20% refunded.
  STATUS_ROLL=$(( (RANDOM % 10) ))
  if (( STATUS_ROLL < 8 )); then
    STATUS="confirmed"
  else
    STATUS="refunded"
  fi

  # Pick 1–3 random items from the catalogue.
  NUM_ITEMS=$(( (RANDOM % 3) + 1 ))
  ITEM_JSON_LIST=""
  SUBTOTAL="0"

  for j in $(seq 1 "${NUM_ITEMS}"); do
    PRODUCT_IDX=$(( RANDOM % ${#PRODUCTS[@]} ))
    PRODUCT="${PRODUCTS[$PRODUCT_IDX]}"
    PRODUCT_NAME="${PRODUCT%%|*}"
    UNIT_PRICE="${PRODUCT##*|}"
    QUANTITY=$(( (RANDOM % 3) + 1 ))

    LINE_TOTAL=$(echo "scale=2; ${UNIT_PRICE} * ${QUANTITY}" | bc)
    SUBTOTAL=$(echo "scale=2; ${SUBTOTAL} + ${LINE_TOTAL}" | bc)

    ITEM_JSON="{\"M\":{\"name\":{\"S\":\"${PRODUCT_NAME}\"},\"quantity\":{\"N\":\"${QUANTITY}\"},\"unit_price\":{\"N\":\"${UNIT_PRICE}\"}}}"
    if [[ -z "${ITEM_JSON_LIST}" ]]; then
      ITEM_JSON_LIST="${ITEM_JSON}"
    else
      ITEM_JSON_LIST="${ITEM_JSON_LIST},${ITEM_JSON}"
    fi
  done

  # Roughly equal chance of each discount tier (including none).
  DISCOUNT_IDX=$(( RANDOM % 4 ))
  DISCOUNT_CODE="${DISCOUNT_CODES[$DISCOUNT_IDX]}"
  DISCOUNT_RATE="${DISCOUNT_RATES[$DISCOUNT_IDX]}"
  DISCOUNT_AMOUNT=$(echo "scale=2; ${SUBTOTAL} * ${DISCOUNT_RATE}" | bc)
  # Ensure two decimal places for monetary values.
  SUBTOTAL=$(printf "%.2f" "${SUBTOTAL}")
  DISCOUNT_AMOUNT=$(printf "%.2f" "${DISCOUNT_AMOUNT}")
  TOTAL=$(echo "scale=2; ${SUBTOTAL} - ${DISCOUNT_AMOUNT}" | bc)
  TOTAL=$(printf "%.2f" "${TOTAL}")

  # Build the DynamoDB item JSON. Omit discount_code when empty.
  DISCOUNT_ATTR=""
  if [[ -n "${DISCOUNT_CODE}" ]]; then
    DISCOUNT_ATTR=",\"discount_code\":{\"S\":\"${DISCOUNT_CODE}\"}"
  fi

  ITEM_DOC=$(cat <<EOF
{
  "id":              {"S": "${ORDER_ID}"},
  "items":           {"L": [${ITEM_JSON_LIST}]},
  "subtotal":        {"N": "${SUBTOTAL}"}${DISCOUNT_ATTR},
  "discount_amount": {"N": "${DISCOUNT_AMOUNT}"},
  "total":           {"N": "${TOTAL}"},
  "status":          {"S": "${STATUS}"},
  "created_at":      {"S": "${CREATED_AT}"},
  "updated_at":      {"S": "${UPDATED_AT}"}
}
EOF
)

  aws dynamodb put-item \
    --table-name "${TABLE_NAME}" \
    --item "${ITEM_DOC}" \
    --region "${REGION}" \
    "${PROFILE_FLAG[@]}"

  echo "  [${i}/${TOTAL_ORDERS}] Created order ${ORDER_ID} — status: ${STATUS}, total: £${TOTAL}${DISCOUNT_CODE:+ (${DISCOUNT_CODE})}"
done

echo ""
echo "Seeding complete. ${TOTAL_ORDERS} orders written to '${TABLE_NAME}'."
