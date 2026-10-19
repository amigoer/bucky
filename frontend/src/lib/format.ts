export function formatBytes(value: number, locale = "en"): string {
  if (!Number.isFinite(value) || value <= 0) return "0 B";
  const units = ["B", "KiB", "MiB", "GiB", "TiB", "PiB"];
  const index = Math.min(Math.floor(Math.log(value) / Math.log(1024)), units.length - 1);
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: index === 0 ? 0 : 1 }).format(value / 1024 ** index)} ${units[index]}`;
}

export function exactBytes(value: number, locale = "en"): string {
  return `${new Intl.NumberFormat(locale, { maximumFractionDigits: 0 }).format(Math.max(0, value))} B`;
}

export function formatDate(value: string, locale = "en"): string {
  const date = new Date(value);
  if (!value || !Number.isFinite(date.getTime()) || date.getUTCFullYear() <= 1) return "—";
  return new Intl.DateTimeFormat(locale, {
    year: "numeric",
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function basename(key: string): string {
  return key.replace(/\/$/, "").split("/").at(-1) || key;
}

export function displayType(key: string, contentType = ""): string {
  const mime = contentType.split(";")[0].trim();
  if (mime) return mime;
  const ext = extension(key);
  return ext ? ext.toUpperCase() : "—";
}

export function extension(key: string): string {
  const name = basename(key);
  const dot = name.lastIndexOf(".");
  return dot > 0 ? name.slice(dot + 1).toLowerCase() : "";
}

export type PreviewKind = "image" | "video" | "audio" | "pdf" | "text" | "file";

export function previewKind(key: string, contentType = ""): PreviewKind {
  const mime = contentType.toLowerCase().split(";")[0];
  if (mime === "image/svg+xml") return "text";
  if (mime.startsWith("image/")) return "image";
  if (mime.startsWith("video/")) return "video";
  if (mime.startsWith("audio/")) return "audio";
  if (mime === "application/pdf") return "pdf";
  if (mime.startsWith("text/") || /(?:json|xml|javascript|yaml)/.test(mime)) return "text";
  const ext = extension(key);
  if (["png", "jpg", "jpeg", "gif", "webp", "avif", "bmp"].includes(ext)) return "image";
  if (["mp4", "webm", "mov", "m4v"].includes(ext)) return "video";
  if (["mp3", "wav", "ogg", "flac", "m4a"].includes(ext)) return "audio";
  if (ext === "pdf") return "pdf";
  if (
    [
      "txt",
      "md",
      "json",
      "jsonl",
      "csv",
      "yaml",
      "yml",
      "xml",
      "svg",
      "html",
      "css",
      "js",
      "ts",
      "tsx",
      "go",
      "rs",
      "log",
      "toml",
      "ini",
      "sh",
    ].includes(ext)
  )
    return "text";
  return "file";
}

export function progress(completed: number, size: number): number {
  if (!Number.isFinite(completed) || !Number.isFinite(size) || size <= 0) return 0;
  return Math.min(100, Math.max(0, (completed / size) * 100));
}

export function breadcrumbs(prefix: string): { name: string; prefix: string }[] {
  const parts = prefix.split("/").filter(Boolean);
  return parts.map((name, index) => ({ name, prefix: `${parts.slice(0, index + 1).join("/")}/` }));
}

/** Seconds as m:ss or h:mm:ss; "—" when unknown. */
export function formatDuration(seconds: number): string {
  if (!Number.isFinite(seconds) || seconds < 0) return "—";
  const total = Math.round(seconds);
  const h = Math.floor(total / 3600);
  const m = Math.floor((total % 3600) / 60);
  const s = String(total % 60).padStart(2, "0");
  return h ? `${h}:${String(m).padStart(2, "0")}:${s}` : `${m}:${s}`;
}

export function formatTime(value: string | number, locale = "en"): string {
  const date = new Date(value);
  if (!Number.isFinite(date.getTime())) return "—";
  return new Intl.DateTimeFormat(locale, {
    month: "short",
    day: "numeric",
    hour: "2-digit",
    minute: "2-digit",
  }).format(date);
}

export function parentPrefix(prefix: string): string {
  const trimmed = prefix.replace(/\/$/, "");
  const slash = trimmed.lastIndexOf("/");
  return slash < 0 ? "" : trimmed.slice(0, slash + 1);
}
