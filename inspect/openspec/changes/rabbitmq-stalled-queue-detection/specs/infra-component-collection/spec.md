## ADDED Requirements

### Requirement: RabbitMQ stalled 队列采集

系统 SHALL 在 `/api/queues` 请求中包含 `message_stats.ack_details.rate` 与 `message_stats.publish_details.rate` 两列，并据此在 collector 阶段筛出"消费停滞"队列。

判定条件（在 collector 完成，全部 MUST 同时满足才追加到 `RabbitMQStatus.StalledQueues`）：

- `messages > 0`
- `ack_rate < Thresholds.RabbitMQStalledAckRateMax`
- `publish_rate > Thresholds.RabbitMQStalledPublishRateMin`
- 队列所在 vhost 不在 `Thresholds.RabbitMQStalledVHostBlacklist` 中

当 API 返回的队列对象不含 `message_stats` 子对象时,`ack_rate` 与 `publish_rate` MUST 视为 `0`。此时由于 `publish_rate > min` 条件不成立(默认 min=0)，该队列 MUST NOT 被追加到 `StalledQueues`。

`RabbitMQQueue` 模型 MUST 新增 `AckRate float64` 与 `PublishRate float64` 字段以承载这两个速率,供下游 checker 渲染 Value 使用。

#### Scenario: 卡死的消费者被识别为 stalled
- **WHEN** 队列 `{vhost: prod, queue: q1, messages: 50, consumers: 2, ack_rate: 0.0, publish_rate: 1.2}`
- **THEN** `CollectRabbitMQ` SHALL 把该队列追加到 `StalledQueues`

#### Scenario: 无消费者队列也走 stalled 路径
- **WHEN** 队列 `{vhost: prod, queue: q2, messages: 10, consumers: 0, ack_rate: 0.0, publish_rate: 0.5}`
- **THEN** `CollectRabbitMQ` SHALL 把该队列追加到 `StalledQueues`（替代旧的 `NoConsumerQueues` 路径）

#### Scenario: 低流量稳态队列豁免
- **WHEN** 队列 `{messages: 3, ack_rate: 0.0, publish_rate: 0.0}`（无新增也无消费）
- **THEN** `CollectRabbitMQ` MUST NOT 把该队列追加到 `StalledQueues`

#### Scenario: 健康消费的队列豁免
- **WHEN** 队列 `{messages: 100, ack_rate: 5.0, publish_rate: 5.1}`
- **THEN** `CollectRabbitMQ` MUST NOT 把该队列追加到 `StalledQueues`（ack_rate 远高于阈值）

#### Scenario: vhost 黑名单豁免
- **WHEN** 队列所在 vhost ∈ `Thresholds.RabbitMQStalledVHostBlacklist`，且其余 stalled 条件全部满足
- **THEN** `CollectRabbitMQ` MUST NOT 把该队列追加到 `StalledQueues`

#### Scenario: message_stats 缺失视为 ack_rate=0、publish_rate=0
- **WHEN** API 返回的队列对象不含 `message_stats` 子对象，且 `messages > 0`
- **THEN** collector MUST 把 `AckRate` 与 `PublishRate` 当作 `0.0` 解析
- **AND** 该队列 MUST NOT 进入 `StalledQueues`（因为 `publish_rate > min` 不成立）

## REMOVED Requirements

### Requirement: RabbitMQ 无消费者队列采集（隐含于先前 collector 筛选行为）
**Reason**: `consumers == 0 ⟹ ack_rate == 0`，"无消费者"是"消费停滞"的真子集。改由新的 stalled 规则统一覆盖，避免双重告警与字段歧义。
**Migration**: collector 层 `RabbitMQStatus.NoConsumerQueues` 切片移除，调用方改读 `StalledQueues`。原 `Thresholds.RabbitMQNoConsumerVHostBlacklist` 字段在 1 个版本周期内仍读取并复制到 `RabbitMQStalledVHostBlacklist`（带 stderr deprecation 警告），随后删除。

## MODIFIED Requirements

### Requirement: collector 阈值判定下沉到 checker

系统 SHALL 把"是否告警"的阈值比较职责下沉到 checker。collector MUST 仅负责采集与
必要的"汇总切片"产出（如 ExceedingQueues / StalledQueues），MUST NOT 在采集阶段
回填 Status 字段。例外：已存在的 `RabbitMQQueueBacklog` 与
`RabbitMQStalledAckRateMax / RabbitMQStalledPublishRateMin / RabbitMQStalledVHostBlacklist`
仍由 collector 作为筛选切片的判定条件。

#### Scenario: ES heap/RAM 由 checker 判定
- **WHEN** ES 节点 `HeapPercent == 90`
- **THEN** `CollectES` MUST 不在该节点上设置任何 `Status` 字段
- **AND** `CheckES` SHALL 根据 `Thresholds.ESHeapPercent` 决定 Status

#### Scenario: RabbitMQ 切片筛选保留
- **WHEN** RabbitMQ 队列 `prod_bk_monitorv3.celery.MessageCount == 360547` 且 `Thresholds.RabbitMQQueueBacklog == 1000`
- **THEN** `CollectRabbitMQ` SHALL 仍然把该队列追加到 `ExceedingQueues`（保留现有筛选行为）
- **AND** `CheckRabbitMQ` SHALL 把该队列转换为一条 Warn CheckResult

#### Scenario: RabbitMQ stalled 切片筛选保留
- **WHEN** RabbitMQ 队列 `{messages: 50, ack_rate: 0.0, publish_rate: 1.0}` 且 `Thresholds.RabbitMQStalledAckRateMax == 0.01`
- **THEN** `CollectRabbitMQ` SHALL 把该队列追加到 `StalledQueues`
- **AND** `CheckRabbitMQ` SHALL 把该队列转换为一条 Warn CheckResult
