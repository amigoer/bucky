<div align="center">
  <h1>Bucky</h1>

  <h3>一个桌面应用，管好你所有的对象存储</h3>

  <p>从 Cloudflare R2 开始，在一处浏览、上传、预览和分享文件。<br>后续逐步接入 AWS S3、阿里云 OSS 等存储，你和 AI agent 都使用同一套流程。</p>

  <p>
    <img src="https://img.shields.io/badge/status-early%20development-F59E0B?style=flat-square&labelColor=1A1A1E" alt="状态：早期开发中">
    <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-3F3F46?style=flat-square&labelColor=1A1A1E" alt="平台：macOS、Windows、Linux">
    <a href="#交给-ai-agent"><img src="https://img.shields.io/badge/agents-CLI%20%C2%B7%20MCP%20%C2%B7%20Skill-3F3F46?style=flat-square&labelColor=1A1A1E" alt="给 agent 用：CLI、MCP 与 Skill"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-3F3F46?style=flat-square&labelColor=1A1A1E" alt="Apache-2.0 许可证"></a>
  </p>

  <p>
    <a href="#能做什么">功能</a>&nbsp;&nbsp;·&nbsp;&nbsp;
    <a href="#支持的存储">支持的存储</a>&nbsp;&nbsp;·&nbsp;&nbsp;
    <a href="#交给-ai-agent">给 agent 用</a>&nbsp;&nbsp;·&nbsp;&nbsp;
    <a href="#路线图">路线图</a>&nbsp;&nbsp;·&nbsp;&nbsp;
    <a href="README.md">English</a>
  </p>
</div>

<br>

> [!NOTE]
> Bucky 仍在早期开发中，尚未发布。第一版只接 Cloudflare R2，下列功能均为开发目标；其他存储类型将在后续版本逐步接入。

## 为什么用 Bucky

每家云都有自己的控制台、自己的工具，做的却是同样几件事。Bucky 把它们放到一起。

- **开源免费，全平台可用。** macOS、Windows、Linux 用的是同一个应用。
- **先做好 R2，再逐类扩展。** 第一版聚焦 R2，后续接入 AWS S3、阿里云 OSS 等服务，继续使用同一套浏览和传输流程。
- **为 AI agent 而生。** 命令行工具、MCP server 和 Agent Skill 三件套，让 Claude Code、Codex、Cursor 直接用你在应用里配好的连接操作存储桶。
- **破坏性操作先预览。** 删除、移动、覆盖之前，Bucky 都会先列出将要发生的变化，等你确认；不管操作来自你自己还是 agent，都是如此。
- **密钥只属于你。** 凭证存放在系统钥匙串里，不会出现在日志和 agent 的输出中。没有任何遥测。

## 能做什么

R2 首版围绕以下使用流程开发：

- **浏览**：把存储桶当文件夹来逛，有面包屑、列表和网格视图、排序、搜索和信息面板。分页加载，让大文件夹保持响应。
- **预览**：图片、音视频、PDF、文本、代码和 JSON。视频边下边播，不用等完整下载。
- **上传下载**：直接拖入文件或整个文件夹。大文件自动分片上传；传输可以暂停、继续、重试，断网或重启后接着传，按服务支持的能力校验并显示已验证的内容。
- **分享**：给单个文件生成最长 7 天有效的临时链接（默认 1 小时）；也可以使用已经绑定到桶的公开域名生成公开链接。
- **图床**：写博客和文档时按一下快捷键，就把剪贴板里的图片传上去；按 `{year}/{month}/{hash}.{ext}` 这样的模板命名，可选压缩和转 WebP，Markdown 链接自动复制好。
- **用量统计**：按文件夹和文件类型统计数量与大小，在后台计算，不耽误你继续操作。
- **批量整理**：在同一连接内复制、移动和删除文件，移动、删除、覆盖都先预览。

Bucky 支持中英双语和明暗两套主题，常用操作都有快捷键。

## 支持的存储

| 存储 | 计划版本 |
| --- | --- |
| Cloudflare R2 | v0.1，首个版本 |
| AWS S3 | v0.2 |
| 阿里云 OSS | v0.3 |
| 自定义 S3 endpoint、MinIO、RustFS、Backblaze B2、Wasabi、DigitalOcean Spaces | v0.4，逐家验证 |
| 腾讯云 COS、七牛云 Kodo | v0.5，逐类型交付 |
| Azure Blob Storage、Google Cloud Storage 等 | 后续 |

目前尚无可用的存储接入。R2 连接会支持直接指定桶名，使用桶级密钥时不必授予列出全部桶的权限。各家独有功能会在基础文件操作验证后逐步增加。

## 交给 AI agent

用大白话说出需求，agent 借助 Bucky 帮你办好：

> 把 `backups/` 下 90 天前的文件移到 `archive/`。

目标流程是先找出文件、展示计划，等你同意后再移动并汇报。只有验证过人工确认流程的客户端才会开放写执行，其他客户端只开放读取和计划。

> 把 `dist` 上传到 R2，给我一个 7 天有效的链接。

如果上传的是 `dist/` 这样的目录，会返回其中各个文件的链接清单，每个链接分别设置有效期；目录本身不是单个可下载对象。

- **兼容你在用的工具。** Claude Code、Codex、Cursor 以及其他 MCP 客户端都能用。在设置里一键安装 Skill，可以装到全局，也可以只装到某个项目。
- **提示词里不用贴密钥。** agent 直接用你在应用里配好的连接，全程看不到你的密钥。
- **默认安全。** 读取直接执行；删除、移动、覆盖一律先出计划，等你确认。
- **需要时设为只读。** 把连接标记为只读，任何 agent 都改不了里面的内容。
- **也能写进脚本。** `bucky` 命令能做 agent 能做的一切，支持 JSON 输出，方便你自己写脚本。

## 路线图

| 版本 | 内容 |
| --- | --- |
| v0.1 | R2 完整流程：浏览、可靠传输、预览、分享、图床和 Agent |
| v0.2 | AWS S3，沿用桌面和 Agent 工作流程 |
| v0.3 | 阿里云 OSS |
| v0.4 | 自定义 S3 endpoint 和经验证的兼容服务 |
| v0.5 | 腾讯云 COS、七牛云 Kodo |
| 后续 | 更多存储类型、跨云复制/同步、安全体检和自然语言批量操作 |
| v1.0 方向 | 把存储桶挂载成本地磁盘，在访达和资源管理器里使用 |

优先完成存储类型接入。至少两种服务稳定后再规划跨云流程，自动更新及 Homebrew、winget、Scoop 分发也在后续安排。

## 下载

目前还没有发布版本。第一个版本会在 [GitHub Releases](https://github.com/amigoer/bucky/releases) 提供 macOS、Windows、Linux 安装包。在本仓库页面选择 Watch → Custom → Releases，发布时就会收到通知。

## 反馈

想法、需求和 bug 都欢迎提到 [GitHub Issues](https://github.com/amigoer/bucky/issues)。

[Apache-2.0](LICENSE) © 2026 [Amigoer](https://github.com/amigoer)
