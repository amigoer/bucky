# ADR-004：桌面端改为纯 Go 原生界面（MyGo）

- 状态：已采纳
- 日期：2026-10-20
- 相关：[ADR-001](0001-tech-stack.md)、[ADR-002](0002-r2-first-storage.md)、[开发架构](../architecture.md)、[SPEC](../SPEC.md)

## 背景

ADR-001 选定的桌面外壳是 Wails v3 + React：Go 侧只做参数转换，界面全部由 React、Tailwind 和 shadcn/ui 渲染，运行时依赖系统 WebView。实现到 v0.1 中段后，这条路线暴露出几个与产品定位冲突的成本：

- 交付物是「Go 二进制 + 一整套前端构建产物」。启动要先解析 HTML、执行 JS、等 React 挂载，冷启动与内存占用都远高于本地优先工具应有的水平。
- 三平台的一致性由 WebView 承担：WKWebView、WebKitGTK、WebView2 在字体、滚动、拖放、输入法和文件选择上的差异都要单独处理，Linux 还必须装 WebKitGTK。
- 类型安全是「生成」出来的：Go 结构体变化后要重新生成绑定，前端才能编译。断链只能在构建或运行时发现。
- 为了一个对象存储客户端，仓库长期维护两套语言、两套依赖树（pnpm + Go）、两套 lint 与测试命令。

Amigoer 明确要求：完整使用 [MyGo](https://github.com/egoist/mygo) 重构，界面全部用 Go 原生实现，放弃 WebView。

## 决策

### 桌面界面用 MyGo 的原生 UI 工具包

MyGo 同时支持「窗口里放网页」和「窗口里放原生界面」。本项目只用后者：`mygo.NewWindow` 的 `Content` 取 `ui.View(app.view)`，窗口不创建 WebView，不加载页面，不生成 TypeScript 绑定。

- 版本：`github.com/egoist/mygo`，以伪版本锁定到具体提交（当前 `v0.5.6-0.20261020115920-163d50e2acc1`）；升级单独成提交并跑完整回归。
- MyGo 为纯 Go、无 cgo，因此桌面端与 CLI 都可以 `CGO_ENABLED=0` 构建并交叉编译；这修正了 ADR-001 中「桌面端在 macOS 和 Linux 上需要 cgo」的记录。
- 平台要求随之下降：Linux 只需 GTK 3，不再需要 WebKitGTK；Windows 不再需要 WebView2 Runtime。

### 视图是状态的函数，直接调用服务层

界面用 `internal/desktop` 一个包实现，`view(*ui.Context)` 每帧从 `App` 的状态构建界面。窗口内不再有 IPC 边界：视图直接调用 `service.Service` 的方法，因此

- `internal/app`（Wails 绑定层）、`internal/macwindow`（交通灯定位）、`frontend/`（React 工程）全部删除；
- 长任务在 goroutine 中执行，结果通过 `Window.Update` 回到主线程；
- 服务的错误是 `*core.Error`，界面直接读 `Code` 做本地化，不再需要跨边界的 JSON 编组。

分层保持不变：入口适配（`internal/desktop`、`internal/cli`、`internal/agent`）→ 应用服务（`internal/service`）→ 核心契约（`internal/core`）→ 存储适配（`internal/drivers`）。只有入口适配层依赖 MyGo。

### 界面状态与本地化落在 Go 侧

- 偏好设置（主题、强调色、语言、视图、排序、限速、并发、图床默认值）写入 `<数据目录>/ui.json`，不再依赖浏览器 `localStorage`。
- 中英文文案从原 i18n 词表导出为 `internal/desktop/locales/{en,zh}.json`，随二进制嵌入；`{{name}}` 占位符与 `_one`/`_other` 复数规则由 `internal/desktop/i18n.go` 处理。
- 图标取自 Lucide，以 SVG 形式放在 `internal/desktop/icons/`（ISC 许可，附 LICENSE），`ui.Icon` 按文字颜色绘制。

### 原生界面做不到的事如实降级

WebView 能内嵌播放视频、音频和 PDF，原生工具包不能。预览面板因此按类型降级：

- 图片：取预签名链接、下载并解码后在窗口内绘制（PNG、JPEG、GIF、WebP）。
- 文本 / 代码 / JSON：读取前 1 MiB 并在窗口内等宽显示。
- 视频、音频、PDF 及其他二进制：显示元数据，并提供下载与「在浏览器中打开预签名链接」。

图床的 WebP 转换使用纯 Go 的 `nativewebp` 编码器，产出的是**无损** WebP，与前端原先的有损 WebP 在体积上不等价，这一点在文档中如实记录。

### 打包与开发命令

MyGo 自带 `mygo` CLI（`go tool mygo`），负责开发运行、打包 `.app`/`.dmg`/安装包和签名。项目根目录的 `mygo.json` 保存应用名、标识符 `io.amigoer.bucky`、版本和输出目录 `build/`；`resources/icon.png` 是应用图标。Taskfile 的 `dev`、`build:desktop` 改为调用 `go tool mygo`。

## 对已有决策的影响

- 替代 ADR-001 中的「桌面外壳 Wails v3」「前端 React / TypeScript / Vite / Tailwind / shadcn/ui」「前端工程 pnpm / Node / Biome / Vitest」「桌面端需要 cgo」等条目。
- ADR-001 中继续有效的部分：单一 Go module、Go 1.27、AWS SDK for Go v2、系统钥匙串 + SQLite、Cobra、官方 MCP Go SDK、testcontainers-go、Taskfile、golangci-lint。
- ADR-002 的四层结构与 R2 首版范围不受影响，只把「Go + Wails v3 + React」的表述更正为「Go + MyGo 原生界面」。
- ADR-003 的登录流程不变；它原本就发生在 Go 侧，现在只是把「通知前端」换成更新界面状态。

## 备选方案

- **保留 Wails 但换掉 React**：仍然依赖 WebView 与绑定生成，启动与内存问题不变。
- **用 Wails 的其它原生 UI 方案**：v3 没有成熟的原生控件集，等于自研工具包。
- **改用 Fyne / Gio / Wails + 自绘**：都能做到纯 Go，但缺少本项目需要的桌面控件（侧栏、表格、分栏、原生菜单、系统对话框、钥匙串、文件监听）。MyGo 的 `ui` 包一次提供这些，且与 CLI 共用同一套纯 Go 构建约束。
- **继续维持双栈**：短期改动最小，但把上面列出的长期成本一直留在仓库里。

## 后果

- 交付物是单一 Go 二进制，冷启动与内存占用回到本地优先工具的水平；仓库只剩一门语言、一套依赖与命令。
- 类型安全由编译器保证：视图直接调用服务，Go 结构体变化会在编译期暴露。
- 代价是放弃浏览器能力：内嵌音视频/PDF 播放、富文本编辑、图表库和 CSS 生态都不再可用，需要原生重写或降级（见上）。预览面板的降级行为必须逐平台验证。
- MyGo 目前是 0.5.x，API 仍在演进；伪版本锁定 + 独立升级提交是控制风险的手段。
- 界面测试改为 `ui.NewTester` 的无窗口渲染（`internal/desktop/view_test.go`），设置 `BUCKY_SNAPSHOT_DIR` 时会把每页渲染成 PNG，便于人工核对布局。
- 设计稿目录 `design/` 保留为视觉参考，不再参与构建；`frontend/` 已删除。
