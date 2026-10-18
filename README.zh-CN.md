<div align="center">
  <h1>Bucky</h1>

  <h3>一个桌面应用，管好你所有的对象存储</h3>

  <p>Cloudflare R2、AWS S3、阿里云 OSS、腾讯云 COS、七牛云以及任何 S3 兼容存储，<br>浏览、上传、预览、分享都在一处完成，还能放心地交给 AI agent 帮忙。</p>

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
> Bucky 仍在早期开发中，尚未发布。本页介绍的是正在开发的第一个版本。给仓库点个 Star 或 Watch，发布时第一时间收到通知。

## 为什么用 Bucky

每家云都有自己的控制台、自己的工具，做的却是同样几件事。Bucky 把它们放到一起。

- **开源免费，全平台可用。** macOS、Windows、Linux 用的是同一个应用。
- **国内外存储一站管理。** R2、S3、OSS、COS、七牛等连接并排放在一起。国内云走各自的原生接口，CDN 刷新这类独有功能也能直接用。
- **为 AI agent 而生。** 命令行工具、MCP server 和 Agent Skill 三件套，让 Claude Code、Codex、Cursor 直接用你在应用里配好的连接操作存储桶。
- **你不点头，什么都不会变。** 删除、移动、覆盖之前，Bucky 都会先列出将要发生的变化，等你确认；不管操作来自你自己还是 agent，都是如此。
- **密钥只属于你。** 凭证存放在系统钥匙串里，不会出现在日志和 agent 的输出中。没有任何遥测。

## 能做什么

- **浏览**：把存储桶当文件夹来逛，有面包屑、列表和网格视图、排序、搜索和信息面板。十万个文件的文件夹也能流畅滚动。
- **预览**：图片、音视频、PDF、文本、代码和 JSON。视频边下边播，不用等完整下载。
- **上传下载**：直接拖入文件或整个文件夹。大文件自动分片上传；传输可以暂停、继续、重试，断网或重启后接着传，传完自动校验。
- **分享**：生成有效期自定的临时链接（默认 1 小时）、公开链接，或者用你自己的域名和 CDN 域名，一键复制。
- **图床**：写博客和文档时按一下快捷键，就把剪贴板里的图片传上去；按 `{year}/{month}/{hash}.{ext}` 这样的模板命名，可选压缩和转 WebP，Markdown 链接自动复制好。
- **用量统计**：按文件夹和文件类型统计数量与大小，在后台计算，不耽误你继续操作。
- **批量整理**：一次删除、移动、重命名大量文件，每一处变化都先给你预览。

Bucky 支持中英双语和明暗两套主题，常用操作都有快捷键。

## 支持的存储

| 服务商 | 能用到什么 |
| --- | --- |
| Cloudflare R2、AWS S3、Backblaze B2、Wasabi、DigitalOcean Spaces、MinIO | 内置预设：选好服务商，填上密钥，即可连接 |
| 阿里云 OSS、腾讯云 COS、七牛云 Kodo | 内置预设，外加 CDN 刷新等各家独有功能 |
| 其他任何 S3 兼容服务 | 填写自定义 endpoint 连接 |

每个连接只提供服务商真正支持的功能，不会点到一半才报错。已经在用 AWS CLI？Bucky 可以直接导入 `~/.aws/credentials` 里的配置。

## 交给 AI agent

用大白话说出需求，agent 借助 Bucky 帮你办好：

> 把 `backups/` 下 90 天前的文件移到 `archive/`。

agent 会先找出这些文件，把执行计划拿给你看，等你同意后再移动，最后汇报结果。

> 把 `dist` 上传到 R2，给我一个 7 天有效的链接。

一句话，一个链接。

- **兼容你在用的工具。** Claude Code、Codex、Cursor 以及其他 MCP 客户端都能用。在设置里一键安装 Skill，可以装到全局，也可以只装到某个项目。
- **提示词里不用贴密钥。** agent 直接用你在应用里配好的连接，全程看不到你的密钥。
- **默认安全。** 读取直接执行；删除、移动、覆盖一律先出计划，等你确认。
- **需要时设为只读。** 把连接标记为只读，任何 agent 都改不了里面的内容。
- **也能写进脚本。** `bucky` 命令能做 agent 能做的一切，支持 JSON 输出，方便你自己写脚本。

## 路线图

| 版本 | 内容 |
| --- | --- |
| v0.1 | 本页介绍的全部功能 |
| v0.2 | 在两个云之间复制和同步，开始前先预估流量费用 |
| v0.3 | 安全体检：找出公开可读的桶、过宽的访问策略和未加密的存储 |
| v0.4 | 用一句话描述批量操作，预览计划后再执行 |
| v1.0 | 把存储桶挂载成本地磁盘，在访达和资源管理器里直接使用 |

此外还会陆续支持 Azure Blob Storage、Google Cloud Storage、华为云 OBS、又拍云等更多服务商，以及自动更新和通过 Homebrew、winget、Scoop 安装。

## 下载

目前还没有发布版本。第一个版本会在 [GitHub Releases](https://github.com/amigoer/bucky/releases) 提供 macOS、Windows、Linux 安装包。在本仓库页面选择 Watch → Custom → Releases，发布时就会收到通知。

## 反馈

想法、需求和 bug 都欢迎提到 [GitHub Issues](https://github.com/amigoer/bucky/issues)。

[Apache-2.0](LICENSE) © 2026 [amigoer](https://github.com/amigoer)
