export const PATH_TEMPLATES = [
  "images/{yyyy}/{mm}/{hash}.{ext}",
  "images/{yyyy}{mm}{dd}-{name}.{ext}",
  "screenshots/{name}.{ext}",
] as const;

export interface TemplateValues {
  date: Date;
  hash: string;
  name: string;
  ext: string;
}

const pad = (n: number) => String(n).padStart(2, "0");

/** Expands a quick-upload path template; a suffix such as "-1" goes before the extension. */
export function renderTemplate(template: string, values: TemplateValues, suffix = ""): string {
  const { date } = values;
  const safeName = sanitizeName(values.name) || "image";
  let key = template
    .replaceAll("{yyyy}", String(date.getFullYear()))
    .replaceAll("{mm}", pad(date.getMonth() + 1))
    .replaceAll("{dd}", pad(date.getDate()))
    .replaceAll("{hh}", pad(date.getHours()))
    .replaceAll("{hash}", values.hash.slice(0, 8))
    .replaceAll("{name}", safeName)
    .replaceAll("{ext}", values.ext.toLowerCase());
  if (suffix) {
    const slash = key.lastIndexOf("/");
    const dot = key.lastIndexOf(".");
    key = dot > slash ? `${key.slice(0, dot)}${suffix}${key.slice(dot)}` : `${key}${suffix}`;
  }
  return key.replace(/^\/+/, "").replace(/\/{2,}/g, "/");
}

export function sanitizeName(name: string): string {
  const base = name.replace(/\.[^.]*$/, "");
  return base
    .trim()
    .replace(/[\s/\\?#%*:|"<>]+/g, "-")
    .replace(/-+/g, "-")
    .replace(/^-|-$/g, "")
    .slice(0, 80);
}

export function markdownImage(name: string, url: string): string {
  const alt = name.replace(/[[\]]/g, "");
  return `![${alt}](${url})`;
}
