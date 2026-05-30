## MODIFIED Requirements

### Requirement: RabbitMQ 检查

系统 SHALL 对 `RabbitMQ` 产生以下 CheckResult，全部为 `Warn`：

- `RabbitMQ.Error` 非空
- `ClusterPartition == true`
- `AbnormalConnections > 0`
- `QueuesError` 非空
- 节点 `MemAlarm == true`
- 节点 `DiskFreeAlarm == true`
- `ExceedingQueues` 中每个队列各一条
- `StalledQueues` 中每个队列各一条

`ExceedingQueues / StalledQueues` 的筛选与 vhost 黑名单仍由 collector 完成，
checker 直接将切片转为 CheckResult，不重新判定。

`StalledQueues` 对应的 CheckResult MUST：
- Field 为 `rabbitmq.<vhost>.<queue>.stalled`
- Value 形如 `"<N> msgs / <M> consumers / ack=<rate>/s / pub=<rate>/s"`
- Threshold 形如 `"ack < <ε>/s, pub > <min>/s"`，由 checker 从 `Thresholds.RabbitMQStalledAckRateMax` 与 `RabbitMQStalledPublishRateMin` 拼出

旧的 `rabbitmq.<vhost>.<queue>.no_consumer` Field MUST NOT 再被产生。

#### Scenario: 队列积压逐项展开
- **WHEN** `ExceedingQueues` 包含 `{vhost: prod_bk_monitorv3, queue: celery, MessageCount: 360547}`
- **THEN** Checker SHALL 产生一条 Warn CheckResult，Field 形如 `rabbitmq.{vhost}.{queue}.backlog`，Value 含消息数

#### Scenario: 队列停滞逐项展开
- **WHEN** `StalledQueues` 包含 `{vhost: prod, queue: q1, MessageCount: 50, Consumers: 2, AckRate: 0.0, PublishRate: 1.2}`
- **THEN** Checker SHALL 产生一条 Warn CheckResult
- **AND** Field 等于 `rabbitmq.prod.q1.stalled`
- **AND** Value 包含 `50 msgs`、`2 consumers`、`ack=0/s`、`pub=1.2/s`
- **AND** Threshold 形如 `ack < 0.01/s, pub > 0/s`

#### Scenario: 节点内存告警
- **WHEN** 节点告警列表中某节点 `MemAlarm == true`
- **THEN** 该节点 SHALL 产生一条 Warn CheckResult

#### Scenario: stalled 与 backlog 并存
- **WHEN** 某队列同时进入 `ExceedingQueues` 与 `StalledQueues`（堆积大且消费停滞）
- **THEN** Checker SHALL 产生两条独立的 Warn CheckResult：一条 `.backlog`、一条 `.stalled`

## REMOVED Requirements

### Requirement: RabbitMQ 0 消费者检查项
**Reason**: 与新的 stalled 规则在语义上完全重叠（无消费者 ⟹ ack_rate=0），合并为统一的 `stalled` 字段以避免双重告警与下游解析歧义。
**Migration**: 邮件/HTML/JSON 中原 `rabbitmq.<vhost>.<queue>.no_consumer` 字段不再出现，等价场景由 `rabbitmq.<vhost>.<queue>.stalled` 表达，Value 中 `consumers=0` 可直观区分"无消费者"与"消费者卡死"。
