# 桌面端对抗性评审（MyGo 原生界面）

- 日期：2026-10-20
- 范围：`internal/desktop`（26 个源文件，约 8,000 行）、`cmd/bucky-desktop`
- 对照：`design/`（设计基线与原型）、[ADR-004](../adr/0004-native-go-desktop-ui.md)、`docs/SPEC.md`、项目 `CLAUDE.md`
- 方法：通读全部桌面端代码；用 `BUCKY_SNAPSHOT_DIR` 渲染 18 张无窗口快照（明暗各一）；在 macOS 上运行 `task dev`，在真实窗口里点击走查；在 1280×800 下打开设计原型逐页对照
- 状态：只记录问题，本轮不改代码

严重度：

| 级别 | 含义 |
| --- | --- |
| P0 | 数据或安全语义错误、核心流程不可用 |
| P1 | 明确的功能缺陷，或界面给出错误信息 |
| P2 | 视觉与交互问题 |
| P3 | 代码质量与可维护性 |

## 0. 结论

1. **设计基线没有落地。** `design/README.md` 规定的暖中性底色、赤陶单一品牌色 `#B45A36`、衬线展示字体、页面标题与表格工具栏分层，在原生实现里全部没有。现在的界面是冷灰底 + 系统蓝 + 一排小号无衬线文字，这是“丑”的主要原因，不是个别控件的问题。
2. **原生移植丢了功能，并且没有记录。** 本地化词表里约 130 个键没有被任何代码使用，对应的是 React 版本有、原生版本没有的功能：计划逐项结果、Cloudflare 登录主入口、表单校验文案、传输表头等。`design/README.md` 第 12 节写的“实现状态”仍然描述 React 版本。
3. **异步状态没有统一的请求代际模型。** 约 90 处 `go func` + `a.update` 各自处理过期结果，有的比较位置，有的比较 key，有的什么都不比较。删除/移动计划、分享、预览、用量扫描都因此有串台问题，其中计划串台属于 P0。
4. **现有测试只证明“能渲染”。** `view_test.go` 没有任何行为断言，快照不含标题栏，所以下面的 P0/P1 问题和红绿灯重叠都不会被现有测试发现。

## 1. P0：数据、安全语义与核心流程

**F1 计划结果会写进另一个计划。** `createPlan` 和 `executePlan` 的 goroutine 捕获 `state := &a.plan`，没有计划代际或 ID 校验（`plan.go:55`、`plan.go:450`）。关闭一个仍在生成的计划、再打开另一个，旧计划的条目和 ID 会写进新对话框：标题和位置是新的，列表和要执行的计划 ID 是旧的。执行过程中对话框仍然响应 Esc 和关闭按钮（`plan.go:161`、`plan.go:186`），关闭后执行结果同样会落进下一次打开的计划。

**F2 执行后不展示逐项结果，危险按钮可以再按一次。** `executePlan` 只用 `plan.Results` 数失败数量、弹一个 toast，然后把对话框恢复成同一份审阅页，“永久删除 N 个对象”按钮仍在（`plan.go:466`）。这不符合 `CLAUDE.md` 里“保留逐项结果”的要求。词表里的 `plan.ok`、`plan.fail`、`plan.skip`、`plan.conflict`、`plan.changed` 都没有使用。

**F3 覆盖上传的计划会顶掉正在审阅的计划。** `queuePlan` 对已经带 `plan` 的请求直接调用 `showPlan`，绕过了队列（`plan.go:25`）。一次拖入多个需要覆盖的文件，只有最后一个计划可见，其余计划静默过期；如果用户正在审阅删除计划，它也会被替换掉（`browse.go:675`）。

**F4 已保存的限速和并发在重启后不生效。** `applyTransferSettings` 用 `transfers.loaded` 判断是否已经应用过（`transfers.go:40`），但启动时 `loadTasks` 就把 `loaded` 置为 true（`browse.go:618`）。结果是：界面下拉框显示上次保存的“5 MiB/s”，传输引擎实际不限速、使用默认并发。

**F5 快速上传的文件内容与扩展名不一致。** `applyQuickImage` 把 `quick.source` 置空且从不赋值（`quick.go:347`），所以切换“压缩 / WebP”后 `reencodeQuick` 什么也不做（`quick.go:379`）。上传时扩展名按开关的当前值决定（`quick.go:442`），于是：

- 先选图再打开 WebP，上传的是 JPEG 字节，key 是 `.webp`。
- 两个开关都关闭时保留原始字节（可能是 PNG/GIF），扩展名却固定为 `.jpg`（`quick.go:387`）。
- 只开“压缩”时 PNG 的透明通道和 GIF 动画被丢弃，界面不显示处理后的体积。

**F6 新建连接走不到 Cloudflare 登录。** 切换到 Cloudflare 流程的唯一入口在 `if state.editing != ""` 里（`connect.go:242`），只有编辑已有的手填连接时才出现。`design/README.md` 第 12 节记录的“Cloudflare 登录为主入口”在原生版里不存在；`connection.cloudflare.feature*`、`step*`、`errorTitle`、`setupRequired` 等 30 多个文案键都没有使用。

**F7 关闭连接页会泄漏登录会话。** `StartCloudflareSignIn` 返回之前关闭页面时，`SessionID` 还是空的，`closeConnect` 无法取消（`connect.go:67`）。之后 goroutine 回来仍会写状态并打开系统浏览器（`connect.go:521`），回环端口 8976 一直被占用，下一次登录返回 `conflict`。

## 2. P1：功能缺陷与错误信息

| # | 问题 | 位置 |
| --- | --- | --- |
| F8 | Quick Look 左右切换时不重置 `bitmap` 和 `fetching`：新图片显示成上一张；如果上一张还在下载，新图片永远转圈 | `preview.go:27`、`preview.go:196` |
| F9 | 分享对话框没有请求代际，快速切换有效期时旧结果覆盖新结果（选择器是 7 天，链接是 15 分钟）；`openShare` 里的 `if` 分支两边都设成 `temporary`，是空操作 | `share.go:17`、`share.go:32` |
| F10 | “加载更多”失败时写入 `browse.err`，整页已加载的列表被替换成错误页 | `browse.go:127`、`browse.go:417` |
| F11 | 每完成一个上传就 `loadEntries` 整页重载：分页丢失、滚动回顶、闪 spinner；批量上传 N 个文件就重载 N 次 | `browse.go:603` |
| F12 | 上传中途失败时，只要已有部分结果，错误就被丢弃；`pendingToast` 只有一个槽，“已排队”和“请审阅计划”同时出现时只显示后者 | `browse.go:657`、`app.go:281` |
| F13 | 拖入大目录时逐个同步调用 `StartTransfer`，没有数量上限、没有进度，使用 `context.Background()`，无法取消 | `files.go:94`、`browse.go:654` |
| F14 | 切换位置后旧的用量扫描继续写入进度和结果，范围徽标却显示新位置 | `usage.go:12`、`usage.go:224` |
| F15 | 失败的传输不显示原因，`TransferTask.Error` 没有被界面读取 | `transfers.go:202` |
| F16 | 保存连接、Cloudflare 连接按钮没有防重复点击（`busy` 没有用于 `Disabled`），双击会创建两个连接或铸两次令牌 | `connect.go:146`、`connect.go:340` |
| F17 | 快速上传遇到冲突时，面板里显示“覆盖 / 重命名”，同时又排队一个上传计划；快速上传层画在计划层之上，把计划挡住 | `quick.go:469`、`app.go:557` |
| F18 | 剪贴板没有图片时，把空状态提示“Paste or choose an image”当错误显示；拖入文件在 UI 线程同步读取（最大 64 MiB） | `quick.go:318`、`quick.go:134` |
| F19 | 设置 → 快捷键有 5 行标签和按键左右颠倒（真实窗口已确认）；Windows 下显示 `CtrlF`、`Ctrl⌃S`，而实际快捷键是 Ctrl+Shift+S | `settings.go:436` |
| F20 | 设置 → 集成里的 CLI 示例无法运行：`bucky ls r2://bucket/prefix --profile "Acme assets"`，CLI 实际语法是 `r2://profile/bucket/prefix`，也没有 `--profile` 参数 | `settings.go:26`、`internal/cli/commands.go:117` |
| F21 | `tn` 先压入原始整数 `count`，调用方传入的 `formatCount` 结果被 `strings.NewReplacer` 忽略，千分位分隔不生效 | `i18n.go:87` |
| F22 | `systemLanguage` 只读 `LANG` 等环境变量，从 Finder 启动的 macOS 应用没有这些变量，中文用户默认看到英文界面 | `prefs.go:74` |
| F23 | 十余处面向用户的错误硬编码英文（如 `Enter a connection name`、`A plan can contain up to 1,000 objects`），`errorText` 对 `invalid_argument` 直接显示原文，中文界面会出现英文（真实窗口已确认）；词表里对应的 `plan.pickSame`、`plan.tooMany`、`connection.accountIdError`、`connection.keysBoth`、`connection.publicUrlError` 没有使用 | `connect.go:402`、`plan.go:108`、`files.go:35` |
| F24 | 分栏视图共用一个选择集，但 `selectedEntries` 只看当前目录，在父列里选中的项不参与预览和操作；`loadColumn` 最多读 4 页，超出部分静默截断 | `browse_views.go:264`、`browse.go:170` |
| F25 | 排序菜单的方向项显示的是当前方向，不是点击后的动作，含义模糊；搜索框在 `Changed` 和 `Submitted` 时各调用一次 `setQuery` | `browse_toolbar.go:111`、`browse_toolbar.go:60` |
| F26 | Esc 行为不一致：连接页不响应 Esc（真实窗口已确认），分享、计划、快速上传响应；计划执行中按 Esc 也会关闭 | `connect.go:81`、`plan.go:161` |
| F27 | 侧栏连接行单击折叠时反而去加载桶（条件写反）；双击会先触发两次折叠，再打开编辑页 | `sidebar.go:88` |
| F28 | `pruneProfiles` 找到第一个失效连接就 `break`，删除多个连接后历史里仍有残留 | `browse.go:37` |
| F29 | 图片预览使用没有超时的 `http.DefaultClient`，上下文是应用级的，关闭预览不会取消；快速翻图会并发下载多张最大 32 MiB 的图片 | `preview.go:234` |
| F30 | 分享文件夹会展开成最多 1,000 条预签名链接，设计明确写“never a folder” | `share.go:42` |
| F31 | 删除连接的确认框只写“You cannot undo this.”，没有说明会删除钥匙串条目和 Cloudflare 令牌 | `connect.go:466` |
| F32 | 侧栏“快速上传”在没有连接、或没有配置公开 URL 的连接时仍然可点，点开后只有一条警告 | `sidebar.go:177`、`quick.go:146` |

## 3. P2：视觉与交互（“为什么丑”）

### 3.1 设计系统

**V1 颜色令牌完全没用设计稿。** `newPalette` 用的是 Apple 风格的冷灰（浅色 `#f4f4f6` / `#1d1d1f`，深色 `#1e1e20`），默认强调色为蓝，另有 7 种强调色可选（`theme.go:84`、`prefs.go:18`、`prefs.go:61`）。设计基线是暖白、浅米色、赤陶 `#B45A36`、暖黑，成功、警告、危险色只表达语义。

**V2 没有字体层级。** 欢迎页标题是 22 px 无衬线粗体，没有使用设计规定的衬线展示字体；代码里出现 10.5 到 22 之间十几种字号字面量，没有字号刻度；数字列没有等宽数字。

**V3 按钮体系混乱。** `ghostButton` 只把背景设成透明，`ui.Button` 默认的边框还在，所以侧栏 `+`、侧栏开关、后退/前进、各处关闭按钮都是带边框的小方块（`app.go:536`）；排序按钮是实心灰底（`browse_toolbar.go:98`）；上传按钮不是主按钮。同一行工具栏里有三种按钮样式。

**V4 颜色、阴影、圆角散落在各文件里。** 对话框阴影 `Shadow(0, 24, 60, 0, …)` 复制了 7 次，R2 图标是写死的橙色 `Oklch(0.68, 0.17, 50)`（`sidebar.go:166`），不随主题变化。

### 3.2 窗口外壳与侧栏

**V5 工具栏全部挤在 52 px 标题行里。** 后退/前进、面包屑、搜索、视图切换、排序、上传都在同一行（`app.go:476`），没有设计里的页面标题（桶名 26 px + 连接状态点）和表格工具栏（对象数、体积、排序、视图）。最小窗口宽 960、侧栏 248 时，内容区只有约 700 px，控件已经占掉约 600 px，深层路径的面包屑没有截断策略。

**V6 侧栏底部按钮溢出被裁掉。** “快速上传”设为 `Grow(1)`，再加两个 32 px 图标按钮，超出了 228 px 的内容宽，设置图标右半边被切掉（真实窗口已确认，`sidebar.go:176`）。

**V7 传输托盘在没有任何传输时显示“完成”。** 进度比例默认是 1.0，空闲时显示绿色对勾、“Transfers complete”和满格绿条；有需要注意的传输时是满格黄条（`sidebar.go:206`）。托盘把排队算作“运行中”，“2 transfers running · 9:30 left”和传输页的“1 running · 1 queued”、行内的“0:01 left”对不上。

**V8 侧栏缺少空状态，大部分可点区域不是按钮。** 没有连接时，侧栏只有一个“Connections”标签。设置导航行、连接行、桶行、网格图块都用 `Row.Clicked` 实现，没有按钮角色，在辅助功能树里不可见（真实窗口的 AX 树已确认）。

### 3.3 页面

| # | 页面 | 问题 |
| --- | --- | --- |
| V9 | 首次使用 | 只有 logo、标题、一句话和按钮，竖直居中在大片空白里；缺设计里的侧栏空状态、“密钥只存在系统钥匙串”的安全承诺和三项能力说明；标题行只剩一个孤零零的侧栏开关（`welcome.go`） |
| V10 | 列表 | 选中行是整行饱和主色加白字，深色模式下非常刺眼；文件图标不随选中反色，对比度不足；没有复选框和行操作；Kind 列字号与其他列不同；列表日期是 `2006-01-02 15:04`，用量页是 `Jan 2 15:04` |
| V11 | 网格 | 96×68 的小图块；图片没有缩略图，用斜条纹占位；没有体积和类型；选中态只靠描边 |
| V12 | 分栏 | 父列里不高亮当前路径上的文件夹，看不出自己在哪；单击文件夹不展开下一列，必须双击；右侧经常是一整列空白 |
| V13 | 选择栏 | 相对整个窗口居中而不是相对内容面板，整体偏左（`actions.go:59`）；深色模式下 HUD 颜色 `RGBA(30,30,32)` 与背景 `#1e1e20` 几乎相同；按钮各自带边框 |
| V14 | 添加连接 | 覆盖整个窗口，macOS 红绿灯压在页头图标和副标题上（真实窗口已确认，`connect.go:81`）；右列控件宽度不一（输入框 320、分段 300、下拉 200/260），右缘不齐；说明文字贴在卡片左缘，不在对应字段下方；标签与占位文字重复；没有 Cancel；Secret 输入框没有显示/隐藏切换 |
| V15 | Cloudflare 表单 | 卡片里有蓝色主按钮“Connect”，卡片外还有蓝色主按钮“Save”，两个主行动并列 |
| V16 | 设置 | 内容列用 `MaxWidth` 加自动外边距（`settings.go:40`），各分区左缘不同（General 576、Appearance 610、About 536、Shortcuts 752），切换分区时内容跳动；页面标题“Settings”和分区标题重复；General 里放了一个“Open usage”跳转按钮 |
| V17 | 用量 | 说明文字放在第一行；按扩展名的条形图沿用类型色，`.jpg` 和 `.png` 同色、File 和 Archive 同灰，分段无法区分；只统计数量没有体积（设计要求按体积）；“Completeness: Complete”冗余；未打开桶时页面几乎是空的 |
| V18 | 传输 | 内容列居中留出大片空白；行内只显示桶名，不显示路径和方向文字；已完成行的满格绿条冗余；没有“清除已完成”和“在 Finder 中显示” |
| V19 | 删除计划 | 标题“Delete 2 items”和按钮“Permanently delete 2 objects”用词不一致；Cancel 在上、全宽红色按钮在下的竖排布局让危险按钮成为最显眼的元素；正文暴露开发者措辞“A plan ID isn't consent.”；不显示计划 ID |
| V20 | 对话框 | 分享对话框标题沿用菜单文案“Share…”（带省略号）；Quick Look 固定 720×320，大窗口下预览很小，且图片和文本预览里没有下载、分享按钮 |
| V21 | 搜索 | 工具栏搜索框（当前目录内 Find）和 ⌘F / ⌘K 面板（只跳转已加载的连接、桶和文件夹，不搜对象）概念重叠；面板里鼠标悬停会抢走键盘选中项 |

## 4. P3：代码质量

- **C1 渲染期副作用。** `ensureBuckets`、`startPendingSearch`、`loadColumn`、`loadSettings`、`applyTransferSettings`、`paletteMatches → ensureBuckets` 都在构建视图时发起网络请求或改状态。
- **C2 异步模型不统一。** 约 90 处 `go func` 和 `a.update`，没有统一的请求代际或取消；`context.Background()` 出现 3 次（`browse.go:654`、`quick.go:480`、`usage.go:228`）。F1、F7、F8、F9、F14 都是这个问题的具体表现。
- **C3 数据竞争。** `a.svc.Transfers.OnUpdate` 和 `a.svc.OnCloudflareSignIn` 在 UI 线程赋值，却由服务的 goroutine 读取（`browse.go:579`、`connect.go:501`）；`OnUpdate` 在启动后才挂上，启动阶段的传输事件会丢失。
- **C4 重复代码。** 限速和并发选择器在传输页、设置页各写一遍（`transfers.go:77`、`settings.go:222`）；`loadBuckets` 与 `ensureBuckets` 重复；7 个浮层的遮罩、面板、页头结构复制粘贴；`joinLines` 重新实现了 `strings.Join`。
- **C5 死代码与失真注释。** `_ = root`、`_ = list`、`_ = state`、`_ = width`、`concurrency := …; _ = concurrency`；`// page returns` 写在 `content` 上，`// shortcutKeys` 写在 `shortcuts` 上；`copyText` 的注释说会报告失败，实际不处理；`statusOrder` 的注释说需要注意的排在前面，实际失败排第 5；定义了 `sidebarMin` 和 `sidebarMax`，侧栏却不能拖动；`substitute` 的注释写 `{name}`，实际格式是 `{{name}}`。
- **C6 偏好写盘过于频繁。** `rememberColumns` 每帧比较列布局，拖动列宽时每帧在 UI 线程同步写一次 `ui.json`。
- **C7 词表与代码脱节。** 532 个键里，排除动态拼接的 `errors.*`、`kinds.*` 等，约 130 个键从未使用；同时代码里又硬编码英文。
- **C8 测试没有行为断言。** `view_test.go` 只检查渲染不 panic 并输出 PNG。F1–F5、F8、F19 都不会被发现；快照不含标题栏，红绿灯重叠这类问题天然漏测（`docs/testing.md` 已承认）。
- **C9 单体状态。** `App` 持有 20 多个子状态，每个浮层都依赖全局可变的 `a.plan`、`a.share` 等，难以单独测试。
- **C10 版本号两处维护。** `settings.go:12` 的 `Version = "0.1.0"` 和 `mygo.json` 各写一份。

## 5. 文档与实现不一致

- `design/README.md` 说“颜色令牌、间距与文案仍以此目录为准”，实现另起了一套（见 V1–V4）。
- `design/README.md` 第 12 节“实现状态”描述的是 React 版本：整窗两栏的添加连接流程、Cloudflare 主入口、⌘F 全库搜索、顶部 Sheet 删除确认、Quick Look 媒体分支。原生版大多没有这些功能，文档没有更新；第 9 节仍然引用 `frontend/src/style.css` 和 Wails。
- `docs/testing.md` 记录“原生界面三平台人工走查尚未进行”。本次只走查了 macOS，并且已经发现标题栏重叠、侧栏裁切等问题。

## 6. 建议的修复顺序

1. **P0 与异步模型**：先建立统一的请求代际 / 取消机制（C2），再修 F1–F7；计划对话框补上逐项结果，执行中禁止关闭。每个修复配一条行为测试。
2. **设计令牌层**：按 `design/README.md` 第 3 节重建颜色、字号刻度、圆角、阴影和按钮变体（primary / secondary / ghost / destructive）。需要先决定：保留多强调色并默认赤陶，还是只保留单一品牌色。
3. **窗口外壳**：拆分标题行、页面标题和表格工具栏；修侧栏底部溢出、托盘空闲态和侧栏空状态；所有可点击行改为带角色的按钮。
4. **逐页重做**：按设计 5.x 节重做首次使用、工作区、连接（恢复 Cloudflare 主入口、避开红绿灯）、计划、传输、设置、用量。
5. **补测试与文档**：为计划、分享、预览、快速上传的状态机写单元测试；快照加入标题栏占位；更新 `design/README.md` 第 9、12 节和 `docs/testing.md`。

## 7. 本次没有覆盖

- 真实 R2 连接：浏览、上传、分享、计划等需要连接的页面只看了无窗口快照（测试假数据），没有在真实窗口里操作。
- 深色模式只看了快照，没有在真实窗口里切换。
- Windows 和 Linux 没有构建或运行。
- 键盘全流程、读屏器、窄窗口（960 px）布局只做了代码层面的推断，没有实测。
