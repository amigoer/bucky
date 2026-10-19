import type { TFunction } from "i18next";
import {
  AlertTriangleIcon,
  ArrowRightLeftIcon,
  CheckIcon,
  ChevronUpIcon,
  ClockIcon,
  FolderIcon,
  LoaderCircleIcon,
  SquareArrowOutUpRightIcon,
  Trash2Icon,
  UploadIcon,
  XIcon,
} from "lucide-react";
import { type ReactNode, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Checkbox } from "@/components/ui/checkbox";
import { Dialog, DialogContent, DialogFooter, DialogHeader } from "@/components/ui/dialog";
import { Popover, PopoverContent, PopoverTrigger } from "@/components/ui/popover";
import { Spinner } from "@/components/ui/spinner";
import { expandEntries, failureMessage } from "@/lib/actions";
import { api, type OperationPlan } from "@/lib/api";
import { type Entry, entriesFromPages, sortEntries } from "@/lib/entries";
import { errorCode, errorMessage } from "@/lib/errors";
import { formatBytes, formatDuration, parentPrefix } from "@/lib/format";
import { useNow } from "@/lib/hooks";
import { type Location, type PlanDraft, useNav } from "@/lib/nav";
import { isFinished, overwriteCount, resultCounts, resultState, secondsLeft } from "@/lib/plan";
import { invalidateBucket, useListing } from "@/lib/queries";
import { cn } from "@/lib/utils";

export function PlanSheet() {
  const draft = useNav((s) => s.plan);
  const key = draft
    ? `${draft.operation}:${draft.plan?.id ?? ""}:${draft.location.prefix}:${draft.entries.map((entry) => entry.key).join("|")}`
    : "";
  return draft ? <PlanSheetInner key={key} draft={draft} /> : null;
}

const ICONS = {
  delete: Trash2Icon,
  copy: SquareArrowOutUpRightIcon,
  move: ArrowRightLeftIcon,
  upload: UploadIcon,
};

function PlanSheetInner({ draft }: { draft: PlanDraft }) {
  const { t, i18n } = useTranslation();
  const closePlan = useNav((s) => s.closePlan);
  const queued = useNav((s) => s.planQueue.length);
  const now = useNow(1000, true);
  const [plan, setPlan] = useState<OperationPlan | null>(draft.plan ?? null);
  const [target, setTarget] = useState("");
  const [creating, setCreating] = useState(false);
  const [executing, setExecuting] = useState(false);
  const [confirmed, setConfirmed] = useState(false);
  const [error, setError] = useState<unknown>(null);
  const started = useRef(false);

  const folder = draft.entries.some((entry) => entry.folder);
  const sameFolder = draft.operation !== "delete" && target === draft.location.prefix;
  const insideFolder =
    draft.operation !== "delete" &&
    draft.entries.some(
      (entry) => entry.folder && (target === entry.key || target.startsWith(entry.key)),
    );
  const invalidTarget = sameFolder || insideFolder;

  const create = async () => {
    setCreating(true);
    setError(null);
    try {
      const refs = await expandEntries(draft.location, draft.entries);
      if (refs.length === 0) {
        setError(new Error(t("common.none")));
        return;
      }
      const next = await api.CreatePlan({
        operation: draft.operation,
        sources: refs,
        target: {
          profileId: draft.location.profileId,
          bucket: draft.location.bucket,
          key: draft.operation === "delete" ? "" : target,
        },
        localPath: "",
        base: draft.operation === "delete" ? "" : draft.location.prefix,
      });
      setPlan(next);
      setConfirmed(false);
    } catch (cause) {
      setError(cause);
      toast.error(failureMessage(cause));
    } finally {
      setCreating(false);
    }
  };

  // biome-ignore lint/correctness/useExhaustiveDependencies: build the delete plan once when the sheet opens
  useEffect(() => {
    if (draft.operation !== "delete" || plan || started.current) return;
    started.current = true;
    // The delete plan is built once when the sheet opens.
    void create();
  }, []);

  const regenerate = async () => {
    setError(null);
    if (draft.operation === "upload" && plan?.items[0]?.target && plan.items[0].localPath) {
      setCreating(true);
      try {
        const next = await api.CreatePlan({
          operation: "upload",
          sources: [],
          target: plan.items[0].target,
          localPath: plan.items[0].localPath,
          base: "",
        });
        setPlan(next);
        setConfirmed(false);
      } catch (cause) {
        setError(cause);
      } finally {
        setCreating(false);
      }
      return;
    }
    setPlan(null);
    setConfirmed(false);
    if (draft.operation === "delete") {
      started.current = true;
      void create();
    } else {
      await create();
    }
  };

  const execute = async () => {
    if (!plan) return;
    setExecuting(true);
    setError(null);
    try {
      const next = await api.ExecutePlan(plan.id, true);
      setPlan(next);
      invalidateBucket(draft.location.profileId, draft.location.bucket);
      if (draft.operation !== "copy") useNav.getState().select([], null);
      if (next.status === "completed") toast.success(t("toast.planDone"));
      else if (next.status === "partial") toast(t("toast.planPartial"));
    } catch (cause) {
      setError(cause);
    } finally {
      setExecuting(false);
    }
  };

  const count = plan ? plan.items.length : draft.entries.length;
  const left = plan ? secondsLeft(plan, now) : 0;
  const expired = !!plan && plan.status === "pending" && left <= 0;
  const overwrites = plan ? overwriteCount(plan) : 0;
  const finished = plan ? isFinished(plan) : false;
  const Icon = ICONS[draft.operation];
  const title =
    draft.operation === "delete"
      ? t("plan.titleDelete", { count })
      : draft.operation === "copy"
        ? t("plan.titleCopy", { count })
        : draft.operation === "move"
          ? t("plan.titleMove", { count })
          : t("plan.titleUpload", { count });
  const destination =
    plan?.operation === "upload" ? (plan.items[0]?.target?.key ?? "") : target || "/";

  return (
    <Dialog
      open
      onOpenChange={(open) => {
        if (!open && !executing) closePlan();
      }}
    >
      <DialogContent placement="sheet" className="w-[520px]">
        <DialogHeader
          icon={<Icon />}
          iconClassName={
            draft.operation === "delete" ? "bg-destructive-soft text-destructive" : undefined
          }
          title={title}
          description={
            draft.operation === "delete" ? undefined : t("plan.to", { place: destination })
          }
          close={!executing}
        >
          {queued > 0 ? (
            <Badge variant="secondary" className="mr-1 flex-none">
              {t("plan.nextPlan", { count: queued })}
            </Badge>
          ) : null}
        </DialogHeader>

        <div className="flex min-h-0 flex-1 flex-col gap-3 overflow-y-auto px-4.5 pb-4 text-[12.5px]">
          {creating ? (
            <div className="flex h-28 flex-col items-center justify-center gap-2.5 text-muted-foreground">
              <Spinner className="size-5" />
              <span>{t("plan.creating")}</span>
            </div>
          ) : error ? (
            <div className="flex flex-col gap-2 rounded-lg bg-destructive-soft px-3 py-2.5 text-destructive">
              <span className="font-semibold">{t("states.errorTitle")}</span>
              <span className="text-[11.5px] leading-relaxed">{describeError(error, t)}</span>
              {plan ? null : (
                <Button
                  variant="secondary"
                  size="sm"
                  className="mt-1 self-start"
                  onClick={() => void regenerate()}
                >
                  {t("common.retry")}
                </Button>
              )}
            </div>
          ) : !plan ? (
            <>
              <div className="grid grid-cols-[70px_minmax(0,1fr)] items-center gap-x-2 gap-y-2">
                <span className="text-faint">{t("plan.source")}</span>
                <span className="truncate">{describeSources(draft.entries, t)}</span>
                {draft.operation === "delete" ? null : (
                  <>
                    <span className="text-faint">{t("plan.destination")}</span>
                    <DestinationPicker
                      location={draft.location}
                      value={target}
                      onChange={setTarget}
                    />
                  </>
                )}
              </div>
              {invalidTarget ? (
                <p className="rounded-md bg-warning-soft px-2.5 py-1.5 text-[11.5px] text-warning-text">
                  {sameFolder ? t("plan.pickSame") : t("plan.pickInvalid")}
                </p>
              ) : null}
            </>
          ) : (
            <>
              <div className="grid grid-cols-3 gap-2">
                <Stat label={t("plan.objects")} value={String(count)} />
                <Stat label={t("plan.size")} value={formatBytes(plan.bytes, i18n.language)} />
                <Stat
                  label={t("plan.overwrite")}
                  value={String(overwrites)}
                  tone={overwrites > 0 ? "warn" : undefined}
                />
              </div>

              {draft.operation !== "delete" || draft.entries.length > 1 ? (
                <div className="grid grid-cols-[70px_minmax(0,1fr)] items-center gap-x-2 gap-y-2">
                  {draft.operation === "upload" ? (
                    <>
                      <span className="text-faint">{t("plan.destination")}</span>
                      <span className="truncate" title={destination}>
                        {destination || "/"}
                      </span>
                    </>
                  ) : (
                    <>
                      <span className="text-faint">{t("plan.source")}</span>
                      <span className="truncate">{describeSources(draft.entries, t)}</span>
                      <span className="text-faint">{t("plan.destination")}</span>
                      <span className="truncate" title={destination}>
                        {destination}
                      </span>
                    </>
                  )}
                </div>
              ) : null}

              <div className="flex flex-wrap items-center gap-2 text-[11px] text-muted-foreground">
                {plan.status === "pending" ? (
                  expired ? (
                    <Badge variant="warning">{t("plan.expiredChip")}</Badge>
                  ) : (
                    <span className="flex items-center gap-1">
                      <ClockIcon className="size-3" />
                      {t("plan.expiresIn", { time: formatDuration(left) })}
                    </span>
                  )
                ) : plan.status === "completed" ? (
                  <Badge variant="success">{t("plan.allDone")}</Badge>
                ) : plan.status === "partial" ? (
                  <Badge variant="warning">
                    {t("plan.partial", {
                      count: resultCounts(plan).fail + resultCounts(plan).conflict,
                    })}
                  </Badge>
                ) : plan.status === "interrupted" ? (
                  <Badge variant="warning">{t("plan.interrupted")}</Badge>
                ) : null}
                {draft.operation === "upload" ? (
                  <span>{t("plan.localFiles", { count: plan.items.length })}</span>
                ) : null}
              </div>

              {expired ? (
                <p className="rounded-md bg-warning-soft px-2.5 py-1.5 text-[11.5px] text-warning-text">
                  {t("plan.expired")}
                </p>
              ) : null}

              {plan.status === "pending" ? (
                <div className="flex flex-col gap-1.5">
                  {draft.operation === "delete" ? (
                    <Warning>{t("plan.warnDelete")}</Warning>
                  ) : draft.operation === "move" ? (
                    <Warning>{t("plan.warnMove")}</Warning>
                  ) : null}
                  {overwrites > 0 && draft.operation !== "delete" ? (
                    <Warning>{t("plan.warnOverwrite", { count: overwrites })}</Warning>
                  ) : null}
                  {folder ? <Warning>{t("plan.warnFolders")}</Warning> : null}
                  <p className="px-0.5 text-[11px] leading-relaxed text-faint">
                    {t("plan.idNote")}
                  </p>
                </div>
              ) : null}

              {finished ? <Results plan={plan} /> : null}
            </>
          )}
        </div>

        <DialogFooter className="justify-between">
          <span className="text-[10.5px] text-faint">{expired ? t("plan.expiredChip") : null}</span>
          <div className="flex items-center gap-2">
            {plan?.status === "pending" && !expired ? (
              <div className="mr-1 flex cursor-default items-center gap-2 text-[11.5px] text-muted-foreground">
                <Checkbox
                  checked={confirmed}
                  onCheckedChange={(value) => setConfirmed(value === true)}
                  disabled={executing}
                />
                <span className="max-w-[260px] leading-tight">{t("plan.confirm")}</span>
              </div>
            ) : null}
            <Button variant="secondary" size="sm" disabled={executing} onClick={closePlan}>
              {finished ? t("common.close") : t("common.cancel")}
            </Button>
            {!plan && !creating ? (
              <Button size="sm" disabled={invalidTarget} onClick={() => void create()}>
                {t("plan.create")}
              </Button>
            ) : plan?.status === "pending" ? (
              expired ? (
                <Button size="sm" onClick={() => void regenerate()}>
                  {t("plan.regenerate")}
                </Button>
              ) : draft.operation === "delete" ? (
                <Button
                  variant="destructive"
                  size="sm"
                  disabled={!confirmed || executing}
                  onClick={() => void execute()}
                >
                  {executing ? <LoaderCircleIcon className="size-3.5 animate-spin" /> : null}
                  {executing
                    ? t("plan.running", { done: 0, total: count })
                    : t("plan.executeDelete", { count })}
                </Button>
              ) : (
                <Button size="sm" disabled={!confirmed || executing} onClick={() => void execute()}>
                  {executing ? <LoaderCircleIcon className="size-3.5 animate-spin" /> : null}
                  {executing ? t("plan.running", { done: 0, total: count }) : t("plan.execute")}
                </Button>
              )
            ) : finished ? (
              <Button size="sm" onClick={closePlan}>
                {t("common.done")}
              </Button>
            ) : null}
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}

function describeSources(entries: Entry[], t: TFunction) {
  if (entries.length === 1) return entries[0].name;
  return t("common.items", { count: entries.length });
}

function describeError(error: unknown, t: TFunction) {
  const code = errorCode(error);
  if (code === "conflict") return t("plan.changed");
  if (code === "expired") return t("plan.expired");
  return failureMessage(error);
}

function Stat({ label, value, tone }: { label: string; value: string; tone?: "warn" }) {
  return (
    <div className="flex flex-col gap-0.5 rounded-lg bg-field px-3 py-2">
      <span className="text-[10.5px] font-medium text-faint">{label}</span>
      <span
        className={cn("tabular text-[15px] font-semibold", tone === "warn" && "text-warning-text")}
      >
        {value}
      </span>
    </div>
  );
}

function Warning({ children }: { children: ReactNode }) {
  return (
    <p className="flex gap-1.5 rounded-md bg-warning-soft px-2.5 py-1.5 text-[11.5px] leading-relaxed text-warning-text">
      <AlertTriangleIcon className="mt-0.5 size-3.5 flex-none" />
      <span>{children}</span>
    </p>
  );
}

function Results({ plan }: { plan: OperationPlan }) {
  const { t } = useTranslation();
  const counts = resultCounts(plan);
  return (
    <div className="flex flex-col gap-2">
      <div className="flex flex-wrap items-center gap-1.5">
        {counts.ok > 0 ? (
          <Badge variant="success">
            <CheckIcon /> {t("plan.ok")} {counts.ok}
          </Badge>
        ) : null}
        {counts.fail > 0 ? (
          <Badge variant="destructive">
            <XIcon /> {t("plan.fail")} {counts.fail}
          </Badge>
        ) : null}
        {counts.skip > 0 ? (
          <Badge variant="secondary">
            {t("plan.skip")} {counts.skip}
          </Badge>
        ) : null}
        {counts.conflict > 0 ? (
          <Badge variant="warning">
            {t("plan.conflict")} {counts.conflict}
          </Badge>
        ) : null}
      </div>
      <div className="max-h-[240px] overflow-y-auto rounded-lg border">
        {(plan.results ?? []).map((result, index) => {
          const state = resultState(result);
          const name = result.target?.key || result.source?.key || String(index + 1);
          return (
            <div
              key={`${result.source.profileId}:${result.source.bucket}:${result.source.key}`}
              className="flex items-start gap-2 border-b px-2.5 py-1.5 text-[11.5px] last:border-b-0"
            >
              <span
                className={cn(
                  "mt-1 size-1.5 flex-none rounded-full",
                  state === "ok" && "bg-success",
                  state === "fail" && "bg-destructive",
                  state === "skip" && "bg-faint",
                  state === "conflict" && "bg-warning",
                  state === "pending" && "bg-muted-foreground",
                )}
              />
              <span className="min-w-0 flex-1 truncate" title={name}>
                {name}
              </span>
              <span className="flex-none text-faint">
                {state === "ok"
                  ? t("plan.ok")
                  : state === "fail"
                    ? t("plan.fail")
                    : state === "skip"
                      ? t("plan.skip")
                      : state === "conflict"
                        ? t("plan.conflict")
                        : t("common.loading")}
              </span>
              {result.error ? (
                <span
                  className="max-w-[180px] truncate text-destructive"
                  title={errorMessage(result.error)}
                >
                  {errorMessage(result.error)}
                </span>
              ) : null}
            </div>
          );
        })}
      </div>
    </div>
  );
}

function DestinationPicker({
  location,
  value,
  onChange,
}: {
  location: Location;
  value: string;
  onChange: (value: string) => void;
}) {
  const { t } = useTranslation();
  const [open, setOpen] = useState(false);
  const [cursor, setCursor] = useState(value);
  useEffect(() => {
    if (open) setCursor(value);
  }, [open, value]);

  return (
    <Popover open={open} onOpenChange={setOpen}>
      <PopoverTrigger asChild>
        <Button variant="secondary" size="sm" className="w-fit max-w-full justify-start">
          <FolderIcon className="size-3.5" />
          <span className="truncate">{value || "/"}</span>
        </Button>
      </PopoverTrigger>
      <PopoverContent align="start" className="w-[320px] p-0">
        <div className="flex h-9 items-center gap-1 border-b px-2">
          <Button
            variant="ghost"
            size="icon-sm"
            disabled={cursor === ""}
            onClick={() => setCursor(parentPrefix(cursor))}
            aria-label={t("common.back")}
          >
            <ChevronUpIcon className="size-3.5" />
          </Button>
          <span className="min-w-0 flex-1 truncate text-[12px]" title={cursor || "/"}>
            {cursor || "/"}
          </span>
        </div>
        <DestinationFolders location={location} cursor={cursor} onOpen={setCursor} />
        <div className="flex items-center justify-between gap-2 border-t px-2 py-1.5">
          <span className="truncate text-[11px] text-faint">{t("plan.pick")}</span>
          <Button
            size="sm"
            onClick={() => {
              onChange(cursor);
              setOpen(false);
            }}
          >
            {t("common.open")}
          </Button>
        </div>
      </PopoverContent>
    </Popover>
  );
}

function DestinationFolders({
  location,
  cursor,
  onOpen,
}: {
  location: Location;
  cursor: string;
  onOpen: (prefix: string) => void;
}) {
  const { t } = useTranslation();
  const { data, isPending, hasNextPage, fetchNextPage, isFetchingNextPage } = useListing({
    ...location,
    prefix: cursor,
  });
  const folders = useMemo(
    () =>
      sortEntries(
        entriesFromPages(data?.pages ?? [], cursor).filter((entry) => entry.folder),
        "name",
        1,
      ),
    [data, cursor],
  );

  return (
    <div className="max-h-[220px] min-h-[80px] overflow-y-auto p-1">
      {isPending ? (
        <div className="grid h-20 place-items-center">
          <LoaderCircleIcon className="size-4 animate-spin text-faint" />
        </div>
      ) : folders.length === 0 ? (
        <p className="px-2 py-2 text-[11.5px] text-faint">{t("common.none")}</p>
      ) : (
        folders.map((entry) => (
          <button
            key={entry.key}
            type="button"
            onClick={() => onOpen(entry.key)}
            className="flex h-7.5 w-full items-center gap-2 rounded-md px-2 text-left text-[12.5px] hover:bg-hover"
          >
            <FolderIcon className="size-3.5 flex-none text-[var(--kind-folder-fg)]" />
            <span className="min-w-0 flex-1 truncate">{entry.name}</span>
          </button>
        ))
      )}
      {hasNextPage ? (
        <button
          type="button"
          onClick={() => void fetchNextPage()}
          disabled={isFetchingNextPage}
          className="h-7 w-full rounded-md text-[11px] text-muted-foreground hover:bg-hover"
        >
          {isFetchingNextPage ? t("browser.loadingMore") : t("browser.loadMore")}
        </button>
      ) : null}
    </div>
  );
}
