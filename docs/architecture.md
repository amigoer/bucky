# Bucky 开发架构

- 状态：已采纳，v0.1 实现进行中
- 更新日期：2026-10-18
- 决策依据：[ADR-002](adr/0002-r2-first-storage.md)
- 需求与验收：[SPEC](SPEC.md)
- 开发顺序：[开发规划](development-plan.md)

## 1. 架构目标

Bucky 是本地优先的对象存储客户端。采用 Go + Wails v3 + React，一个本地核心承载桌面端、CLI 和 MCP 的业务逻辑。第一版 v0.1 只接 Cloudflare R2，先完成一个存储类型的完整体验，再逐类接入 AWS S3、阿里云 OSS 等服务。

架构保持简单：单一 Go module、进程内函数调用、SQLite 持久化、系统钥匙串保存凭证。不引入远程业务服务器、微服务、消息队列、运行时插件或依赖注入框架；不为尚未接入的厂商建立空实现。

桌面应用、CLI 和 MCP 按各自入口启动本地进程，共用代码与本地配置，不要求桌面应用先运行。首版不额外维护本地常驻服务或自定义 RPC 协议。

## 2. 四层结构

```text
React desktop UI / bucky CLI / MCP tools
                    |
            Entry adapters
                    |
        Application services (service)
                    |
          Storage contracts (core)
                    |
       Provider adapters (drivers)
                    |
           Cloud storage APIs

Application services also use profile, transfer and store.
Credentials are resolved through the system keychain.
```

| 层 | 职责 | 不承担的内容 |
| --- | --- | --- |
| 入口 | React 交互；Wails 参数和事件转换；CLI 参数和 JSON；MCP 工具输入输出 | 存储操作规则、凭证持久化、上传调度 |
| `service` | 连接测试、浏览、预览、分享、操作计划、只读限制，组合传输和本地数据 | 云 SDK 的具体调用、Wails 类型 |
| `core` | 连接和对象模型、存储接口、能力、统一错误 | SDK、数据库、桌面运行时 |
| `drivers` | 将统一操作翻译为对应云 API，处理 endpoint、签名、分页和兼容差异 | 界面交互、业务确认、全局任务队列 |

`cmd` 负责显式创建依赖并传给各入口。`service` 使用 `core` 接口，由组合入口注入具体驱动；`core` 不反向引用 `service`、`drivers` 或基础设施。

只有桌面入口和 `internal/app` 可以依赖 Wails。React 只调用生成的绑定，不直接请求云 API，也不保存存储密钥。SDK 返回值在驱动内转换，不能直接成为前端、CLI 或 MCP 的数据结构。

## 3. 目录与模块

以下是随功能逐步创建的目标目录，不是要求 Phase 0 一次生成全部文件。

```text
bucky/
├── cmd/
│   ├── bucky-desktop/     # Desktop composition root
│   └── bucky/             # CLI and MCP composition root
├── internal/
│   ├── core/             # Models, storage contracts and errors
│   ├── service/          # Shared application operations and plans
│   ├── drivers/
│   │   ├── s3compat/     # Shared S3 protocol implementation
│   │   └── providers/    # Type registration and validated presets
│   ├── profile/          # Connection configuration and keychain access
│   ├── transfer/         # Queue, streaming and resumable transfers
│   ├── store/            # SQLite schema, migrations and repositories
│   ├── app/              # Wails bindings and event conversion
│   └── agent/            # MCP transport and Skill installation
├── frontend/
│   └── src/
│       ├── features/     # Connections, browser, transfers and settings
│       ├── components/   # Shared UI components
│       └── lib/          # Bindings wrapper, query setup and i18n
├── build/                # Platform build resources and tasks
├── test/integration/     # Local backends and opt-in cloud tests
├── docs/
└── Taskfile.yml
```

R2 图床、分享和用量先作为 `service` 内的具体用例，出现可复用的独立职责后再拆包。OSS 接入时新增 `drivers/oss/`；COS、Kodo 和非 S3 协议适配器到对应版本再创建。

Wails 的入口路径和绑定生成在 Phase 0 验证；如果模板只能在根目录稳定工作，按 ADR-001 调整桌面入口位置，其余分层不变。

## 4. 按存储类型接入

产品中的存储类型、底层协议和已验证支持范围分别管理。共用协议不代表已经支持所有使用该协议的产品。

| 连接类型 | 服务 | 实现方式 | 交付顺序 |
| --- | --- | --- | --- |
| `r2` | Cloudflare R2 | `s3compat` + R2 endpoint、region、校验策略和能力配置 | v0.1，唯一可选择的类型 |
| `s3` | AWS S3 普通桶 | 复用 `s3compat`，增加 AWS region、凭证解析和能力配置 | v0.2 |
| `oss` | 阿里云 OSS | 独立原生适配器，使用 OSS Go SDK v2 | v0.3 |
| `s3-compatible` | 自定义 S3 endpoint、MinIO、RustFS、B2、Wasabi、Spaces 等 | 复用 `s3compat`，按测试记录提供预设和兼容选项 | v0.4 |
| `cos`、`kodo` | 腾讯云 COS、七牛 Kodo | 按各家协议及原生能力接入，验证后确定复用边界 | v0.5 |
| `azure-blob`、`gcs` 等 | Azure Blob、Google Cloud Storage 等 | 实现相同业务契约的独立适配器 | 后续，逐类交付 |

注册方式用普通 Go 工厂和类型表即可。`provider_type` 是持久化的稳定标识，未知类型返回明确错误；未实现的类型不出现在新增连接界面。SDK 和额外依赖只在相应类型开发时引入。

R2 和 AWS S3 的差异在配置与驱动内处理，不复制浏览、上传和删除的业务流程。OSS 的基础读写也走原生适配器，不强制经由 S3 再叠一层；CDN 刷新属于另一个产品 API，到 OSS 增强阶段再接入。

## 5. 存储契约

先定义首版真正使用的小接口，不一次性设计覆盖所有云产品的大接口。具体 Go 签名在 Phase 1 落地，以下约束作为验收依据。

| 契约 | 操作与语义 |
| --- | --- |
| 基础对象读写 | 按 prefix 和 delimiter 分页列举、对象元数据、带范围读取、流式写入、按对象返回结果的删除 |
| 桶发现（可选） | 列举当前凭证允许发现的桶；不支持或无权限时仍允许指定桶名连接 |
| 预签名（可选） | 返回链接、所需请求头与实际过期时间 |
| 分片上传（可选） | 初始化、上传分片、列已完成分片、完成和中止；状态可恢复 |
| 服务端复制（可选） | 在已验证的范围内复制；大对象按厂商限制使用分片复制 |
| 厂商增强（后续） | 版本、生命周期、CDN 刷新、图片处理等按需增加独立接口 |

`core` 至少定义 `Profile`、`Bucket`、`ObjectRef`、`ObjectInfo`、`ListPage`、`TransferTask`、`OperationPlan` 和统一错误。对象身份包含连接 ID、桶和完整 key，不能只用文件名。

能力只描述驱动已实现的操作，不承诺当前密钥拥有权限。可选接口与能力声明必须一致；实际请求仍可能返回无权限、限流或功能不支持，入口展示具体原因。只读连接的写限制由 `service` 统一执行。

SDK 的分页 token 当作不透明字符串使用；不能根据 token、ETag 或对象 key 推断跨厂商行为。ETag 不统一当作 MD5。

## 6. R2 首版边界

| 项目 | 设计 |
| --- | --- |
| 身份配置 | Account ID、R2 S3 Access Key ID、Secret Access Key；可指定桶名与只读模式。不是 Cloudflare 通用管理 API token |
| Endpoint | 默认 `https://<ACCOUNT_ID>.r2.cloudflarestorage.com`；受地域管辖的桶使用官方对应 endpoint，作为独立连接保存 |
| Region | `auto`；请求签名、寻址和校验配置集中在 R2 适配配置内 |
| 连接测试 | 指定桶时验证该桶的对象列举权限；不要求 Admin 权限或列举全部桶成功 |
| 支持范围 | 已存在桶内的浏览、元数据、上传下载、分片、复制/移动/删除、预览和链接分享 |
| 不纳入首版 | 创建/删除桶、Cloudflare 账号管理、自动修改桶 CORS、CDN 配置、ACL 和 AWS 专属管理界面 |
| 分享 | 预签名下载默认 1 小时，R2 上限 7 天；自定义公开域名另行配置，只对已开放访问的对象生成公开链接 |
| 预览 | 有界文本读取；音视频和 PDF 使用短期链接按需读取。浏览器直接访问遵循桶 CORS，不通过放宽整个桶权限来解决 |
| 校验 | 按 R2 官方兼容表设置；分片 ETag 用于协议交互，不作为整个文件的内容哈希 |

预签名链接只能使用 R2 S3 API 域名，不能替换为自定义域名。生成链接是本地签名，不构成远端连通性或读取权限验证；界面不因此显示“连接测试成功”。预签名链接本身也是临时访问凭据，不写入日志或持久化缓存。

## 7. 关键业务流程

### 浏览与搜索

入口传连接 ID、桶、prefix 和分页游标 → `service` 解析连接并创建驱动 → 请求单页结果 → 返回中立模型。前端用 TanStack Query 管理分页，TanStack Virtual 控制渲染规模。

当前页排序、已加载结果筛选和遍历整个 prefix 的搜索要明确区分。全量搜索是可取消的后台遍历，不能把第一页排序结果称为整个桶的排序。用量统计同样逐页计算，展示统计时间、扫描范围和未完成状态，不预先建立全量对象索引。

### 上传与恢复

文件选择或 CLI 输入 → `service` 检查只读和目标冲突 → 必要时生成覆盖计划 → `transfer` 建立任务 → 驱动上传 → 保存结果并发送进度。

- 流式传输，不把整个大文件读入内存；首版采用固定并发上限和简单任务队列。
- SQLite 保存任务、upload ID、分片编号、ETag、局部校验信息和源文件指纹。恢复前检查源文件是否变化，并与远端 `ListParts` 对账。
- 暂停保留可恢复分片；取消尝试中止远端分片并记录清理失败；崩溃后的 `running` 状态转为等待恢复，不能直接视作成功。
- 支持时使用服务端可验证的校验和，并保存本地哈希；无法验证完整内容时必须注明校验级别。5 GB 验收通过重新下载计算哈希验证，不用 ETag 替代。
- SDK 负责单次请求的有限重试，传输层负责任务恢复；避免多层无限重试。对象写入完成结果不明时，先核对远端再决定是否重试。

### 复制、移动、删除与覆盖

`service` 枚举并生成不可变计划 → 展示对象、字节数、覆盖项和操作代价 → 用户确认 → 按 plan ID 执行并保存逐项结果。

计划绑定连接、桶、精确对象列表、目标和可取得的对象状态，设置有效期。执行时重新检查只读状态及变化；厂商支持条件请求时使用条件保护，不支持时说明检查与写入之间的竞态，不能承诺跨对象事务。

plan ID 只是计划标识，不是人工授权证明。GUI 使用用户确认事件，CLI 使用明确的执行授权；MCP 的人工确认边界在 Phase 5 按客户端验证，不把模型提交的确认参数作为凭据。无法可靠确认的客户端只开放读取和计划。

移动采用复制后删除。大对象复制超过单次 API 限制时走分片复制，不能复制失败后仍删除源对象。目标复制经可用的元数据/校验规则核对后才删除源对象；部分成功独立记录，不以“一次 API 调用”假设批量原子性。首版仅实现同一连接内的操作，不开放跨连接或跨云复制。

### 预览、分享与图床

预览和分享调用相同读取/预签名契约；链接包含 key 的正确 URL 编码。文件夹不是单个对象，不能给整个 `dist/` 签一个下载链接；目录上传返回逐对象链接或清单，大批量时返回清单路径。

图床复用传输任务与分享逻辑，仅增加剪贴板入口、命名模板和可选图片处理；Markdown 链接必须区分临时链接与已配置的公开域名链接。

## 8. 本地数据与进程边界

| 数据 | 保存位置 | 规则 |
| --- | --- | --- |
| 连接元数据 | SQLite `profiles` | 稳定 ID、类型、账号标识、endpoint、region、指定桶、公开域名、只读标记、凭证引用 |
| 密钥及临时凭证 | 系统钥匙串 | 不进入 SQLite、前端状态、日志或导出配置；钥匙串不可用时明确失败，不回退为明文文件 |
| 任务与分片 | SQLite `transfers`、`transfer_parts` | 用事务保存状态；进度事件节流，不能每个网络包都写库 |
| 执行计划与结果 | SQLite `operation_plans`、`operation_results` | 不可变目标、有效期、执行状态；重复执行返回已有状态，防止同一计划并发执行 |
| 用量缓存 | SQLite，Phase 4 再加 | 连接、桶、prefix、统计时间和扫描范围；允许失效重算 |

表名是设计建议，迁移随使用阶段增加，不在 Phase 0 预建未来业务表。

桌面端、CLI 和 MCP 使用相同的系统应用数据目录。SQLite 使用事务、WAL 和适当的等待超时；数据库事务不能覆盖云网络请求。OS 文件锁保证同一数据目录一次只有一个进程执行写任务或接管恢复，其他入口返回“任务忙”，不会隐式启动第二个 worker。锁在进程退出后释放，恢复需重新核对持久化状态；首版不建设跨进程任务调度系统。

CLI 在没有桌面钥匙串的系统上可显式读取进程环境凭证，仅用于该次运行，不持久化、不放在命令行参数中。平台差异在 `profile` 内封装，不能绕开只读检查。AWS 凭证链和 profiles 导入留到 AWS S3 接入时实现。

## 9. 错误、取消与测试

统一错误至少区分配置无效、凭证不可用、无权限、对象不存在、冲突、不支持、需要确认、任务忙、限流、暂时网络故障和已取消。保留可安全展示的厂商错误码与 request ID，不直接输出含签名的完整请求。

网络操作传递 `context`、超时与取消；长任务通过任务 ID 查询状态，前端进度事件只是通知，重新打开页面仍能读到最新状态。可选能力缺失、权限不足与网络失败不能混为“连接无效”。

测试按三层执行：

1. 无网络单元测试验证用例、只读限制、计划、状态迁移和错误转换。
2. MinIO 与 RustFS 运行相同的 S3 协议契约测试，验证分页、流式读写、复制、删除和分片；它们是测试后端，不是 v0.1 支持声明。
3. 真实 R2 的显式集成测试验证 endpoint、桶级权限、签名、校验、预览、续传和链接有效期。普通 PR 不读取云密钥，云验证使用独立测试桶及限定 prefix，并在发布前记录结果。

后续每接入一种存储类型，复用相同的业务契约测试，再补该类型的差异测试和真实服务验证。不能只换 endpoint 就宣布支持，也不能用本地 S3 测试代替 OSS 原生适配器验证。

## 10. 官方参考

- [R2 S3 API 兼容性](https://developers.cloudflare.com/r2/api/s3/api/)
- [R2 API 凭证与地域 endpoint](https://developers.cloudflare.com/r2/api/tokens/)
- [R2 预签名链接](https://developers.cloudflare.com/r2/api/s3/presigned-urls/)
- [AWS S3 概念与桶类型](https://docs.aws.amazon.com/AmazonS3/latest/userguide/Welcome.html)
- [阿里云 OSS Go SDK v2](https://github.com/aliyun/alibabacloud-oss-go-sdk-v2)

这些资料约束接入方式；具体 SDK 方法与厂商限制仍需在相应实现阶段再次核实。
