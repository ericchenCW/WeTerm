## ADDED Requirements

### Requirement: RabbitMQ stalled 队列 ack 速率上限

系统 SHALL 支持 `INSPECT_RABBITMQ_STALLED_ACK_RATE_MAX`（浮点数,单位 msgs/sec）用于设定"消费停滞"判定的 ack 速率上限阈值 ε。env 未设置时默认 `0.01`。值 MUST > 0，非法值（≤ 0 或非数字）时 `Config.Load()` MUST 返回错误。

#### Scenario: 默认 0.01
- **WHEN** `INSPECT_RABBITMQ_STALLED_ACK_RATE_MAX` 未设置
- **THEN** `Config.Thresholds.RabbitMQStalledAckRateMax` 等于 `0.01`

#### Scenario: env 覆盖
- **WHEN** `INSPECT_RABBITMQ_STALLED_ACK_RATE_MAX=0.5`
- **THEN** `Config.Thresholds.RabbitMQStalledAckRateMax` 等于 `0.5`

#### Scenario: 非法值
- **WHEN** `INSPECT_RABBITMQ_STALLED_ACK_RATE_MAX=abc` 或 `INSPECT_RABBITMQ_STALLED_ACK_RATE_MAX=0`
- **THEN** `Config.Load()` 返回错误

### Requirement: RabbitMQ stalled 队列 publish 速率下限

系统 SHALL 支持 `INSPECT_RABBITMQ_STALLED_PUBLISH_RATE_MIN`（浮点数,单位 msgs/sec）用于设定 stalled 判定中 publish 速率必须超过的下限。env 未设置时默认 `0.0`（要求 publish_rate > 0,豁免低流量稳态队列）。

设为负数（如 `-1`）时 MUST 视为关闭 publish_rate 条件——只要 ack_rate < ε 且 messages > 0 即触发 stalled。

#### Scenario: 默认 0.0
- **WHEN** `INSPECT_RABBITMQ_STALLED_PUBLISH_RATE_MIN` 未设置
- **THEN** `Config.Thresholds.RabbitMQStalledPublishRateMin` 等于 `0.0`

#### Scenario: env 覆盖
- **WHEN** `INSPECT_RABBITMQ_STALLED_PUBLISH_RATE_MIN=0.05`
- **THEN** `Config.Thresholds.RabbitMQStalledPublishRateMin` 等于 `0.05`

#### Scenario: 负值表示关闭该条件
- **WHEN** `INSPECT_RABBITMQ_STALLED_PUBLISH_RATE_MIN=-1`
- **THEN** stalled 判定不再要求 `publish_rate > min`，只要其他条件满足即触发

## MODIFIED Requirements

### Requirement: RabbitMQ 0 消费者 vhost 黑名单

系统 SHALL 支持 `INSPECT_RABBITMQ_STALLED_VHOST_BLACKLIST`（逗号分隔字符串）以指定一组在"队列消费停滞"检查中需被忽略的 vhost。该黑名单仅作用于 stalled 告警，队列堆积阈值告警（`INSPECT_RABBITMQ_QUEUE_BACKLOG_THRESHOLD`）对所有 vhost 仍照常生效。env 未设置时默认包含 `bk_bknodeman`。

**向后兼容（过渡 1 个版本周期）**：若 `INSPECT_RABBITMQ_STALLED_VHOST_BLACKLIST` 未设置且 `INSPECT_RABBITMQ_NO_CONSUMER_VHOST_BLACKLIST` 已设置，则系统 SHALL 沿用旧变量的值并在 stderr 打印 deprecation 警告。同时设置两个变量时以新变量为准。

#### Scenario: 默认黑名单包含 bk_bknodeman
- **WHEN** `INSPECT_RABBITMQ_STALLED_VHOST_BLACKLIST` 与 `INSPECT_RABBITMQ_NO_CONSUMER_VHOST_BLACKLIST` 均未设置
- **THEN** `Config.Thresholds.RabbitMQStalledVHostBlacklist` 等于 `["bk_bknodeman"]`

#### Scenario: env 覆盖黑名单
- **WHEN** `INSPECT_RABBITMQ_STALLED_VHOST_BLACKLIST=foo,bar`
- **THEN** `Config.Thresholds.RabbitMQStalledVHostBlacklist` 等于 `["foo", "bar"]`（完全替换默认值）

#### Scenario: 黑名单 vhost 下队列停滞不告警
- **WHEN** vhost `bk_bknodeman` 下某队列满足 stalled 条件
- **THEN** 该队列不产生 stalled 告警

#### Scenario: 黑名单 vhost 下队列堆积仍告警
- **WHEN** vhost `bk_bknodeman` 下某队列 `messages` 超过 `INSPECT_RABBITMQ_QUEUE_BACKLOG_THRESHOLD`
- **THEN** 仍产生队列堆积告警

#### Scenario: 旧变量过渡兼容
- **WHEN** `INSPECT_RABBITMQ_STALLED_VHOST_BLACKLIST` 未设置且 `INSPECT_RABBITMQ_NO_CONSUMER_VHOST_BLACKLIST=legacy_vhost`
- **THEN** `Config.Thresholds.RabbitMQStalledVHostBlacklist` 等于 `["legacy_vhost"]`
- **AND** stderr 输出 deprecation 警告

#### Scenario: 同时设置新旧变量以新为准
- **WHEN** `INSPECT_RABBITMQ_STALLED_VHOST_BLACKLIST=new` 且 `INSPECT_RABBITMQ_NO_CONSUMER_VHOST_BLACKLIST=old`
- **THEN** `Config.Thresholds.RabbitMQStalledVHostBlacklist` 等于 `["new"]`
