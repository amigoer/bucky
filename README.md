<div align="center">
  <h1>Bucky</h1>

  <h3>One desktop app for all your object storage</h3>

  <p>Browse, upload, preview and share your files on Tencent Cloud COS, Alibaba Cloud OSS, AWS S3 and Cloudflare R2.<br>More providers will follow, with the same workflow for you and your AI agent.</p>

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
> Tencent Cloud COS, Alibaba Cloud OSS, AWS S3 and Cloudflare R2. So far only R2 is implemented:
> its core workflow passes local protocol tests against MinIO and RustFS, but it has not yet been
> verified against a real R2 account. COS, OSS and S3 are not implemented yet.
> Interfaces and behavior may change; other storage providers will follow in later releases.

## Why Bucky

Every cloud has its own console, its own tool, and its own way of doing the same few things.
Bucky brings them together.

- **Free and open source, on every desktop.** One app for macOS, Windows and Linux.
- **Four providers in the first release.** Tencent Cloud COS, Alibaba Cloud OSS, AWS S3 and
  Cloudflare R2 share one browsing and transfer workflow. Other providers will follow the same way.
- **Made for AI agents.** A command line tool, an MCP server and an Agent Skill let Claude Code,
  Codex or Cursor work with your buckets, through the connections you have already set up.
- **Review destructive changes first.** Before anything is deleted, moved or overwritten, Bucky
  shows exactly what will change and waits for you to confirm, whether you clicked or an agent
  asked.
- **Your keys stay yours.** Credentials live in your system keychain and never appear in logs or
  agent output. No telemetry.

## What you can do

The first release is being built around these workflows, the same for every provider.
Browsing, transfers with resume, previews, sharing, usage and confirmed plans are implemented
for R2 in the current development build; image hosting and one-click Skill installation are still
in progress.

- **Browse** buckets as folders, with breadcrumbs, list and grid views, sorting, search and an
  info panel. Paginated lists are designed to keep large folders responsive.
- **Preview** images, audio, video, PDFs, text, code and JSON. Videos stream, so there is no
  waiting for a full download.
- **Upload and download** files and whole folders by dragging them in. Large files go up in
  parts. Transfers can be paused, resumed and retried, and carry on after a dropped connection
  or a restart. Integrity checks show what was verified using the checks the service supports.
- **Share** a temporary link for a single file, valid for one hour by default and up to the
  limit your provider allows. Public links use a domain you have already connected to your bucket.
- **Host images** for your blog or docs: press a shortcut to upload the image on your clipboard,
  name it with a pattern like `{year}/{month}/{hash}.{ext}`, optionally compress it or convert
  it to WebP, and get a Markdown link ready to paste.
- **See usage** by folder and file type, counted in the background while you keep working.
- **Clean up in bulk.** Copy, move or delete files within the same connection, with a preview
  before anything is moved, deleted or overwritten.

Bucky speaks English and Chinese, comes in light and dark themes, and has keyboard shortcuts
throughout.

## Supported storage

| Storage | Planned release | Status |
| --- | --- | --- |
| Tencent Cloud COS | v0.1: the first release | Not implemented yet |
| Alibaba Cloud OSS | v0.1: the first release | Not implemented yet |
| AWS S3 | v0.1: the first release | Not implemented yet |
| Cloudflare R2 | v0.1: the first release | Implemented, not yet verified on a real account |
| Custom S3-compatible endpoints, MinIO, RustFS, Backblaze B2, Wasabi, DigitalOcean Spaces | Later: verified service by service | Planned |
| Qiniu Kodo | Later | Planned |
| Azure Blob Storage, Google Cloud Storage and more | Later | Planned |

The first release ships only when all four providers pass the same checks against their real
services. Every provider's connections will accept an explicit bucket name, so credentials
limited to one bucket do not need permission to list every bucket.

R2 support is implemented but not yet verified against a real Cloudflare account. The protocol
contract and service workflows pass against MinIO and RustFS. R2 connections can also be added
by signing in to Cloudflare in the browser; Bucky then mints an R2
token limited to the bucket you pick. That flow is covered by tests against a fake Cloudflare API
but has not been run against a real account yet.
Provider-specific features will be added after each provider's basic file operations are verified.

## Let your agent help

Ask in plain words, and your agent gets it done through Bucky:

> Move everything under `backups/` older than 90 days to `archive/`.

The intended flow is to find the files, show you the plan, wait for your go-ahead, then move
them and report back. Write access will be enabled only for clients with a verified user
confirmation flow; other clients will be limited to reads and plans.

> Upload `dist` to R2, then give me links that work for 7 days.

For a folder such as `dist/`, the result is a list of links to its files, each with its own
expiry. A folder is not a single downloadable object.

- **Works with the tools you use.** Claude Code, Codex, Cursor and other MCP clients. Bucky
  installs its Skill for you from Settings, for one project or everywhere.
- **No keys in prompts.** Agents use the connections you set up in the app and never see your
  credentials.
- **Safe by default.** Reading happens right away. Deleting, moving and overwriting always come
  back to you as a plan first. The current MCP server exposes reads and plan creation only;
  writing stays behind an explicit human confirmation in the app or CLI.
- **Read-only when you want it.** Mark a connection read-only and no agent can change anything
  in it.
- **Scriptable too.** The `bucky` command does everything an agent can, with JSON output for
  your own scripts.

## Roadmap

| Version | Coming |
| --- | --- |
| v0.1 | Tencent Cloud COS, Alibaba Cloud OSS, AWS S3 and Cloudflare R2, each with browsing, reliable transfers, previews, sharing, image hosting and agents |
| Next | Custom S3 endpoints and verified compatible services |
| Later | Qiniu Kodo and more storage types, cross-cloud copy and sync, security checkups and natural-language bulk operations |
| v1.0 direction | Mount a bucket as a drive in Finder and File Explorer |

Storage integrations come first. Cross-cloud workflows will be planned once at least two
providers are stable. Automatic updates and Homebrew, winget and Scoop distribution will follow.

## Download

There is no release yet. The first one will bring installers for macOS, Windows and Linux on
[GitHub Releases](https://github.com/amigoer/bucky/releases). Choose Watch → Custom → Releases
on this repository to get notified.

## Build from source

You need Go 1.27.1 or later and [Task](https://taskfile.dev).

```sh
task run      # run the desktop app once
task build    # build the CLI and package the desktop app
```

## Feedback

Ideas, requests and bug reports are welcome in
[GitHub Issues](https://github.com/amigoer/bucky/issues).

[Apache-2.0](LICENSE) © 2026 [Amigoer](https://github.com/amigoer)
