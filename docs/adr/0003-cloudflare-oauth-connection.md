# ADR-003：浏览器 Cloudflare 登录作为连接的可选入口

- 状态：已采纳
- 日期：2026-10-19
- 相关：[ADR-002](0002-r2-first-storage.md)、[开发架构](../architecture.md)、[SPEC](../SPEC.md)

## 背景

手动填写 Account ID、Access Key ID 和 Secret Access Key 要求用户先到 Cloudflare 控制台自行创建 R2 令牌，步骤多且容易因权限范围选错而失败。竞品（Blob0）已经支持点击后跳转浏览器完成授权，回来后直接得到可用连接。

Cloudflare 自 2026 年起允许第三方创建 OAuth 客户端，但只支持授权码 + PKCE，没有 device flow；新建客户端默认私有，仅父级账号成员可授权。更关键的是：**OAuth 访问令牌不能用于 R2 的 S3 对象操作**，R2 只接受 R2 API 令牌（Access Key ID 为令牌 `id`，Secret 为令牌 `value` 的 SHA-256 十六进制）。

因此“浏览器登录”本身不能直接产生可用的存储凭证，必须再通过管理 API 铸一把令牌。

## 决策

### 登录负责铸令牌，S3 仍只认 R2 令牌

1. 桌面端在 `127.0.0.1:<port>` 起临时回环监听，生成 PKCE 与 `state`，用系统浏览器打开授权页。
2. 回调交换 access/refresh token，随后 `GET /accounts` 取账号列表、`GET /accounts/{id}/r2/buckets` 取桶列表。
3. 用户在 Bucky 里选桶（可选只读）。`POST /accounts/{id}/tokens` 铸一把**限定单桶**的 R2 令牌：资源串 `com.cloudflare.edge.r2.bucket.<ACCOUNT_ID>_<JURISDICTION>_<BUCKET>`，权限组按运行时查询的 `Workers R2 Storage Bucket Item Write`（只读用 `... Read`）。
4. 派生 S3 凭证，**先通过真实 S3 列举该桶验证**，成功后才写入钥匙串并保存连接。

OAuth 访问令牌只存在于内存会话，不落盘、不回前端；刷新令牌和铸出的令牌 ID 进系统钥匙串，与 S3 凭证分开保存。对象访问永远走 S3，OAuth 令牌不用于 S3。

### 凭证来源第三种取值

`Profile.CredentialSource` 增加 `cloudflare`，语义与 `keychain` 相同（S3 凭证在钥匙串中），额外约束是连接必须指定单桶。`normalizeProfile` 强制这一点；`SaveProfile` 拒绝该来源携带手填密钥。

### 失败与撤销

- 铸令牌成功但验证或保存失败时，best-effort 删除刚铸出的令牌，不留孤儿。
- 重新授权先铸新令牌，保存成功后再删除旧令牌。
- 断开连接需要显式确认：刷新 access token，`DELETE /accounts/{id}/tokens/{tokenID}`，撤销 refresh token，清理钥匙串；连接本身保留，可再次授权。
- 同一时间只允许一个登录会话（回调端口固定）；`Service.Close` 取消所有未完成登录并释放端口。

### 客户端配置

OAuth Client ID、回调地址和 scope 存在 SQLite `settings` 表，可在设置页填写，环境变量 `BUCKY_CLOUDFLARE_CLIENT_ID`、`BUCKY_CLOUDFLARE_REDIRECT_URI`、`BUCKY_CLOUDFLARE_SCOPES` 可覆盖。放在 Go 侧而不是前端本地存储，是为了 CLI 与 MCP 以后复用同一份配置。

私有客户端只允许该 Cloudflare 账号成员授权，自用足够。公开发布需要域名 TXT 验证且不可回退，首版不要求。

## 后果

- 用户无需手动创建 R2 令牌；权限范围由 Bucky 控制且限定到单桶。
- 连接依赖一个额外的 Cloudflare 敏感权限 `Account API Tokens Write`；未勾选时无法自动铸令牌，界面必须给出明确指引。
- 客户端对 Cloudflare 管理 API 有了真实依赖，需按官方文档验证回环回调、scope 字符串和令牌请求体；这些在真实账号验证前视为未验证。
- 手动 S3 密钥入口保留为后备路径，CLI 与环境变量凭据不受影响。
