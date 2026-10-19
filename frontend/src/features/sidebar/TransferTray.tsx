import { ArrowDownUpIcon, CheckIcon, ChevronRightIcon, TriangleAlertIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { formatBytes, formatDuration } from "@/lib/format";
import { useNav } from "@/lib/nav";
import { isActiveTask, useTasks } from "@/lib/queries";
import { cn } from "@/lib/utils";

export function TransferTray() {
  const { t, i18n } = useTranslation();
  const { data: tasks = [] } = useTasks();
  const setPage = useNav((s) => s.setPage);
  const page = useNav((s) => s.page);

  const active = tasks.filter((x) => isActiveTask(x) || x.status === "paused");
  const moving = tasks.filter(isActiveTask);
  const attention = tasks.filter((x) => x.status === "failed" || x.status === "paused");
  const speed = moving.reduce((sum, x) => sum + (x.bytesPerSecond || 0), 0);
  const size = active.reduce((sum, x) => sum + x.size, 0);
  const done = active.reduce((sum, x) => sum + Math.min(x.completed, x.size), 0);
  const ratio = size > 0 ? done / size : 1;

  let title: string;
  let sub: string;
  let tone: "accent" | "warn" | "ok";
  if (moving.length > 0) {
    title = t("tray.active", { count: moving.length });
    sub = t("tray.activeSub", {
      speed: formatBytes(speed, i18n.language),
      eta: formatDuration(speed > 0 ? (size - done) / speed : Number.NaN),
    });
    tone = "accent";
  } else if (attention.length > 0) {
    title = t("tray.attention", { count: attention.length });
    sub = t("tray.attentionSub");
    tone = "warn";
  } else {
    title = t("tray.idle");
    sub = t("tray.idleSub");
    tone = "ok";
  }
  const percent = tone === "accent" ? ratio : 1;
  const ring =
    tone === "accent" ? "var(--primary)" : tone === "warn" ? "var(--warning)" : "var(--success)";
  const Icon =
    tone === "accent" ? ArrowDownUpIcon : tone === "warn" ? TriangleAlertIcon : CheckIcon;

  return (
    <button
      type="button"
      onClick={() => setPage("transfers")}
      className={cn(
        "flex w-full flex-col gap-1.75 rounded-xl bg-background px-2.5 py-2.25 text-left shadow-card transition-[box-shadow,transform] duration-150 hover:shadow-card-hover active:scale-[0.985]",
        page === "transfers" && "shadow-[0_0_0_1.5px_var(--primary)]",
      )}
    >
      <span className="flex items-center gap-2.5">
        <span
          className="relative grid size-7.5 flex-none place-items-center"
          style={{
            color:
              tone === "accent"
                ? "var(--primary-text)"
                : tone === "warn"
                  ? "var(--warning-text)"
                  : "var(--success-text)",
          }}
        >
          <span
            className="absolute inset-0 rounded-full transition-[background] duration-500"
            style={{
              background: `conic-gradient(${ring} 0 ${percent * 360}deg, var(--muted) ${percent * 360}deg)`,
              mask: "radial-gradient(circle, transparent 11.5px, #000 12px)",
              WebkitMask: "radial-gradient(circle, transparent 11.5px, #000 12px)",
            }}
          />
          <Icon className="size-3" strokeWidth={2.4} />
        </span>
        <span className="flex min-w-0 flex-1 flex-col gap-px">
          <span className="truncate text-[12.5px] font-semibold">{title}</span>
          <span className="truncate text-[11px] tabular text-muted-foreground">{sub}</span>
        </span>
        <ChevronRightIcon className="size-3 flex-none text-faint" strokeWidth={3} />
      </span>
      <span className="h-0.75 overflow-hidden rounded-full bg-muted">
        <span
          className="block h-full rounded-full transition-[width] duration-500 ease-linear"
          style={{ width: `${percent * 100}%`, background: ring }}
        />
      </span>
    </button>
  );
}
