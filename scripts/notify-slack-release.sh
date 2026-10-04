#!/usr/bin/env bash

set -euo pipefail

TAG="${1:-}"
WEBHOOK_URL="${SLACK_WEBHOOK_URL:-}"
REPOSITORY="${GITHUB_REPOSITORY:-hostodo/odo-cli}"
SERVER_URL="${GITHUB_SERVER_URL:-https://github.com}"

if [[ ! "${TAG}" =~ ^v[0-9]+\.[0-9]+\.[0-9]+$ ]]; then
  echo "Error: release tag must use vMAJOR.MINOR.PATCH format." >&2
  exit 1
fi

if [[ -z "${WEBHOOK_URL}" ]]; then
  echo "Error: SLACK_WEBHOOK_URL is not configured." >&2
  exit 1
fi

RELEASE_URL="${SERVER_URL}/${REPOSITORY}/releases/tag/${TAG}"
PAYLOAD="$(jq -n \
  --arg title "odo CLI ${TAG} released" \
  --arg url "${RELEASE_URL}" \
  '{
    text: $title,
    blocks: [
      {
        type: "header",
        text: {type: "plain_text", text: $title}
      },
      {
        type: "section",
        text: {
          type: "plain_text",
          text: "New macOS, Linux, and Windows builds are available."
        }
      },
      {
        type: "actions",
        elements: [
          {
            type: "button",
            text: {type: "plain_text", text: "View release"},
            url: $url,
            action_id: "view_odo_cli_release"
          }
        ]
      }
    ]
  }')"

curl \
  --fail-with-body \
  --silent \
  --show-error \
  --retry 3 \
  --retry-all-errors \
  --header 'Content-Type: application/json' \
  --data-binary "${PAYLOAD}" \
  "${WEBHOOK_URL}"

echo
echo "Slack release notification sent for ${TAG}."
