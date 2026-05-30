# 将 weops-inspect 与 procmon 集成进 WeTerm（monorepo）设计

- 日期：2026-05-30
- 状态：已批准（待写实现计划）
- 涉及仓库：
  - 主仓 `WeTerm`（Go，TUI，tview）
  - `/Users/eric/workspace/weops-inspect`（Go，一次性巡检 CLI）
  - `/Users/eric/workspace/claude_space/script`（即 procmon：Go 采集器 + Python 报告器）

## 1. 背景与目标

WeTerm 当前是基于 tview 的交互式终端运维工具，菜单驱动、SSH 出去执行脚本/命令。
本次目标是把另外两个独立项目的能力收编进 WeTerm，使 WeTerm 成为**统一运维入口**：

- **weops-inspect**：蓝鲸平台一次性全量巡检，覆盖 host/es/redis/mongo/rabbitmq/
  bkmonitor，带规则判定，产出 HTML/JSON 报告与告警邮件。能力强于 WeTerm 现有的
  弱 healthcheck（仅 host/mysql/consul）。
- **procmon**：按进程粒度采集 Linux 资源占用的常驻 agent（远端 cron）+ 本地 Python
  报告器，产出自包含 HTML 报告，用于架构评审/扩容决策。

## 2. 核心设计决策（已确认）

| 决策点 | 选择 |
|--------|------|
| 集成形态 | 全源码收编为 monorepo（单仓维护） |
| Go module 组织 | go.work 多 module 并存（各保留 go.mod） |
| 目录布局 | 子项目各自独立目录（`inspect/`、`procmon/`） |
| 历史保留 | 不保留，直接复制文件（去各自 .git） |
| 原仓处理 | 原封不动保留作备份，只复制 |
| 数据产物 | 只搬源码，jsonl/html/xlsx/data/logs/.pyc 等全部不进仓，写入 .gitignore |
| 菜单入口 | 主菜单新增「平台巡检」「进程监控」两个顶级项 |
| inspect 运行时 | **in-process** 调 Go 包（深度融合） |
| procmon/reporter 运行时 | **shell-out**（运行模型与语言决定，无法进 WeTerm 进程） |
| 旧 healthcheck | 用 inspect 替掉（删除 `pages/healthcheck/` 与 `index/health.go`） |
| procmon 二进制 | 预编译 + `//go:embed` 嵌入 WeTerm |
| 「服务概览」vs「平台巡检」 | 服务概览=轻量速查；平台巡检=全量报告，共用 inspect 引擎 |
| `utils/ssh.go` 统一 | 分两步：本轮先不动，留后续 |

## 3. 整体架构

WeTerm 升级为统一运维入口，但按各子项目的运行模型采用**两种集成深度**：

- **inspect**（本地 Go，与 WeTerm 同构同进程）→ in-process 深度融合。
- **procmon + reporter**（远端常驻 Go agent + 本地 Python）→ shell-out 编排。

### 仓库布局

```
WeTerm/
├── go.work                    # 新增：编排 weterm + inspect 两个 Go module
├── go.mod                     # module weterm（import 路径不变）
├── main.go  cmd/  index/  pages/  utils/  model/   # WeTerm 原有
│
├── inspect/                   # ← weops-inspect 整体搬入（去 .git、去预编译二进制）
│   ├── go.mod                 #   module weops-inspect（import 路径不改）
│   ├── checker/ collector/ notify/ render/ ssh/ config/ model/ lock/
│   └── （原 main.go 改为薄壳，调用新抽出的库函数）
│
└── procmon/                   # ← claude_space/script 搬入（只搬源码）
    ├── cmd/procmon/           #   Go 采集器，独立 module（不进 go.work，保持极轻依赖）
    │   └── go.mod             #   module procmon
    └── reporter/              #   Python 报告器（源码 + requirements.txt）
```

### 关键布局决策理由

- **procmon 的 Go module 故意不纳入 go.work**：它要保持「零依赖、编 ~2MB 静态二进制
  分发到任意 Linux」的特性，纳入 workspace 会让它沾上 WeTerm/inspect 的大依赖树。
  它在仓里仅作为源码 + 构建目标存在。
- **inspect module 纳入 go.work**：使 WeTerm 能 `import "weops-inspect/checker"` 直接
  in-process 调用。
- 三个原仓原封不动保留作备份，只复制不删。
- 数据产物全部不进仓，写入 `.gitignore`。

## 4. 平台巡检（inspect）in-process 融合

### 入口改造

inspect 现为独立 CLI（`main.go` 里 `flag.Parse` → `os.Exit`）。改造为可被库调用：

- 在 `inspect/` 下新增库函数，签名形如：
  `inspect.Run(ctx context.Context, cfg *config.Config, progress chan<- string) (*model.InspectReport, error)`
- 把现在散在 `main.go` 的三阶段流程（采集主机指标 → 规则判定 → 渲染报告）抽进去。
- 原 `main.go` 保留为薄壳调用该库函数，**inspect 仍可独立编译运行**（不破坏其现有
  CI 与单测）。

### 数据流

```
「平台巡检」菜单项
   │ 选中
   ▼
pages/inspect/inspect.go
   │ 复用 WeTerm 进程内已 godotenv.Load 的 env → inspect.config.Load()
   │ 起 goroutine 调 inspect.Run(ctx, cfg, progressChan)
   ▼                                  │ Phase1 采集（SSH 出去）
TUI TextView 实时显示进度  ◀──────────┤ Phase2 规则判定
   │                                  │ Phase3 生成 model.InspectReport
   ▼ Run 返回                          ▼
渲染结果：①TUI 内表格展示 CheckResult 汇总
        ②落地 HTML 报告到输出目录，提示路径
```

### 关键设计点

1. **配置桥接（天然红利）**：inspect 用 `BK_*`/`INSPECT_*` 环境变量；WeTerm 启动时
   已 `godotenv.Load` 了 `/data/install/bin/*/*.env`。两者是同一套环境变量体系，
   inspect 的 `config.Load()` 直接复用同一份进程环境变量，无需额外桥接代码。
2. **进度反馈**：inspect 现用 `fmt.Fprintf(os.Stderr, "[1/3]...")`。改造为通过
   callback/channel 上报，WeTerm 接进 `QueueUpdateDraw` 实时刷 TUI（复用现有
   `ShowShellExecutePage` 的流式输出模式）。
3. **替换旧 healthcheck**：删除 `pages/healthcheck/`（host/mysql/consul/service/
   datalink）与 `index/health.go` 的「服务概览」旧实现，统一由 inspect 的 checker 提供。
4. **取消与超时**：复用 WeTerm 现有 `model.CancelFunc` + ESC 返回主菜单机制；
   `inspect.Run` 接受 `context.Context`，ESC 时 cancel。

## 5. 进程监控（procmon + reporter）shell-out 编排

procmon 的运行模型决定它不可能进 WeTerm 进程：采集器常驻每台目标机跑 cron，
报告器是 Python。WeTerm 作为**编排者**管理其生命周期，命中 WeTerm 已有 SSH 分发能力。

```
WeTerm「进程监控」菜单
   ├─①「部署采集器」─► procmon 二进制 SCP 到目标机 + 写 cron.d（复用 CopyFileBySSH/RunSSH）
   ├─②「拉取数据」───► 从各目标机 rsync/scp 回本地 data 目录
   ├─③「生成报告」───► 本地 exec: python3 -m procmon_report --data-dir ... --out report.html
   │                    输出流进 TUI，完成后提示 HTML 路径
   └─④「卸载采集器」─► 删远端 cron + 二进制（对应 uninstall.sh）
```

### 关键设计点

1. **procmon 二进制来源**：用 `//go:embed` 把预编译的 `procmon-linux-amd64`（静态 ~2MB）
   嵌进 WeTerm，部署时释放并 SCP 到目标机——与 WeTerm 现有 embed 资源套路一致。
   仓里通过 `make build-procmon` 从 `procmon/cmd/procmon` 静态编译重新生成，嵌入文件放
   `pages/procmon/assets/procmon-linux-amd64`。默认只嵌 linux-amd64，其他架构按需再加。
2. **reporter 调用**：shell-out 到 `python3 -m procmon_report`。约束：要求本地装了
   python3 + `reporter/requirements.txt` 依赖。WeTerm 调用前**预检** python3 与依赖
   是否就位，缺失时在 TUI 给出 `pip install` 指引而非崩溃。reporter 源码进仓，但其
   Python 依赖不由 WeTerm 管理。
3. **目标机列表**：复用 WeTerm 已有 `BK_*_IP_COMMA` 环境变量，不引入新的主机配置来源。
4. **数据目录**：本地 data 目录与报告输出路径，沿用 WeTerm 现有「输出到约定目录并
   提示路径」的做法。

## 6. 菜单结构与数据流整合

整合后主菜单（新增 2 项、改造 1 项，其余不动）：

```
主菜单
├── 服务概览          ← 改造：删旧 healthcheck，改调 inspect 快速子集（TUI 内展示健康态）
├── 平台巡检 ★新增     ← inspect 全量：采集→规则判定→HTML 报告+告警
│   ├── 执行全量巡检
│   └── 打开最近报告
├── 进程监控 ★新增     ← procmon 编排（shell-out）
│   ├── 部署采集器
│   ├── 拉取数据
│   ├── 生成报告
│   └── 卸载采集器
├── 信息收集          ← 不动
├── 配置管理          ← 不动
├── 常用操作          ← 不动
├──（AIO 一体机）     ← 不动（AIO=true 时出现）
└── 退出
```

### 两条数据流对照

```
【平台巡检】in-process —— 数据全在内存，类型安全
  WeTerm env ──► inspect.config.Load() ──► CollectAllHosts(SSH)
       ──► checker.Check*() ──► []model.CheckResult ──► ①TUI表格 ②render→HTML

【进程监控】shell-out —— 跨进程，靠文件/exec 交换
  WeTerm ──SCP procmon──► 目标机cron采集──jsonl──► WeTerm rsync回本地
       ──exec python3 reporter──► report.html ──► TUI 提示路径
```

### 新增/删除的 WeTerm 内部文件（遵循现有 pages/<域>/ + index/<域>.go 模式）

新增：
- `pages/inspect/inspect.go` —— 平台巡检页（调 inspect 库、进度流、结果展示）
- `pages/procmon/procmon.go` —— 进程监控页（SSH 编排、reporter shell-out、预检）
- `pages/procmon/assets/procmon-linux-amd64` —— 嵌入的采集器二进制
- `index/inspect.go`、`index/procmon.go` —— 两个新菜单定义
- `go.work` —— 编排 weterm + inspect module

改造：
- `index/root.go` 的 `mainMenuItems`（加 2 项、改 1 项）
- inspect `main.go` → 薄壳 + 新库函数

删除：
- `index/health.go`
- `pages/healthcheck/`（host/mysql/consul/service/datalink + assets）

## 7. 错误处理

贯穿三处集成边界：

- **inspect in-process**：`inspect.Run` 返回 `error`，WeTerm 在 TUI 展示而非 panic
  （顺带修掉现有 healthcheck 里的裸 `panic(err)`）。单台主机采集失败不中断整体
  （保留 inspect 现有 per-host error 机制）。ESC 经 context cancel 干净退出。
- **procmon shell-out**：每步（SCP/rsync/python）检查退出码与 stderr，失败在 TUI
  显红字提示；调 reporter 前预检 python3 与依赖，缺失给出 `pip install` 指引而非崩溃。
- **配置缺失**：`BK_*_IP_COMMA` 等环境变量为空时，菜单项给出明确提示而非空跑。

## 8. 测试策略

- inspect、procmon 各自现有单测原样保留并继续跑：
  - `cd inspect && go test ./...`
  - procmon 独立 `cd procmon/cmd/procmon && go test ./...`
  - reporter `cd procmon/reporter && pytest`
- WeTerm 新增胶水层（配置桥接、进度回调适配）加针对性单测。
- 验收：以下均通过——
  - `go build ./...`（weterm）
  - `cd inspect && go build ./...`
  - `make build-procmon`
  - 且 inspect 仍能独立 `go run` 出报告。

## 9. 迁移步骤（实现阶段大致顺序，细节留给 writing-plans）

1. 建 `go.work`，复制 `inspect/`（去 .git、去预编译二进制），确认 inspect 独立编译 +
   测试通过。
2. 复制 `procmon/`（只源码），`make build-procmon` 产出嵌入二进制，更新 `.gitignore`。
3. 改造 inspect 入口为库函数，WeTerm 接入「平台巡检」菜单。
4. 删旧 healthcheck，「服务概览」改调 inspect 子集。
5. 接入「进程监控」菜单（SSH 编排 + reporter shell-out + 预检）。
6. 全量验收 + 提交。

## 10. 范围边界（本轮不做）

- 统一 `utils/ssh.go`（分两步，留后续）。
- 将 procmon module 纳入 go.work。
- 清理原仓 / 加废弃说明。
- 将 Python 依赖纳入 WeTerm 管理。
- 嵌入 amd64 以外架构的 procmon 二进制（按需再加）。

## 11. 已知风险

- inspect 入口改造若与其现有 CI 假设冲突，需同步调整其 CI 配置。
- 嵌入二进制使 WeTerm 体积增加 ~2MB；procmon 源码改动后需 `make build-procmon`
  重新生成嵌入文件，否则分发的是旧二进制（需在文档/Makefile 中明确这一约定）。
- reporter 依赖本地 Python 环境，跨环境可移植性弱于纯 Go 路径——已通过运行前预检
  缓解，但无法消除环境依赖。
