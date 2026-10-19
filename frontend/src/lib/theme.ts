import type { Accent } from "./store";

/** OKLCH hue and chroma per accent; style.css derives the primary tokens from them. */
export const ACCENTS: Record<Accent, { hue: number; chroma: number }> = {
  blue: { hue: 255, chroma: 0.165 },
  indigo: { hue: 275, chroma: 0.18 },
  purple: { hue: 305, chroma: 0.17 },
  pink: { hue: 355, chroma: 0.17 },
  orange: { hue: 48, chroma: 0.15 },
  green: { hue: 152, chroma: 0.13 },
  graphite: { hue: 260, chroma: 0.012 },
};

const accentOf = (accent: Accent) => ACCENTS[accent] ?? ACCENTS.blue;

/** Matches --primary in style.css for the given scheme. */
export function accentColor(accent: Accent, dark = false): string {
  const { hue, chroma } = accentOf(accent);
  return `oklch(${dark ? 0.63 : 0.56} ${chroma} ${hue})`;
}

export function applyAppearance(dark: boolean, accent: Accent) {
  const root = document.documentElement;
  const { hue, chroma } = accentOf(accent);
  root.classList.toggle("dark", dark);
  root.style.colorScheme = dark ? "dark" : "light";
  root.style.setProperty("--brand-hue", String(hue));
  root.style.setProperty("--brand-chroma", String(chroma));
}
