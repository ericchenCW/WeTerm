## Context

weops-inspect 是一个 cron 周期触发的"单次跑"巡检工具（推荐 `*/5 * * * *`），架构上 collector → checker → notify 三段。RabbitMQ 队列层告警目前由 `collector/rabbitmq.go` 在采集阶段筛出 `ExceedingQueues`（msgs ≥ backlog 阈值）与 `NoConsumerQueues`（consumers=0 且 msgs>0，带 vhost 黑名单），`checker/rabbitmq.go` 仅把它们映射成 Warn CheckResult。"抗抖动"由 `notify/persistence.go` 在 notify 阶段实现，N 次连续巡检都命中才真正发邮件（默认 N=2）。

现状漏掉了一个真实场景：**consumer 连接仍在、但 worker 不再 ack 消息**（GC 卡死、死锁、外部依赖 hang、池子打满）。这种情况下队列既不到 backlog 阈值，也不算"无消费者"，要等堆积到 10k 才告警，常常已经晚了。RabbitMQ Management API 在 `/api/queues` 的 `message_stats` 子对象里暴露了 `ack_details.rate` 与 `publish_details.rate`，可以直接读出"最近窗口的 ack 速率/发布速率"，是更直接的产能信号。

约束：
- 不能引入常驻进程或独立采样器——工具必须保持 cron 单次跑形态。
- 不能新增非 curl 网络栈——现有 RabbitMQ collector 走 `exec.CommandContext("curl", ...)` 与 `disable_stats=true&enable_queue_totals=true`，保持一致风格。
- 中大型集群上 management API 的 `message_stats` 是已知性能热点；任何拉法都需要先量化。
- 告警下游（邮件/HTML/JSON）面向值班运维，字段语义变化要在发布说明里同步告知。

## Goals / Non-Goals

**Goals:**
- 补上"consumer 卡死"盲区——通过 ack 速率信号在堆积越过 backlog 阈值前就告警。
- 用一条更通用的"stalled"规则替换 `no_consumer`（语义包含且更强）。
- 复用已有 `Persistence.ConsecutiveRuns` 机制实现"持续 N 次确认"，不新增时间窗逻辑。
- 通过 `publish_rate > 0` 条件，避免对"低流量稳态队列"误报。
- 保留 `backlog` 规则（与 stalled 正交，捕捉"产能跟得上但仍堆积"的洪峰）。

**Non-Goals:**
- 不引入分钟级的独立采样循环——"5 分钟"由 cron 间隔 × N 表达。
- 不针对单个队列做更细的"消费速率应当 ≥ 发布速率的 X%"动态比较（实现复杂、阈值难调，可作为后续 follow-up）。
- 不替换 backlog 规则。
- 不改造 notify 持续性确认算法本身。

## Decisions

### D1：用 `ack_details.rate < ε AND messages > 0 AND publish_rate > 0` 作为停滞判定

**选择**：三条件合取。`ε` 默认 0.01（每秒），意为"近 100 秒消费不到 1 条"；`publish_rate > 0` 用于自动豁免低流量稳态队列。

**备选 A：仅 `ack_rate < ε AND messages > 0`**——更简单但对"心跳/补偿/parking"队列误报严重，需要硬编码白名单或反复加 vhost 黑名单。

**备选 B：动态比较 `ack_rate < α × publish_rate`**——理论上更准，但 RabbitMQ stats 窗口是滚动 5/60s 平均，瞬态抖动严重；α 难调；首次发布风险高。

**取舍**：A 太弱、B 太复杂，C（本方案）三条件合取在简单与鲁棒间取得平衡。`publish_rate > 0` 的阈值通过 `RabbitMQStalledPublishRateMin` 暴露，置为 -1 可关闭该条件回到 A 形态。

### D2：替换而非新增——删除 `no_consumer` 规则

**选择**：`consumers == 0 ⟹ ack_rate == 0`，是 stalled 的真子集，独立保留只会产生双重告警与歧义。

**备选**：保留 `no_consumer` 与 `stalled` 并存，让运维同事看出"是没人消费还是消费者卡了"。

**取舍**：报告里我们能从 Value 中带上 `consumers=N`，"无消费者 vs 卡死"在邮件正文里仍能区分（`consumers=0` 一目了然），不需要靠 Field 名区分。删除可以避免签名层的双重折叠规则。**这是 BREAKING 变更**，发布说明里必须提示。

### D3：阈值 ε 默认 0.01/s，可通过环境变量覆盖

**选择**：`Thresholds.RabbitMQStalledAckRateMax float64`，env `INSPECT_RABBITMQ_STALLED_ACK_RATE_MAX`，默认 `0.01`。

**理由**：RabbitMQ stats 默认窗 5s 滚动平均，0.01/s ≈ "近 100s 消费 < 1 条"，等同 idle；阈值上限留 0.01 是为了对"刚好在每 60s 消费一条心跳"的边缘情况留余量。配合 `publish_rate > 0` 通常已经足够避免误报。

### D4：collector 阶段做筛选，checker 仅映射（沿用现有架构约定）

**选择**：`CollectRabbitMQ` 把 `RabbitMQQueue` 中超过阈值 ε 且通过黑名单的队列追加到 `RabbitMQStatus.StalledQueues`；`CheckRabbitMQ` 仅迭代 `StalledQueues` 转成 Warn CheckResult。

**理由**：`infra-component-collection` spec 已有此例外约定（"汇总切片在 collector 阶段筛选，但 Status 字段必须 checker 回填"），且 backlog/no_consumer 已经是这种模式，保持一致。

### D5：`message_stats` 缺失语义

**选择**：当 `messages > 0` 且 API 返回不含 `message_stats` 子对象时，视为 `ack_rate == 0`、`publish_rate == 0` 并触发判定中**仅以 `ack_rate` 部分**生效——即仍需 `publish_rate > 0` 才上报。

**理由**：刚创建、从未发生过 deliver/ack 的队列会缺整段 message_stats；这类队列的 publish_rate 也是 0，自然被 D1 的第三条件豁免，无需特殊代码。"有消息但 message_stats 缺失"通常是 RabbitMQ stats 还没刷新或队列被清空后立即重新填入，等下一轮巡检即可。

### D6：复用 `notify.Persistence`，不新增时间窗

**选择**：用 `PendingKey(host, "rabbitmq.<vhost>.<queue>.stalled")` 自动接入既有的 N 次确认。

**等效"5 分钟"换算**：cron `*/5` × N=2 ≈ 10 分钟最坏首次告警延迟；如要严格 5 分钟，文档建议把 cron 调到 `*/1` 并设 N=5，或保持 N=1（牺牲抗抖动）。

### D7：Field 命名 `rabbitmq.<vhost>.<queue>.stalled`，Value 携带四个数值

**Value 格式**：`"<messages> msgs / <consumers> consumers / ack=<rate>/s / pub=<rate>/s"`。
**Threshold 渲染**：`"ack < 0.01/s, pub > 0/s"`（在 checker 阶段从 Thresholds 拼出，与现有 backlog 的 `> 10000 msgs` 风格一致）。
**签名归一化**：`alert-notification` 在 `rabbitmq.<vhost>.<queue>.stalled` 上沿用 vhost 折叠，得到 `rabbitmq.<vhost>.stalled`，与 backlog 互不合并（已有规则机制覆盖）。

### D8：vhost 黑名单迁移路径

**选择**：
- 新增 `Thresholds.RabbitMQStalledVHostBlacklist`（默认 `["bk_bknodeman"]`，与现有 no_consumer 默认值相同）。
- 新增环境变量 `INSPECT_RABBITMQ_STALLED_VHOST_BLACKLIST`。
- **过渡期**：若新变量未设、旧 `INSPECT_RABBITMQ_NO_CONSUMER_VHOST_BLACKLIST` 已设，则沿用旧值并在 stderr 打 deprecation 警告；新旧都设以新值为准。保留过渡 1 个版本周期后删除旧变量。

## Risks / Trade-offs

- **management API 性能开销** → 在中大型集群灰度先用 `curl -w '%{size_download}\n'` 量化 `/api/queues` 响应大小；若涨幅 > 30%，把 `disable_stats=true` 去掉的影响也要量；必要时降级为"按 vhost 分批拉"。
- **低流量合法队列误报**（如 5 分钟才一条的心跳） → `publish_rate > 0` 一般可豁免；万一仍有特例，通过 `RabbitMQStalledVHostBlacklist` 兜底，并在 README 文档化建议。
- **消费者偶发空转（GC 暂停、批处理 flush 间隔）** → N=2 持续性确认能过滤一过性 < ε 抖动；ε 设 0.01 而非更高也是为了不被瞬态 0.1/s 抖到。
- **删除 no_consumer 字段名是 BREAKING** → 发布说明里硬置顶提示；保留旧环境变量做过渡期。下游若有人解析邮件文本/JSON 字段名，需要同步通知。
- **`message_stats` 字段缺失** → D5 已处理，配合 `publish_rate > 0` 条件自然避免新建队列误报。
- **stats 窗口与 cron 窗口不对齐** → RabbitMQ stats 默认滚动 5s/60s/600s，与 cron 5min 不强耦合；N=2 的二次确认天然吸收对齐误差。

## Migration Plan

1. **代码落地**（按 tasks.md 顺序）。
2. **本地/QA 验证**：用现有 `checker/rabbitmq_test.go` 与新增 collector 测试覆盖 stalled 判定与 message_stats 缺失场景。
3. **预发集群灰度**：开启新规则跑 2-3 个 cron 周期，对比 ExceedingQueues / StalledQueues 与人工观察，确认无明显误报。
4. **量化 API 开销**：跑 `curl -w '%{size_download} %{time_total}\n'` 对比启用前后 `/api/queues` 响应大小与耗时，记录在 PR description。
5. **生产发布**：
   - 默认 `RabbitMQStalledPublishRateMin = 0.0`（启用条件）。
   - 发布说明顶部明确告知"`no_consumer` 字段被替换为 `stalled`"，并指出对应的环境变量迁移路径。
6. **回滚**：把 `INSPECT_RABBITMQ_STALLED_ACK_RATE_MAX` 设成 `999999`（实际上禁用 stalled 规则），等价于关回 backlog-only；同时无法瞬时恢复 no_consumer，须发紧急 hotfix 还原 collector/checker。

## Open Questions

- ε=0.01 是否对实际现场过松/过紧？建议落地前抓一份生产 `/api/queues` 快照统计 `ack_details.rate` 分布再定档。
- 是否一并把 ExceedingQueues 的 Value 也加上 `ack_rate` 字段（信息含量更高，但属于额外 scope，倾向放到下一个 change）。
- 是否在 stalled Value 里展示 RabbitMQ stats 的窗口长度（avg_rate vs rate 区分）——若启用 `?msg_rates_age=60&msg_rates_incr=10` 这类参数会让 management 端更稳定，但增加耦合，先不引入。
