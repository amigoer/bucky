import { create } from "zustand";
import { persist } from "zustand/middleware";
import type { SortKey } from "./entries";

export type Theme = "light" | "dark" | "system";
export type Accent = "blue" | "indigo" | "purple" | "pink" | "orange" | "green" | "graphite";
export type Language = "en" | "zh";
export type View = "list" | "grid" | "columns";
export type CopyFormat = "markdown" | "url";

interface Prefs {
  theme: Theme;
  accent: Accent;
  language: Language;
  view: View;
  sortKey: SortKey;
  sortDir: 1 | -1;
  /** MiB/s; 0 means unlimited. */
  rate: number;
  concurrency: number;
  sidebarHidden: boolean;
  collapsed: Record<string, boolean>;
  quick: {
    profileId: string;
    bucket: string;
    template: number;
    compress: boolean;
    webp: boolean;
    format: CopyFormat;
  };
  set: (patch: Partial<Omit<Prefs, "set" | "setQuick">>) => void;
  setQuick: (patch: Partial<Prefs["quick"]>) => void;
}

const systemLanguage = (): Language =>
  typeof navigator !== "undefined" && navigator.language.toLowerCase().startsWith("zh")
    ? "zh"
    : "en";

export const usePrefs = create<Prefs>()(
  persist(
    (set) => ({
      theme: "system",
      accent: "blue",
      language: systemLanguage(),
      view: "list",
      sortKey: "name",
      sortDir: 1,
      rate: 0,
      concurrency: 2,
      sidebarHidden: false,
      collapsed: {},
      quick: {
        profileId: "",
        bucket: "",
        template: 0,
        compress: true,
        webp: true,
        format: "markdown",
      },
      set: (patch) => set(patch),
      setQuick: (patch) => set((s) => ({ quick: { ...s.quick, ...patch } })),
    }),
    {
      name: "bucky.ui.v1",
      version: 2,
      migrate: (persisted, version) => {
        const old = (persisted ?? {}) as Partial<Prefs>;
        if (version < 2) {
          return {
            theme: old.theme,
            language: old.language,
            rate: old.rate,
            view: old.view,
          } as Prefs;
        }
        return old as Prefs;
      },
      partialize: ({ set: _set, setQuick: _setQuick, ...rest }) => rest,
    },
  ),
);
