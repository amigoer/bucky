import { create } from "zustand";
import { persist } from "zustand/middleware";

type Theme = "light" | "dark" | "system";
type Language = "en" | "zh";
type View = "list" | "grid";

interface UIState {
  theme: Theme;
  language: Language;
  view: View;
  rate: number;
  transfersOpen: boolean;
  setTheme: (theme: Theme) => void;
  setLanguage: (language: Language) => void;
  setView: (view: View) => void;
  setRate: (rate: number) => void;
  setTransfersOpen: (open: boolean) => void;
}

export const useUI = create<UIState>()(
  persist(
    (set) => ({
      theme: "system",
      language:
        typeof navigator !== "undefined" && navigator.language.startsWith("zh") ? "zh" : "en",
      view: "list",
      rate: 0,
      transfersOpen: false,
      setTheme: (theme) => set({ theme }),
      setLanguage: (language) => set({ language }),
      setView: (view) => set({ view }),
      setRate: (rate) => set({ rate }),
      setTransfersOpen: (transfersOpen) => set({ transfersOpen }),
    }),
    {
      name: "bucky.ui.v1",
      partialize: ({ theme, language, view, rate }) => ({ theme, language, view, rate }),
    },
  ),
);
