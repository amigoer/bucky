<div align="center">
  <h1>Bucky</h1>

  <h3>One desktop app for all your object storage</h3>

  <p>Browse, upload, preview and share your files, starting with Cloudflare R2.<br>AWS S3, Alibaba Cloud OSS and more will follow, with the same workflow for you and your AI agent.</p>

  <p>
    <img src="https://img.shields.io/badge/status-early%20development-F59E0B?style=flat-square&labelColor=1A1A1E" alt="Status: early development">
    <img src="https://img.shields.io/badge/platform-macOS%20%7C%20Windows%20%7C%20Linux-3F3F46?style=flat-square&labelColor=1A1A1E" alt="Platforms: macOS, Windows, Linux">
    <a href="#let-your-agent-help"><img src="https://img.shields.io/badge/agents-CLI%20%C2%B7%20MCP%20%C2%B7%20Skill-3F3F46?style=flat-square&labelColor=1A1A1E" alt="For agents: CLI, MCP and Skill"></a>
    <a href="LICENSE"><img src="https://img.shields.io/badge/license-Apache--2.0-3F3F46?style=flat-square&labelColor=1A1A1E" alt="Apache-2.0 license"></a>
  </p>

  <p>
    <a href="#what-you-can-do">Features</a>&nbsp;&nbsp;·&nbsp;&nbsp;
    <a href="#supported-storage">Storage</a>&nbsp;&nbsp;·&nbsp;&nbsp;
    <a href="#let-your-agent-help">For agents</a>&nbsp;&nbsp;·&nbsp;&nbsp;
    <a href="#roadmap">Roadmap</a>&nbsp;&nbsp;·&nbsp;&nbsp;
    <a href="README.zh-CN.md">简体中文</a>
  </p>
</div>

<br>

> [!NOTE]
> Bucky is in early development and has not been released yet. The first release targets
> Cloudflare R2 only. The features below are planned; other storage providers will follow in
> separate releases.

## Why Bucky

Every cloud has its own console, its own tool, and its own way of doing the same few things.
Bucky brings them together.

- **Free and open source, on every desktop.** One app for macOS, Windows and Linux.
- **Start with R2, grow from there.** The first release focuses on R2. AWS S3, Alibaba Cloud
  OSS and other providers will follow, keeping the same browsing and transfer workflow.
- **Made for AI agents.** A command line tool, an MCP server and an Agent Skill let Claude Code,
  Codex or Cursor work with your buckets, through the connections you have already set up.
- **Review destructive changes first.** Before anything is deleted, moved or overwritten, Bucky
  shows exactly what will change and waits for you to confirm, whether you clicked or an agent
  asked.
- **Your keys stay yours.** Credentials live in your system keychain and never appear in logs or
  agent output. No telemetry.

## What you can do

The R2 release is being built around these workflows:

- **Browse** buckets as folders, with breadcrumbs, list and grid views, sorting, search and an
  info panel. Paginated lists are designed to keep large folders responsive.
- **Preview** images, audio, video, PDFs, text, code and JSON. Videos stream, so there is no
  waiting for a full download.
- **Upload and download** files and whole folders by dragging them in. Large files go up in
  parts. Transfers can be paused, resumed and retried, carry on after a dropped connection or a
  restart, and are verified when they finish.
- **Share** a temporary link for a single file, valid for up to seven days (one hour by
  default). Public links use a domain you have already connected to your bucket.
- **Host images** for your blog or docs: press a shortcut to upload the image on your clipboard,
  name it with a pattern like `{year}/{month}/{hash}.{ext}`, optionally compress it or convert
  it to WebP, and get a Markdown link ready to paste.
- **See usage** by folder and file type, counted in the background while you keep working.
- **Clean up in bulk.** Copy, move or delete files within the same connection, with a preview
  before anything is moved, deleted or overwritten.

Bucky speaks English and Chinese, comes in light and dark themes, and has keyboard shortcuts
throughout.

## Supported storage

| Storage | Planned release |
| --- | --- |
| Cloudflare R2 | v0.1: the first release |
| AWS S3 | v0.2 |
| Alibaba Cloud OSS | v0.3 |
| Custom S3-compatible endpoints, MinIO, RustFS, Backblaze B2, Wasabi, DigitalOcean Spaces | v0.4: verified service by service |
| Tencent Cloud COS, Qiniu Kodo | v0.5: one provider at a time |
| Azure Blob Storage, Google Cloud Storage and more | Later |

No provider is available yet. R2 connections will support an explicit bucket name, so you
can use bucket-scoped credentials without permission to list every bucket. Provider-specific
features will be added after each provider's basic file operations are verified.

## Let your agent help

Ask in plain words, and your agent gets it done through Bucky:

> Move everything under `backups/` older than 90 days to `archive/`.

The agent finds the files, shows you the plan, waits for your go-ahead, then moves them and
reports back.

> Upload `dist` to R2 and give me a link that works for 7 days.

For a folder such as `dist/`, the result is a list of links to its files, each with its own
expiry. A folder is not a single downloadable object.

- **Works with the tools you use.** Claude Code, Codex, Cursor and other MCP clients. Bucky
  installs its Skill for you from Settings, for one project or everywhere.
- **No keys in prompts.** Agents use the connections you set up in the app and never see your
  credentials.
- **Safe by default.** Reading happens right away. Deleting, moving and overwriting always come
  back to you as a plan first.
- **Read-only when you want it.** Mark a connection read-only and no agent can change anything
  in it.
- **Scriptable too.** The `bucky` command does everything an agent can, with JSON output for
  your own scripts.

## Roadmap

| Version | Coming |
| --- | --- |
| v0.1 | A complete R2 workflow: browsing, reliable transfers, previews, sharing, image hosting and agents |
| v0.2 | AWS S3, with the same desktop and agent workflows |
| v0.3 | Alibaba Cloud OSS |
| v0.4 | Custom S3 endpoints and verified compatible services |
| v0.5 | Tencent Cloud COS and Qiniu Kodo |
| Later | More storage types, cross-cloud copy and sync, security checkups and natural-language bulk operations |
| v1.0 direction | Mount a bucket as a drive in Finder and File Explorer |

Storage integrations come first. Cross-cloud workflows will be planned once at least two
providers are stable. Automatic updates and Homebrew, winget and Scoop distribution will follow.

## Download

There is no release yet. The first one will bring installers for macOS, Windows and Linux on
[GitHub Releases](https://github.com/amigoer/bucky/releases). Choose Watch → Custom → Releases
on this repository to get notified.

## Feedback

Ideas, requests and bug reports are welcome in
[GitHub Issues](https://github.com/amigoer/bucky/issues).

[Apache-2.0](LICENSE) © 2026 [Amigoer](https://github.com/amigoer)
