import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { Brand } from "@/components/brand";
import { SidebarToggle } from "@/components/page-header";
import { isMac } from "@/components/platform";
import { Toaster } from "@/components/ui/sonner";
import { Spinner } from "@/components/ui/spinner";
import { TooltipProvider } from "@/components/ui/tooltip";
import { Browser } from "@/features/browser/Browser";
import { ConnectionDialog } from "@/features/connection/ConnectionDialog";
import { PlanSheet } from "@/features/plan/PlanSheet";
import { QuickUpload } from "@/features/quick-upload/QuickUpload";
import { QuickLook } from "@/features/quicklook/QuickLook";
import { SettingsPage } from "@/features/settings/SettingsPage";
import { ShareDialog } from "@/features/share/ShareDialog";
import { Sidebar } from "@/features/sidebar/Sidebar";
import { TransfersPage } from "@/features/transfers/TransfersPage";
import { UsagePage } from "@/features/usage/UsagePage";
import { Welcome } from "@/features/welcome/Welcome";
import { api, backendKind, backendReady, isDesktop } from "@/lib/api";
import { useEvent, useMediaQuery } from "@/lib/hooks";
import i18n from "@/lib/i18n";
import { useNav } from "@/lib/nav";
import { useTransferEvents } from "@/lib/queries";
import { usePrefs } from "@/lib/store";
import { applyAppearance } from "@/lib/theme";
import { cn } from "@/lib/utils";

export default function App() {
  const { t } = useTranslation();
  const [ready, setReady] = useState(false);
  const [kind, setKind] = useState<"desktop" | "mock" | "none">(isDesktop() ? "desktop" : "none");
  const page = useNav((s) => s.page);
  const location = useNav((s) => s.location);
  const theme = usePrefs((s) => s.theme);
  const accent = usePrefs((s) => s.accent);
  const language = usePrefs((s) => s.language);
  const rate = usePrefs((s) => s.rate);
  const concurrency = usePrefs((s) => s.concurrency);
  const systemDark = useMediaQuery("(prefers-color-scheme: dark)");

  useEffect(() => {
    let alive = true;
    void backendReady.then(() => {
      if (!alive) return;
      setKind(backendKind());
      setReady(true);
    });
    return () => {
      alive = false;
    };
  }, []);

  const dark = theme === "dark" || (theme === "system" && systemDark);
  useEffect(() => applyAppearance(dark, accent), [dark, accent]);

  useEffect(() => {
    void i18n.changeLanguage(language);
    document.documentElement.lang = language === "zh" ? "zh-CN" : "en";
  }, [language]);

  useEffect(() => {
    if (!ready || kind === "none") return;
    void api.SetRate(rate * 1024 * 1024).catch(() => {});
    void api.SetConcurrency(concurrency).catch(() => {});
  }, [ready, kind, rate, concurrency]);

  const onKeyDown = useEvent((event: KeyboardEvent) => {
    if (event.isComposing) return;
    const nav = useNav.getState();
    const overlay = nav.quickLook || nav.share || nav.plan || nav.connection || nav.quickUpload;
    const mod = event.metaKey || event.ctrlKey;
    const key = event.key.toLowerCase();
    const prefs = usePrefs.getState();

    if (mod && !event.altKey && (key === "f" || key === "k") && nav.location && !overlay) {
      event.preventDefault();
      nav.setPage("browse");
      nav.setSearchOpen(true);
      return;
    }
    if (mod && key === "," && !overlay) {
      event.preventDefault();
      nav.setPage("settings");
      return;
    }
    if (mod && event.shiftKey && key === "t" && !overlay) {
      event.preventDefault();
      nav.setPage("transfers");
      return;
    }
    if (
      ((isMac && event.ctrlKey && event.metaKey && key === "s") ||
        (!isMac && event.ctrlKey && event.shiftKey && key === "s")) &&
      !overlay
    ) {
      event.preventDefault();
      prefs.set({ sidebarHidden: !prefs.sidebarHidden });
      return;
    }
    if (
      ((isMac && event.metaKey && event.altKey) || (!isMac && event.ctrlKey && event.altKey)) &&
      key === "u" &&
      !overlay
    ) {
      event.preventDefault();
      nav.setPage("browse");
      nav.openQuickUpload({});
      return;
    }
    if (mod && !overlay && nav.page === "browse") {
      if (event.key === "[") {
        event.preventDefault();
        nav.goBack();
      } else if (event.key === "]") {
        event.preventDefault();
        nav.goForward();
      }
    }
  });

  useEffect(() => {
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onKeyDown]);

  useEffect(() => {
    const onPaste = (event: ClipboardEvent) => {
      const nav = useNav.getState();
      if (nav.quickUpload || nav.plan || nav.share || nav.connection || nav.quickLook) return;
      if (nav.page !== "browse") return;
      const file = [...(event.clipboardData?.items ?? [])]
        .find((item) => item.type.startsWith("image/"))
        ?.getAsFile();
      if (!file) return;
      event.preventDefault();
      nav.openQuickUpload({ file });
    };
    document.addEventListener("paste", onPaste);
    return () => document.removeEventListener("paste", onPaste);
  }, []);

  if (!ready) {
    return (
      <div className="grid h-full place-items-center bg-background">
        <Spinner className="size-5 text-muted-foreground" />
      </div>
    );
  }

  if (kind === "none") {
    return (
      <div className="grid h-full place-items-center bg-background px-8">
        <div className="flex max-w-[420px] flex-col items-center gap-2.5 text-center">
          <Brand className="mb-2 scale-125" />
          <h1 className="text-[16px] font-semibold">{t("app.desktopTitle")}</h1>
          <p className="text-[12.5px] leading-relaxed text-muted-foreground">
            {t("app.desktopDesc")}
          </p>
        </div>
      </div>
    );
  }

  return (
    <TooltipProvider>
      <div className="relative flex h-full overflow-hidden bg-sidebar">
        <Sidebar />
        <main className="m-2 flex min-w-0 flex-1 flex-col overflow-hidden rounded-2xl border bg-background shadow-panel">
          {page === "browse" ? (
            location ? (
              <Browser />
            ) : (
              <Welcome />
            )
          ) : page === "transfers" ? (
            <TransfersPage />
          ) : page === "usage" ? (
            <UsagePage />
          ) : (
            <SettingsPage />
          )}
        </main>

        {/* Lives outside both headers so it stays put when the sidebar collapses. */}
        <div className={cn("absolute top-3 z-30", isMac ? "left-[82px]" : "left-4")}>
          <SidebarToggle />
        </div>

        <QuickLook />
        <ShareDialog />
        <PlanSheet />
        <ConnectionDialog />
        <QuickUpload />
        <TransferSync />
        <Toaster />
      </div>
    </TooltipProvider>
  );
}

function TransferSync() {
  useTransferEvents();
  return null;
}
