#!/bin/bash -x
LOG_FILE=$1
RUN_ID="$2"
SLACK_WEBHOOK_URL="$3"
API_TOKEN_TO_GITHUB="$4"

if [ "${LOG_FILE}" == "" ]; then
   echo "Usage: $0 LOG_FILE RUN_ID SLACK_WEBHOOK_URL API_TOKEN_TO_GITHUB"
   exit 1
fi

cat ${LOG_FILE} | tail -5 |head -3 > /tmp/attachment.txt
TEST_STATUS=`cat /tmp/attachment.txt|tail -1`
TEST_TOTALS=`cat /tmp/attachment.txt|head -1`

cat /tmp/attachment.txt

TIME_STAMP="`date +%s`"

if [ "${TEST_STATUS}" != "OK" ]; then
  HEALTH=":x:"
else
  HEALTH=":white_check_mark:"
fi

SUITE_ID=`curl -s -H "Authorization: token ${API_TOKEN_TO_GITHUB}" \
        -H "Accept: application/vnd.github.v3+json" \
        https://api.github.com/repos/couchbaselabs/smallcb/actions/runs/${RUN_ID} |jq '.check_suite_id'`

cat <<EOT > /tmp/slack_message.json
{
    "type": "mrkdwn",
    "text": "Playground <https://couchbase.live|couchbase.live> health: ${HEALTH}",
    "attachments": [
        {
            "fallback": "${TEST_TOTALS},${TEST_STATUS}",
            "color": "#36a64f",
            "pretext": "${TEST_TOTALS}\n\n${TEST_STATUS}",
            "title": "Download action logs",
            "title_link": "https://github.com/couchbaselabs/smallcb/suites/${SUITE_ID}/logs",
            "footer": "couchbase.live",
            "footer_icon": "https://www.couchbase.com/webfiles/1629373386042/images/favicon.ico",
            "ts": ${TIME_STAMP}
        }
    ]
}
EOT

cat /tmp/slack_message.json

# Send to different channels/webhooks based on tests success or failures
SLACK_WEBHOOK_URL_SUCCESS="`echo ${SLACK_WEBHOOK_URL}|cut -f1 -d','`"
SLACK_WEBHOOK_URL_FAILURES="`echo ${SLACK_WEBHOOK_URL}|cut -f2 -d','`"

if [ "${TEST_STATUS}" == "OK" ]; then
  SLACK_WEBHOOK_URL_MSG="${SLACK_WEBHOOK_URL_SUCCESS}"
else
  SLACK_WEBHOOK_URL_MSG="${SLACK_WEBHOOK_URL_FAILURES}"
fi
curl -s -X POST -H 'Content-type: application/json' --data @/tmp/slack_message.json ${SLACK_WEBHOOK_URL_MSG}
