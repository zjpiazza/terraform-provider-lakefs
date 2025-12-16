#!/bin/bash
# Setup script for LakeFS acceptance testing
# This script waits for LakeFS to be ready and sets up initial admin credentials

set -e

# Base URL for LakeFS (without /api/v1)
LAKEFS_BASE_URL="${LAKEFS_BASE_URL:-http://localhost:8000}"
MAX_RETRIES=30
RETRY_INTERVAL=2

echo "Waiting for LakeFS to be ready at ${LAKEFS_BASE_URL}..."

# Wait for LakeFS to be ready
retries=0
until curl -s "${LAKEFS_BASE_URL}/api/v1/healthcheck" > /dev/null 2>&1; do
  retries=$((retries + 1))
  if [ $retries -ge $MAX_RETRIES ]; then
    echo "Error: LakeFS did not become ready in time"
    exit 1
  fi
  echo "Waiting for LakeFS... (attempt ${retries}/${MAX_RETRIES})"
  sleep $RETRY_INTERVAL
done

echo "LakeFS is ready!"

# Setup initial admin user (or check if already setup)
echo "Setting up LakeFS admin user..."
CREDENTIALS=$(curl -s -X POST "${LAKEFS_BASE_URL}/api/v1/setup_lakefs" \
  -H "Content-Type: application/json" \
  -d '{"username": "admin"}')

# Check if already initialized
if echo "$CREDENTIALS" | grep -q "already initialized"; then
  echo "LakeFS is already configured."
  if [ -f ".env.test" ]; then
    echo "Using existing .env.test file."
    exit 0
  else
    echo "Error: LakeFS is initialized but .env.test is missing."
    echo "To get fresh credentials, restart the container:"
    echo "  make testacc-down && make testacc-up"
    exit 1
  fi
fi

ACCESS_KEY_ID=$(echo "$CREDENTIALS" | jq -r '.access_key_id')
SECRET_ACCESS_KEY=$(echo "$CREDENTIALS" | jq -r '.secret_access_key')

if [ "$ACCESS_KEY_ID" == "null" ] || [ -z "$ACCESS_KEY_ID" ]; then
  echo "Error: Failed to get credentials from LakeFS"
  echo "Response: $CREDENTIALS"
  exit 1
fi

echo ""
echo "============================================"
echo "LakeFS Setup Complete!"
echo "============================================"
echo ""
echo "Add these to your environment or .env file:"
echo ""
echo "export LAKEFS_ENDPOINT=${LAKEFS_BASE_URL}/api/v1"
echo "export LAKEFS_ACCESS_KEY_ID=${ACCESS_KEY_ID}"
echo "export LAKEFS_SECRET_ACCESS_KEY=${SECRET_ACCESS_KEY}"
echo "export TF_ACC=1"
echo ""
echo "Or run acceptance tests directly with:"
echo ""
echo "LAKEFS_ENDPOINT=${LAKEFS_BASE_URL}/api/v1 \\"
echo "LAKEFS_ACCESS_KEY_ID=${ACCESS_KEY_ID} \\"
echo "LAKEFS_SECRET_ACCESS_KEY=${SECRET_ACCESS_KEY} \\"
echo "make testacc"
echo ""

# Optionally write to .env file
if [ "${WRITE_ENV_FILE:-false}" == "true" ]; then
  cat > .env.test <<EOF
export LAKEFS_ENDPOINT=${LAKEFS_BASE_URL}/api/v1
export LAKEFS_ACCESS_KEY_ID=${ACCESS_KEY_ID}
export LAKEFS_SECRET_ACCESS_KEY=${SECRET_ACCESS_KEY}
export TF_ACC=1
EOF
  echo "Credentials written to .env.test"
  echo "Source with: source .env.test"
fi
