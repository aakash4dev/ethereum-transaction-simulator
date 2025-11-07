#!/bin/bash
# Script to create GitHub issue using GitHub CLI

gh issue create \
  --title "Fix nonce conflicts in parallel transaction operations" \
  --body "$(cat .github/ISSUE_TEMPLATE.md)" \
  --label "bug,enhancement" \
  --assignee "@me"

echo "Issue created successfully!"
