## Why

现有 RabbitMQ 队列异常检测有两条规则：`messages ≥ 10000` 触发 backlog 告警，`consumers == 0 && messages > 0` 触发 no_consumer 告警。这套规则有一个明显盲区——**当消费者连接还在、但 worker 已经卡死/不在 ack** 时，队列既不到 backlog 阈值也不算"没有消费者"，告警就会一直沉默直到堆积越过 10000。运维同事希望换一种更直接的"消费产能"信号来判断队列是否异常。

引入 RabbitMQ Management API 暴露的 `message_stats.ack_details.rate`（最近窗口内每秒 ack 的消息数）作为"停滞"信号，配合"持续 N 次巡检都停滞"的二次确认，覆盖上述盲区，同时不替换 backlog（产能跟不上的洪峰场景仍由 backlog 兜底）。

## What Changes

- **新增"队列停滞"检测规则**：当某队列 `ack_rate < 阈值ε`（默认 0.01/s）AND `messages > 0` AND `publish_rate > 0`，连续 N 次巡检都满足时，产出一条 Warn CheckResult，Field 形如 `rabbitmq.<vhost>.<queue>.stalled`。
  - `publish_rate > 0` 的条件用于自动豁免"低流量稳态"队列（如心跳/补偿队列在闲时无新消息进入，本来就该 ack=0）。
- **替换 no_consumer 规则**：`consumers == 0 && messages > 0` 是 ack_rate < ε 的真子集，删除独立的 `no_consumer` 规则与对应的 `rabbitmq.<vhost>.<queue>.no_consumer` Field。卡死场景统一由新的 `stalled` Field 表达。**BREAKING**：邮件/HTML/JSON 报告中不再出现 `no_consumer` 字段名。
- **复用持续性确认机制**：不新增独立的"5 分钟"时间窗，沿用 `notify.Persistence.ConsecutiveRuns`（默认 N=2，配合推荐 cron `*/5 * * * *` 等效于 ~10 分钟门槛；如需精确 5 分钟可调 N=1 或缩短 cron）。
- **Collector API 调整**：`/api/queues` 请求新增 columns `message_stats.ack_details.rate` 与 `message_stats.publish_details.rate`，保留 `disable_stats=true`（验证后若仍能返回 message_stats 字段则维持；否则去掉并文档化对 management 节点的开销影响）。
- **新增阈值与豁免配置**：
  - `Thresholds.RabbitMQStalledAckRateMax`（float，默认 0.01）
  - `Thresholds.RabbitMQStalledPublishRateMin`（float，默认 0.0，>0 触发；设为 -1 可关闭 publish_rate 条件）
  - `Thresholds.RabbitMQStalledVHostBlacklist`（替换/复用现有 `RabbitMQNoConsumerVHostBlacklist` 语义，保留同名环境变量做迁移）
- **签名归一化扩展**：`alert-notification` 的 RabbitMQ 队列级 Field 折叠规则新增 `stalled` 后缀，删除 `no_consumer` 后缀。
- **`message_stats` 缺失语义**：当 `messages > 0` 且 API 返回不包含 `message_stats` 时，视为 `ack_rate == 0` 参与判定（队列有消息却从未活动过 = 真异常）。

## Capabilities

### New Capabilities
（无）

### Modified Capabilities
- `infra-component-collection`：RabbitMQ 采集请求新增 `message_stats.ack_details.rate` / `publish_details.rate` 列；筛选切片由 `NoConsumerQueues` 改为 `StalledQueues`（仍由 collector 阶段做阈值/黑名单过滤，与现有架构约定一致）。
- `platform-checks`：删除 `rabbitmq.<vhost>.<queue>.no_consumer` Warn 产出，新增 `rabbitmq.<vhost>.<queue>.stalled` Warn 产出，Value 含 `ack_rate`、`publish_rate`、`messages`、`consumers`。
- `threshold-config`：新增 `RabbitMQStalledAckRateMax`、`RabbitMQStalledPublishRateMin` 阈值与 `RabbitMQStalledVHostBlacklist`（迁移自 `RabbitMQNoConsumerVHostBlacklist`）。
- `alert-notification`：签名归一化把 `rabbitmq.<vhost>.<queue>.stalled` 折叠为 `rabbitmq.<vhost>.stalled`；删除 `no_consumer` 折叠规则。

## Impact

- **代码**：
  - `collector/rabbitmq.go`：调整 columns、新增解析 `message_stats.ack_details.rate` / `publish_details.rate`、把 `NoConsumerQueues` 替换为 `StalledQueues`。
  - `model/`：`RabbitMQQueue` 新增 `AckRate float64`、`PublishRate float64` 字段；`RabbitMQStatus.NoConsumerQueues` → `StalledQueues`。
  - `checker/rabbitmq.go`：去掉 no_consumer 分支，新增 stalled 分支，Field 与 Value 格式变化。
  - `config/config.go`：新增三个阈值字段与对应环境变量（建议命名 `INSPECT_RABBITMQ_STALLED_ACK_RATE_MAX`、`INSPECT_RABBITMQ_STALLED_PUBLISH_RATE_MIN`、`INSPECT_RABBITMQ_STALLED_VHOST_BLACKLIST`），保留 `INSPECT_RABBITMQ_NO_CONSUMER_VHOST_BLACKLIST` 为向后兼容别名 1 个版本周期。
  - `notify/signature.go`：正则与折叠规则更新（删 `no_consumer`、加 `stalled`）。
  - `render/`、`output/`、邮件模板：列名/字段渲染同步。
- **API 与外部依赖**：`/api/queues` 响应大小增加（每队列多两个 `message_stats` 子对象）；需在中大型集群灰度验证 management 节点 CPU 开销，必要时回退到不带 `disable_stats=true`。
- **告警下游**：邮件/HTML 中 `no_consumer` 行项将消失，被 `stalled` 替代；本变更落地后第一次跑会出现"告警类型迁移"，需在发布说明里向运维同事提示。
- **持续性确认**：复用 `notify/persistence.go`，无变更；`PendingKey` 仍按 `host|field` 维度，新 Field 自然进入既有抗抖动流程。
- **文档**：README"告警类型"小节、`docs/` 下相关说明同步更新。
