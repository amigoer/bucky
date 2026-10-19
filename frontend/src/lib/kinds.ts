import { extension } from "./format";

export type Kind =
  | "folder"
  | "image"
  | "video"
  | "audio"
  | "pdf"
  | "text"
  | "code"
  | "json"
  | "archive"
  | "font"
  | "other";

export const KINDS: Kind[] = [
  "folder",
  "image",
  "video",
  "audio",
  "pdf",
  "text",
  "code",
  "json",
  "archive",
  "font",
  "other",
];

const BY_EXTENSION: Record<string, Kind> = {
  png: "image",
  jpg: "image",
  jpeg: "image",
  webp: "image",
  gif: "image",
  avif: "image",
  bmp: "image",
  ico: "image",
  svg: "image",
  heic: "image",
  mp4: "video",
  mov: "video",
  webm: "video",
  m4v: "video",
  mkv: "video",
  avi: "video",
  mp3: "audio",
  wav: "audio",
  m4a: "audio",
  ogg: "audio",
  flac: "audio",
  aac: "audio",
  opus: "audio",
  pdf: "pdf",
  md: "text",
  markdown: "text",
  txt: "text",
  log: "text",
  csv: "text",
  tsv: "text",
  ini: "text",
  conf: "text",
  env: "text",
  ts: "code",
  tsx: "code",
  js: "code",
  jsx: "code",
  mjs: "code",
  cjs: "code",
  css: "code",
  scss: "code",
  html: "code",
  htm: "code",
  xml: "code",
  go: "code",
  rs: "code",
  py: "code",
  rb: "code",
  java: "code",
  kt: "code",
  swift: "code",
  c: "code",
  h: "code",
  cpp: "code",
  sh: "code",
  sql: "code",
  yaml: "code",
  yml: "code",
  toml: "code",
  json: "json",
  jsonl: "json",
  ndjson: "json",
  geojson: "json",
  gz: "archive",
  tgz: "archive",
  zip: "archive",
  tar: "archive",
  rar: "archive",
  "7z": "archive",
  bz2: "archive",
  xz: "archive",
  zst: "archive",
  dmg: "archive",
  pkg: "archive",
  iso: "archive",
  woff: "font",
  woff2: "font",
  ttf: "font",
  otf: "font",
};

export function kindOf(key: string, contentType = "", folder = false): Kind {
  if (folder || key.endsWith("/")) return "folder";
  const byExt = BY_EXTENSION[extension(key)];
  if (byExt) return byExt;
  const mime = contentType.toLowerCase().split(";")[0].trim();
  if (mime.startsWith("image/")) return "image";
  if (mime.startsWith("video/")) return "video";
  if (mime.startsWith("audio/")) return "audio";
  if (mime === "application/pdf") return "pdf";
  if (mime.includes("json")) return "json";
  if (mime.startsWith("font/")) return "font";
  if (mime.startsWith("text/")) return "text";
  if (/(zip|gzip|tar|compressed|x-7z|x-rar)/.test(mime)) return "archive";
  return "other";
}

/** Kind for an extension key from a usage scan, such as ".png" or "(none)". */
export function kindOfExtension(ext: string): Kind {
  if (!ext.startsWith(".")) return "other";
  return BY_EXTENSION[ext.slice(1)] ?? "other";
}

export type PreviewMode = "image" | "video" | "audio" | "pdf" | "text" | "none";

export function previewMode(kind: Kind): PreviewMode {
  switch (kind) {
    case "image":
    case "video":
    case "audio":
    case "pdf":
      return kind;
    case "text":
    case "code":
    case "json":
      return "text";
    default:
      return "none";
  }
}

export const tileStyle = (kind: Kind) => ({
  background: `var(--kind-${kind}-bg)`,
  color: `var(--kind-${kind}-fg)`,
});

export const CHART_COLOR: Record<Kind, string> = {
  image: "oklch(0.62 0.13 42)",
  video: "oklch(0.6 0.13 320)",
  audio: "oklch(0.62 0.1 200)",
  pdf: "oklch(0.58 0.15 25)",
  json: "oklch(0.6 0.1 150)",
  text: "oklch(0.62 0.1 250)",
  code: "oklch(0.56 0.1 160)",
  archive: "oklch(0.55 0.03 262)",
  font: "oklch(0.66 0.08 30)",
  other: "oklch(0.72 0.02 262)",
  folder: "var(--folder-stroke)",
};
