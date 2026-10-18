# ADR-001：技术栈选型

- 状态：已采纳
- 日期：2026-10-18

## 背景

SPEC 的「技术栈」一节给出了方向性选型，并要求查清各项的当前状态后写入本 ADR。作者已拍板桌面外壳用 Wails v3、前端用 React。本 ADR 记录最终选型和 2026-10-18 的核实结果。

下文版本号是核实当天的最新版，作为 Phase 0 锁定依赖的基线。届时如有新的补丁版本可以直接采用，并同步更新本表；跨次版本升级单独提交。

## 决策

### 桌面外壳：Wails v3

- 版本：`github.com/wailsapp/wails/v3 v3.0.0-beta.31`，前端运行时 `@wailsio/runtime 3.0.0-beta.31`，两者必须同版本。
- 现状：v3 仍是 beta，GitHub 上标为 prerelease，大约每周发一个 beta；v2 的最新稳定版是 v2.17.0。
- 选择理由：v3 是 Wails 的主力开发线；原生支持多窗口和系统托盘，图床常驻、独立传输窗口都用得上；绑定改为 Service 模式并生成 TypeScript 类型，契合 `app` 层「只做参数转换」的分层。
- 约束：
  - 锁定精确的 beta 版本，禁止使用 `latest`；升级 Wails 单独成一个提交，并跑完整回归。
  - 只有 `cmd/bucky-desktop` 和 `internal/app` 可以 import Wails，core 及以下各层不得依赖它，从而把 v3 API 变动的影响面限制在这两处。
  - 构建统一用 Wails v3 自带的 Taskfile（go-task v3.54.0），不再另建 Makefile；Phase 0 的验收命令由 `make dev` 改为 `task dev`。
  - Wails v3 模板默认把 `main` 包放在仓库根目录。我们保留 `cmd/bucky-desktop/`，在生成的 `build/` Taskfile 中改构建路径；Phase 0 验证 `wails3 dev` 和绑定生成在这种布局下可用，不可行就退回根目录 `main.go` 并更新 SPEC。
  - 桌面端在 macOS 和 Linux 上需要 cgo。CLI `bucky` 不依赖 Wails，必须能以 `CGO_ENABLED=0` 构建和交叉编译。

### Go

- Go 1.27（核实时最新为 1.27.2）；Wails v3 beta.31 要求 Go 1.26 及以上。
- 单一 Go module：`github.com/amigoer/bucky`。

### 前端：React

| 用途 | 选型 | 版本 |
| --- | --- | --- |
| UI 框架 | React | 19.3.1 |
| 语言 | TypeScript | 7.0.2 |
| 构建 | Vite + @vitejs/plugin-react | 8.3.4 / 6.1.2 |
| 样式 | Tailwind CSS + @tailwindcss/vite | 4.3.3 |
| 组件 | shadcn/ui（CLI `shadcn`），图标 lucide-react | 4.22.0 / 1.59.0 |
| 虚拟滚动 | @tanstack/react-virtual | 3.14.21 |
| 异步数据 | @tanstack/react-query | 5.104.4 |
| 界面状态 | zustand | 5.0.15 |
| 国际化 | i18next + react-i18next | 26.5.2 / 17.1.0 |
| Lint 与格式化 | Biome | 2.5.15 |
| 单元测试 | Vitest | 5.0.5 |
| 包管理 | pnpm，Node 24 LTS | Node 24.22.0 |

- Wails 的 React 模板仍是 React 18 + TypeScript 5，Phase 0 生成后升级到上表版本。
- 数据分工：调用 Wails 绑定拿到的远端数据走 TanStack Query（缓存、分页、取消）；纯界面状态（选中项、视图模式、面板开关）走 Zustand。
- 选 TypeScript 7 + Biome：typescript-eslint 8.71.2 只支持 TypeScript `<6.1.0`，无法与 TS 7 搭配。Biome 一个工具包办 lint 和格式化，也不依赖 TS 编译器 API。
- shadcn 的 Vite 安装步骤会改 tsconfig 的路径别名，Phase 0 要确认这套配置在 TS 7 下可用。
- 路由库暂不引入，Phase 3 根据页面结构再定。

### 后端依赖

| 用途 | 模块 | 版本 |
| --- | --- | --- |
| S3 兼容层 | github.com/aws/aws-sdk-go-v2（+ service/s3、config） | v1.47.3（s3 v1.114.3，config v1.33.9） |
| 阿里云 OSS | github.com/aliyun/alibabacloud-oss-go-sdk-v2 | v1.6.1 |
| 腾讯云 COS | github.com/tencentyun/cos-go-sdk-v5 | v0.7.75 |
| 七牛 | github.com/qiniu/go-sdk/v7 | v7.29.0 |
| 钥匙串 | github.com/zalando/go-keyring | v0.2.8 |
| SQLite | modernc.org/sqlite | v1.60.1 |
| CLI | github.com/spf13/cobra | v1.10.2 |
| MCP | github.com/modelcontextprotocol/go-sdk | v1.8.0 |
| 日志 | log/slog | 标准库 |
| 测试 | github.com/stretchr/testify、github.com/testcontainers/testcontainers-go | v1.12.1 / v0.44.0 |
| Lint | github.com/golangci/golangci-lint/v2 | v2.15.0 |

- **分片上传不用 SDK 的 manager**：`feature/s3/manager` 在 v1.23.14 中已标记 Deprecated，由仍处于 v0.4.x 的 `feature/s3/transfermanager` 取代。两者都只在进程内存里维护分片状态，满足不了「杀进程后续传」的验收。transfer 引擎直接调用 service/s3 的 `CreateMultipartUpload`、`UploadPart`、`ListParts`、`CompleteMultipartUpload` 和 `AbortMultipartUpload`，把 upload id 和已完成的分片写进 SQLite。这两个包都不引入。
- **完整性校验**：service/s3 的 `Options` 提供 `RequestChecksumCalculation` 和 `ResponseChecksumValidation`，取值包括 `aws.RequestChecksumCalculationWhenRequired` 和 `aws.ResponseChecksumValidationWhenRequired`。Phase 1 按厂商预设配置，并让集成测试覆盖。
- **阿里云用 v2 SDK**：旧版 `aliyun-oss-go-sdk` 最后一次发版是 2023-12，不再使用。
- **CDN 刷新另需 SDK**：阿里云和腾讯云的 CDN 刷新属于各自 CDN 产品的 OpenAPI，不在 OSS / COS SDK 里，需要额外引入 `github.com/alibabacloud-go/cdn-20180510` 和 `github.com/tencentcloud/tencentcloud-sdk-go/tencentcloud/cdn`；七牛 go-sdk 自带 `cdn` 包。这些到 Phase 6 再核实版本，Phase 0 不引入。
- **钥匙串**：go-keyring 在三个平台上都不需要 cgo。macOS 通过 stdin 把密码交给 `security -i`，不会出现在进程参数里；单条命令上限 4096 字节，存 access key 足够。Linux 依赖 D-Bus Secret Service，没有桌面会话的服务器上用不了，Phase 1 需要决定这种环境下 CLI 从哪里取凭证。

### 集成测试后端：MinIO + RustFS

- MinIO 社区版仓库已于 2026-04 归档并停止维护。核实当天，Docker Hub 上的 `minio/minio` 已无法匿名拉取，quay.io 同样要求认证。
- 同一套集成测试对两个后端各跑一遍：
  - RustFS：`rustfs/rustfs:1.0.3`（Apache-2.0，活跃维护）。
  - MinIO：`cgr.dev/chainguard/minio`，由 Chainguard 基于最后一版社区源码持续构建。免费版只提供 `latest` tag，Phase 0 按 digest 锁定；备用镜像是已冻结的 `bitnamilegacy/minio:2025.7.23-debian-12-r5`。
- 测试代码用 testcontainers-go 的通用容器启动后端，不用其 minio 模块（它的默认镜像来自已停更的 quay.io）。另提供 docker compose 供本地手动调试。
- MinIO 仍保留为厂商预设，因为存量的自建实例很多。

## 备选方案

- **Wails v2.17.0**：API 稳定，但新能力都在 v3 上开发，日后从 v2 迁移到 v3 的成本高于现在直接上 v3。
- **Vue 3 + shadcn-vue**：SPEC 原先的默认选项。按作者决定改用 React，shadcn/ui 和 TanStack 系列都以 React 为第一支持目标。
- **TypeScript 6.0 + ESLint + typescript-eslint + Prettier**：规则最全、生态最成熟，但停留在上一代 TypeScript，迟早要再迁移一次。
- **aws `feature/s3/manager` 或 `transfermanager`**：前者已弃用，后者仍是 v0.x，而且两者都不支持跨进程续传。
- **只用 MinIO 旧镜像**：镜像不再更新，来源也在收紧，长期看是一个死依赖。
- **只用 RustFS**：测不到 MinIO 的行为差异，而 MinIO 是首发预设之一。

## 后果

- Wails v3 在 beta 期间 API 可能变化，需要持续跟进版本；影响面被限制在 `cmd/bucky-desktop` 和 `internal/app` 两处。
- 自己管理分片状态会增加 transfer 引擎的复杂度，但这是续传验收的前提，也是 v0.2 远端到远端复制要复用的能力。
- Biome 没有基于类型信息的 lint 规则，需要靠 TypeScript 严格模式和测试来弥补。
- MinIO 镜像依赖第三方构建，digest 失效或拉取受限时要更换来源；RustFS 保证即使 MinIO 暂时不可用，CI 仍有一个后端可跑。
- CI 要跑两遍集成测试，耗时会增加几分钟。
