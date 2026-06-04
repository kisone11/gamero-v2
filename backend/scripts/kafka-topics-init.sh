#!/usr/bin/env bash
# scripts/kafka-topics-init.sh
# 预创建 Gamero 所需的 Kafka Topic（可选，Kafka 已开启 auto.create.topics.enable）
# 手动创建可控制分区数和副本数。
#
# 用法：
#   ./scripts/kafka-topics-init.sh [BOOTSTRAP_SERVER]
#   默认 BOOTSTRAP_SERVER = localhost:9093
#
set -euo pipefail

BOOTSTRAP="${1:-localhost:9093}"
PARTITIONS="${PARTITIONS:-3}"
REPLICATION="${REPLICATION:-1}"
PREFIX="gamero."

TOPICS=(
  "comment.created"
  "post.liked"
  "log.comment.created"
  "hot_score.refresh"
  "recruit.expiry_check"
  "user.followed"
  "project.updated"
  "crowdfund.goal_met"
  "post.published"
  "crowdfund.success"
  "milestone.completed"
)

echo "→ Kafka Bootstrap: ${BOOTSTRAP}"
echo "→ Partitions: ${PARTITIONS}, Replication: ${REPLICATION}"
echo ""

for TOPIC in "${TOPICS[@]}"; do
  FULL_TOPIC="${PREFIX}${TOPIC}"
  echo -n "  Creating topic [${FULL_TOPIC}] ... "
  kafka-topics.sh \
    --bootstrap-server "${BOOTSTRAP}" \
    --create \
    --if-not-exists \
    --topic "${FULL_TOPIC}" \
    --partitions "${PARTITIONS}" \
    --replication-factor "${REPLICATION}" \
    2>&1 | tail -1
done

echo ""
echo "✓ All topics created. Listing:"
kafka-topics.sh --bootstrap-server "${BOOTSTRAP}" --list | grep "^${PREFIX}" || true
