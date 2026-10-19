import { PanelLeftIcon } from "lucide-react";
import type * as React from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Tip } from "@/components/ui/tooltip";
import { usePrefs } from "@/lib/store";
import { cn } from "@/lib/utils";
import { isMac } from "./platform";
import { WindowControls } from "./window-controls";

export function SidebarToggle() {
  const { t } = useTranslation();
  const hidden = usePrefs((s) => s.sidebarHidden);
  const set = usePrefs((s) => s.set);
  return (
    <Tip label={`${hidden ? t("sidebar.show") : t("sidebar.hide")} ⌃⌘S`}>
      <Button
        variant="ghost"
        size="icon"
        aria-label={hidden ? t("sidebar.show") : t("sidebar.hide")}
        onClick={() => set({ sidebarHidden: !hidden })}
      >
        <PanelLeftIcon className="size-4" />
      </Button>
    </Tip>
  );
}

/** The 52px draggable bar at the top of every page. */
export function PageHeader({
  children,
  className,
}: {
  children?: React.ReactNode;
  className?: string;
}) {
  const hidden = usePrefs((s) => s.sidebarHidden);
  return (
    <header
      className={cn(
        "drag-region relative z-10 flex h-[var(--titlebar-height)] flex-none items-center gap-1.5 border-b bg-background pr-3.5 pl-3",
        hidden && (isMac ? "pl-[82px]" : "pl-4"),
        className,
      )}
    >
      {hidden ? (
        <div className="mr-1 flex items-center animate-fade">
          <SidebarToggle />
        </div>
      ) : null}
      {children}
      {isMac ? null : <WindowControls />}
    </header>
  );
}

export function PageTitle({ children }: { children: React.ReactNode }) {
  return <span className="flex-none whitespace-nowrap text-[15px] font-[650]">{children}</span>;
}
