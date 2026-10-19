// In-memory backend for `vite dev` in a plain browser. Never bundled in builds.
import type {
  Backend,
  Bucket,
  CloudflareSignIn,
  EventSource,
  ObjectInfo,
  ObjectRef,
  OperationPlan,
  PlanItem,
  Profile,
  TransferTask,
  Usage,
  UsageProgress,
} from "./api";
import { models } from "./api";

const { core, service, app } = models;
const MiB = 1024 * 1024;
const now = Date.now();
const day = 86_400_000;

const sleep = (ms: number) => new Promise((resolve) => setTimeout(resolve, ms));
const fail = (code: string, message: string) =>
  Promise.reject({ message, cause: { code, message } });
const id = () => Math.random().toString(16).slice(2, 14);

interface StoredObject {
  size: number;
  modified: number;
  contentType: string;
  metadata?: Record<string, string>;
}

const TEXT: Record<string, string> = {
  "README.md":
    "# media\n\nStatic assets and media files.\n\n- images/  site imagery\n- videos/  product videos\n- docs/    documents and reports\n\nUse lowercase kebab-case file names.\n",
  "config.json":
    '{\n  "cdn": "https://cdn.example.dev",\n  "cache": { "images": 31536000, "html": 300 },\n  "variants": ["480w", "960w", "1920w"],\n  "webp": true\n}\n',
  "app.log": Array.from(
    { length: 40 },
    (_, i) =>
      `2026-10-18T14:${String(10 + (i % 50)).padStart(2, "0")}:0${i % 10}Z INFO  worker-${i % 4} processed batch ${4810 + i} in ${120 + i * 7}ms`,
  ).join("\n"),
};

function contentType(key: string): string {
  const ext = key.split(".").pop()?.toLowerCase() ?? "";
  return (
    {
      png: "image/png",
      jpg: "image/jpeg",
      webp: "image/webp",
      svg: "image/svg+xml",
      mp4: "video/mp4",
      mp3: "audio/mpeg",
      pdf: "application/pdf",
      json: "application/json",
      md: "text/markdown",
      log: "text/plain",
      txt: "text/plain",
      gz: "application/gzip",
      woff2: "font/woff2",
    }[ext] ?? "application/octet-stream"
  );
}

function seed(): Map<string, Map<string, StoredObject>> {
  const media = new Map<string, StoredObject>();
  const put = (key: string, size: number, ageDays: number, metadata?: Record<string, string>) =>
    media.set(key, {
      size,
      modified: now - ageDays * day,
      contentType: contentType(key),
      metadata,
    });
  put("hero.png", 2.4 * MiB, 0.1, {
    "cache-control": "public, max-age=31536000",
    author: "design-team",
  });
  put("cover.webp", 684 * 1024, 3);
  put("logo.svg", 8 * 1024, 6);
  put("intro.mp4", 184 * MiB, 6);
  put("podcast-ep12.mp3", 48.2 * MiB, 9);
  put("spec.pdf", 3.6 * MiB, 12);
  put("config.json", 2.1 * 1024, 0.3);
  put("README.md", 4.8 * 1024, 20);
  put("app.log", 3.2 * MiB, 0.05);
  put("brand-sans.woff2", 96 * 1024, 60);
  put("backup-2026-10-17.sql.gz", 2.1 * 1024 * MiB, 1);
  put("docs/report.pdf", 3.6 * MiB, 15);
  put("docs/release-notes.md", 12 * 1024, 8);
  put("docs/data.json", 88 * 1024, 8);
  put("videos/launch-4k.mp4", 2.3 * 1024 * MiB, 4);
  put("empty/", 0, 30);
  for (let i = 1; i <= 1500; i++) {
    put(
      `images/IMG_${String(i).padStart(4, "0")}${i % 5 === 0 ? ".webp" : ".jpg"}`,
      (0.8 + ((i * 37) % 30) / 10) * MiB,
      i * 0.13,
    );
  }
  const assets = new Map<string, StoredObject>();
  assets.set("index.html", { size: 14 * 1024, modified: now - day, contentType: "text/html" });
  assets.set("css/app.css", { size: 48 * 1024, modified: now - day, contentType: "text/css" });
  assets.set("js/app.js", {
    size: 312 * 1024,
    modified: now - day,
    contentType: "text/javascript",
  });
  const backups = new Map<string, StoredObject>();
  backups.set("db/db-2026-10-16.sql.gz", {
    size: 2 * 1024 * MiB,
    modified: now - 2 * day,
    contentType: "application/gzip",
  });
  backups.set("manifest.json", {
    size: 3.2 * 1024,
    modified: now - day,
    contentType: "application/json",
  });
  const cdn = new Map<string, StoredObject>();
  cdn.set("logo.svg", { size: 8 * 1024, modified: now - 5 * day, contentType: "image/svg+xml" });
  cdn.set("og.png", { size: 320 * 1024, modified: now - 5 * day, contentType: "image/png" });
  cdn.set("fonts/inter.woff2", {
    size: 112 * 1024,
    modified: now - 11 * day,
    contentType: "font/woff2",
  });
  return new Map([
    ["p1/media", media],
    ["p1/assets", assets],
    ["p1/builds", new Map()],
    ["p2/prod-backups", backups],
    ["p3/media-cdn", cdn],
  ]);
}

function svgImage(label: string, hue: number): string {
  const svg = `<svg xmlns="http://www.w3.org/2000/svg" width="1200" height="800" viewBox="0 0 1200 800"><defs><linearGradient id="g" x1="0" y1="0" x2="1" y2="1"><stop offset="0" stop-color="hsl(${hue} 70% 62%)"/><stop offset="1" stop-color="hsl(${(hue + 40) % 360} 65% 42%)"/></linearGradient></defs><rect width="1200" height="800" fill="url(#g)"/><circle cx="900" cy="230" r="120" fill="rgba(255,255,255,.25)"/><text x="80" y="700" font-family="-apple-system,Helvetica" font-size="64" font-weight="700" fill="rgba(255,255,255,.9)">${label}</text></svg>`;
  return `data:image/svg+xml;charset=utf-8,${encodeURIComponent(svg)}`;
}

export function createMockBackend(): { backend: Backend; events: EventSource } {
  const profiles: Profile[] = [
    core.Profile.createFrom({
      id: "p1",
      name: "Acme assets",
      providerType: "r2",
      accountId: "4f1c9e2a7d0b4c3e8a9f1b2c3d4e5f60",
      jurisdiction: "default",
      bucket: "",
      publicUrl: "https://cdn.example.dev",
      readOnly: false,
      credentialSource: "keychain",
      createdAt: new Date(now - 30 * day).toISOString(),
      updatedAt: new Date(now - day).toISOString(),
    }),
    core.Profile.createFrom({
      id: "p2",
      name: "Prod backups",
      providerType: "r2",
      accountId: "9a8b7c6d5e4f3a2b1c0d9e8f7a6b5c4d",
      jurisdiction: "eu",
      bucket: "prod-backups",
      publicUrl: "",
      readOnly: true,
      credentialSource: "environment",
      createdAt: new Date(now - 60 * day).toISOString(),
      updatedAt: new Date(now - 2 * day).toISOString(),
    }),
    core.Profile.createFrom({
      id: "p3",
      name: "CDN assets",
      providerType: "r2",
      accountId: "4f1c9e2a7d0b4c3e8a9f1b2c3d4e5f60",
      jurisdiction: "default",
      bucket: "media-cdn",
      publicUrl: "https://cdn.example.dev",
      readOnly: false,
      credentialSource: "cloudflare",
      createdAt: new Date(now - 8 * day).toISOString(),
      updatedAt: new Date(now - 8 * day).toISOString(),
    }),
  ];
  const store = seed();
  const tasks = new Map<string, TransferTask>();
  const plans = new Map<string, OperationPlan>();
  const cancels = new Map<string, () => void>();
  const listeners = {
    transfer: new Set<(task: TransferTask) => void>(),
    usage: new Set<(p: UsageProgress) => void>(),
    cloudflare: new Set<(update: CloudflareSignIn) => void>(),
  };
  let concurrency = 2;
  let rate = 0;
  let settings = service.Settings.createFrom({
    cloudflareClientId: "",
    cloudflareRedirectUri: "http://127.0.0.1:8976/oauth/callback",
    cloudflareScopes: [
      "account-settings.read",
      "workers-r2.read",
      "workers-r2.write",
      "account-api-tokens.write",
    ],
  });
  const cloudflareSessions = new Map<string, { accountId: string; timer?: number }>();
  const authorized = new Set<string>(["p1"]);
  const emitCloudflare = (update: CloudflareSignIn) => {
    for (const cb of listeners.cloudflare) cb(service.CloudflareSignIn.createFrom({ ...update }));
  };

  const bucketMap = (profileId: string, bucket: string) => {
    const m = store.get(`${profileId}/${bucket}`);
    if (!m) throw { message: "Bucket not found", cause: { code: "not_found" } };
    return m;
  };
  const info = (r: ObjectRef, o: StoredObject): ObjectInfo =>
    core.ObjectInfo.createFrom({
      profileId: r.profileId,
      bucket: r.bucket,
      key: r.key,
      size: Math.round(o.size),
      etag: `"${id()}${id()}"`.slice(0, 34),
      lastModified: new Date(o.modified).toISOString(),
      contentType: o.contentType,
      metadata: o.metadata ?? {},
    });
  const profile = (pid: string) => profiles.find((p) => p.id === pid);
  const writable = (pid: string) => {
    if (profile(pid)?.readOnly)
      throw { message: "This connection is read-only", cause: { code: "read_only" } };
  };
  const emit = (task: TransferTask) => {
    tasks.set(task.id, task);
    for (const cb of listeners.transfer) cb(core.TransferTask.createFrom({ ...task }));
  };

  const newTask = (direction: string, r: ObjectRef, localPath: string, size: number) => {
    const task = core.TransferTask.createFrom({
      id: id(),
      direction,
      object: r,
      localPath,
      status: "queued",
      size,
      completed: 0,
      bytesPerSecond: 0,
      partSize: size > 16 * MiB ? 8 * MiB : 0,
      createdAt: new Date().toISOString(),
      updatedAt: new Date().toISOString(),
    });
    emit(task);
    return task;
  };

  setInterval(() => {
    const all = [...tasks.values()];
    let running = all.filter((t) => t.status === "running").length;
    for (const t of all) {
      if (t.status === "queued" && running < concurrency) {
        running++;
        emit(core.TransferTask.createFrom({ ...t, status: "running" }));
      }
    }
    for (const t of [...tasks.values()].filter((x) => x.status === "running")) {
      const cap = rate > 0 ? rate / Math.max(1, running) : Number.POSITIVE_INFINITY;
      const speed = Math.min(cap, (20 + Math.random() * 30) * MiB);
      const completed = Math.min(t.size, t.completed + speed * 0.5);
      const done = completed >= t.size;
      if (done && t.direction === "upload") {
        bucketMap(t.object.profileId, t.object.bucket).set(t.object.key, {
          size: t.size,
          modified: Date.now(),
          contentType: contentType(t.object.key),
        });
      }
      emit(
        core.TransferTask.createFrom({
          ...t,
          completed,
          bytesPerSecond: done ? 0 : speed,
          status: done ? "completed" : "running",
          updatedAt: new Date().toISOString(),
        }),
      );
    }
  }, 500);

  newTask(
    "download",
    core.ObjectRef.createFrom({ profileId: "p1", bucket: "media", key: "intro.mp4" }),
    "/Users/demo/Downloads/intro.mp4",
    184 * MiB,
  );
  const failed = newTask(
    "upload",
    core.ObjectRef.createFrom({ profileId: "p1", bucket: "media", key: "banner-dark.png" }),
    "/Users/demo/Desktop/banner-dark.png",
    1.2 * MiB,
  );
  emit(
    core.TransferTask.createFrom({
      ...failed,
      status: "failed",
      completed: 0.3 * MiB,
      error: { code: "timeout", message: "Operation timed out", retryable: true },
    }),
  );
  const paused = newTask(
    "download",
    core.ObjectRef.createFrom({ profileId: "p1", bucket: "media", key: "videos/launch-4k.mp4" }),
    "/Users/demo/Downloads/launch-4k.mp4",
    2.3 * 1024 * MiB,
  );
  emit(core.TransferTask.createFrom({ ...paused, status: "paused", completed: 1.41 * 1024 * MiB }));
  const done = newTask(
    "upload",
    core.ObjectRef.createFrom({ profileId: "p1", bucket: "media", key: "hero.png" }),
    "/Users/demo/Desktop/hero.png",
    2.4 * MiB,
  );
  emit(core.TransferTask.createFrom({ ...done, status: "completed", completed: 2.4 * MiB }));

  const backend: Backend = {
    async Profiles() {
      await sleep(80);
      return profiles.map((p) => core.Profile.createFrom({ ...p }));
    },
    async SaveProfile(input) {
      await sleep(300);
      const p = core.Profile.createFrom({ ...input.profile });
      if (!p.name.trim())
        return fail("invalid_argument", "A connection name of 1 to 100 characters is required");
      if (!/^[0-9a-f]{32}$/i.test(p.accountId))
        return fail("invalid_argument", "The R2 account ID must contain 32 hexadecimal characters");
      if (!p.id) {
        p.id = id();
        p.createdAt = new Date().toISOString();
        if (p.bucket) store.set(`${p.id}/${p.bucket}`, new Map());
        else for (const b of ["assets", "media"]) store.set(`${p.id}/${b}`, new Map());
        profiles.push(p);
      } else {
        const i = profiles.findIndex((x) => x.id === p.id);
        profiles[i] = p;
        if (p.bucket && !store.has(`${p.id}/${p.bucket}`))
          store.set(`${p.id}/${p.bucket}`, new Map());
      }
      p.updatedAt = new Date().toISOString();
      return p;
    },
    async RemoveProfile(pid, confirmed) {
      await sleep(200);
      if (!confirmed) return fail("confirmation_required", "Confirm removal");
      profiles.splice(
        profiles.findIndex((p) => p.id === pid),
        1,
      );
    },
    async TestConnection(pid) {
      return backend.Buckets(pid);
    },
    async TestDraft(input) {
      await sleep(700);
      if (input.secretAccessKey === "bad") return fail("forbidden", "SignatureDoesNotMatch");
      if (input.profile.bucket) return [core.Bucket.createFrom({ name: input.profile.bucket })];
      return ["assets", "media", "builds"].map((name) => core.Bucket.createFrom({ name }));
    },
    async Buckets(pid) {
      await sleep(150);
      const out: Bucket[] = [];
      for (const key of store.keys()) {
        const [p, b] = key.split("/");
        if (p === pid)
          out.push(
            core.Bucket.createFrom({ name: b, createdAt: new Date(now - 90 * day).toISOString() }),
          );
      }
      return out;
    },
    async List(req) {
      await sleep(180);
      const m = bucketMap(req.profileId, req.bucket);
      const rows: { key: string; prefix: boolean }[] = [];
      const seen = new Set<string>();
      for (const key of [...m.keys()].filter((k) => k.startsWith(req.prefix)).sort()) {
        const rest = key.slice(req.prefix.length);
        const slash = req.delimiter ? rest.indexOf(req.delimiter) : -1;
        if (slash >= 0) {
          const p = req.prefix + rest.slice(0, slash + 1);
          if (!seen.has(p)) {
            seen.add(p);
            rows.push({ key: p, prefix: true });
          }
        } else rows.push({ key, prefix: false });
      }
      const start = req.cursor ? Number(req.cursor) : 0;
      const limit = req.limit || 1000;
      const page = rows.slice(start, start + limit);
      return core.ListPage.createFrom({
        objects: page
          .filter((r) => !r.prefix)
          .map((r) =>
            info(core.ObjectRef.createFrom({ ...req, key: r.key }), m.get(r.key) as StoredObject),
          ),
        prefixes: page.filter((r) => r.prefix).map((r) => r.key),
        nextCursor: start + limit < rows.length ? String(start + limit) : "",
      });
    },
    async Find(req) {
      await sleep(400);
      const m = bucketMap(req.profileId, req.bucket);
      const keys = [...m.keys()].filter((k) => k.startsWith(req.prefix)).sort();
      const objects: ObjectInfo[] = [];
      let i = req.cursor ? Number(req.cursor) : 0;
      let scanned = 0;
      while (i < keys.length) {
        const key = keys[i++];
        scanned++;
        if (key.toLowerCase().includes(req.contains.toLowerCase())) {
          objects.push(
            info(core.ObjectRef.createFrom({ ...req, key }), m.get(key) as StoredObject),
          );
        }
        if (scanned % 1000 === 0 && objects.length >= (req.limit || 1000)) break;
      }
      return core.FindResult.createFrom({
        objects,
        scanned,
        complete: i >= keys.length,
        nextCursor: i >= keys.length ? "" : String(i),
      });
    },
    async Stat(r) {
      await sleep(60);
      const o = bucketMap(r.profileId, r.bucket).get(r.key);
      if (!o) return fail("not_found", "The object does not exist");
      return info(r, o);
    },
    async ReadText(r) {
      await sleep(250);
      const name = r.key.split("/").pop() ?? r.key;
      const o = bucketMap(r.profileId, r.bucket).get(r.key);
      const text = TEXT[name] ?? `// ${name}\n`;
      return core.TextPreview.createFrom({
        text,
        contentType: o?.contentType ?? "text/plain",
        truncated: (o?.size ?? 0) > MiB,
      });
    },
    async Share(r, seconds, publicLink) {
      await sleep(120);
      const p = profile(r.profileId);
      if (publicLink) {
        if (!p?.publicUrl)
          return fail("configuration", "Configure an existing public bucket domain first");
        return core.ShareLink.createFrom({
          url: `${p.publicUrl}/${r.key.split("/").map(encodeURIComponent).join("/")}`,
          headers: {},
          expiresAt: "",
          public: true,
        });
      }
      const ext = r.key.split(".").pop() ?? "";
      const url = ["png", "jpg", "webp", "svg"].includes(ext)
        ? svgImage(r.key.split("/").pop() ?? "", (r.key.length * 47) % 360)
        : `https://${r.bucket}.${p?.accountId}.r2.cloudflarestorage.com/${encodeURI(r.key)}?X-Amz-Expires=${seconds}&X-Amz-Signature=${id()}${id()}`;
      return core.ShareLink.createFrom({
        url,
        headers: {},
        expiresAt: new Date(Date.now() + seconds * 1000).toISOString(),
        public: false,
      });
    },
    async Usage(pid, bucket, prefix, requestID) {
      const m = bucketMap(pid, bucket);
      const keys = [...m.keys()].filter((k) => k.startsWith(prefix));
      const u: Usage = core.Usage.createFrom({
        profileId: pid,
        bucket,
        prefix,
        objects: 0,
        bytes: 0,
        types: {},
        counts: {},
        scannedAt: new Date().toISOString(),
        complete: false,
      });
      let cancelled = false;
      cancels.set(requestID, () => {
        cancelled = true;
      });
      for (let i = 0; i < keys.length; i++) {
        const o = m.get(keys[i]) as StoredObject;
        const dot = keys[i].lastIndexOf(".");
        const ext = dot > keys[i].lastIndexOf("/") ? keys[i].slice(dot).toLowerCase() : "(none)";
        u.objects++;
        u.bytes += Math.round(o.size);
        u.types[ext] = (u.types[ext] ?? 0) + Math.round(o.size);
        u.counts[ext] = (u.counts[ext] ?? 0) + 1;
        if (i % 100 === 99) {
          await sleep(160);
          if (cancelled) break;
          for (const cb of listeners.usage)
            cb(app.UsageProgress.createFrom({ requestId: requestID, usage: { ...u } }));
        }
      }
      cancels.delete(requestID);
      u.complete = !cancelled;
      u.scannedAt = new Date().toISOString();
      return u;
    },
    async CancelRequest(rid) {
      cancels.get(rid)?.();
    },
    async StartTransfer(req) {
      await sleep(120);
      if (req.direction === "upload") {
        writable(req.object.profileId);
        if (bucketMap(req.object.profileId, req.object.bucket).has(req.object.key)) {
          const plan = await backend.CreatePlan(
            core.PlanRequest.createFrom({
              operation: "upload",
              target: req.object,
              localPath: req.localPath,
              sources: [],
            }),
          );
          return service.TransferResult.createFrom({ plan });
        }
        return service.TransferResult.createFrom({
          task: newTask("upload", req.object, req.localPath, 3.4 * MiB),
        });
      }
      const o = bucketMap(req.object.profileId, req.object.bucket).get(req.object.key);
      return service.TransferResult.createFrom({
        task: newTask("download", req.object, req.localPath, o?.size ?? MiB),
      });
    },
    async UploadPaths(pid, bucket, prefix, paths) {
      writable(pid);
      const out = [];
      for (const p of paths) {
        const name = p.split("/").pop() ?? p;
        out.push(
          await backend.StartTransfer(
            core.TransferRequest.createFrom({
              direction: "upload",
              object: { profileId: pid, bucket, key: `${prefix.replace(/^\/+/, "")}${name}` },
              localPath: p,
            }),
          ),
        );
      }
      return out;
    },
    async UploadData(r, base64) {
      writable(r.profileId);
      await sleep(200);
      if (bucketMap(r.profileId, r.bucket).has(r.key)) {
        const plan = await backend.CreatePlan(
          core.PlanRequest.createFrom({
            operation: "upload",
            target: r,
            localPath: `/staged/${id()}`,
            sources: [],
          }),
        );
        return service.TransferResult.createFrom({ plan });
      }
      const size = Math.floor((base64.length * 3) / 4);
      return service.TransferResult.createFrom({
        task: newTask("upload", r, `/staged/${id()}`, size),
      });
    },
    async DownloadTo(dir, sources, base) {
      await sleep(150);
      return sources.map((s) => {
        const o = bucketMap(s.profileId, s.bucket).get(s.key);
        const rel =
          base && s.key.startsWith(base)
            ? s.key.slice(base.length)
            : (s.key.split("/").pop() ?? s.key);
        return app.DownloadResult.createFrom({
          key: s.key,
          task: newTask("download", s, `${dir}/${rel}`, o?.size ?? MiB),
        });
      });
    },
    async Tasks() {
      await sleep(60);
      return [...tasks.values()].map((t) => core.TransferTask.createFrom({ ...t }));
    },
    async ResumeTransfer(tid) {
      const t = tasks.get(tid);
      if (!t) return fail("not_found", "Task not found");
      const next = core.TransferTask.createFrom({ ...t, status: "queued", error: null });
      emit(next);
      return next;
    },
    async PauseTransfer(tid) {
      const t = tasks.get(tid);
      if (t) emit(core.TransferTask.createFrom({ ...t, status: "paused", bytesPerSecond: 0 }));
    },
    async CancelTransfer(tid) {
      const t = tasks.get(tid);
      if (t) emit(core.TransferTask.createFrom({ ...t, status: "cancelled", bytesPerSecond: 0 }));
    },
    async SetRate(bytesPerSecond) {
      rate = bytesPerSecond;
    },
    async SetConcurrency(n) {
      concurrency = Math.min(8, Math.max(1, n));
      return concurrency;
    },
    async CreatePlan(req) {
      await sleep(350);
      const items: PlanItem[] = [];
      let bytes = 0;
      if (req.operation === "upload") {
        const existing = bucketMap(req.target.profileId, req.target.bucket).get(req.target.key);
        items.push(
          core.PlanItem.createFrom({
            source: { profileId: "", bucket: "", key: "", size: 0 },
            target: req.target,
            existing: existing ? info(req.target, existing) : null,
            localPath: req.localPath,
            localSize: 3.4 * MiB,
          }),
        );
        bytes = 3.4 * MiB;
      } else {
        for (const s of req.sources) {
          writable(s.profileId);
          const o = bucketMap(s.profileId, s.bucket).get(s.key);
          if (!o) return fail("not_found", `The object ${s.key} does not exist`);
          let target: ObjectRef | null = null;
          let existing: ObjectInfo | null = null;
          if (req.operation !== "delete") {
            const rel = req.base
              ? s.key.slice(req.base.length)
              : req.sources.length > 1
                ? (s.key.split("/").pop() ?? "")
                : "";
            target = core.ObjectRef.createFrom({ ...req.target, key: req.target.key + rel });
            const ex = store.get(`${target.profileId}/${target.bucket}`)?.get(target.key);
            existing = ex ? info(target, ex) : null;
          }
          items.push(core.PlanItem.createFrom({ source: info(s, o), target, existing }));
          bytes += o.size;
        }
      }
      const plan = core.OperationPlan.createFrom({
        id: id() + id(),
        operation: req.operation,
        items,
        bytes,
        status: "pending",
        warning: "",
        results: [],
        createdAt: new Date().toISOString(),
        expiresAt: new Date(Date.now() + 15 * 60_000).toISOString(),
      });
      plans.set(plan.id, plan);
      return plan;
    },
    async ExecutePlan(pid, confirmed) {
      const plan = plans.get(pid);
      if (!plan) return fail("not_found", "Plan not found");
      if (!confirmed) return fail("confirmation_required", "Confirm the plan");
      if (Date.parse(plan.expiresAt) < Date.now()) return fail("expired", "The plan expired");
      plan.status = "running";
      for (const item of plan.items) {
        await sleep(item.source.key.includes("IMG_") ? 15 : 220);
        if (plan.operation === "upload" && item.target) {
          const task = newTask("upload", item.target, item.localPath ?? "", item.localSize ?? MiB);
          plan.results.push({
            source: item.source,
            target: item.target,
            status: `task:${task.id}`,
          });
          continue;
        }
        const src = bucketMap(item.source.profileId, item.source.bucket);
        const obj = src.get(item.source.key);
        if (!obj) {
          plan.results.push({
            source: item.source,
            target: item.target,
            status: "failed",
            error: {
              code: "not_found",
              message: "The object no longer exists",
              retryable: false,
            },
          });
          continue;
        }
        if (item.target)
          bucketMap(item.target.profileId, item.target.bucket).set(item.target.key, {
            ...obj,
            modified: Date.now(),
          });
        if (plan.operation !== "copy") src.delete(item.source.key);
        plan.results.push({
          source: item.source,
          target: item.target,
          status: "completed",
        });
      }
      plan.status = plan.results.some((r) => r.error) ? "partial" : "completed";
      return core.OperationPlan.createFrom({ ...plan });
    },
    async GetPlan(pid) {
      const plan = plans.get(pid);
      if (!plan) return fail("not_found", "Plan not found");
      return core.OperationPlan.createFrom({ ...plan, results: [...plan.results] });
    },
    async PickFiles(directory) {
      await sleep(200);
      return directory
        ? ["/Users/demo/Desktop/screenshots"]
        : ["/Users/demo/Desktop/hero.png", "/Users/demo/Desktop/notes.txt"];
    },
    async PickDirectory() {
      await sleep(200);
      return "/Users/demo/Downloads";
    },
    async PickDownload(name) {
      await sleep(200);
      return `/Users/demo/Downloads/${name}`;
    },
    async CopyText(text) {
      await navigator.clipboard?.writeText(text).catch(() => {});
    },
    async Settings() {
      await sleep(60);
      return service.Settings.createFrom({ ...settings });
    },
    async SaveSettings(input) {
      await sleep(200);
      const next = service.Settings.createFrom({ ...input });
      if (/\s/.test(next.cloudflareClientId))
        return fail("invalid_argument", "The Cloudflare OAuth client ID cannot contain spaces");
      if (!/^http:\/\/(127\.0\.0\.1|localhost|\[::1\]):\d+\//.test(next.cloudflareRedirectUri))
        return fail(
          "invalid_argument",
          "The OAuth redirect URI must be a loopback HTTP address such as http://127.0.0.1:8976/oauth/callback",
        );
      settings = next;
      return service.Settings.createFrom({ ...settings });
    },
    async StartCloudflareSignIn() {
      await sleep(150);
      if (!settings.cloudflareClientId)
        return fail(
          "configuration",
          "Add the Cloudflare OAuth client ID in settings before signing in",
        );
      const sessionId = id();
      const authUrl = `https://dash.cloudflare.com/oauth2/auth?response_type=code&client_id=${encodeURIComponent(settings.cloudflareClientId)}&redirect_uri=${encodeURIComponent(settings.cloudflareRedirectUri)}&state=${id()}&code_challenge=mock&code_challenge_method=S256`;
      const timer = window.setTimeout(() => {
        const session = cloudflareSessions.get(sessionId);
        if (!session) return;
        session.accountId = profiles[0].accountId;
        emitCloudflare(
          service.CloudflareSignIn.createFrom({
            sessionId,
            state: "authorized",
            authUrl,
            redirectUri: settings.cloudflareRedirectUri,
            accounts: [
              { id: profiles[0].accountId, name: "Acme Inc" },
              { id: profiles[1].accountId, name: "Acme Labs" },
            ],
          }),
        );
      }, 1500);
      cloudflareSessions.set(sessionId, { accountId: "", timer });
      return service.CloudflareSignIn.createFrom({
        sessionId,
        state: "waiting",
        authUrl,
        redirectUri: settings.cloudflareRedirectUri,
        accounts: [],
      });
    },
    async CloudflareSignInStatus(sessionId) {
      const session = cloudflareSessions.get(sessionId);
      if (!session) return fail("not_found", "This sign-in attempt is no longer available");
      return service.CloudflareSignIn.createFrom({
        sessionId,
        state: session.accountId ? "authorized" : "waiting",
        authUrl: "",
        redirectUri: settings.cloudflareRedirectUri,
        accounts: session.accountId ? [{ id: session.accountId, name: "Acme Inc" }] : [],
      });
    },
    async CancelCloudflareSignIn(sessionId) {
      const session = cloudflareSessions.get(sessionId);
      if (!session) return;
      if (session.timer) window.clearTimeout(session.timer);
      cloudflareSessions.delete(sessionId);
      emitCloudflare(
        service.CloudflareSignIn.createFrom({
          sessionId,
          state: "failed",
          authUrl: "",
          redirectUri: settings.cloudflareRedirectUri,
          accounts: [],
          error: { code: "cancelled", message: "Operation cancelled" },
        }),
      );
    },
    async CloudflareBuckets(sessionId, accountId) {
      await sleep(600);
      if (!cloudflareSessions.has(sessionId))
        return fail("not_found", "Start the Cloudflare sign-in again");
      if (!accountId) return fail("invalid_argument", "Select a Cloudflare account first");
      return ["media-cdn", "assets", "builds"].map((name) =>
        core.Bucket.createFrom({ name, createdAt: new Date(now - 30 * day).toISOString() }),
      );
    },
    async ConnectCloudflare(input) {
      await sleep(1200);
      const session = cloudflareSessions.get(input.sessionId);
      if (!session) return fail("not_found", "Start the Cloudflare sign-in again");
      if (!input.bucket) return fail("invalid_argument", "Select a bucket");
      cloudflareSessions.delete(input.sessionId);
      const p = core.Profile.createFrom({
        id: id(),
        name: input.name,
        providerType: "r2",
        accountId: input.accountId,
        jurisdiction: input.jurisdiction || "default",
        bucket: input.bucket,
        publicUrl: input.publicUrl,
        readOnly: input.readOnly,
        credentialSource: "cloudflare",
        createdAt: new Date().toISOString(),
        updatedAt: new Date().toISOString(),
      });
      if (!store.has(`${p.id}/${p.bucket}`)) store.set(`${p.id}/${p.bucket}`, new Map());
      profiles.push(p);
      authorized.add(p.id);
      return p;
    },
    async CloudflareStatus(profileId) {
      await sleep(80);
      const p = profile(profileId);
      if (p?.credentialSource !== "cloudflare")
        return service.CloudflareStatus.createFrom({ authorized: false });
      return service.CloudflareStatus.createFrom({
        authorized: authorized.has(profileId),
        tokenName: `Bucky ${p.name} ${p.bucket}`,
        authorizedAt: p.updatedAt,
      });
    },
    async ReauthorizeCloudflare(profileId) {
      await sleep(900);
      const p = profile(profileId);
      if (!p) return fail("not_found", "The connection does not exist");
      if (p.credentialSource !== "cloudflare")
        return fail("unsupported", "This connection does not use Cloudflare sign-in");
      authorized.add(profileId);
      return core.Profile.createFrom({ ...p });
    },
    async DisconnectCloudflare(profileId, confirmed) {
      await sleep(400);
      if (!confirmed)
        return fail("confirmation_required", "Confirm disconnecting this Cloudflare connection");
      authorized.delete(profileId);
    },
    async OpenExternal() {
      // The mock sign-in completes on its own, so nothing is opened.
    },
  };

  const events: EventSource = {
    onTransferUpdated: (cb) => {
      listeners.transfer.add(cb);
      return () => listeners.transfer.delete(cb);
    },
    onUsageProgress: (cb) => {
      listeners.usage.add(cb);
      return () => listeners.usage.delete(cb);
    },
    onFilesDropped: () => () => {},
    onCloudflareSignIn: (cb) => {
      listeners.cloudflare.add(cb);
      return () => listeners.cloudflare.delete(cb);
    },
  };
  return { backend, events };
}
