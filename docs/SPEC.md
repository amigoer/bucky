# Bucky 需求与开发规范

## 使用方式

本文档即仓库中的 `docs/SPEC.md`，把下面的启动指令发给 Claude Code 即可开始。项目定名 Bucky（原暂名 Bucket Studio）。

首条启动指令：

```text
请完整阅读 docs/SPEC.md，这是本项目的需求与开发规范。
1. 先用你自己的话复述项目定位、v0.1 范围和整体架构，指出你认为有歧义或有风险的地方。
2. 然后只执行「Phase 0：工程脚手架」，动手前先给出详细计划和验收标准，等我确认。
3. 在仓库根目录创建 CLAUDE.md，记录关键约定和技术决策，之后每个阶段结束时更新。
```

后续每个阶段的指令：

```text
继续执行 Phase N。先对照 docs/SPEC.md 列出本阶段的任务清单和验收标准，等我确认后再写代码。
```

## 角色与工作规则

你是一名资深 Go 工程师兼桌面应用开发者，负责从零实现本项目。代码质量要经得起面试官逐行审阅，因为这个仓库也是作者的求职作品。

1. 分阶段推进。每个阶段开始前输出任务清单和验收标准，等我确认后再写代码。
2. 不确定第三方库的 API 时，先读官方文档或模块源码，禁止凭记忆编造接口。所有依赖锁定具体版本。
3. 优先使用纯 Go 依赖，避免 cgo（挂载阶段除外）。
4. 核心包写单元测试，存储相关功能写基于 MinIO 的集成测试。仓库里不得出现任何真实密钥。
5. 小步提交，使用 Conventional Commits，一个提交只做一件事。
6. 重要技术决策写成 ADR，放在 `docs/adr/`，包含背景、决策、备选方案和后果。
7. 删除、覆盖、移动等破坏性操作，在代码层面必须先生成执行计划，经确认后再执行。
8. 不得复刻 Blob0 或任何竞品的界面、文案、图标，设计要有自己的视觉语言。
9. 遇到本文档没写清的产品决策，列出选项和你的建议，等我拍板，不要自行扩大范围。
10. 每个阶段结束时跑通 lint 和全部测试，更新 CLAUDE.md 和 README 的相关部分，给出阶段总结与下一步建议。

## 项目定位与竞品

Bucky 是一款开源、免费、全平台、AI 原生的对象存储桌面客户端：一个客户端管理国内外所有对象存储，并让 AI agent 能安全地操作存储桶。

目标用户有四类：日常管理 R2、S3、OSS、COS 的开发者；用对象存储做图床的博客和文档作者；需要跨云迁移的团队；用 Claude Code、Codex、Cursor 的 agent 用户。

设计思路沿用作者上一个项目 MQ Studio：本地优先的桌面应用，可插拔驱动，每个驱动声明自己的能力，界面只展示该能力支持的功能。两者为姊妹项目。

| 产品 | 形态 | 短板（即我们的机会） |
| --- | --- | --- |
| [Blob0](https://blob0.com/) | 闭源，仅 macOS，16.9 美元买断，带 CLI 和 Agent Skill | 不开源、不免费、没有 Windows 和 Linux、没有国内云原生能力、不能挂载 |
| [Can](https://github.com/100gle/can) | 开源，Go + Wails + React | 仍是 MVP/Beta，厂商适配不完整 |
| [Brows3](https://github.com/rgcsekaraa/brows3) | 开源，Rust + Tauri | 偏海外 S3 生态，没有 AI 和 agent 能力 |
| [AnySCP](https://github.com/macnev2013/anySCP) | 开源，SSH、SFTP、S3 三合一 | S3 只是附属功能 |
| Cyberduck、S3 Browser、各云厂商官方工具 | 老牌工具 | 界面老旧、各管一家，S3 Browser 只支持 Windows |

差异化只有四点，所有功能取舍都围绕它们：

- 全平台、开源、免费。
- 国内外全覆盖，国内云走原生 SDK，能用上 CDN 刷新、图片处理等独有能力。
- AI 原生：CLI、MCP、Skill 三件套，默认安全。
- 后续版本提供跨云迁移和挂载为本地磁盘。

## 技术栈

后端全部用 Go，桌面外壳用 Wails，前端默认 Vue 3。下表中标注「核实」的项，在 Phase 0 查清当前状态后写入 ADR-001。

| 层 | 选型 | 说明 |
| --- | --- | --- |
| 语言 | Go 最新稳定版 | 在 go.mod 中写明版本 |
| 桌面框架 | Wails | 核实 v2 与 v3 当前状态，优先选生产可用的稳定版 |
| 前端 | Vue 3 + TypeScript + Vite + Tailwind CSS + shadcn-vue | 若改用 React + shadcn/ui，全项目统一即可；十万级对象列表必须用虚拟滚动 |
| S3 兼容层 | aws-sdk-go-v2（s3 与分片上传 manager） | 所有厂商的基础读写都走这一层 |
| 国内云原生能力 | 阿里云 OSS Go SDK、腾讯云 cos-go-sdk-v5、七牛 go-sdk | 只用于 CDN 刷新、图片处理等独有功能；核实各 SDK 最新版本 |
| 凭证 | 系统钥匙串（如 zalando/go-keyring） | 本地只存非敏感的连接元数据 |
| 本地数据 | SQLite，纯 Go 驱动 modernc.org/sqlite | 存连接配置、传输历史、统计缓存 |
| CLI | cobra | 与桌面端共用 core |
| MCP | 官方 Go SDK github.com/modelcontextprotocol/go-sdk | 核实最新版本与 API |
| 日志 | log/slog | 日志中禁止出现密钥 |
| 测试 | testing + testify，testcontainers-go 或 docker compose 启动 MinIO | 集成测试进 CI |
| 构建发布 | GitHub Actions 三平台矩阵构建 | macOS 签名与公证放到 v0.2 之后 |

## 架构与目录结构

核心引擎与界面完全解耦：桌面 GUI、CLI、MCP 三个入口共用同一个 core，保证 agent 能做的事和桌面端永远一致。

```text
bucky/
├── cmd/
│   ├── bucky-desktop/     # 桌面应用入口（Wails）
│   └── bucky/             # CLI 入口，含 `bucky mcp` 子命令
├── internal/
│   ├── core/              # 领域模型、Driver 接口、Capability、错误类型
│   ├── drivers/
│   │   ├── s3compat/      # 通用 S3 兼容驱动 + 各厂商预设
│   │   ├── aliyunoss/     # 阿里云 OSS 原生能力
│   │   ├── tencentcos/    # 腾讯云 COS 原生能力
│   │   └── qiniu/         # 七牛原生能力
│   ├── profile/           # 连接配置、钥匙串、导入 ~/.aws/credentials
│   ├── transfer/          # 传输引擎：队列、并发、分片、断点续传、校验、限速
│   ├── ops/               # 高层操作：批量删除、移动、复制，统一 Plan/Execute
│   ├── share/             # 预签名链接、公开链接、自定义域名
│   ├── imagebed/          # 图床：路径模板、压缩、Markdown 链接
│   ├── agent/             # MCP server、Skill 生成与安装
│   ├── store/             # SQLite 持久化
│   └── app/               # 暴露给 Wails 前端的服务层，只做参数转换
├── frontend/              # Vue 3 前端
├── docs/                  # SPEC.md、architecture.md、adr/
├── test/integration/      # 基于 MinIO 的集成测试
└── Makefile
```

三条硬性约定：

- `app` 层只做参数转换和事件推送，不写业务逻辑。
- 所有破坏性操作统一走 Plan/Execute 两步：先生成计划（涉及哪些对象、多少字节、是否覆盖），再执行。GUI 的确认框、CLI 的 `--dry-run`、MCP 的预览工具都基于同一个 Plan。
- 长任务（传输、统计、批量操作）都可取消、可查询进度，状态持久化到 SQLite，应用重启后可恢复。

## 驱动接口设计

驱动层沿用 MQ Studio 的「接口 + 能力声明」模式：基础读写是所有驱动必须实现的接口，厂商独有功能做成可选接口，界面根据声明的能力决定显示什么。下面是方向性草图，具体签名由你在 Phase 1 细化，并写成 ADR-002。

```go
type Capability string

const (
    CapPresign        Capability = "presign"
    CapMultipart      Capability = "multipart"
    CapServerSideCopy Capability = "server_side_copy"
    CapVersioning     Capability = "versioning"
    CapStorageClass   Capability = "storage_class"
    CapLifecycle      Capability = "lifecycle"
    CapCORS           Capability = "cors"
    CapBucketPolicy   Capability = "bucket_policy"
    CapCDNRefresh     Capability = "cdn_refresh"
    CapImageProcess   Capability = "image_process"
)

type Driver interface {
    Name() string
    Capabilities() CapabilitySet
    ListBuckets(ctx context.Context) ([]Bucket, error)
    List(ctx context.Context, in ListInput) (ListPage, error) // prefix + delimiter + 分页 token
    Stat(ctx context.Context, ref ObjectRef) (ObjectInfo, error)
    Get(ctx context.Context, ref ObjectRef, rng *Range) (io.ReadCloser, ObjectInfo, error)
    Put(ctx context.Context, in PutInput) (ObjectInfo, error)
    Multipart(ctx context.Context, ref ObjectRef, opts MultipartOptions) (MultipartUpload, error)
    Copy(ctx context.Context, src, dst ObjectRef) error
    DeleteBatch(ctx context.Context, bucket string, keys []string) ([]DeleteResult, error)
    Presign(ctx context.Context, in PresignInput) (string, error)
}

// 厂商独有能力用可选接口，调用方通过类型断言获取
type CDNRefresher interface {
    RefreshCDN(ctx context.Context, urls []string) error
}
```

首发支持的厂商预设：Cloudflare R2、AWS S3、MinIO、阿里云 OSS、腾讯云 COS、七牛 Kodo、Backblaze B2、Wasabi、DigitalOcean Spaces。每个预设包含 endpoint 模板、region 规则、path-style 或 virtual-host 寻址方式，这些细节一律以厂商官方文档为准，实现时逐一核实，不要凭印象填写。

兼容性提醒：aws-sdk-go-v2 较新的版本默认会计算并发送完整性校验（CRC 类 checksum），部分 S3 兼容服务可能因此报错。请核实这一行为，按厂商把校验配置为仅在必需时计算，并让集成测试覆盖到。

## v0.1 功能范围与验收标准

v0.1 只做下表十个模块，目标是界面精致、传输可靠、agent 好用。跨云同步、安全体检、挂载等全部留到后续版本，不得提前实现。

| 模块 | 必须做到 | 验收标准 |
| --- | --- | --- |
| 连接管理 | 厂商预设、自定义 endpoint、测试连接、多配置切换、导入 ~/.aws/credentials | MinIO 和 R2 均能连通；密钥只存在系统钥匙串 |
| 浏览 | 桶列表、按前缀当文件夹浏览、面包屑、列表与网格视图、排序、当前前缀内搜索、元数据面板 | 10 万对象的前缀滚动流畅，分页加载不阻塞界面 |
| 预览 | 图片、音视频（用预签名链接流式播放）、PDF、文本/代码/JSON（有大小上限） | 不完整下载即可预览大视频；超限文件给出明确提示 |
| 文件操作 | 拖拽上传文件和文件夹、下载、新建文件夹、重命名和移动（复制 + 删除，提示代价）、批量删除 | 所有破坏性操作先展示计划，二次确认后执行 |
| 传输队列 | 并发数可配、大文件自动分片、暂停/继续/取消/重试、断点续传、速度与剩余时间、完成后校验 | 5 GB 文件上传中途断网或杀进程，重启后能续传且校验通过 |
| 分享 | 预签名链接（自定义有效期）、公开链接、按连接配置自定义域名或 CDN 域名 | 一键复制链接；预签名默认有效期 1 小时 |
| 用量面板 | 按前缀和文件类型统计数量与大小，后台计算并缓存，显示统计时间 | 统计不阻塞界面，可随时取消 |
| 图床 | 全局快捷键上传剪贴板图片、路径模板如 `{year}/{month}/{hash}.{ext}`、可选压缩与转 WebP、自动复制 Markdown 链接 | 网络正常时，从截图到拿到链接不超过 3 秒 |
| Agent 接入 | CLI、MCP server、Agent Skill，细节见下一节 | 见下一节的验收场景 |
| 基础体验 | 中英双语、明暗主题、键盘快捷键、友好的空状态与错误提示 | 两种主题、两种语言下逐页走查无错位 |

## AI 原生：CLI、MCP 与 Agent Skill

Agent 能力是首发卖点，三件套共用 core，复用桌面端已配置的连接，默认安全：读操作直接执行，写操作先出计划。

**CLI（命令名 `bucky`）**

| 命令 | 作用 |
| --- | --- |
| `bucky profiles list` | 列出已配置的连接（不含任何密钥） |
| `bucky ls <profile>:<bucket>/<prefix>` | 列出对象 |
| `bucky find` | 按 `--older-than`、`--larger-than`、`--name` 通配、`--sort` 查找对象 |
| `bucky stat` | 查看对象元数据 |
| `bucky cp` | 本地与远端互传，或远端到远端（含跨厂商） |
| `bucky mv` / `bucky rm` | 移动、删除，非交互环境默认只出计划 |
| `bucky presign --expires 7d` | 生成预签名链接 |
| `bucky du` | 统计前缀的数量与大小 |
| `bucky upload-image` | 图床上传，返回 Markdown 链接 |

全局约定：

- 所有命令支持 `--json`，输出结构稳定并在 `docs/cli-schema.md` 中写明；错误同样以 JSON 输出。
- 退出码明确约定：0 成功，1 一般错误，2 参数错误，3 部分失败，4 需要确认。
- `mv`、`rm` 和会覆盖已有对象的写入，在非交互环境下默认等同 `--dry-run`，必须显式加 `--yes` 才执行。
- 连接可在 GUI 中标记为「只读」，被标记的连接在 CLI 和 MCP 中无法写入。
- 密钥永远不出现在输出、日志和任何提示词里。

**MCP server（`bucky mcp`，stdio 传输）**

工具清单：`list_profiles`、`list_objects`、`find_objects`、`stat_object`、`read_text_object`（有大小上限）、`presign`、`usage_summary`、`upload_file`、`plan_operation`、`execute_plan`。复制、移动、删除必须两步走：`plan_operation` 返回计划和 plan id，`execute_plan` 只接受 plan id。按 MCP 规范为每个工具设置只读、破坏性等注解（核实当前规范字段名）。

**Agent Skill**

生成一份 SKILL.md，教 agent 按固定流程使用 CLI：先 `find` 或 `ls` 定位，再 `--dry-run` 出计划，把计划展示给用户，确认后才执行。GUI 设置页提供一键安装，可检测本机的 Claude Code、Codex、Cursor，支持全局或项目级安装；各工具的 skills 目录路径以官方文档为准，实现时核实。

**验收场景**：在 Claude Code 中说「把 backups/ 下 90 天前的文件移到 archive/」，agent 能完成定位、出计划、等待确认、执行、汇报结果的全流程；再说「把 dist 上传到 R2，给我 7 天有效的链接」，一次完成。这两段就是 README 的演示素材。

## 非功能要求与工程质量

这个仓库同时是求职作品，工程质量标准和功能同等重要。下面的数字是目标值，实测结果要写进 README。

| 维度 | 要求 |
| --- | --- |
| 性能 | 冷启动目标 2 秒内；空闲内存目标 150 MB 以内；安装包尽量小 |
| 安全 | 默认无遥测；密钥只存系统钥匙串；日志脱敏；预签名链接默认短有效期；删除二次确认 |
| 可靠性 | 传输状态持久化，崩溃后可恢复；网络调用都有超时和指数退避重试；区分可重试与不可重试错误 |
| 兼容性 | 每个厂商预设都有集成测试或手工验证记录；S3 兼容差异集中在 s3compat 驱动内处理 |
| 代码质量 | golangci-lint 全绿；core、transfer、ops 三个包测试覆盖率目标 70% 以上；transfer 引擎有基准测试 |
| 文档 | `docs/architecture.md` 讲清分层与数据流；重要决策都有 ADR；CLI 输出结构有文档 |
| CI | 每个 PR 跑 lint、单元测试、MinIO 集成测试、三平台构建 |
| 界面 | 自己的设计语言；加载骨架屏、空状态、错误态齐全；动效克制。同类产品被夸最多的就是颜值，首发不能糙 |

## 开发阶段与里程碑

目标是 6 到 8 周发布 v0.1.0，趁同类产品正在热度上抢占时间窗口。顺序是先引擎后界面：CLI 先跑通，能最早验证 core 的正确性。时间紧张时，Phase 6 先只做阿里云 OSS，其余放到 v0.2。

| 阶段 | 内容 | 交付与验收 |
| --- | --- | --- |
| Phase 0 工程脚手架 | 初始化仓库、go.mod、Wails 与前端脚手架、Makefile、golangci-lint、GitHub Actions、docker compose 启动 MinIO、CLAUDE.md、ADR-001 技术栈 | `make dev` 能跑起空壳应用；CI 全绿 |
| Phase 1 core 与 S3 兼容驱动 | Driver 接口与能力声明、s3compat 驱动与厂商预设、profile 与钥匙串、CLI 的 ls、stat、find、presign 和 `--json` | MinIO 集成测试通过；CLI 能连通真实的 R2 |
| Phase 2 传输引擎与 ops | 队列、并发、分片、断点续传、校验、限速；Plan/Execute；CLI 的 cp、mv、rm、du | 断网和杀进程续传测试通过；dry-run 输出正确 |
| Phase 3 GUI 浏览 | 连接管理、对象浏览器、元数据面板、传输面板、拖拽上传 | 10 万对象前缀滚动流畅 |
| Phase 4 预览、分享、图床、用量 | 对应 v0.1 范围表中的四个模块 | 逐项满足范围表中的验收标准 |
| Phase 5 Agent 接入 | MCP server、Skill 生成与一键安装、只读连接 | Claude Code 中跑通 AI 原生一节的两个验收场景 |
| Phase 6 国内云原生能力 | 阿里云 OSS、腾讯云 COS、七牛的原生驱动与独有能力声明 | 每家至少完成连接、浏览、上传、CDN 刷新的手工验证记录 |
| Phase 7 打磨与发布 | 性能实测、中英文 README、演示 GIF、安装说明、三平台安装包 | 打 tag v0.1.0，Release 页面可下载三平台安装包 |

## v0.2 之后的路线图

这些功能 v0.1 一律不做，这里只写清方向，避免架构上把路堵死。

| 版本 | 主题 | 内容 |
| --- | --- | --- |
| v0.2 | 跨云复制与同步 | 两个连接之间按大小、ETag、修改时间对比差异；复制与单向同步；迁移任务可暂停恢复；执行前预估流量费用 |
| v0.3 | 安全体检 | 扫描公开可读的桶、过宽的桶策略、未加密存储；生成体检报告 |
| v0.4 | 自然语言批量操作 | 用户说一句话，生成 Plan，预览后执行；模型可配置为 Claude、OpenAI 兼容接口或本地模型 |
| 持续 | 图床生态 | 评估兼容 PicGo 的本地上传接口（先核实其协议），让 Typora、Obsidian 用户零成本迁移；Obsidian 插件 |
| 持续 | 更多驱动 | Azure Blob、Google Cloud Storage（非 S3 协议）、华为云 OBS、又拍云 |
| 持续 | 分发 | 自动更新、macOS 签名与公证、Homebrew、winget、Scoop |

对架构的影响：v0.2 的跨云复制依赖驱动层的流式读写和传输引擎的远端到远端模式，Phase 1 和 Phase 2 设计接口时就要考虑进去。

## v1.0：挂载为本地磁盘

v1.0 的重磅功能是把存储桶挂载成本地磁盘，在访达侧边栏和资源管理器里像普通文件夹一样使用。它不进 v0.1，但 core 的设计要为它留好接口。

**文件系统层**：优先评估以库的方式复用 rclone 的 VFS 层（Go 编写、MIT 协议，自带本地缓存和延迟回写），不要从零实现。评估结论写 ADR，包括依赖体积、许可证、与自有驱动层如何对接。

| 平台 | 方案 | 说明 |
| --- | --- | --- |
| macOS（默认） | NFS 回环挂载 | 参考 rclone nfsmount：进程内起 NFS 服务，用系统自带 NFS 客户端挂载，用户无需安装任何驱动 |
| macOS（可选） | FUSE-T 的 FSKit 后端 | 需要 macOS 26 及以上，用户需安装 FUSE-T |
| macOS（暂不支持） | macFUSE 的 FSKit 后端 | 社区反馈目前读写仍不可用 |
| macOS（长期） | File Provider 扩展 | iCloud 同款体验，需用 Swift 编写扩展、开发者账号签名，Go 核心通过进程间通信对接 |
| Windows | WinFsp 挂载为盘符 | 用户需安装 WinFsp；长期评估 Cloud Files API |
| Linux | FUSE | 标准方案 |

必须做到：

- 先只读，再读写。读写模式必须有本地缓存和回写队列，界面上显示尚未同步的文件。
- 默认过滤 `.DS_Store`、`._*` 等系统文件；对挂载卷禁用 Spotlight 索引和缩略图预取。
- 显示缓存占用（可设上限），以及本次挂载的请求次数和流量估算，避免用户收到意外账单。
- 文档写清适用场景（素材库、归档、浏览和拖拽、在线打开大文件）和不适用场景（数据库、git 仓库、`node_modules`、频繁读写小文件）。

参考资料：[rclone nfsmount 说明](https://nix-community.github.io/home-manager/options/home-manager/programs/rclone.html)、[FUSE-T 的 FSKit 后端实测](https://forum.rclone.org/t/macos-rclone-mount-with-fuse-t-via-fskit/53608)、[macFUSE 的 FSKit 后端现状](https://forum.rclone.org/t/macfuse-fskit-backend/53037)。

## README 与发布素材

README 是拿 star 的第一现场，Phase 7 按下面的要求制作，不能临时凑。

- 首屏：一句话定位、演示 GIF（Claude Code 通过 MCP 完成「上传 dist 并生成 7 天有效链接」）、三平台下载按钮。
- 对比表：与 Blob0、Cyberduck、S3 Browser、云厂商官方工具对比，只写可核实的事实，措辞中立。
- 能力矩阵：支持的厂商 × 支持的功能，由驱动的能力声明自动生成，避免文档和代码不一致。
- 中英文两版 README；安装说明包含 macOS 未签名应用的打开方法。
- 截图：明暗两套主题，使用真实但脱敏的数据。
- 写明实测的安装包体积、冷启动时间和内存占用。

发布节奏：先在 X 上的中文独立开发者圈、V2EX、即刻首发，再投 HelloGitHub，最后上 Product Hunt 和 Hacker News。MQ Studio 与本项目的 README 互相导流。

命名：定名 Bucky（原暂名 Bucket Studio），GitHub 仓库为 amigoer/bucky，CLI 命令为 `bucky`。Phase 0 时检查 Homebrew 包名和域名是否可用，有冲突就列出备选方案等我决定。
