import {
  ArrowDownToLineIcon,
  ArrowDownUpIcon,
  ArrowUpFromLineIcon,
  PauseIcon,
  PlayIcon,
  XIcon,
} from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { EmptyState } from "@/components/empty-state";
import { PageHeader, PageTitle } from "@/components/page-header";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Progress } from "@/components/ui/progress";
import { Segmented } from "@/components/ui/segmented";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { api, type TransferTask } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { basename, formatBytes, formatDuration, progress } from "@/lib/format";
import { isActiveTask, keys, useTasks } from "@/lib/queries";
import { queryClient } from "@/lib/query";
import { usePrefs } from "@/lib/store";
import { cn } from "@/lib/utils";

type Filter = "all" | "active" | "done" | "attention";
const RATES = [0, 1, 2, 5, 10, 20, 50];

const PRIORITY: Record<string, number> = {
  running: 0,
  cancelling: 1,
  queued: 2,
  paused: 3,
  failed: 4,
  completed: 5,
  cancelled: 6,
};

export function TransfersPage() {
  const { t, i18n } = useTranslation();
  const { data: tasks = [], isPending } = useTasks();
  const rate = usePrefs((s) => s.rate);
  const concurrency = usePrefs((s) => s.concurrency);
  const set = usePrefs((s) => s.set);
  const [filter, setFilter] = useState<Filter>("all");
  const [cancelling, setCancelling] = useState<TransferTask | null>(null);

  const running = tasks.filter((task) => task.status === "running");
  const queued = tasks.filter((task) => task.status === "queued");
  const speed = running.reduce((sum, task) => sum + task.bytesPerSecond, 0);

  const filtered = useMemo(() => {
    const keep = (task: TransferTask) =>
      filter === "all"
        ? true
        : filter === "active"
          ? isActiveTask(task)
          : filter === "done"
            ? task.status === "completed"
            : task.status === "failed" || task.status === "paused";
    return tasks
      .filter(keep)
      .sort(
        (a, b) =>
          (PRIORITY[a.status] ?? 9) - (PRIORITY[b.status] ?? 9) ||
          b.createdAt.localeCompare(a.createdAt),
      );
  }, [tasks, filter]);

  const act = async (action: () => Promise<unknown>) => {
    try {
      await action();
      await queryClient.invalidateQueries({ queryKey: keys.tasks });
    } catch (error) {
      toast.error(errorMessage(error));
    }
  };

  const changeConcurrency = (value: string) => {
    const next = Number(value);
    set({ concurrency: next });
    void act(() => api.SetConcurrency(next));
  };

  const changeRate = (value: string) => {
    const next = Number(value) * 1024 * 1024;
    set({ rate: Number(value) });
    void act(() => api.SetRate(next));
  };

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader>
        <PageTitle>{t("transfers.title")}</PageTitle>
      </PageHeader>
      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="mx-auto flex w-full max-w-[920px] flex-col gap-4 px-6 py-5">
          <div className="flex flex-wrap items-center gap-x-3 gap-y-1 text-[12px] text-muted-foreground">
            <span>
              {t("transfers.summary", {
                running: running.length,
                queued: queued.length,
                speed: formatBytes(speed, i18n.language),
              })}
            </span>
            <span className="ml-auto flex items-center gap-2">
              <span className="text-[11px] text-faint">{t("transfers.concurrency")}</span>
              <Select value={String(concurrency)} onValueChange={changeConcurrency}>
                <SelectTrigger className="h-7 w-[64px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {[1, 2, 3, 4, 5, 6, 7, 8].map((value) => (
                    <SelectItem key={value} value={String(value)}>
                      {value}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
              <span className="text-[11px] text-faint">{t("transfers.limit")}</span>
              <Select value={String(rate)} onValueChange={changeRate}>
                <SelectTrigger className="h-7 w-[112px]">
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {RATES.map((value) => (
                    <SelectItem key={value} value={String(value)}>
                      {value === 0
                        ? t("transfers.unlimited")
                        : `${value} ${t("transfers.perSecond")}`}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </span>
          </div>

          <Segmented<Filter>
            value={filter}
            onValueChange={setFilter}
            className="w-[340px]"
            options={[
              { value: "all", label: t("transfers.all") },
              { value: "active", label: t("transfers.active") },
              { value: "done", label: t("transfers.done") },
              { value: "attention", label: t("transfers.attention") },
            ]}
          />

          {isPending ? (
            <div className="grid h-40 place-items-center">
              <Spinner className="size-5" />
            </div>
          ) : filtered.length === 0 ? (
            <div className="grid place-items-center py-16">
              <EmptyState
                icon={<ArrowDownUpIcon />}
                title={tasks.length === 0 ? t("transfers.empty") : t("transfers.emptyFiltered")}
              />
            </div>
          ) : (
            <div className="overflow-hidden rounded-xl bg-card shadow-card">
              {filtered.map((task) => (
                <TaskRow
                  key={task.id}
                  task={task}
                  onPause={() => void act(() => api.PauseTransfer(task.id))}
                  onResume={() => void act(() => api.ResumeTransfer(task.id))}
                  onCancel={() => setCancelling(task)}
                />
              ))}
            </div>
          )}
        </div>
      </div>

      <AlertDialog
        open={!!cancelling}
        onOpenChange={(open) => (!open ? setCancelling(null) : null)}
      >
        <AlertDialogContent>
          <AlertDialogTitle>{t("transfers.cancel")}</AlertDialogTitle>
          <AlertDialogDescription>{t("transfers.cancelConfirm")}</AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel asChild>
              <Button variant="secondary" size="sm">
                {t("common.cancel")}
              </Button>
            </AlertDialogCancel>
            <AlertDialogAction asChild>
              <Button
                variant="destructive"
                size="sm"
                onClick={() => {
                  const task = cancelling;
                  setCancelling(null);
                  if (task) void act(() => api.CancelTransfer(task.id));
                }}
              >
                {t("transfers.cancel")}
              </Button>
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function TaskRow({
  task,
  onPause,
  onResume,
  onCancel,
}: {
  task: TransferTask;
  onPause: () => void;
  onResume: () => void;
  onCancel: () => void;
}) {
  const { t, i18n } = useTranslation();
  const percent = progress(task.completed, task.size);
  const active = task.status === "running";
  const remaining =
    active && task.bytesPerSecond > 0
      ? (task.size - task.completed) / task.bytesPerSecond
      : Number.NaN;
  const parts =
    task.partSize > 0 && task.size > 0
      ? {
          total: Math.ceil(task.size / task.partSize),
          done: Math.min(
            Math.ceil(task.size / task.partSize),
            Math.floor(task.completed / task.partSize),
          ),
        }
      : null;
  const tone =
    task.status === "completed"
      ? "success"
      : task.status === "failed"
        ? "destructive"
        : task.status === "paused" || task.status === "cancelling"
          ? "warning"
          : task.status === "running"
            ? "default"
            : "secondary";
  const Icon = task.direction === "upload" ? ArrowUpFromLineIcon : ArrowDownToLineIcon;

  return (
    <div className="group flex items-center gap-3 border-b px-3.5 py-2.5 last:border-b-0 hover:bg-hover">
      <span
        className={cn(
          "grid size-7 flex-none place-items-center rounded-lg",
          task.direction === "upload"
            ? "bg-primary-soft text-primary-text"
            : "bg-muted text-muted-foreground",
        )}
      >
        <Icon className="size-3.75" />
      </span>
      <div className="flex min-w-0 flex-1 flex-col gap-1">
        <div className="flex min-w-0 items-center gap-2">
          <span className="truncate text-[12.5px] font-medium" title={task.localPath}>
            {basename(task.object.key)}
          </span>
          <Badge variant={tone} className="flex-none">
            {t(`transfers.status.${task.status}` as "transfers.status.queued")}
          </Badge>
          {parts ? (
            <span className="flex-none text-[10.5px] text-faint">
              {t("transfers.parts", { done: parts.done, total: parts.total })}
            </span>
          ) : null}
          <span className="ml-auto flex-none tabular text-[11px] text-muted-foreground">
            {formatBytes(task.completed, i18n.language)}
            {task.size > 0 ? ` / ${formatBytes(task.size, i18n.language)}` : ""}
          </span>
        </div>
        <Progress
          value={percent}
          striped={task.status === "running"}
          className="h-1"
          indicatorClassName={cn(
            task.status === "failed" && "bg-destructive",
            task.status === "paused" && "bg-warning",
            task.status === "completed" && "bg-success",
          )}
        />
      </div>
      <div className="flex w-[120px] flex-none flex-col items-end gap-0.5 text-[10.5px] text-faint">
        <span className="tabular text-muted-foreground">
          {active && task.bytesPerSecond > 0
            ? `${formatBytes(task.bytesPerSecond, i18n.language)}/s`
            : `${Math.round(percent)}%`}
        </span>
        {active && Number.isFinite(remaining) ? (
          <span className="tabular">
            {t("transfers.left", { time: formatDuration(remaining) })}
          </span>
        ) : task.status === "queued" ? (
          <span>{t("transfers.status.queued")}</span>
        ) : null}
      </div>
      <div className="flex w-[56px] flex-none items-center justify-end gap-0.5 opacity-0 transition-opacity group-hover:opacity-100">
        {task.status === "running" || task.status === "queued" ? (
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={t("transfers.pause")}
            onClick={onPause}
          >
            <PauseIcon className="size-3.5" />
          </Button>
        ) : task.status === "paused" || task.status === "failed" ? (
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={t("transfers.resume")}
            onClick={onResume}
          >
            <PlayIcon className="size-3.5" />
          </Button>
        ) : null}
        {task.status !== "completed" &&
        task.status !== "cancelled" &&
        task.status !== "cancelling" ? (
          <Button
            variant="ghost"
            size="icon-sm"
            aria-label={t("transfers.cancel")}
            onClick={onCancel}
          >
            <XIcon className="size-3.5" />
          </Button>
        ) : null}
      </div>
    </div>
  );
}
