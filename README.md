<div align="center">
  <h1>Bucky</h1>

  <h3>One desktop app for all your object storage</h3>

  <p>Browse, upload, preview and share files on Cloudflare R2, AWS S3, Alibaba Cloud OSS, Tencent Cloud COS,<br>Qiniu and any S3-compatible service, and let your AI agent lend a hand, safely.</p>

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
> Bucky is in early development and has not been released yet. This page describes the first
> version as it is being built. Star or watch the repository to hear when it ships.

## Why Bucky

Every cloud has its own console, its own tool, and its own way of doing the same few things.
Bucky brings them together.

- **Free and open source, on every desktop.** One app for macOS, Windows and Linux.
- **Every provider in one place.** Connections to R2, S3, OSS, COS, Qiniu and more sit side by
  side. The Chinese clouds connect through their own APIs, so extras such as CDN cache refresh
  are there too.
- **Made for AI agents.** A command line tool, an MCP server and an Agent Skill let Claude Code,
  Codex or Cursor work with your buckets, through the connections you have already set up.
- **Nothing changes without your say.** Before anything is deleted, moved or overwritten, Bucky
  shows exactly what will change and waits for you to confirm, whether you clicked or an agent
  asked.
- **Your keys stay yours.** Credentials live in your system keychain and never appear in logs or
  agent output. No telemetry.

## What you can do

- **Browse** buckets as folders, with breadcrumbs, list and grid views, sorting, search and an
  info panel. Folders holding a hundred thousand files still scroll smoothly.
- **Preview** images, audio, video, PDFs, text, code and JSON. Videos stream, so there is no
  waiting for a full download.
- **Upload and download** files and whole folders by dragging them in. Large files go up in
  parts. Transfers can be paused, resumed and retried, carry on after a dropped connection or a
  restart, and are verified when they finish.
- **Share** a link that expires when you choose (one hour by default), a public link, or a link
  on your own domain or CDN, copied in one click.
- **Host images** for your blog or docs: press a shortcut to upload the image on your clipboard,
  name it with a pattern like `{year}/{month}/{hash}.{ext}`, optionally compress it or convert
  it to WebP, and get a Markdown link ready to paste.
- **See usage** by folder and file type, counted in the background while you keep working.
- **Clean up in bulk.** Delete, move or rename many files at once, with a preview of every
  change first.

Bucky speaks English and Chinese, comes in light and dark themes, and has keyboard shortcuts
throughout.

## Supported storage

| Provider | What you get |
| --- | --- |
| Cloudflare R2, AWS S3, Backblaze B2, Wasabi, DigitalOcean Spaces, MinIO | Ready-made presets: pick the provider, add your keys, connect |
| Alibaba Cloud OSS, Tencent Cloud COS, Qiniu Kodo | Presets plus provider-only features such as CDN cache refresh |
| Any other S3-compatible service | Connect with a custom endpoint |

Each connection only offers what its provider actually supports, so nothing fails halfway.
Already using the AWS CLI? Bucky can import the profiles in `~/.aws/credentials`.

## Let your agent help

Ask in plain words, and your agent gets it done through Bucky:

> Move everything under `backups/` older than 90 days to `archive/`.

The agent finds the files, shows you the plan, waits for your go-ahead, then moves them and
reports back.

> Upload `dist` to R2 and give me a link that works for 7 days.

One request, one link.

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
| v0.1 | Everything on this page |
| v0.2 | Copy and sync between two clouds, with an estimate of transfer costs before you start |
| v0.3 | A security checkup that finds public buckets, overly broad policies and unencrypted storage |
| v0.4 | Describe a bulk change in plain words, review the plan, then run it |
| v1.0 | Mount a bucket as a drive in Finder and File Explorer |

Along the way: more providers such as Azure Blob Storage, Google Cloud Storage, Huawei Cloud OBS
and Upyun, automatic updates, and installs through Homebrew, winget and Scoop.

## Download

There is no release yet. The first one will bring installers for macOS, Windows and Linux on
[GitHub Releases](https://github.com/amigoer/bucky/releases). Choose Watch → Custom → Releases
on this repository to get notified.

## Sister project

[MQ Studio](https://github.com/amigoer/mq-studio) does for message queues what Bucky does for
object storage: one local-first desktop app for RocketMQ, Kafka, RabbitMQ, Pulsar and more.

## Feedback

Ideas, requests and bug reports are welcome in
[GitHub Issues](https://github.com/amigoer/bucky/issues).

[Apache-2.0](LICENSE) © 2026 [amigoer](https://github.com/amigoer)
