import {
  ArrowRightLeftIcon,
  DownloadIcon,
  Share2Icon,
  SquareArrowOutUpRightIcon,
  Trash2Icon,
  XIcon,
} from "lucide-react";
import type * as React from "react";
import { useTranslation } from "react-i18next";
import { downloadEntries } from "@/lib/actions";
import type { Entry } from "@/lib/entries";
import { type Location, useNav } from "@/lib/nav";
import { cn } from "@/lib/utils";

export function SelectionBar({
  location,
  selected,
  readOnly,
}: {
  location: Location;
  selected: Entry[];
  readOnly: boolean;
}) {
  const { t } = useTranslation();
  const select = useNav((s) => s.select);
  const openShare = useNav((s) => s.openShare);
  const openPlan = useNav((s) => s.openPlan);
  if (selected.length === 0) return null;

  const icon = "size-3.5 flex-none";

  return (
    <div className="pointer-events-none absolute inset-x-0 bottom-4 z-30 flex justify-center px-4">
      <div className="pointer-events-auto flex h-10 items-center gap-0.5 rounded-xl bg-hud pl-3.5 pr-1.5 text-white shadow-[0_10px_30px_rgb(0_0_0/0.28)] backdrop-blur-xl animate-fade">
        <span className="pr-1 text-[12px] font-medium whitespace-nowrap">
          {t("selection.summary", { count: selected.length })}
        </span>
        <span className="mx-1 h-4 w-px bg-white/16" />
        <BarButton
          label={t("menu.download")}
          onClick={() => void downloadEntries(location, selected)}
        >
          <DownloadIcon className={icon} />
        </BarButton>
        <BarButton
          label={t("menu.share")}
          onClick={() => openShare({ location, entries: selected })}
        >
          <Share2Icon className={icon} />
        </BarButton>
        {readOnly ? null : (
          <>
            <span className="mx-1 h-4 w-px bg-white/16" />
            <BarButton
              label={t("selection.copy")}
              onClick={() => openPlan({ operation: "copy", location, entries: selected })}
            >
              <SquareArrowOutUpRightIcon className={icon} />
            </BarButton>
            <BarButton
              label={t("selection.move")}
              onClick={() => openPlan({ operation: "move", location, entries: selected })}
            >
              <ArrowRightLeftIcon className={icon} />
            </BarButton>
            <BarButton
              label={t("common.delete")}
              destructive
              onClick={() => openPlan({ operation: "delete", location, entries: selected })}
            >
              <Trash2Icon className={icon} />
            </BarButton>
          </>
        )}
        <span className="mx-1 h-4 w-px bg-white/16" />
        <BarButton label={t("selection.clear")} onClick={() => select([], null)}>
          <XIcon className={icon} />
        </BarButton>
      </div>
    </div>
  );
}

function BarButton({
  label,
  onClick,
  destructive,
  children,
}: {
  label: string;
  onClick: () => void;
  destructive?: boolean;
  children: React.ReactNode;
}) {
  return (
    <button
      type="button"
      onClick={onClick}
      title={label}
      className={cn(
        "flex h-7 items-center gap-1.5 rounded-md px-2 text-[12px] text-white/85 transition-colors hover:bg-white/14 hover:text-white",
        destructive && "hover:bg-destructive hover:text-white",
      )}
    >
      {children}
      <span>{label}</span>
    </button>
  );
}
