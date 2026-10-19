import { useQuery } from "@tanstack/react-query";
import { ChevronRightIcon, DownloadIcon, ExpandIcon, LoaderCircleIcon, XIcon } from "lucide-react";
import { useEffect, useMemo, useRef } from "react";
import { useTranslation } from "react-i18next";
import { FileTile } from "@/components/file-tile";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { Tip } from "@/components/ui/tooltip";
import { downloadEntries } from "@/lib/actions";
import { api } from "@/lib/api";
import { entriesFromPages, sortEntries } from "@/lib/entries";
import { basename, formatBytes, formatDate } from "@/lib/format";
import { kindOf, previewMode } from "@/lib/kinds";
import { type Location, useNav } from "@/lib/nav";
import { useListing, usePresigned } from "@/lib/queries";
import { cn } from "@/lib/utils";
import type { EntryViewProps } from "./types";

const COLUMN_W = 230;
const PREVIEW_W = 300;

export function ColumnsView({ selection, onSelect, onOpen, location, readOnly }: EntryViewProps) {
  const prefixes = useMemo(() => {
    const parts = location.prefix.split("/").filter(Boolean);
    const out = [""];
    parts.forEach((_, index) => {
      out.push(`${parts.slice(0, index + 1).join("/")}/`);
    });
    return out;
  }, [location.prefix]);

  const scrollRef = useRef<HTMLDivElement>(null);
  // biome-ignore lint/correctness/useExhaustiveDependencies: scroll to the newly opened column when the path changes
  useEffect(() => {
    const element = scrollRef.current;
    if (element) element.scrollTo({ left: element.scrollWidth, behavior: "smooth" });
  }, [location.prefix]);

  const selected = [...selection];
  const selectedKey = selected.length === 1 && selected[0] !== location.prefix ? selected[0] : null;

  return (
    <div className="flex min-h-0 flex-1">
      <div ref={scrollRef} className="flex min-h-0 min-w-0 flex-1 overflow-x-auto">
        {prefixes.map((prefix) => (
          <Column
            key={prefix || "root"}
            location={{ ...location, prefix }}
            currentPrefix={location.prefix}
            selection={selection}
            onSelect={onSelect}
            onOpen={onOpen}
            header={
              prefix === ""
                ? location.bucket
                : (prefix.replace(/\/$/, "").split("/").at(-1) ?? prefix)
            }
          />
        ))}
      </div>
      {selectedKey ? (
        <ColumnsPreview
          location={location}
          entryKey={selectedKey}
          onClose={() => useNav.getState().select([], null)}
          readOnly={readOnly}
        />
      ) : (
        <div
          className="hidden flex-none border-l bg-sidebar/40 xl:block"
          style={{ width: PREVIEW_W }}
          aria-hidden
        />
      )}
    </div>
  );
}

function Column({
  location,
  currentPrefix,
  selection,
  onSelect,
  onOpen,
  header,
}: {
  location: Location;
  currentPrefix: string;
  selection: Set<string>;
  onSelect: EntryViewProps["onSelect"];
  onOpen: EntryViewProps["onOpen"];
  header: string;
}) {
  const { t } = useTranslation();
  const { data, isPending, error, fetchNextPage, hasNextPage, isFetchingNextPage } =
    useListing(location);
  const entries = useMemo(
    () => sortEntries(entriesFromPages(data?.pages ?? [], location.prefix), "name", 1),
    [data, location.prefix],
  );
  const navigate = useNav((s) => s.navigate);

  if (isPending) {
    return (
      <div className="grid flex-none place-items-center border-r" style={{ width: COLUMN_W }}>
        <Spinner />
      </div>
    );
  }

  return (
    <div
      className="flex h-full flex-none flex-col border-r bg-background"
      style={{ width: COLUMN_W }}
    >
      <div className="flex h-8 flex-none items-center px-3 text-[11px] font-semibold text-faint">
        <span className="truncate">{header}</span>
      </div>
      <div className="min-h-0 flex-1 overflow-y-auto px-1.5 pb-2">
        {error ? (
          <p className="px-2 py-1 text-[11.5px] text-destructive">{t("states.errorTitle")}</p>
        ) : entries.length === 0 ? (
          <p className="px-2 py-1 text-[11.5px] text-faint">{t("common.none")}</p>
        ) : (
          entries.map((entry) => {
            const selected = selection.has(entry.key);
            const inPath =
              entry.folder && currentPrefix.startsWith(entry.key) && currentPrefix !== entry.key;
            return (
              <div
                key={entry.key}
                role="option"
                aria-selected={selected}
                data-entry-key={entry.key}
                tabIndex={-1}
                onKeyDown={(event) => {
                  if (event.key === "Enter" || event.key === " ") {
                    event.preventDefault();
                    onSelect(entry, {
                      metaKey: event.metaKey,
                      ctrlKey: event.ctrlKey,
                      shiftKey: event.shiftKey,
                    });
                  }
                }}
                onClick={(event) => {
                  if (entry.folder && !event.metaKey && !event.shiftKey) {
                    navigate({ ...location, prefix: entry.key });
                    return;
                  }
                  onSelect(entry, {
                    metaKey: event.metaKey,
                    ctrlKey: event.ctrlKey,
                    shiftKey: event.shiftKey,
                  });
                }}
                onDoubleClick={() => onOpen(entry)}
                className={cn(
                  "flex h-7 cursor-default items-center gap-2 rounded-md px-2 text-[12.5px]",
                  selected
                    ? "bg-primary text-primary-foreground"
                    : inPath
                      ? "bg-muted"
                      : "hover:bg-hover",
                )}
              >
                <FileTile kind={entry.kind} size={18} selected={selected} />
                <span className="min-w-0 flex-1 truncate" title={entry.name}>
                  {entry.name}
                </span>
                {entry.folder ? (
                  <ChevronRightIcon
                    className={cn("size-3 flex-none", selected ? "text-white/70" : "text-faint")}
                  />
                ) : null}
              </div>
            );
          })
        )}
        {hasNextPage ? (
          <button
            type="button"
            onClick={() => void fetchNextPage()}
            disabled={isFetchingNextPage}
            className="mt-1 flex h-7 w-full items-center justify-center gap-1.5 rounded-md text-[11.5px] text-muted-foreground hover:bg-hover"
          >
            {isFetchingNextPage ? <LoaderCircleIcon className="size-3 animate-spin" /> : null}
            {isFetchingNextPage ? t("browser.loadingMore") : t("browser.loadMore")}
          </button>
        ) : null}
      </div>
    </div>
  );
}

function ColumnsPreview({
  location,
  entryKey,
  onClose,
  readOnly,
}: {
  location: Location;
  entryKey: string;
  onClose: () => void;
  readOnly: boolean;
}) {
  const { t, i18n } = useTranslation();
  const setQuickLook = useNav((s) => s.setQuickLook);
  const { data: info } = useQuery({
    queryKey: ["stat", location.profileId, location.bucket, entryKey],
    queryFn: () =>
      api.Stat({ profileId: location.profileId, bucket: location.bucket, key: entryKey }),
    staleTime: 60_000,
  });
  const contentType = info?.contentType ?? "";
  const mode = previewMode(kindOf(entryKey, contentType));
  const presigned = usePresigned(
    location,
    entryKey,
    info?.etag ?? "",
    mode === "image" || mode === "video" || mode === "audio" || mode === "pdf",
  );

  return (
    <aside
      className="flex h-full flex-none flex-col border-l bg-background"
      style={{ width: PREVIEW_W }}
    >
      <div className="flex h-8 flex-none items-center gap-1 pl-3 pr-1.5">
        <span className="min-w-0 flex-1 truncate text-[11px] font-semibold text-faint">
          {t("browser.previewKey")}
        </span>
        <Tip label={t("menu.quickLook")}>
          <Button variant="ghost" size="icon-sm" onClick={() => setQuickLook(entryKey)}>
            <ExpandIcon className="size-3" />
          </Button>
        </Tip>
        <Tip label={t("menu.download")}>
          <Button
            variant="ghost"
            size="icon-sm"
            onClick={() =>
              void downloadEntries(location, [
                {
                  key: entryKey,
                  name: basename(entryKey),
                  folder: false,
                  size: info?.size ?? 0,
                  modified: "",
                  contentType: "",
                  etag: "",
                  kind: kindOf(entryKey, contentType),
                },
              ])
            }
          >
            <DownloadIcon className="size-3" />
          </Button>
        </Tip>
        <Button variant="ghost" size="icon-sm" onClick={onClose} aria-label={t("common.close")}>
          <XIcon className="size-3" />
        </Button>
      </div>

      <div className="mx-3 aspect-[4/3] flex-none overflow-hidden rounded-lg bg-tile-image">
        {presigned.data ? (
          mode === "image" ? (
            <img
              src={presigned.data.url}
              alt=""
              draggable={false}
              className="size-full object-contain"
            />
          ) : mode === "video" ? (
            // biome-ignore lint/a11y/useMediaCaption: user media has no caption tracks
            <video src={presigned.data.url} controls className="size-full object-contain" />
          ) : mode === "audio" ? (
            <div className="grid size-full place-items-center p-3">
              {/* biome-ignore lint/a11y/useMediaCaption: user media has no caption tracks */}
              <audio src={presigned.data.url} controls className="w-full" />
            </div>
          ) : mode === "pdf" ? (
            <iframe src={presigned.data.url} title={basename(entryKey)} className="size-full" />
          ) : (
            <TextPreview location={location} entryKey={entryKey} />
          )
        ) : (
          <div className="grid size-full place-items-center">
            <FileTile kind={kindOf(entryKey, contentType)} size={56} />
          </div>
        )}
      </div>

      <div className="flex flex-col gap-2 px-3 py-3 text-[12px]">
        <div className="truncate font-semibold" title={entryKey} data-selectable>
          {basename(entryKey)}
        </div>
        <dl className="flex flex-col gap-1.5 text-[11.5px]">
          <Row
            label={t("quicklook.size")}
            value={info ? formatBytes(info.size, i18n.language) : "—"}
          />
          <Row
            label={t("quicklook.modified")}
            value={info ? formatDate(info.lastModified, i18n.language) : "—"}
          />
          <Row label={t("quicklook.contentType")} value={contentType || "—"} />
          <Row label={t("browser.previewKey")} value={entryKey} selectable />
        </dl>
        {readOnly ? (
          <p className="text-[11px] leading-relaxed text-faint">{t("browser.readOnlyBanner")}</p>
        ) : null}
      </div>
    </aside>
  );
}

function Row({ label, value, selectable }: { label: string; value: string; selectable?: boolean }) {
  return (
    <div className="flex gap-3">
      <dt className="w-20 flex-none text-faint">{label}</dt>
      <dd className={cn("min-w-0 flex-1 truncate", selectable && "select-text")} title={value}>
        {value}
      </dd>
    </div>
  );
}

function TextPreview({ location, entryKey }: { location: Location; entryKey: string }) {
  const { data, isPending, error } = useQuery({
    queryKey: ["text", location.profileId, location.bucket, entryKey],
    queryFn: () =>
      api.ReadText({ profileId: location.profileId, bucket: location.bucket, key: entryKey }),
    staleTime: 5 * 60_000,
    retry: false,
  });
  if (isPending) {
    return (
      <div className="grid size-full place-items-center">
        <Spinner />
      </div>
    );
  }
  if (error || !data) {
    return <div className="grid size-full place-items-center p-3 text-xs text-faint">—</div>;
  }
  return (
    <pre className="size-full overflow-auto p-2.5 font-mono text-[10px] leading-relaxed whitespace-pre-wrap select-text">
      {data.text}
    </pre>
  );
}
