# CLI 输出与退出码

- 更新日期：2026-10-18
- 适用于：`bucky` v0.1 开发版
- 命令帮助：`bucky --help`、`bucky <command> --help`

## 1. 基本约定

所有命令都可以加全局参数 `--json`。默认文本输出面向人阅读；`--json` 输出面向脚本，字段名和错误码保持稳定。

- 成功时，stdout 输出一个 JSON 对象：`{"data": ...}`。
- 失败时，stdout 输出一个 JSON 对象：`{"error": {...}}`，同时返回非零退出码。
- 文本模式下，结果写到 stdout，错误写到 stderr，只显示经过清洗的错误码和消息。
- 凭证只从系统钥匙串或 `BUCKY_R2_*` 环境变量读取，不接受命令行密钥参数。
- 输出不包含访问密钥、签名 URL 的内部签名参数或 SDK 原始错误。
- 需要确认的写操作返回退出码 4，不执行任何远端修改。

远程对象地址格式：

```text
r2://<连接 ID 或连接名>/<桶名>/<对象 key>
```

key 中的空格、`#`、`%`、`+`、非 ASCII 字符等必须按 URL path segment 规则百分号编码。Bucky 会保留解码后的 key 字节。

## 2. JSON 结构

成功：

```json
{"data": []}
```

失败：

```json
{
  "error": {
    "code": "confirmation_required",
    "message": "Review the saved plan, then authorize execution with bucky execute <id> --yes",
    "retryable": false
  }
}
```

部分失败的计划仍属于执行结果，因此只输出计划对象（`data`），不额外输出 `error`；退出码为 3，计划中的 `results[].error` 列出逐项失败原因。

常见错误码：

| 错误码 | 含义 |
| --- | --- |
| `invalid_argument` | 参数、路径或条件不合法 |
| `configuration` | 本地配置缺失或不可用 |
| `credentials_unavailable` | 凭证缺失、被拒绝或已过期 |
| `permission_denied` | 凭证无权执行该操作 |
| `not_found` | 本地记录或远端对象不存在 |
| `conflict` | 对象或本地文件在计划生成后发生变化 |
| `read_only` | 连接被标记为只读 |
| `confirmation_required` | 需要人工确认 |
| `expired` | 计划已过期 |
| `unsupported` | 当前驱动或服务不支持该能力 |
| `partial` | 传输或计划只完成了一部分 |
| `cancelled` | 操作被取消 |
| `timeout` | 网络或服务请求超时 |
| `rate_limited` | 服务端限流，可重试 |
| `network` | 无法连接存储服务，可重试 |
| `verification_failed` | 传输后的完整性校验失败 |
| `source_changed` | 本地源文件在传输过程中变化 |
| `internal` | 未分类的本地错误 |

## 3. 退出码

| 退出码 | 含义 |
| --- | --- |
| 0 | 成功 |
| 1 | 一般错误，包括 `not_found`、`conflict`、`read_only`、网络和内部错误 |
| 2 | 参数错误、配置错误、凭证不可用、未知命令或非法参数数量 |
| 3 | 部分失败；计划或任务已经产生逐项结果 |
| 4 | 需要确认；没有执行破坏性操作 |

## 4. 命令与输出

### 连接

| 命令 | 说明 |
| --- | --- |
| `bucky profiles list` | 列出本地连接 |
| `bucky profiles add --name NAME --account ACCOUNT_ID [--bucket BUCKET] [--readonly] [--environment]` | 新增 R2 连接。默认交互读取两个密钥；`--environment` 读取 `BUCKY_R2_ACCESS_KEY_ID`、`BUCKY_R2_SECRET_ACCESS_KEY` 和可选的 `BUCKY_R2_SESSION_TOKEN` |
| `bucky profiles test ID` | 测试连接并列出可见桶；指定桶时只列出该桶 |
| `bucky profiles show ID` | 显示连接元数据，不包含凭证 |
| `bucky profiles remove ID --yes` | 删除本地连接和钥匙串条目；没有 `--yes` 时退出码 4 |

`--account` 必须是 32 位十六进制 Cloudflare 账户 ID。R2 endpoint 会按账户 ID 和 jurisdiction 生成。

### 浏览与读取

| 命令 | 说明 |
| --- | --- |
| `bucky ls REMOTE [--limit N] [--cursor C] [--recursive]` | 列举对象和文件夹；默认每页 200，最大 1000 |
| `bucky stat REMOTE` | 读取对象元数据 |
| `bucky find REMOTE [--contains TEXT] [--before RFC3339] [--min-size B] [--max-size B] [--limit N] [--cursor C]` | 遍历 prefix 搜索；结果包含 `complete` 和 `nextCursor` |
| `bucky presign REMOTE [--seconds N] [--public]` | 生成预签名读取链接；默认 3600 秒，最大 604800 秒 |
| `bucky du REMOTE` | 统计 prefix 下对象数、字节数和扩展名分布 |

`find` 的 `complete=false` 表示只扫描了部分 prefix。已加载页面的前端筛选与全量 prefix 搜索不是同一概念。

### 传输与对象操作

| 命令 | 说明 |
| --- | --- |
| `bucky cp LOCAL REMOTE` | 上传文件；目标已存在时返回覆盖计划 |
| `bucky cp REMOTE LOCAL` | 下载到本地路径 |
| `bucky cp REMOTE_A REMOTE_B` | 同一连接内复制；未加 `--yes` 时只生成计划 |
| `bucky cp LOCAL REMOTE --recursive` | 上传目录，逐文件创建任务 |
| `bucky cp ... --yes` | 授权执行必要的覆盖计划 |
| `bucky mv REMOTE_A REMOTE_B [--yes]` | 生成移动计划；`--yes` 时执行 |
| `bucky rm REMOTE [REMOTE...] [--yes]` | 生成删除计划；`--yes` 时执行 |

上传覆盖、复制、移动和删除都先绑定当前对象状态并生成带有效期的计划。计划 ID 只是标识符，不是用户同意；`--yes` 是当前命令的人工确认。

`cp LOCAL REMOTE` 遇到已存在对象时，会在文本模式打印计划并以退出码 4 结束；JSON 模式只输出计划对象。重新执行并加 `--yes` 才会覆盖。

### 计划与任务

| 命令 | 说明 |
| --- | --- |
| `bucky plan ID` | 读取已保存的计划，包含逐项来源、目标、大小和执行结果 |
| `bucky execute ID --yes` | 执行已经人工审阅的计划；没有 `--yes` 时退出码 4 |
| `bucky transfers list` | 列出持久化的传输任务 |
| `bucky transfers pause ID` | 暂停任务并保留可恢复状态 |
| `bucky transfers cancel ID` | 取消任务并尝试清理远端分片和临时文件 |
| `bucky transfers resume ID` | 恢复任务并等待完成 |

进程退出不会删除任务。重新运行 `transfers resume` 前，Bucky 会重新核对本地源文件指纹和远端已上传分片。

### Agent 与 MCP

| 命令 | 说明 |
| --- | --- |
| `bucky skill print` | 输出内嵌的 Agent Skill 文本 |
| `bucky mcp` | 在本进程的 stdin/stdout 上运行只读 MCP server |

MCP 工具只包含读取和计划创建。没有执行计划、上传、下载、删除、复制或移动工具。写操作必须由人通过 GUI 或 `bucky execute` 完成。

## 5. 示例

列举：

```bash
bucky ls r2://backup/prod-assets/photos/ --json --limit 100
```

生成删除计划：

```bash
bucky rm r2://backup/prod-assets/old/logo.png --json
```

输出计划后，如果状态是 `pending`，退出码为 4。人工检查后执行：

```bash
bucky execute 5f0c... --yes --json
```

上传并处理覆盖：

```bash
bucky cp ./dist/app.js r2://backup/prod-assets/app.js --json
```

若目标已存在，第一条命令返回计划；确认大小和目标后重新执行：

```bash
bucky cp ./dist/app.js r2://backup/prod-assets/app.js --yes --json
```

## 6. 密钥与环境变量

交互式添加连接时，CLI 会提示输入 Access Key ID 和 Secret Access Key。也可以设置：

```bash
export BUCKY_R2_ACCESS_KEY_ID=...
export BUCKY_R2_SECRET_ACCESS_KEY=...
export BUCKY_R2_SESSION_TOKEN=...
```

使用 `bucky profiles add --environment` 保存一个只引用环境变量的连接。密钥不会写入 SQLite，也不会出现在 `profiles show`、日志或 JSON 错误中。

本地数据目录由 `BUCKY_DATA_DIR` 指定；未指定时使用系统配置目录下的 `Bucky`。同一数据目录由进程文件锁保护。
