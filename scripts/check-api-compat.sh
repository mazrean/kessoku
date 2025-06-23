#!/bin/bash
set -e

# Script to check API compatibility between versions
# Usage: ./scripts/check-api-compat.sh <base_version> [target_version]

BASE_VERSION=${1:-"HEAD~1"}
TARGET_VERSION=${2:-"."}

echo "🔍 Checking API compatibility..."
echo "Base version: $BASE_VERSION"
echo "Target version: $TARGET_VERSION"

# Build the apicompat tool
echo "📦 Building apicompat tool..."
cd tools
go build -o ../bin/apicompat .
cd ..

# Ensure bin directory exists
mkdir -p bin

# Run the API compatibility check
echo "🔍 Running compatibility check..."
./bin/apicompat apicompat "$BASE_VERSION" "$TARGET_VERSION"

echo "✅ API compatibility check completed!"