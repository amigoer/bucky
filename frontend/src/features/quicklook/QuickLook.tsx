import { useQuery } from "@tanstack/react-query";
import {
  ChevronLeftIcon,
  ChevronRightIcon,
  DownloadIcon,
  FileIcon,
  RefreshCwIcon,
  Share2Icon,
  XIcon,
} from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { EmptyState } from "@/components/empty-state";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { Spinner } from "@/components/ui/spinner";
import { Tip } from "@/components/ui/tooltip";
import { downloadEntries } from "@/lib/actions";
import { api } from "@/lib/api";
import type { Entry } from "@/lib/entries";
import { entriesFromPages, sortEntries } from "@/lib/entries";
import { errorCode, errorMessage } from "@/lib/errors";
import { basename, formatBytes, formatDate } from "@/lib/format";
import { useNow } from "@/lib/hooks";
import { kindOf, previewMode } from "@/lib/kinds";
import { type Location, useNav } from "@/lib/nav";
import { useListing, usePresigned } from "@/lib/queries";
import { usePrefs } from "@/lib/store";
import { cn } from "@/lib/utils";

export function QuickLook() {
  const objectKey = useNav((s) => s.quickLook);
  const location = useNav((s) => s.location);
  if (!objectKey || !location) return null;
  return <QuickLookInner objectKey={objectKey} location={location} />;
}

function QuickLookInner({ objectKey, location }: { objectKey: string; location: Location }) {
  const { t, i18n } = useTranslation();
  const key = objectKey;
  const siblings = useNav((s) => s.quickLookList);
  const setQuickLook = useNav((s) => s.setQuickLook);
  const openShare = useNav((s) => s.openShare);
  const sortKey = usePrefs((s) => s.sortKey);
  const sortDir = usePrefs((s) => s.sortDir);
  const { data: listing } = useListing(location);

  const fallback = sortEntries(
    entriesFromPages(listing?.pages ?? [], location.prefix).filter((entry) => !entry.folder),
    sortKey,
    sortDir,
  );
  const list =
    siblings && siblings.length > 0 ? siblings.filter((entry) => !entry.folder) : fallback;
  const index = list.findIndex((entry) => entry.key === key);
  const known = index >= 0 ? list[index] : null;
  const name = known?.name || basename(key);

  const { data: info } = useQuery({
    queryKey: ["stat", location.profileId, location.bucket, key],
    queryFn: () => api.Stat({ profileId: location.profileId, bucket: location.bucket, key }),
    staleTime: 30_000,
    retry: false,
  });
  const contentType = info?.contentType ?? known?.contentType ?? "";
  const kind = kindOf(key, contentType);
  const mode = previewMode(kind);
  const entry: Entry =
    known ??
    ({
      key,
      name,
      folder: false,
      size: info?.size ?? 0,
      modified: info?.lastModified ?? "",
      contentType,
      etag: info?.etag ?? "",
      kind,
    } satisfies Entry);

  const step = (delta: number) => {
    const next = list[index + delta];
    if (next) setQuickLook(next.key, list);
  };

  return (
    <Dialog open onOpenChange={(open) => (!open ? setQuickLook(null) : undefined)}>
      <DialogContent
        className="h-[min(580px,calc(100vh-104px))] w-[min(880px,calc(100vw-96px))]"
        onKeyDown={(event) => {
          const target = event.target as HTMLElement;
          const interactive = [
            "BUTTON",
            "INPUT",
            "TEXTAREA",
            "SELECT",
            "A",
            "VIDEO",
            "AUDIO",
          ].includes(target.tagName);
          if (event.key === " " && !interactive && !target.isContentEditable) {
            event.preventDefault();
            setQuickLook(null);
          } else if (event.key === "ArrowLeft" && !interactive) {
            event.preventDefault();
            step(-1);
          } else if (event.key === "ArrowRight" && !interactive) {
            event.preventDefault();
            step(1);
          }
        }}
      >
        <DialogTitle className="sr-only">{name}</DialogTitle>
        <DialogDescription className="sr-only">{t("quicklook.note")}</DialogDescription>
        <div className="flex h-11 flex-none items-center gap-2 border-b pl-3.5 pr-2">
          <span className="min-w-0 truncate text-[13px] font-semibold" title={key}>
            {name}
          </span>
          <Badge variant="secondary" className="flex-none">
            {t(`kinds.${kind}`)}
          </Badge>
          <div className="ml-auto flex flex-none items-center gap-0.5">
            <Tip label={t("common.back")}>
              <Button
                variant="ghost"
                size="icon-sm"
                disabled={index <= 0}
                onClick={() => step(-1)}
                aria-label={t("common.back")}
              >
                <ChevronLeftIcon className="size-3.5" />
              </Button>
            </Tip>
            <Tip label={t("common.forward")}>
              <Button
                variant="ghost"
                size="icon-sm"
                disabled={index < 0 || index >= list.length - 1}
                onClick={() => step(1)}
                aria-label={t("common.forward")}
              >
                <ChevronRightIcon className="size-3.5" />
              </Button>
            </Tip>
            <span className="mx-1 h-4 w-px bg-border" />
            <Tip label={t("menu.download")}>
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={() => void downloadEntries(location, [entry])}
                aria-label={t("menu.download")}
              >
                <DownloadIcon className="size-3.5" />
              </Button>
            </Tip>
            <Tip label={t("menu.share")}>
              <Button
                variant="ghost"
                size="icon-sm"
                onClick={() => openShare({ location, entries: [entry] })}
                aria-label={t("menu.share")}
              >
                <Share2Icon className="size-3.5" />
              </Button>
            </Tip>
            <DialogClose asChild>
              <Button variant="ghost" size="icon-sm" aria-label={t("common.close")}>
                <XIcon className="size-3.5" />
              </Button>
            </DialogClose>
          </div>
        </div>

        <div className="flex min-h-0 flex-1">
          <div
            className={cn(
              "relative flex min-w-0 flex-1 items-center justify-center overflow-hidden",
              mode === "text" ? "bg-background" : "bg-[#0c0c0e]",
              mode === "pdf" && "bg-[#1a1a1c]",
            )}
            key={key}
          >
            <Preview location={location} entry={entry} mode={mode} contentType={contentType} />
          </div>

          <aside className="flex w-[268px] flex-none flex-col gap-3.5 overflow-y-auto border-l bg-popover p-3.5">
            <section className="flex flex-col gap-1.5">
              <Row label={t("quicklook.key")} value={key} mono />
              <Row
                label={t("quicklook.size")}
                value={info ? formatBytes(info.size, i18n.language) : "—"}
              />
              <Row label={t("quicklook.contentType")} value={contentType || "—"} />
              <Row
                label={t("quicklook.modified")}
                value={info ? formatDate(info.lastModified, i18n.language) : "—"}
              />
              <Row label={t("quicklook.etag")} value={info?.etag || "—"} mono />
            </section>
            {info?.metadata && Object.keys(info.metadata).length > 0 ? (
              <section className="flex flex-col gap-1.5 border-t pt-3">
                <h3 className="text-[11px] font-semibold text-faint">{t("quicklook.metadata")}</h3>
                {Object.entries(info.metadata)
                  .sort(([a], [b]) => a.localeCompare(b))
                  .map(([metaKey, value]) => (
                    <Row key={metaKey} label={metaKey} value={value ?? ""} mono />
                  ))}
              </section>
            ) : null}
            <p className="mt-auto text-[10.5px] leading-relaxed text-faint">
              {t("quicklook.note")}
            </p>
          </aside>
        </div>

        <div className="flex h-8 flex-none items-center gap-2 border-t bg-sidebar px-3.5 text-[10.5px] text-faint">
          <span>{t("quicklook.hint")}</span>
          {index >= 0 ? (
            <span className="ml-auto tabular">
              {index + 1} / {list.length}
            </span>
          ) : null}
        </div>
      </DialogContent>
    </Dialog>
  );
}

function Preview({
  location,
  entry,
  mode,
  contentType,
}: {
  location: Location;
  entry: Entry;
  mode: ReturnType<typeof previewMode>;
  contentType: string;
}) {
  const { t, i18n } = useTranslation();
  const now = useNow(30_000, true);
  const link = usePresigned(
    location,
    entry.key,
    entry.etag,
    mode === "image" || mode === "video" || mode === "audio" || mode === "pdf",
  );
  const text = useQuery({
    queryKey: ["text", location.profileId, location.bucket, entry.key],
    queryFn: () =>
      api.ReadText({
        profileId: location.profileId,
        bucket: location.bucket,
        key: entry.key,
      }),
    enabled: mode === "text",
    staleTime: 5 * 60_000,
    retry: false,
  });
  const [loaded, setLoaded] = useState(false);
  const [broken, setBroken] = useState(false);

  if (mode === "none") {
    return (
      <EmptyState
        icon={<FileIcon />}
        title={t("quicklook.unsupported")}
        description={t("quicklook.unsupportedDesc")}
        className="text-[#f5f5f7] [&>span:nth-child(2)]:text-[#f5f5f7] [&>span:nth-child(3)]:text-white/55"
        action={
          <Button
            size="sm"
            variant="secondary"
            onClick={() => void downloadEntries(location, [entry])}
          >
            {t("menu.download")}
          </Button>
        }
      />
    );
  }

  if (mode === "text") {
    if (text.isPending) return <Spinner className="size-5 text-white/60" />;
    if (text.error || !text.data) {
      return <Failed message={errorMessage(text.error)} onRetry={() => void text.refetch()} />;
    }
    return (
      <div className="flex size-full flex-col">
        {text.data.truncated ? (
          <div className="flex-none border-b bg-warning-soft px-3 py-1 text-[10.5px] text-warning-text">
            {t("quicklook.truncated", { size: formatBytes(entry.size, i18n.language) })}
          </div>
        ) : null}
        <pre className="min-h-0 flex-1 overflow-auto p-4 font-mono text-[11.5px] leading-relaxed whitespace-pre-wrap text-foreground select-text">
          {text.data.text}
        </pre>
      </div>
    );
  }

  const url = link.data?.url;
  if (link.error) {
    const code = errorCode(link.error);
    const expired = code === "expired";
    return (
      <Failed
        title={expired ? t("quicklook.expired") : t("quicklook.failed")}
        message={expired ? t("quicklook.expiredDesc") : errorMessage(link.error)}
        onRetry={() => void link.refetch()}
        action={t("quicklook.regenerate")}
      />
    );
  }
  const expired = link.data
    ? Date.parse(link.data.expiresAt) > 0 && Date.parse(link.data.expiresAt) < now
    : false;
  if (expired) {
    return (
      <Failed
        title={t("quicklook.expired")}
        message={t("quicklook.expiredDesc")}
        onRetry={() => void link.refetch()}
        action={t("quicklook.regenerate")}
      />
    );
  }

  if (mode === "image") {
    if (!url) return <Spinner className="size-5 text-white/60" />;
    if (broken) {
      return <Failed message={t("quicklook.failedDesc")} onRetry={() => void link.refetch()} />;
    }
    return (
      <>
        {!loaded ? (
          <div className="absolute inset-0 grid place-items-center">
            <Spinner className="size-5 text-white/60" />
          </div>
        ) : null}
        <img
          src={url}
          alt={entry.name}
          draggable={false}
          onLoad={() => setLoaded(true)}
          onError={() => setBroken(true)}
          className={cn(
            "max-h-full max-w-full object-contain transition-opacity duration-200",
            loaded ? "opacity-100" : "opacity-0",
          )}
        />
        <div className="absolute inset-x-0 bottom-0 flex h-7 items-center bg-black/45 px-3 text-[10.5px] text-white/70">
          {formatBytes(entry.size, i18n.language)} · {contentType || t(`kinds.${entry.kind}`)}
        </div>
      </>
    );
  }

  if (!url) return <Spinner className="size-5 text-white/60" />;

  if (mode === "video") {
    return (
      // biome-ignore lint/a11y/useMediaCaption: user media has no caption tracks
      <video
        key={entry.key}
        src={url}
        controls
        className="max-h-full max-w-full"
        onError={() => setBroken(true)}
      />
    );
  }

  if (mode === "audio") {
    return (
      <div className="flex flex-col items-center gap-4 px-8">
        <span className="grid size-24 place-items-center rounded-3xl bg-white/8 text-white/80">
          <FileIcon className="size-10" />
        </span>
        <span className="max-w-full truncate text-[13px] font-medium text-white/85">
          {entry.name}
        </span>
        {/* biome-ignore lint/a11y/useMediaCaption: user media has no caption tracks */}
        <audio src={url} controls className="w-[320px]" />
      </div>
    );
  }

  return (
    <iframe
      src={url}
      title={entry.name}
      className="size-full border-0 bg-white"
      onError={() => setBroken(true)}
    />
  );
}

function Failed({
  title,
  message,
  onRetry,
  action,
}: {
  title?: string;
  message: string;
  onRetry: () => void;
  action?: string;
}) {
  const { t } = useTranslation();
  return (
    <EmptyState
      icon={<RefreshCwIcon />}
      tone="bad"
      title={title ?? t("quicklook.failed")}
      description={message}
      className="text-[#f5f5f7] [&>span:nth-child(2)]:text-[#f5f5f7] [&>span:nth-child(3)]:text-white/55"
      action={
        <Button size="sm" variant="secondary" onClick={onRetry}>
          {action ?? t("common.retry")}
        </Button>
      }
    />
  );
}

function Row({ label, value, mono }: { label: string; value: string; mono?: boolean }) {
  return (
    <div className="flex gap-2 text-[11.5px]" title={`${label}: ${value}`}>
      <span className="w-[76px] flex-none truncate text-faint">{label}</span>
      <span
        className={cn("min-w-0 flex-1 truncate select-text", mono && "font-mono text-[10.5px]")}
      >
        {value}
      </span>
    </div>
  );
}
