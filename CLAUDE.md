# Bucky development

- Read `docs/SPEC.md`, `docs/architecture.md`, `docs/development-plan.md` and the ADRs before changing behavior.
- Work on `main` during early development. Check, commit and push completed work directly; do not open a PR unless requested.
- Use English for code, comments and commit messages. Follow Conventional Commits without attribution footers.
- Keep the Go module and the four layers: entry adapters, application services, core contracts, provider adapters.
- Only `cmd/bucky-desktop` and `internal/desktop` import MyGo (ADR-004). Cloud SDK types stay inside drivers.
- v0.1 connection types are Tencent COS, Aliyun OSS, AWS S3 and Cloudflare R2 (ADR-005). S3 and R2 share `s3compat`; COS and OSS use their native SDKs. MinIO and RustFS are S3 integration test backends only.
- Never commit credentials. Use the system keychain or explicitly selected process environment credentials.
- Deletion, movement and overwriting require a persisted plan and explicit user confirmation. Enforce read-only connections in services.
- MCP exposes reads and plans until an independent human confirmation mechanism has been verified for a client.
- Preserve streaming, conditional checks, resumable transfer state and per-item results. Never treat ETags as universal content hashes.
- Pin dependencies and test images. Keep `CGO_ENABLED=0` CLI builds working.
- Use `task lint`, `task test`, `task test:integration` and `task build` as appropriate. Record missing cloud or platform verification honestly.
- Do not create releases, tags, deploys or registry uploads without explicit authorization.
