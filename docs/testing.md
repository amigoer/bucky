# 测试与验证记录

- 更新日期：2026-10-18
- 状态：v0.1 实现进行中
- 适用代码：`main` 分支当前提交

本文区分三类结果：已在本机通过的自动检查、已在容器后端通过的协议测试，以及尚未验证的真实环境行为。没有真实凭据或目标平台时，不把本地结果当作真实 R2 或三平台结论。

## 1. 测试环境

| 项目 | 版本或说明 |
| --- | --- |
| Go | 1.27.2 |
| Node.js | 24.22.0 |
| pnpm | 12.14.0 |
| Docker | OrbStack 29.4.0，仅用于本机集成测试 |
| 操作系统 | macOS（Darwin 27.0.0） |

集成测试使用固定镜像摘要：

- MinIO：`cgr.dev/chainguard/minio@sha256:709099319add48dfb44aa596a7070d3ddd8dd04d7ae881765719aec29df9e510`
- RustFS：`docker.io/rustfs/rustfs:1.0.3@sha256:9ca9559533b537d10482fe923288b676ad08f92a7069d9a17327fbae3cab688b`

两个镜像都只作为 S3 协议测试后端，不是 Bucky 支持的产品连接类型。

## 2. 常用命令

Go 单元测试与静态检查：

```bash
go test -race ./internal/... ./cmd/...
go vet ./...
golangci-lint run ./...
```

前端检查与生产构建：

```bash
pnpm --dir frontend typecheck
pnpm --dir frontend lint
pnpm --dir frontend test
pnpm --dir frontend build
```

需要 Docker 的协议与端到端测试：

```bash
go test -tags integration -count=1 -timeout 15m ./test/integration
```

只跑某一个后端：

```bash
go test -tags integration -run 'TestBackendContract/minio' -v ./test/integration
go test -tags integration -run 'TestServiceWorkflows/rustfs' -v ./test/integration
```

无 CGO 的 CLI 构建：

```bash
CGO_ENABLED=0 go build -trimpath -buildvcs=false -o bin/bucky ./cmd/bucky
```

桌面构建与本地运行：

```bash
task build:desktop
task run
```

## 3. 已通过的本地检查

### 3.1 Go 与前端

- `go test -race ./internal/... ./cmd/...`：通过。当前单元测试覆盖 SQLite 存储、传输本地路径安全与目标文件检查。
- `go vet ./...`：通过。
- `golangci-lint run ./...`：0 个问题。
- `tsc -b`：通过。
- Biome：退出码 0；仅剩 CSS 的 `noImportantStyles`（减少动态效果块）和 `noDescendingSpecificity` 警告。
- Vitest：1 个测试文件、5 个测试通过。
- Vite 生产构建：通过。输出单个 JS chunk，约 466 kB（gzip 约 144 kB），CSS 约 32 kB。
- `CGO_ENABLED=0 go build ./cmd/bucky`：通过。
- Wails 绑定生成：1 个服务、25 个方法、21 个模型、2 个事件。

### 3.2 协议契约（MinIO 与 RustFS）

`TestBackendContract` 对两个后端执行同一组断言：

- 列举分页、文件夹分隔符、递归列举、特殊字符 key、零字节对象。
- `Stat` 元数据与 Content-Type、完整读取、范围读取、`If-Match` 不匹配。
- `If-None-Match: *` 创建保护、`If-Match` 覆盖、过期 ETag 冲突。
- 分片上传的创建、上传、列举、完成、中止与非法分片尺寸。
- 预签名完整读取和范围读取。
- 特殊字符 key 的服务端复制、元数据保留、条件覆盖、过期源冲突。
- 删除的过期 ETag 冲突、缺失对象逐项结果。
- 错误凭证到 `credentials_unavailable` 或 `permission_denied` 的映射。

结果：MinIO 与 RustFS 均通过。

### 3.3 服务层工作流（MinIO 与 RustFS）

`TestServiceWorkflows` 使用真实的 `service.Service`、临时 SQLite、假钥匙串和本地容器：

- 连接保存、连接测试、指定桶。
- 小文件上传、下载、预签名分享。
- 覆盖上传生成计划、未确认拒绝执行、确认后异步上传并核对内容。
- 17 MiB 分片上传与下载哈希一致。
- 复制、移动、删除计划；未确认时退出；执行后源和目标状态正确。
- 计划生成后源对象被外部修改，执行返回 `conflict`，计划保持 `pending`。
- 计划生成后本地文件被修改，执行返回 `conflict`。
- 只读连接拒绝上传和删除计划，但仍允许列举。
- `Find` 的完整性与过滤、`Usage` 的对象数、字节数和类型统计。
- 限速下暂停并恢复下载，最终哈希一致。
- 限速下暂停并恢复 18 MiB 分片上传，列出远端分片后继续，最终下载哈希一致。

结果：MinIO 与 RustFS 均通过。

### 3.4 CLI 与 MCP

CLI 使用临时数据目录做过以下冒烟检查：

- `profiles list` 文本和 JSON 输出。
- `profiles add --environment`、`show`、`remove`。
- 未加 `--yes` 的删除返回 `confirmation_required` 和退出码 4。
- 引用不存在的连接、对象或计划返回退出码 1。
- 参数错误与未知子命令返回退出码 2。
- 构造 `partial` 计划后，`plan ID --json` 仍输出单个 JSON 对象，退出码为 3。
- 未知的密钥类参数不会把值写进输出。

MCP 通过 stdio 完成 `initialize` 与 `tools/list`。当前工具集：

`buckets`、`create_plan`、`find`、`get_plan`、`list`、`presign`、`profiles`、`read_text`、`stat`、`transfers`、`usage`。

工具集中没有上传、下载、删除、复制、移动或执行计划的工具。MCP 只开放读取和计划创建，写执行必须回到 GUI 或 CLI 的人工确认流程。

### 3.5 桌面启动

生产前端最初把 Wails 运行时、生成绑定和模型拆成互相引用的 chunk。WKWebView 的模块求值顺序使 service models 在 core models 的 `Profile` 初始化前执行，连接列表查询报 `undefined is not an object (evaluating 'r.createFrom')`。改为静态导入生成绑定后，生产构建只输出一个 JS chunk，不再存在循环 chunk。

修复后在 WKWebView 测试页中以空连接列表响应启动，连接区正常显示 “Add R2 connection”，没有模块错误。macOS 原生二进制已重新构建并打包为本地 ad-hoc 签名的 `.app`，启动后进程保持运行，系统窗口列表中存在 1279×871 的窗口。

由于当前终端没有屏幕录制和辅助功能权限，没有自动截图，也没有执行点击、拖拽和文件选择。界面视觉与交互仍需人工走查。

## 4. 尚未验证

以下项目没有真实环境结果，不能视为通过：

- **真实 Cloudflare R2**：没有提供凭据。真实 R2 的条件 PUT、CopyObject 目标条件、删除竞态、预签名链接、jurisdiction 和桶级凭证行为均未验证。MinIO/RustFS 通过不代表 R2 通过。
- **5 GiB 以上恢复**：只在本地容器上验证了 18 MiB 分片上传和 2 MiB 下载的暂停/恢复；没有验证 5 GiB 文件、进程被杀、重启后接管或断网重连。
- **Windows 与 Linux**：没有构建或运行。钥匙串、文件选择、拖拽、剪贴板、原子落盘和临时文件删除在该平台的行为未验证。
- **真实网络与 CORS**：没有在 WebView 中打开真实 R2 预签名媒体，CORS、视频范围请求和长时间预签名未验证。
- **大目录性能**：没有构造 10 万对象数据集，虚拟滚动、搜索遍历和用量扫描的交互表现未测量。
- **桌面发布**：仅在 macOS 生成过本地 ad-hoc 签名的 `.app`，没有 Developer ID 签名、公证、DMG 安装包或首次打开流程验证。
- **图床与 Skill 安装器**：剪贴板图片上传、命名模板、压缩/WebP、Markdown 链接和 Skill 一键安装尚未实现。
- **跨平台三平台构建链路**：当前只在 macOS 完成 Wails 原生构建。

## 5. 测试数据与安全约定

- 集成测试只使用容器内临时桶和固定测试密钥，不读取真实凭证。
- 失败信息只输出经过 `core.Error` 清洗的错误码和消息，不输出 SDK 原始错误、签名 URL 或凭证。
- 测试结束后中止未完成的分片上传，再删除对象和桶。
- 每个服务测试使用独立临时数据目录和独立假钥匙串。
- 仓库不得包含真实密钥、签名链接或用户数据。

## 6. 已知限制

- R2 的 CopyObject 目标条件和 DELETE 条件支持不完整。适配器在 R2 模式下使用 HEAD 预检，并在计划中提示检查与写入之间仍可能有竞态。这是有意的保守实现，不是原子操作。
- 单次 PUT 已经提交但取消信号随后到达时，任务可能显示为 cancelled，而远端对象已经存在。重新执行前应以对象状态为准。
- 连接从钥匙串切换到环境凭证后，旧钥匙串条目不会自动删除；可在系统钥匙串中手动清理。
- 列表响应不包含 Content-Type 时，界面根据文件扩展名显示类型标签。
- 下载落盘使用原子链接；Windows 上临时文件删除和句柄语义尚未验证。
