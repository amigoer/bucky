import {
  AlertTriangleIcon,
  FolderIcon,
  KeyRoundIcon,
  LoaderCircleIcon,
  LockIcon,
  SearchIcon,
  SearchXIcon,
  ShieldXIcon,
  UploadCloudIcon,
  WifiOffIcon,
} from "lucide-react";
import { useCallback, useEffect, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { EmptyState } from "@/components/empty-state";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { pickAndUpload, uploadPaths } from "@/lib/actions";
import { events } from "@/lib/api";
import type { Entry, SortKey } from "@/lib/entries";
import { entriesFromFind, entriesFromPages, sortEntries } from "@/lib/entries";
import { errorCode, errorMessage } from "@/lib/errors";
import { useEvent } from "@/lib/hooks";
import { type Location, useNav } from "@/lib/nav";
import { useListing, useProfiles, useSearch } from "@/lib/queries";
import { usePrefs } from "@/lib/store";
import { ColumnsView } from "./ColumnsView";
import { GridView } from "./GridView";
import { ListView } from "./ListView";
import { SelectionBar } from "./SelectionBar";
import { Toolbar } from "./Toolbar";
import type { SelectModifiers } from "./types";

export function Browser() {
  const location = useNav((s) => s.location) as Location;
  const { t } = useTranslation();
  const query = useNav((s) => s.query);
  const selection = useNav((s) => s.selection);
  const anchor = useNav((s) => s.anchor);
  const select = useNav((s) => s.select);
  const setQuery = useNav((s) => s.setQuery);
  const setSearchOpen = useNav((s) => s.setSearchOpen);
  const searchOpen = useNav((s) => s.searchOpen);
  const navigate = useNav((s) => s.navigate);
  const setQuickLook = useNav((s) => s.setQuickLook);
  const openPlan = useNav((s) => s.openPlan);
  const openConnection = useNav((s) => s.openConnection);
  const forgetProfile = useNav((s) => s.forgetProfile);
  const overlay = useNav(
    (s) => !!(s.quickLook || s.share || s.plan || s.connection || s.quickUpload),
  );

  const view = usePrefs((s) => s.view);
  const set = usePrefs((s) => s.set);
  const sortKey = usePrefs((s) => s.sortKey);
  const sortDir = usePrefs((s) => s.sortDir);

  const { data: profiles = [], isPending: profilesPending } = useProfiles();
  const profile = profiles.find((p) => p.id === location.profileId);
  const readOnly = profile?.readOnly ?? false;

  const q = query.trim();
  const listing = useListing(location);
  const search = useSearch(location, q);

  const listingEntries = useMemo(
    () => entriesFromPages(listing.data?.pages ?? [], location.prefix),
    [listing.data, location.prefix],
  );
  const searchEntries = useMemo(
    () =>
      entriesFromFind(
        search.data?.pages.flatMap((page) => page.objects ?? []) ?? [],
        location.prefix,
      ),
    [search.data, location.prefix],
  );
  const entries = useMemo(
    () => sortEntries(q ? searchEntries : listingEntries, sortKey, sortDir),
    [q, searchEntries, listingEntries, sortKey, sortDir],
  );

  const selectedEntries = useMemo(
    () => entries.filter((entry) => selection.includes(entry.key)),
    [entries, selection],
  );
  const selectionSet = useMemo(() => new Set(selection), [selection]);

  const loadedCount = (listing.data?.pages ?? []).reduce(
    (sum, page) => sum + (page.objects?.length ?? 0),
    0,
  );
  const scannedCount =
    search.data?.pages.reduce((sum, page) => sum + (page.objects?.length ?? 0), 0) ?? 0;

  useEffect(() => {
    if (!profilesPending && !profile) forgetProfile(location.profileId);
  }, [profilesPending, profile, forgetProfile, location.profileId]);

  const openEntry = useEvent((entry: Entry) => {
    if (entry.folder) navigate({ ...location, prefix: entry.key });
    else setQuickLook(entry.key, entries);
  });

  const selectEntry = useEvent((entry: Entry, mods: SelectModifiers) => {
    const keys = entries.map((item) => item.key);
    if (mods.shiftKey && anchor) {
      const from = keys.indexOf(anchor);
      const to = keys.indexOf(entry.key);
      if (from >= 0 && to >= 0) {
        const [lo, hi] = from < to ? [from, to] : [to, from];
        select(keys.slice(lo, hi + 1), anchor);
        return;
      }
    }
    if (mods.metaKey || mods.ctrlKey) {
      const has = selection.includes(entry.key);
      select(
        has ? selection.filter((key) => key !== entry.key) : [...selection, entry.key],
        entry.key,
      );
      return;
    }
    select([entry.key], entry.key);
  });

  const loadMore = useCallback(() => {
    if (q) return;
    if (listing.hasNextPage && !listing.isFetchingNextPage) void listing.fetchNextPage();
  }, [q, listing]);

  const sortBy = useCallback(
    (key: SortKey) => {
      set(sortKey === key ? { sortDir: sortDir === 1 ? -1 : 1 } : { sortKey: key, sortDir: 1 });
    },
    [set, sortKey, sortDir],
  );

  const onDrop = useEvent((paths: string[]) => {
    if (paths.length === 0) return;
    if (readOnly) {
      toast.error(t("browser.readOnlyTip"));
      return;
    }
    void uploadPaths(location, paths);
  });

  useEffect(() => events.onFilesDropped(onDrop), [onDrop]);

  const onKeyDown = useEvent((event: KeyboardEvent) => {
    if (overlay) return;
    const target = event.target instanceof HTMLElement ? event.target : null;
    if (
      target &&
      (target.isContentEditable ||
        ["INPUT", "TEXTAREA", "SELECT"].includes(target.tagName.toUpperCase()))
    ) {
      return;
    }
    const mod = event.metaKey || event.ctrlKey;
    const move = (index: number) => {
      if (entries.length === 0) return;
      const clamped = Math.max(0, Math.min(entries.length - 1, index));
      const next = entries[clamped];
      select([next.key], next.key);
    };
    const current = anchor ? entries.findIndex((entry) => entry.key === anchor) : -1;

    if (event.key === "Escape") {
      if (searchOpen) {
        setSearchOpen(false);
        setQuery("");
      } else if (selection.length > 0) {
        select([], null);
      }
      return;
    }
    if (mod && (event.key === "a" || event.key === "A")) {
      event.preventDefault();
      select(
        entries.map((entry) => entry.key),
        entries[0]?.key ?? null,
      );
      return;
    }
    if (mod && ["1", "2", "3"].includes(event.key)) {
      event.preventDefault();
      set({ view: (["list", "grid", "columns"] as const)[Number(event.key) - 1] });
      return;
    }
    if (event.key === " " && !mod) {
      const entry = selectedEntries.length === 1 ? selectedEntries[0] : null;
      if (entry && !entry.folder) {
        event.preventDefault();
        setQuickLook(entry.key, entries);
      }
      return;
    }
    if (event.key === "Enter") {
      const entry =
        selectedEntries.length === 1 ? selectedEntries[0] : current >= 0 ? entries[current] : null;
      if (entry) {
        event.preventDefault();
        openEntry(entry);
      }
      return;
    }
    if ((mod && event.key === "Backspace") || event.key === "Delete") {
      if (!readOnly && selectedEntries.length > 0) {
        event.preventDefault();
        openPlan({ operation: "delete", location, entries: selectedEntries });
      }
      return;
    }
    switch (event.key) {
      case "ArrowDown":
      case "ArrowRight":
        event.preventDefault();
        move(current + 1);
        break;
      case "ArrowUp":
      case "ArrowLeft":
        event.preventDefault();
        move(current - 1);
        break;
      case "Home":
        event.preventDefault();
        move(0);
        break;
      case "End":
        event.preventDefault();
        move(entries.length - 1);
        break;
    }
  });

  useEffect(() => {
    window.addEventListener("keydown", onKeyDown);
    return () => window.removeEventListener("keydown", onKeyDown);
  }, [onKeyDown]);

  const visibleError = q ? search.error : listing.error;
  const pending = q ? search.isPending : listing.isPending;
  const place = location.prefix || location.bucket;
  const viewProps = {
    entries,
    selection: selectionSet,
    focusKey: anchor,
    onSelect: selectEntry,
    onOpen: openEntry,
    onLoadMore: loadMore,
    location,
    readOnly,
  };

  return (
    <div data-file-drop-target className="relative flex min-h-0 flex-1 flex-col">
      <Toolbar location={location} readOnly={readOnly} />

      {readOnly ? (
        <div className="flex h-8 flex-none items-center gap-1.5 border-b bg-warning-soft px-4 text-[11.5px] text-warning-text">
          <LockIcon className="size-3.5 flex-none" />
          <span className="truncate">{t("browser.readOnlyBanner")}</span>
        </div>
      ) : null}

      {q ? (
        <div className="flex h-8 flex-none items-center gap-1.5 border-b px-4 text-[11.5px] text-muted-foreground">
          <SearchIcon className="size-3 flex-none text-faint" />
          <span className="truncate">{t("search.scope", { place })}</span>
          <span className="text-faint">·</span>
          <span className="whitespace-nowrap">
            {search.isFetching
              ? t("search.scanning")
              : t("search.scanned", { count: scannedCount })}
          </span>
          {search.hasNextPage ? (
            <button
              type="button"
              onClick={() => void search.fetchNextPage()}
              disabled={search.isFetchingNextPage}
              className="ml-auto flex flex-none items-center gap-1 rounded px-1.5 py-0.5 text-[11px] font-medium text-primary-text hover:bg-primary-soft disabled:opacity-50"
            >
              {search.isFetchingNextPage ? (
                <LoaderCircleIcon className="size-3 animate-spin" />
              ) : null}
              {t("search.scanMore")}
            </button>
          ) : (
            <span className="ml-auto flex-none text-faint">{t("search.complete")}</span>
          )}
        </div>
      ) : null}

      <div className="relative flex min-h-0 flex-1 flex-col">
        {pending ? (
          <div className="grid flex-1 place-items-center">
            <div className="flex flex-col items-center gap-2.5 text-muted-foreground">
              <Spinner className="size-5" />
              <span className="text-[12.5px]">{t("states.loading")}</span>
            </div>
          </div>
        ) : visibleError ? (
          <div className="grid flex-1 place-items-center">
            <ErrorState
              error={visibleError}
              onRetry={() => void (q ? search.refetch() : listing.refetch())}
              onEdit={profile ? () => openConnection({ id: profile.id }) : undefined}
            />
          </div>
        ) : entries.length === 0 ? (
          <div className="grid flex-1 place-items-center pb-10">
            {q ? (
              <EmptyState
                icon={<SearchXIcon />}
                title={t("states.noResultsTitle", { query: q })}
                description={t("states.noResultsDesc", { place, count: scannedCount })}
              />
            ) : (
              <EmptyState
                icon={<FolderIcon />}
                tone="accent"
                title={
                  location.prefix ? t("states.emptyFolderTitle") : t("states.emptyBucketTitle")
                }
                description={
                  readOnly
                    ? t("states.readOnlyEmpty")
                    : location.prefix
                      ? t("states.emptyFolderDesc", { place })
                      : t("states.emptyBucketDesc")
                }
                action={
                  readOnly ? undefined : (
                    <Button size="sm" onClick={() => void pickAndUpload(location, false)}>
                      {t("browser.upload")}
                    </Button>
                  )
                }
              />
            )}
          </div>
        ) : view === "grid" ? (
          <GridView {...viewProps} />
        ) : view === "columns" ? (
          <ColumnsView {...viewProps} />
        ) : (
          <ListView {...viewProps} sortKey={sortKey} sortDir={sortDir} onSort={sortBy} />
        )}

        <SelectionBar location={location} selected={selectedEntries} readOnly={readOnly} />

        <div className="flex h-7 flex-none items-center gap-2 border-t px-3.5 text-[10.5px] text-faint">
          <span>
            {q
              ? t("search.results", { count: entries.length })
              : t("browser.loaded", { count: loadedCount })}
          </span>
          {q && search.hasNextPage ? (
            <span className="truncate">{t("search.scanning")}</span>
          ) : null}
          {!q && listing.hasNextPage ? (
            <button
              type="button"
              onClick={() => void listing.fetchNextPage()}
              disabled={listing.isFetchingNextPage}
              className="ml-auto flex items-center gap-1 rounded px-1.5 py-0.5 font-medium text-muted-foreground hover:bg-hover disabled:opacity-50"
            >
              {listing.isFetchingNextPage ? t("browser.loadingMore") : t("browser.loadMore")}
            </button>
          ) : null}
        </div>

        <div className="pointer-events-none absolute inset-0 z-40 hidden place-items-center bg-primary-tint backdrop-blur-[1.5px] drop-active:grid">
          <div className="flex flex-col items-center gap-2 rounded-2xl border-2 border-dashed border-primary-line bg-popover/95 px-10 py-7 text-center shadow-sheet">
            <UploadCloudIcon className="size-7 text-primary" />
            <span className="text-[13px] font-semibold">
              {readOnly ? t("browser.dropReadOnly") : t("browser.dropRelease", { place })}
            </span>
          </div>
        </div>
      </div>
    </div>
  );
}

function ErrorState({
  error,
  onRetry,
  onEdit,
}: {
  error: unknown;
  onRetry: () => void;
  onEdit?: () => void;
}) {
  const { t } = useTranslation();
  const code = errorCode(error);
  if (code === "credentials_unavailable" || code === "configuration") {
    return (
      <EmptyState
        icon={<KeyRoundIcon />}
        tone="warn"
        title={t("states.credTitle")}
        description={t("states.credDesc")}
        action={
          <>
            {onEdit ? (
              <Button size="sm" onClick={onEdit}>
                {t("states.editConnection")}
              </Button>
            ) : null}
            <Button size="sm" variant="secondary" onClick={onRetry}>
              {t("common.retry")}
            </Button>
          </>
        }
      />
    );
  }
  if (code === "forbidden") {
    return (
      <EmptyState
        icon={<ShieldXIcon />}
        tone="warn"
        title={t("states.permTitle")}
        description={t("states.permDesc")}
        action={
          <Button size="sm" variant="secondary" onClick={onRetry}>
            {t("common.retry")}
          </Button>
        }
      />
    );
  }
  if (code === "network" || code === "timeout" || code === "protocol") {
    return (
      <EmptyState
        icon={<WifiOffIcon />}
        title={t("states.netTitle")}
        description={t("states.netDesc")}
        action={
          <Button size="sm" variant="secondary" onClick={onRetry}>
            {t("common.retry")}
          </Button>
        }
      />
    );
  }
  return (
    <EmptyState
      icon={<AlertTriangleIcon />}
      tone="bad"
      title={t("states.errorTitle")}
      description={errorMessage(error)}
      action={
        <Button size="sm" variant="secondary" onClick={onRetry}>
          {t("common.retry")}
        </Button>
      }
    />
  );
}
