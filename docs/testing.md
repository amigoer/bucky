# 测试与验证记录

- 更新日期：2026-10-19
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

- `go test -race ./internal/... ./cmd/...`：通过。当前单元测试覆盖 SQLite 存储、传输本地路径安全与目标文件检查、Cloudflare REST/OAuth 驱动和服务层登录流程。
- `go vet ./...`：通过。
- `golangci-lint run ./...`：0 个问题。
- `tsc -b`：通过。
- Biome：退出码 0，无告警；CSS 解析启用 Tailwind 指令。
- Vitest：5 个测试文件、30 个测试通过（格式化、列表语义、计划结果、快速上传模板、Cloudflare 登录状态机与会话取消/重试）。
- Vite 生产构建：通过。输出单个 JS chunk，约 813 kB（gzip 约 245 kB），CSS 约 61 kB（gzip 约 11.5 kB）。
- `CGO_ENABLED=0 go build ./cmd/bucky`：通过。
- Wails 绑定生成：1 个服务、41 个方法、28 个模型、4 个事件。

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

### 3.6 自绘窗口 chrome

桌面窗口现在由前端绘制 52px 标题栏。macOS 保留原生窗口边框，隐藏原生标题文字与背景，并通过 `internal/macwindow` 的 Objective-C 辅助代码把系统红绿灯按钮重新对齐到标题栏中心；Windows 和 Linux 使用无边框窗口，由前端绘制最小化、最大化/还原和关闭按钮。标题栏整体是 `--wails-draggable: drag` 区域，按钮单独排除拖动。

在生产 bundle 上分别以模拟的 darwin 和 windows 环境运行 WKWebView 测试：当时的 40px 标题栏测试已通过。界面重构后标题栏高度调整为 52px，macOS 保留 78px 左侧空间且不渲染自绘按钮，Windows / Linux 渲染 3 个窗口按钮，拖动属性均为 `drag`；新高度的原生窗口交互仍需人工走查。点击 3 个按钮分别发出的窗口方法为 17、39、2，即 `Minimise`、`ToggleMaximise`、`Close`。macOS 原生应用已重新构建并启动，窗口存在。

Windows 和 Linux 的按钮点击、无边框缩放、贴靠和最大化事件没有在真实平台验证；macOS 红绿灯的实际像素位置也没有自动截图，仍需人工走查。由于当前终端没有屏幕录制和辅助功能权限，没有执行点击、拖拽和文件选择。

### 3.7 界面重构验证（Web mock 后端）

使用 `pnpm dev` 的浏览器 mock 后端（不是 Wails 桌面运行时，也不是真实 R2）完成以下走查：

- 亮色与深色主题、侧栏折叠、页面切换与 ⌘F 搜索面板。
- 列表 / 图标 / 分栏视图、面包屑导航、排序、分页加载、全选与单选、浮层选择栏。
- Quick Look 文本 / 图片 / 媒体 / PDF 分支与元数据、分享对话框的临时链接与公开链接切换。
- 删除计划：自动生成、确认前禁用危险按钮、执行后刷新列表并清除选择；逐项结果不再误报空错误。
- 复制 / 移动计划的同目录与“搬入自身”校验；只读连接的只读横幅、禁用上传以及仅保留下载 / 分享。
- 快速上传：粘贴图片、模板路径、WebP 输出、上传后对象出现在目标 prefix、Markdown / URL 复制状态。
- 用量扫描进度与结果、传输页筛选与限速控件、设置页主题 / 语言 / 限速 / 快捷键 / 集成片段。

上述结果只证明前端交互与 mock 契约一致；真实 Wails 事件、R2 响应和平台 WebView 行为仍以第 4 节为准。

### 3.8 Cloudflare 浏览器登录（假服务）

`internal/drivers/cloudflare` 与 `internal/service` 用 `httptest` 假服务覆盖了完整链路，不需要真实 Cloudflare 账号：

- 驱动：基础 URL 与回环回调校验、Bearer 头、状态码到错误码映射、无效信封、令牌请求体（单桶资源串与权限组 ID）、只读令牌选读权限组、非法桶名拒绝、令牌 `value` 到 Secret 的 SHA-256 派生、jurisdiction 请求头、RFC 7636 PKCE 向量、授权 URL 参数、授权码交换、state 不匹配拒绝、用户拒绝授权、回调端口释放、被拒的 grant。
- 服务：登录 → 回调 → 账号与桶解析 → 铸单桶令牌 → 派生凭证 → 经 `Factory` 真实调用一次 S3 列举验证 → 保存连接（`credentialSource=cloudflare`）；刷新令牌只进钥匙串、不出现在数据库记录中；只读连接铸读权限组；验证失败时删除已铸令牌且不落盘；重新授权替换旧令牌；取消登录会关闭回环监听并释放端口；并发登录返回 `conflict`；未配置 Client ID 返回 `configuration`；设置读写与校验；断开连接需确认并删除令牌、撤销刷新令牌、清理钥匙串；删除连接同时清理两项钥匙串条目。

补充回归：保存只读权限变更会创建新令牌、验证并替换原令牌；手填密钥在任何 Cloudflare 令牌操作前被拒绝；改名保留原令牌；钥匙串或数据库写入失败时恢复原条目、保留原连接且只删除新令牌；新连接保存失败不残留凭证；已有连接重新登录到其他账号时在铸令牌前拒绝。

结果：全部通过。真实 Cloudflare 行为见第 4 节。

### 3.9 Cloudflare 登录界面（Web mock 后端）

使用浏览器 mock 后端（不是 Wails 桌面运行时，也不是真实 Cloudflare）走查，mock 的登录在 1.5 秒后自行完成授权：

- 设置页 Integrations 的 Cloudflare 卡片：Client ID 保存与校验、回调地址展示与复制、scope 列表、创建客户端入口。
- 添加连接：整窗左右两栏。左栏是当前这一步（未授权是说明与两个入口、等待授权是授权链接与三个操作、授权后是连接名称与只读开关），右栏随阶段变化（能力说明 → 三步进度 → 真实的账号与存储桶列表）。桶在右栏点选，不再用下拉；连接后侧栏出现新连接并可进入该桶。未配置 Client ID 时留在页面上，左栏给出原因，右栏出现「打开设置」跳到 Integrations。
- 已有连接：未授权时显示「授权缺失」与「重新授权 / 重新登录」，授权后显示令牌名与「重新授权 / 重新登录 / 断开连接」；重新授权与重新登录都在原连接上就地替换授权；断开连接需二次确认，确认后连接保留、状态回到未授权；删除连接需二次确认，删除后连接从侧栏消失。
- 补充回归：等待授权时关闭弹窗，再次打开后可重新登录并选桶；已有连接取消登录后等待状态消失且按钮恢复，再次登录可成功；重新登录后侧栏仍只有一条原连接。mock 现在按 `profileId` 更新原连接，并保留创建时间；缺少授权时刷新会失败，不再假报成功。
- 会话辅助逻辑单元测试覆盖重复点击合并、关闭后晚返回的登录自动取消、取消完成前不启动下一次登录；账号匹配与授权/取消终态独立测试通过。
- 全程无浏览器控制台错误。

上述结果只证明前端交互与 mock 契约一致；真实 Wails 事件、系统浏览器跳转和 Cloudflare 响应仍以第 4 节为准。

## 4. 尚未验证

以下项目没有真实环境结果，不能视为通过：

- **真实 Cloudflare R2**：没有提供凭据。真实 R2 的条件 PUT、CopyObject 目标条件、删除竞态、预签名链接、jurisdiction 和桶级凭证行为均未验证。MinIO/RustFS 通过不代表 R2 通过。
- **真实 Cloudflare OAuth**：没有用真实账号走通。回环回调是否被 Cloudflare 接受、scope 的精确字符串、`POST /accounts/{id}/tokens` 是否接受单桶资源串与运行时查到的权限组 ID、敏感权限 `Account API Tokens Write` 对私有客户端是否可用，均只有官方文档与假服务结果。授权页跳转、系统浏览器打开和撤销后的实际效果也需要人工走查。
- **5 GiB 以上恢复**：只在本地容器上验证了 18 MiB 分片上传和 2 MiB 下载的暂停/恢复；没有验证 5 GiB 文件、进程被杀、重启后接管或断网重连。
- **Windows 与 Linux**：没有构建或运行。钥匙串、文件选择、拖拽、剪贴板、原子落盘和临时文件删除在该平台的行为未验证。
- **真实网络与 CORS**：没有在 WebView 中打开真实 R2 预签名媒体，CORS、视频范围请求和长时间预签名未验证。
- **大目录性能**：没有构造 10 万对象数据集，虚拟滚动、搜索遍历和用量扫描的交互表现未测量。
- **桌面发布**：仅在 macOS 生成过本地 ad-hoc 签名的 `.app`，没有 Developer ID 签名、公证、DMG 安装包或首次打开流程验证。
- **快速上传的真实后端行为**：剪贴板图片上传、命名模板、压缩/WebP、Markdown 链接已在 Web mock 后端走通，尚未在真实 R2 上验证冲突、公开链接和图片转换结果。
- **Skill 安装器**：Skill 一键安装尚未实现。
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
- Cloudflare 登录的回调端口固定（默认 8976）。端口被占用时无法开始登录，需要改回调地址并同步更新 OAuth 客户端。同一时间只允许一个登录会话。
- Cloudflare 连接限定单桶；保存只读权限、桶或地域管辖变更时自动刷新授权并创建、验证替代令牌，成功保存后才删除旧令牌。改名和公开域名不替换令牌；缺少刷新授权时需重新登录。界面中的账号和桶当前固定，不能通过重新登录切换到其他账号。
