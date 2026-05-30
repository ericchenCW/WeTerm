## 1. 阈值与配置（config 包）

- [x] 1.1 在 `config/config.go` 的 `Thresholds` 结构体新增 `RabbitMQStalledAckRateMax float64`、`RabbitMQStalledPublishRateMin float64`、`RabbitMQStalledVHostBlacklist []string`
- [x] 1.2 实现解析函数：`INSPECT_RABBITMQ_STALLED_ACK_RATE_MAX` 默认 0.01，非法值（≤0、非数字）返回错误；`INSPECT_RABBITMQ_STALLED_PUBLISH_RATE_MIN` 默认 0.0，允许负数表示关闭
- [x] 1.3 实现 `RabbitMQStalledVHostBlacklist` 解析，默认 `["bk_bknodeman"]`；新变量未设而旧 `INSPECT_RABBITMQ_NO_CONSUMER_VHOST_BLACKLIST` 已设时，沿用旧值并在 stderr 打 deprecation 警告；两者都设以新变量为准
- [x] 1.4 在 `config/config_test.go` 补全：默认值、env 覆盖、非法值、负数关闭语义、新旧变量过渡四种场景

## 2. Collector（采集 message_stats 速率）

- [x] 2.1 在 `collector/rabbitmq.go` 的 `queues` 请求 columns 中追加 `message_stats.ack_details.rate`、`message_stats.publish_details.rate`
- [x] 2.2 新增辅助函数从队列 JSON 解析嵌套 `message_stats.ack_details.rate` 与 `message_stats.publish_details.rate`，缺失视为 `0.0`
- [x] 2.3 在 `model/` 的 `RabbitMQQueue` 结构体新增 `AckRate float64`、`PublishRate float64` 字段（保留 JSON tag 与零值含义）
- [x] 2.4 在 `RabbitMQStatus` 中把 `NoConsumerQueues` 重命名为 `StalledQueues`（更新所有引用：collector / checker / render / output / 测试）
- [x] 2.5 替换 collector 内的筛选逻辑：删除 `consumers == 0 && msgs > 0` 分支，新增 `msgs > 0 && ack_rate < ackMax && publish_rate > pubMin && !blackV` 分支，命中则追加到 `StalledQueues`
- [x] 2.6 在 `collector/rabbitmq.go`（或现有 collector 测试文件）新增单元测试：卡死消费者、无消费者、低流量稳态、健康消费、vhost 黑名单、`message_stats` 缺失六个场景

## 3. Checker（映射到 CheckResult）

- [x] 3.1 在 `checker/rabbitmq.go` 中删除 `NoConsumerQueues` 循环
- [x] 3.2 新增 `StalledQueues` 循环，产出 Field `rabbitmq.<vhost>.<queue>.stalled`，Value 形如 `<msgs> msgs / <consumers> consumers / ack=<ackRate>/s / pub=<pubRate>/s`
- [x] 3.3 Threshold 字段在 checker 内拼出 `"ack < <ε>/s, pub > <min>/s"`（当 `RabbitMQStalledPublishRateMin < 0` 时省略 pub 部分）
- [x] 3.4 更新 `checker/rabbitmq_test.go` / `checker/rules_test.go` 中所有提及 `no_consumer` 的用例为 `stalled`，并补全 Value/Threshold 字段断言

## 4. 签名归一化（notify/signature.go）

- [x] 4.1 修改 `rabbitmqQueueFieldPattern` 正则：把 `no_consumer` 替换为 `stalled`
- [x] 4.2 更新签名归一化逻辑使其折叠 `rabbitmq.<vhost>.<queue>.stalled` → `rabbitmq.<vhost>.stalled`
- [x] 4.3 更新 `notify/signature_test.go`：把现有"no_consumer 同 vhost 不同队列名同签名"用例迁移为 stalled 版本；保留 backlog vs stalled 互不合并的对照用例
- [x] 4.4 添加新用例：旧字段 `rabbitmq.<vhost>.<queue>.no_consumer` 若意外出现 MUST 保持原样（兼容意料外输入，不被新规则误吸收）

## 5. 报告渲染与输出

- [x] 5.1 在 `render/` 与 `output/` 下搜索并替换所有 `NoConsumerQueues` 字段名引用为 `StalledQueues`
- [x] 5.2 更新 HTML/邮件模板列名与说明文字（"无消费者队列" → "消费停滞队列"）
- [x] 5.3 更新 `notify/email_test.go` / `notify/integration_test.go` / `notify/alerts_test.go` 中相关固定值与断言
- [x] 5.4 确认 JSON 输出 schema 变化已被相关测试覆盖（如有 snapshot 测试需要同步更新）

## 6. 文档与发布说明

- [x] 6.1 更新 `README.md`：环境变量表新增三个 stalled 相关变量；告警类型小节把 "no_consumer" 替换为 "stalled" 并列出新 Value/Threshold 格式示例
- [x] 6.2 在 README "告警类型迁移" 段落加入 BREAKING 提示：邮件/HTML/JSON 中 `no_consumer` 字段被 `stalled` 替换，并给出旧黑名单变量的过渡期说明
- [x] 6.3 在 `docs/` 下新增或更新对应章节，说明 stalled 判定公式、ε/publish_rate_min 调参建议、message_stats 缺失的语义
- [ ] 6.4 在 PR description 模板里加入"已在预发集群验证 `/api/queues` 响应大小与 management 节点 CPU 变化"的勾选项

## 7. 集成验证

- [x] 7.1 本地构建并跑全量 `go test ./...`，确认所有相关测试通过
- [ ] 7.2 用 `curl -w '%{size_download} %{time_total}\n'` 在预发集群对比启用前后 `/api/queues` 响应大小与耗时，记录数据到 PR
- [ ] 7.3 在预发集群跑 2-3 个 cron 周期，检查 stalled / backlog 告警分布与人工观察一致
- [ ] 7.4 验证回滚路径：把 `INSPECT_RABBITMQ_STALLED_ACK_RATE_MAX=999999` 设上后 stalled 规则等效禁用，邮件中只剩 backlog 与集群级告警
