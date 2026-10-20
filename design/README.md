# Bucky UI 设计稿

- 更新日期：2026-10-20
- 状态：设计基线（产品实现为 `internal/desktop` 的 MyGo 原生界面，本目录只保留设计原型与说明）
- 入口：[design/index.html](index.html)
- 设计基线：暖中性底色、单一强调色、52 px 标题栏、8 px 内容面板内缩

本目录只包含设计与原型，不参与产品构建，其中的 HTML/CSS/JS 也不再是产品代码。页面中的连接、对象、传输、计划 ID 和密钥均为演示数据，`bucky-placeholder-secret` 不是真实凭证。桌面端已按 [ADR-004](../docs/adr/0004-native-go-desktop-ui.md) 改为 MyGo 纯 Go 原生界面，颜色令牌、间距与文案仍以此目录为准；原 `frontend/` 实现已删除。

## 1. 使用方式

直接打开：

```bash
open design/index.html
```

或启动一个静态服务器：

```bash
python3 -m http.server 8000 --directory design
```

顶部设计栏包含 8 个页面：工作区、首次使用、新建连接、对象详情、传输队列、删除确认、设置、设计系统。右上角按钮切换亮/暗主题。原型支持这些交互：

- 列表 / 网格切换。
- 单选、全选与清空选择，选择栏计数同步。
- 点击 `hero-aurora.jpg` 或预览按钮进入对象详情。
- 侧栏进入传输、设置和新建连接。
- 设置中的 Light / Dark / System 立即切换主题。
- 设计栏页面切换与左右方向键。

URL 参数便于截图：

- `?capture=1`：隐藏设计栏和页面说明，只保留 1280 × 800 窗口。
- `?theme=light` / `?theme=dark`：指定主题。
- `#workspace`、`#welcome`、`#connection`、`#preview`、`#transfers`、`#delete`、`#settings`、`#system`：指定页面。

窗口中的 `⌘K`、`⌘,` 快捷键提示是设计目标，原型没有实现全局快捷键；侧栏中的传输和设置入口可以点击。首次使用页只展示一条主行动，不伪造可点击的“示例空间”。

## 2. 设计方向

Bucky 是本地优先的桌面工具，界面应该让对象存储操作显得安静、可预期，而不是把云控制台搬进桌面。

- **暖中性底色。** 背景使用暖白与浅米色，避免纯白大面积刺眼；深色主题使用暖黑，而不是蓝黑。
- **单一品牌色。** 赤陶色 `#B45A36` 承担主按钮、选中标记、进度与少量强调。成功、警告和危险色只表达语义。
- **克制的编辑感。** 衬线字体只出现在品牌字标和首次使用页的大标题，建立记忆点；工作界面保持无衬线、紧凑、可扫描。
- **桌面密度。** 控件 32 px 高、表格行 48 px、圆角 8–12 px。信息密度接近 Finder / Linear，而不是移动端卡片流。
- **一处一个主动作。** 每个页面只保留一个主按钮；次要操作退为 outline 或 ghost。批量操作浮在内容上方，不挤占工具栏。
- **危险操作需要上下文。** 删除只通过计划摘要对话框执行，展示对象、体积、有效期与不可撤销提示。

视觉参考对象是 shadcn/ui 的中性主题、Raycast 的窗口质感与 Finder 的浏览效率。标题栏、侧栏和内容区使用同一套 token，macOS 与 Windows / Linux 的窗口 chrome 保持 52 px 高度。

## 3. 设计令牌

颜色变量按 shadcn/ui 的命名约定组织，可直接作为 CSS 变量接入 Tailwind v4。

### 3.1 亮色

```css
:root {
  --background: #faf9f6;
  --foreground: #24211c;
  --card: #ffffff;
  --card-foreground: #24211c;
  --popover: #ffffff;
  --popover-foreground: #24211c;

  --primary: #b45a36;
  --primary-foreground: #fff9f3;
  --primary-hover: #9d4a2a;
  --secondary: #f2efe8;
  --secondary-foreground: #3c372f;
  --muted: #f4f1ea;
  --muted-foreground: #7b756a;
  --accent: #f5e9e1;
  --accent-foreground: #7a3d24;
  --destructive: #b23a2e;
  --destructive-foreground: #fff8f6;

  --border: #e8e3d9;
  --input: #ddd7ca;
  --ring: #b45a36;

  --sidebar: #f5f2eb;
  --sidebar-foreground: #2c2822;
  --sidebar-accent: #ece6da;
  --sidebar-border: #e6e0d4;

  --success: #3f7050;
  --success-soft: #e8f1e9;
  --warning: #a46912;
  --warning-soft: #f7eedb;
}
```

### 3.2 深色

```css
[data-theme="dark"] {
  --background: #161411;
  --foreground: #f1ece3;
  --card: #1d1a16;
  --card-foreground: #f1ece3;
  --popover: #211d18;
  --popover-foreground: #f1ece3;

  --primary: #e09670;
  --primary-foreground: #26130a;
  --primary-hover: #eda77f;
  --secondary: #26221c;
  --secondary-foreground: #ece6dc;
  --muted: #221f19;
  --muted-foreground: #a49c90;
  --accent: #2e231c;
  --accent-foreground: #f2b693;
  --destructive: #e07466;
  --destructive-foreground: #24100d;

  --border: #2f2a22;
  --input: #3a342b;
  --ring: #e09670;

  --sidebar: #12100d;
  --sidebar-foreground: #ded7cc;
  --sidebar-accent: #241f18;
  --sidebar-border: #29241d;

  --success: #8cc39c;
  --success-soft: #1b2a20;
  --warning: #e0ae62;
  --warning-soft: #2f2617;
}
```

深色模式不是把亮色简单反相：主色变为更浅的赤陶，按钮使用深色文字；卡片与背景只拉开约 5% 明度，避免大块纯黑；边框保持低对比，焦点环仍使用 `--ring`。

### 3.3 字体

```css
--font-sans: ui-sans-serif, -apple-system, BlinkMacSystemFont, "Segoe UI",
  "Noto Sans SC", "PingFang SC", sans-serif;
--font-serif: "Iowan Old Style", "Palatino Linotype", Palatino,
  "Book Antiqua", Georgia, "Songti SC", serif;
--font-mono: ui-monospace, "SF Mono", SFMono-Regular, Menlo, Consolas,
  "Liberation Mono", monospace;
```

| 角色 | 字号 / 行高 | 字重 | 用途 |
| --- | --- | --- | --- |
| Serif display | 39 / 1.08 | 600 | 首次使用页标题 |
| Page title | 26 / 1.15 | 650 | 当前桶名 |
| Dialog title | 16.5 / 1.3 | 650 | 对话框标题 |
| Body | 12.8 / 1.5 | 400 | 表格、表单、正文 |
| Small | 11 / 1.5 | 400 | 元数据、辅助说明 |
| Label | 10 / 1.4 | 600 | 全大写分组标题 |
| Mono | 10–12 | 400 | Object key、ETag、版本号 |

衬线字体不用于正文或表单。数字列使用 `font-variant-numeric: tabular-nums`。

### 3.4 圆角、间距与阴影

| 项目 | 值 | 用途 |
| --- | --- | --- |
| 圆角 | 4 px | Checkbox |
| 圆角 | 6–8 px | 图标按钮、输入框、菜单项 |
| 圆角 | 10–12 px | 文件卡片、表格容器、面板 |
| 圆角 | 14–16 px | 对话框、Sheet、应用窗口 |
| 间距 | 4 / 8 / 12 / 16 / 24 / 32 px | 基础刻度 |
| 控件高度 | 28 / 32 / 38 px | 小按钮、默认按钮、主行动 |
| 表格行 | 48 px | 文件夹行含第二行说明 |
| 阴影 | `0 1px 2px` | 卡片、输入框 |
| 阴影 | `0 10px 28px` | 浮动选择栏、菜单 |
| 阴影 | `0 30px 80px` | 对话框、确认计划、窗口 |

不使用彩色发光、玻璃拟态或大面积渐变。唯一的渐变用于图片缩略图与预览占位。

### 3.5 Tailwind v4 接入示例

```css
@theme inline {
  --color-background: var(--background);
  --color-foreground: var(--foreground);
  --color-card: var(--card);
  --color-primary: var(--primary);
  --color-primary-foreground: var(--primary-foreground);
  --color-secondary: var(--secondary);
  --color-muted: var(--muted);
  --color-muted-foreground: var(--muted-foreground);
  --color-accent: var(--accent);
  --color-destructive: var(--destructive);
  --color-border: var(--border);
  --color-input: var(--input);
  --color-ring: var(--ring);
  --color-sidebar: var(--sidebar);
  --font-sans: var(--font-sans);
  --font-serif: var(--font-serif);
  --font-mono: var(--font-mono);
}
```

主题切换使用根元素上的 `data-theme="dark"`。若采用 shadcn/ui 默认的 `.dark` 选择器，保持变量声明一致即可，不需要复制两套组件样式。

## 4. 布局与尺寸

| 区域 | 尺寸 | 说明 |
| --- | --- | --- |
| 应用窗口（设计画板） | 1280 × 800 | 实际窗口可缩放，最小建议 1040 × 680 |
| 标题栏 | 52 px | macOS 保留原生红绿灯；Windows / Linux 自绘控件；整体为拖动区 |
| 侧栏 | 256 px | 连接、桶、传输与设置的唯一入口 |
| 内容左边距 | 24 px | 页面标题、表格容器与操作区对齐 |
| 表格容器 | 自适应宽度 | 圆角 12 px，内部工具栏 48 px，表头 33 px，行 48 px |
| 对象详情 / 传输 Sheet | 432 px | 从标题栏下方滑入，保留窗口 chrome |
| 新建连接 Dialog | 620 px | 两列表单 |
| 删除计划 Dialog | 536 px | 计划摘要 + 危险提示 |
| 设置 Dialog | 700 × 580 px | 左侧 166 px 分区导航 |
| 窗口控件热区 | 46 × 40 px | Windows / Linux；关闭按钮 hover 为红色 |

窄屏策略：

- 1120 px 以下隐藏表格的 Type / Modified 列，搜索框收窄。
- 860 px 以下隐藏侧栏，设计画板转为单页预览；真实桌面应用应限制最小窗口宽度，而不是隐藏主导航。
- 弹窗宽度使用 `max-width: calc(100% - 48px)`，高度使用 `max-height` 与内部滚动。

## 5. 页面说明

### 5.1 工作区 Workspace

目标：在不离开列表的情况下完成浏览、选择和下一步动作。

- 页面标题区：连接名的绿色状态点、桶名、搜索、刷新、Upload 主按钮与下拉。
- 表格工具栏：面包屑、对象数与体积、排序、列表 / 网格切换。
- 表格：Checkbox、Name（文件夹带对象数）、Type、Size、Modified、行操作。表头 sticky，行 hover 与选中态区分。
- 选择栏：浮于内容底部，显示数量、Download、Share、Move、Delete 与清空。对象为空时消失。
- 网格视图：4 列卡片，文件夹、图片缩略图、类型与体积；选中态使用主色描边。
- 加载更多放在表格底部，与“Showing 11 of 128 objects”并列；全量搜索与大目录性能不因视觉设计改变。
- 行点击打开详情。行内 Checkbox 和更多菜单不触发行跳转。

建议的 shadcn/ui 组合：`Table`、`Checkbox`、`Button`、`DropdownMenu`、`Breadcrumb`、`Input`、`Tooltip`。

尚未画出的状态：加载骨架、空文件夹、搜索无结果、权限错误、只读连接禁用上传、全选跨页。实现时按现有查询语义补齐，不把“已加载结果”伪装成全桶结果。

### 5.2 首次使用 First run

目标：在没有连接时给出一个清晰、可信的起点。

- 侧栏连接区显示虚线空状态和 “Add connection”；桶区隐藏。
- 主区使用品牌衬线标题、单句说明和唯一的 “Add R2 connection” 主行动。
- 安全承诺紧邻主行动：“Secrets stay in your system keychain”。
- 下方三项能力说明浏览、临时链接和可恢复传输，不使用假数据或不可用的示例空间。
- 只读与凭证说明按平台替换系统名称；设计稿使用通用的 “system keychain”。

建议组合：`Button` + 自定义空状态。标题是本设计中最强的品牌表达，不建议在普通页面复用 39 px 衬线字号。

### 5.3 新建连接 New connection

目标：一次表单完成 R2 连接，凭证来源与只读保护不被埋在折叠区里。

- Connection name、Account ID、Jurisdiction、Bucket、Public URL。
- Credentials 分段选择：System keychain / Environment。系统钥匙串为默认；环境变量只用于显式选择的运行环境。
- Access key ID 与 Secret access key 两列；Secret 默认隐藏，可临时显示。
- Read-only connection 开关独占一行，解释会阻止上传、移动和删除。
- 底部说明密钥只进入系统钥匙串。Footer 左侧重复“Encrypted at rest”，右侧 Cancel / Save connection。
- 新建状态不直接提供 Test：测试需要已保存的 profile 与凭证。编辑态可在 footer 增加 “Test connection”，与现有服务测试流程一致。
- 校验错误使用输入框下方红色说明，不弹出二次对话框；保存失败保留表单内容。

建议组合：`Dialog`、`Input`、`Select`、`RadioGroup` 或分段按钮、`Switch`、`Alert`、`Label`。

实现时注意：Environment 选项要显示“仅本次运行”的来源说明；切换到环境凭证后不要自动删除旧钥匙串条目，保持当前产品的已知行为。

### 5.4 对象详情 Object details

目标：让预览、元数据和危险操作在同一处完成。

- 右侧 Sheet 432 px，从标题栏下缘开始，不遮盖窗口 chrome。
- 预览区按类型切换：图片、文本（有界读取）、视频 / 音频（短期预签名）、PDF；无 CORS 或无法预览时展示原因和下载入口。
- 标题行包含文件名、类型摘要、Download 主按钮、Share 与更多操作。
- Object key 使用等宽字体与复制按钮；只显示可安全复制的完整 key。
- 元数据：Size、Dimensions、Modified、Uploaded；下方 Tabs 展示详情与自定义元数据。
- ETag 显示截断值，并明确它不等于内容哈希。
- Footer 左侧 Delete object；只读连接或没有权限时禁用并说明原因。
- 关闭 Sheet 回到原列表位置，不重置已加载页面与选择。

建议组合：`Sheet`、`Tabs`、`Badge`、`Separator`、`Button`、`Tooltip`。

### 5.5 传输队列 Transfers

目标：在有限空间里看清进度、速度和下一步。

- Sheet 顶部为标题、活动数和关闭；下方 All / Active / Completed 分段标签与计数。
- 活动项：文件类型图标、名称、百分比、进度条、方向与字节数、速度与剩余时间、暂停和取消。
- 完成项：绿色勾选、名称、方向、体积与时间；提供定位文件的入口。
- 底部虚线区域提示拖放上传到当前 prefix。
- Footer 提供速度限制与设置入口；限速选项与现有服务一致，0 表示不限制。
- 队列需要覆盖等待、暂停、失败、校验中等状态；进度事件只用于通知，重新打开面板以 SQLite 状态为准。

建议组合：`Sheet`、`Tabs`、`Progress`、`Button`、`DropdownMenu`、`Badge`。

可访问性：进度变化应以低频率 `aria-live="polite"` 宣告，避免读屏器每个包都播报；暂停和取消必须有明确标签。

### 5.6 删除确认 Delete plan

目标：把“不可撤销”从一句警告变成可检查的计划。

- AlertDialog 标题明确数量，正文说明将从哪个桶删除。
- 计划卡列出对象、类型图标与体积，底部汇总对象数和总字节数。
- 展示计划 ID、剩余有效期和“执行前再次检查”说明。
- 危险提示说明不可撤销；版本控制相关文案必须根据桶的实际状态生成，不能假设所有桶都没有版本控制。
- Footer 左侧说明逐项结果会被记录，右侧 Cancel 与 Delete 3 objects。
- 只读连接不显示入口；对象在执行前发生变化时返回冲突并保持计划待确认，不以对话框文案承诺原子性。
- 不使用“输入 DELETE”作为默认交互：计划摘要加明确的危险按钮已经构成人工确认；如果后续审计要求更强确认，再在服务层增加独立机制。

建议组合：`AlertDialog`、`Badge`、`Separator`、`Button`。

### 5.7 设置 Settings

目标：把外观、网络和用量收进一个可扩展的桌面设置窗口。

- 700 × 580 px，左侧分区导航：Appearance、Network、Usage；底部显示版本。
- Appearance：Light / Dark / System 三张主题卡，预览窗口即时反映主题；语言选择紧随其后。
- Network：速度限制、预签名链接有效期。说明 0 代表不限速，R2 上限为 7 天。
- Usage：对象数、总体积、最近扫描时间；按类型分段的体积条与图例；扫描完成后可重新扫描。
- 主题切换即时生效并持久化；设置内容超出高度时在内容区滚动，分区导航固定。
- 将来增加账号、更新、诊断时沿用相同分区，不再堆成一个长表单。

建议组合：`Dialog`、自定义分区导航或纵向 `Tabs`、`RadioGroup`（主题卡）、`Select`、`Progress`、`Badge`。

### 5.8 设计系统 Design system

这一页是给实现者的工作台，不作为产品页面交付：

- Color tokens：亮色下的 10 个核心变量及实际取值。
- Typography：衬线显示、页面标题、区块、正文、小字、标签、等宽。
- Shape & spacing：4–16 px 圆角与 4–32 px 间距。
- Components：按钮、表单、选择控件、状态标签、导航、进度、通知、菜单与 Tooltip。
- Window chrome：macOS 原生红绿灯与 Windows / Linux 自绘控件的并排对照。
- shadcn/ui mapping：界面区域到组件组合的对照表。

## 6. shadcn/ui 组件映射

| 界面区域 | 组件 | 说明 |
| --- | --- | --- |
| 窗口 chrome | 自定义 `PageHeader` | 不使用 shadcn 组件；52 px，拖动区，macOS 原生红绿灯 |
| 侧栏 | `ScrollArea`、`Button`、`Badge` | 连接行是自定义组合，激活态使用 sidebar accent |
| 对象表格 | `Table`、`Checkbox`、`DropdownMenu` | 48 px 行高，sticky 表头，TanStack Virtual |
| 面包屑 | `Breadcrumb` | 桶、prefix、当前文件夹 |
| 搜索 | `Input`；后续 `Command` | 当前为 prefix 筛选，未来可加命令面板 |
| 批量操作 | `Button` + 浮动容器 | 浮在选择栏中，不常驻工具栏 |
| 新建连接 | `Dialog`、`Input`、`Select`、`RadioGroup`、`Switch` | 两列紧凑表单，凭证来源是分段选择 |
| 对象详情 | `Sheet`、`Tabs`、`Badge`、`Separator` | 预览优先，删除在 footer |
| 传输 | `Sheet`、`Tabs`、`Progress` | 活动与完成分组，逐项暂停 / 取消 |
| 删除计划 | `AlertDialog`、`Badge` | 计划摘要与明确危险按钮 |
| 设置 | `Dialog`、`RadioGroup`、`Select` | 主题卡作为 radio group，左侧分区导航 |
| 通知 | `Sonner` | 复制、分享、传输完成等一句话结果 |
| 图标 | `lucide-react` | 1.6–1.8 px 描边；控件内 14 px，空状态 23 px |
| 主题 | CSS variables | `data-theme` 或 `.dark`，两套值只换变量 |

组件来自 shadcn/ui 官方实现，不重新造 Button / Dialog / Table 等基础件。业务组合放在 `features/` 下，避免把领域状态塞进通用组件。

## 7. 交互与状态

- 过渡时长 120–220 ms，使用 `ease` 或 `cubic-bezier(.32,.72,0,1)`；同时支持 `prefers-reduced-motion`。
- 行 hover 只改变背景，不改变尺寸；选中行增加主色左边线，避免只靠颜色区分（Checkbox 同时保持可见）。
- 选择栏出现时不推动内容，浮在内容上方；清空选择后立即消失。
- 列表 / 网格切换保留选择状态与当前 prefix，不重新请求数据。
- Dialog 打开时焦点进入第一个可操作控件，Escape 关闭，关闭后焦点返回触发按钮；删除计划默认聚焦 Cancel，而非危险按钮。
- Sheet 作为模态检查器，打开时不改变后台列表的滚动位置。
- 主题切换写入本地设置；System 监听系统外观变化。
- 所有危险操作由服务层重新校验只读状态、对象变化与计划有效期；界面只展示结果与原因。
- 文案使用现有 i18n 结构；设计稿中的英文只作为源文案，中文版保持相同信息层级。

## 8. 可访问性

- 焦点环使用 2 px `--ring` 加 2 px offset，不只用颜色变化表示焦点。
- 图标按钮必须有 `aria-label`；行操作在 hover 出现时，键盘聚焦仍必须可见。
- Checkbox 按对象命名，全选使用 indeterminate 状态；“2 selected”通过可读文本呈现。
- 对话框使用 Radix 提供的焦点陷阱与 `aria-modal`；标题通过 `aria-labelledby` 关联。
- 传输进度使用低频率 live region；失败与冲突用 `role="alert"`。
- 对比度需要以最终色值复测：主按钮使用 `--primary-foreground`，表格次级文字使用 `--muted-foreground`；目标至少 WCAG AA。
- 最小点击目标 24 px；主要按钮 32 px，窗口控件 46 × 40 px。
- 预签名链接、Object key 和 ETag 复制后只短暂提示，不把完整临时链接写进日志。
- 原型未用真实读屏器、键盘全流程或辅助技术测试，实施阶段必须补测。

## 9. 后续实现建议

当前阶段不改产品代码。进入重构时的推荐顺序：

1. **Token 层**：在 `frontend/src/style.css` 或独立 `theme.css` 中落地两套变量，接入 Tailwind v4 `@theme inline`；保持现有明暗主题入口和持久化逻辑。
2. **基础组件**：用 shadcn CLI 引入 Button、Input、Select、Checkbox、Switch、Dialog、AlertDialog、Sheet、Tabs、Table、Progress、Badge、DropdownMenu、Tooltip、ScrollArea、Separator。删除重复的自定义 CSS 之前先确认现有行为覆盖。
3. **应用外壳**：先替换侧栏、页面标题和表格容器；标题栏保持 Wails 自绘方式，不引入第二套窗口逻辑。
4. **工作区**：把表格列、选择状态、列表 / 网格切换和批量操作拆开；对象身份继续使用连接 ID + 桶 + 完整 key，不按文件名选中。
5. **对话框与 Sheet**：连接表单、对象详情、传输、删除计划和设置逐页替换；服务调用和只读检查不动。
6. **细节**：通知使用 Sonner；命令面板（⌘K）与快捷键在基础交互稳定后再加；最后补键盘、读屏器、100k 对象性能和三平台走查。

约束：

- 不改变服务层的计划、确认、只读和逐项结果语义。
- 不把云端 SDK 类型带进前端；继续只调用生成绑定。
- 不为了视觉引入重量级组件库；shadcn 的组件代码在本地，可按需裁剪。
- 标题栏高度仍需与 `cmd/bucky-desktop/main.go` 和 `frontend/src/style.css` 中的 52 px 保持同步。
- 流式与可恢复传输不在 UI 重构中重写。

## 10. 待确认

1. **品牌色**：保留赤陶色，还是改为更中性的石墨 / 靛蓝？当前提案接近现有 `--accent`，改动成本最低。
2. **衬线范围**：仅品牌与首次使用页（当前提案），还是扩展到页面标题？建议保持克制。
3. **列表密度**：48 px 行高对普通用户友好；是否为高频用户提供 40 px 紧凑模式？
4. **默认视图**：当前继续用列表；网格是否只作为图片较多的桶的推荐视图？
5. **批量操作位置**：当前浮动选择栏；是否需要在宽屏下改为表格上方固定工具栏？
6. **对象详情**：固定 432 px 的右侧 Sheet；后续是否需要可拖拽宽度或独立窗口？
7. **删除确认**：当前为计划摘要 + 危险按钮。是否需要更强的二次确认（输入对象数或名称）？
8. **设置结构**：当前拆为 Appearance / Network / Usage 三个分区；是否保留一个“General”入口？
9. **连接凭证**：Environment 凭证入口是否面向所有用户展示，还是仅在 CLI / 高级模式中开放？

## 11. 验证与限制

已在 Chrome 中检查：

- 亮色与深色的工作区、首次使用、新建连接、对象详情、传输、删除确认、设置和设计系统。
- 列表 / 网格切换、行点击进入详情、设计栏切换、主题按钮与设置主题卡、选择计数、清空选择、全选、设置分区滚动。
- 原型在 1400 × 1000 视口和 1280 × 800 应用画板下渲染；窄屏规则只做了基本检查。

尚未验证：

- Safari / WKWebView、Firefox、Windows / Linux 实际渲染。
- 真实读屏器、完整键盘流程与 WCAG 对比度测量。
- 真实 R2 数据、预签名媒体 CORS、100k 对象虚拟滚动与传输进度压力。
- 原型不是产品构建的一部分；产品界面已按本设计在 `frontend/` 中实现。

设计稿中的尺寸、间距和色值为提案值。实现时如果与真实内容或平台行为冲突，以可读性、现有服务语义和三端一致性为准重新调整。

## 12. 实现状态（2026-10-18）

`frontend/` 已完成基于 shadcn/ui 的重构，主要落地内容：

- 列表 / 图标 / 分栏三种浏览视图，面包屑、排序、⌘F 全库搜索与虚拟滚动。
- Quick Look 预览、分享对话框、复制 / 移动 / 删除计划审阅、传输页与用量页。
- 整窗「添加连接」流程、设置页、双语与明暗主题、只读连接限制。
- 剪贴板快速上传、命名模板、WebP 转换与 Markdown / URL 复制。

与原型的有意差异：

- 标题栏高度从 40 px 调整为 52 px，与 Wails 窗口控制按钮和 macOS 红绿灯对齐。
- 添加连接不是 620 px 的两列表单对话框，而是覆盖整个窗口的左右两栏流程：左栏是当前这一步要做什么，右栏随阶段变化（未授权是能力与安全说明，授权后换成真实的账号与存储桶列表）。Cloudflare 登录为主入口，手填 S3 密钥退为同一页的次要按钮。
- 设置不作为 Dialog，而是独立页面，便于扩展和滚动。
- 表格行高使用 34–36 px 的桌面密度，选择栏使用深色 HUD。
- 删除确认使用顶部 Sheet 而不是 AlertDialog，以便展示计划有效期与逐项结果。
- 对象详情以 Quick Look 模态框呈现，分栏视图另有固定预览侧栏。

尚未验证：真实 R2、Windows / Linux WebView、读屏器与 100k 对象性能。详见 `docs/testing.md`。
