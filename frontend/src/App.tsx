import { useInfiniteQuery, useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { useVirtualizer } from "@tanstack/react-virtual";
import {
  AlertTriangle,
  ArrowUpDown,
  Check,
  ChevronRight,
  ChevronUp,
  Clipboard,
  Cloud,
  Copy,
  Download,
  File,
  FileText,
  Film,
  Folder,
  FolderDown,
  FolderUp,
  Gauge,
  Globe,
  Grid3X3,
  HardDrive,
  Image as ImageIcon,
  Languages,
  List,
  LoaderCircle,
  LockKeyhole,
  Moon,
  MoreHorizontal,
  Music,
  Pause,
  Play,
  Plus,
  RefreshCw,
  Search,
  Settings,
  Share2,
  Sun,
  Trash2,
  Upload,
  X,
} from "lucide-react";
import { type ReactNode, useCallback, useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { ConnectionModal } from "./components/ConnectionModal";
import { TitleBar } from "./components/TitleBar";
import {
  Button,
  ConfirmDialog,
  ErrorBanner,
  IconButton,
  Modal,
  Spinner,
  useAsyncState,
} from "./components/ui";
import {
  api,
  type Bucket,
  errorMessage,
  type FindRequest,
  isDesktop,
  type ObjectInfo,
  type OperationPlan,
  onFilesDropped,
  type Profile,
  type ShareLink,
  type TextPreview,
  type TransferTask,
  type Usage,
} from "./lib/api";
import {
  basename,
  breadcrumbs,
  displayType,
  exactBytes,
  formatBytes,
  formatDate,
  previewKind,
  progress,
} from "./lib/format";
import { useUI } from "./lib/store";

type Mode = { kind: "copy" | "move"; objects: ObjectInfo[] } | null;
type ConnectionState = { profile?: Profile } | null;

function useLocale() {
  const language = useUI((state) => state.language);
  return language === "zh" ? "zh-CN" : "en-US";
}

export default function App() {
  const { t, i18n } = useTranslation();
  const locale = useLocale();
  const ui = useUI();
  const queryClient = useQueryClient();
  const desktop = isDesktop();
  const [profileId, setProfileId] = useState("");
  const [bucket, setBucket] = useState("");
  const [prefix, setPrefix] = useState("");
  const [connection, setConnection] = useState<ConnectionState>(null);
  const [removing, setRemoving] = useState<Profile | null>(null);
  const [selection, setSelection] = useState<Set<string>>(new Set());
  const [preview, setPreview] = useState<ObjectInfo | null>(null);
  const [sharing, setSharing] = useState<ObjectInfo | null>(null);
  const [shareLink, setShareLink] = useState<ShareLink | null>(null);
  const [plan, setPlan] = useState<OperationPlan | null>(null);
  const [pendingCount, setPendingCount] = useState(0);
  const planQueue = useRef<OperationPlan[]>([]);
  const planRef = useRef<OperationPlan | null>(null);
  planRef.current = plan;
  const [mode, setMode] = useState<Mode>(null);
  const [settings, setSettings] = useState(false);
  const [transfersOpen, setTransfersOpen] = useState(false);
  const [toast, setToast] = useState("");
  const [search, setSearch] = useState("");
  const [findRequest, setFindRequest] = useState<FindRequest | null>(null);
  const [sort, setSort] = useState<"name" | "size" | "modified">("name");
  const [ascending, setAscending] = useState(true);
  const [executeState, runExecute] = useAsyncState();

  useEffect(() => {
    void i18n.changeLanguage(ui.language);
    document.documentElement.lang = ui.language === "zh" ? "zh-CN" : "en";
  }, [ui.language, i18n]);

  useEffect(() => {
    const apply = () => {
      const dark =
        ui.theme === "dark" ||
        (ui.theme === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches);
      document.documentElement.dataset.theme = dark ? "dark" : "light";
    };
    apply();
    const media = window.matchMedia("(prefers-color-scheme: dark)");
    media.addEventListener("change", apply);
    return () => media.removeEventListener("change", apply);
  }, [ui.theme]);

  useEffect(() => {
    if (!desktop) return;
    void api.SetRate(Math.round(useUI.getState().rate * 1024 * 1024)).catch(() => {});
  }, [desktop]);

  const notify = useCallback((message: string) => {
    setToast(message);
    window.setTimeout(() => setToast((current) => (current === message ? "" : current)), 3200);
  }, []);

  const enqueuePlans = useCallback((plans: OperationPlan[]) => {
    const valid = plans.filter(Boolean);
    if (!valid.length) return;
    setPendingCount((count) => count + valid.length);
    if (planRef.current) {
      planQueue.current.push(...valid);
    } else {
      setPlan(valid[0]);
      planQueue.current.push(...valid.slice(1));
    }
  }, []);

  const closePlan = useCallback(() => {
    setPendingCount((count) => Math.max(0, count - 1));
    setPlan(null);
    const next = planQueue.current.shift();
    if (next) setPlan(next);
  }, []);

  const profiles = useQuery({ queryKey: ["profiles"], queryFn: api.Profiles, enabled: desktop });
  const profile = profiles.data?.find((item) => item.id === profileId) ?? null;

  useEffect(() => {
    if (!profileId && profiles.data?.length) setProfileId(profiles.data[0].id);
    if (profileId && profiles.data && !profiles.data.some((item) => item.id === profileId)) {
      setProfileId(profiles.data[0]?.id ?? "");
    }
  }, [profiles.data, profileId]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: reset browsing state when the connection changes
  useEffect(() => {
    setBucket("");
    setPrefix("");
    setSelection(new Set());
    setFindRequest(null);
    setSearch("");
  }, [profileId]);

  const buckets = useQuery({
    queryKey: ["buckets", profileId],
    queryFn: () => api.Buckets(profileId),
    enabled: desktop && Boolean(profileId),
  });

  useEffect(() => {
    if (profile?.bucket) setBucket(profile.bucket);
    if (!bucket && buckets.data?.length) setBucket(buckets.data[0].name);
    if (bucket && buckets.data && !buckets.data.some((item) => item.name === bucket))
      setBucket(buckets.data[0]?.name ?? "");
  }, [buckets.data, bucket, profile?.bucket]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: reset browsing state when the bucket changes
  useEffect(() => {
    setPrefix("");
    setSelection(new Set());
    setFindRequest(null);
    setSearch("");
  }, [bucket]);

  const list = useInfiniteQuery({
    queryKey: ["objects", profileId, bucket, prefix],
    initialPageParam: "",
    enabled: desktop && Boolean(profileId && bucket) && !findRequest,
    queryFn: ({ pageParam }) =>
      api.List({ profileId, bucket, prefix, delimiter: "/", cursor: pageParam, limit: 200 }),
    getNextPageParam: (page) => page.nextCursor || undefined,
  });

  const found = useInfiniteQuery({
    queryKey: [
      "find",
      findRequest?.profileId,
      findRequest?.bucket,
      findRequest?.prefix,
      findRequest?.contains,
    ],
    initialPageParam: "",
    enabled: desktop && Boolean(findRequest),
    queryFn: ({ pageParam }) => {
      if (!findRequest) throw new Error("search is not active");
      return api.Find({ ...findRequest, cursor: pageParam });
    },
    getNextPageParam: (page) => (page.complete ? undefined : page.nextCursor || undefined),
  });

  const foundObjects = useMemo(
    () => found.data?.pages.flatMap((page) => page.objects) ?? [],
    [found.data],
  );
  const searchScanned = found.data?.pages.reduce((total, page) => total + page.scanned, 0) ?? 0;
  const searchComplete = found.data?.pages.at(-1)?.complete ?? false;

  const objects = useMemo(() => {
    const source: ObjectInfo[] = findRequest
      ? foundObjects
      : (list.data?.pages.flatMap((page) => page.objects) ?? []);
    const unique = new Map<string, ObjectInfo>();
    for (const object of source) {
      if (!object.key || object.key.endsWith("/")) continue;
      unique.set(object.key, object);
    }
    return [...unique.values()].sort((a, b) => {
      const factor = ascending ? 1 : -1;
      if (sort === "size") return ((a.size ?? 0) - (b.size ?? 0)) * factor;
      if (sort === "modified")
        return (Date.parse(a.lastModified) - Date.parse(b.lastModified)) * factor;
      return basename(a.key).localeCompare(basename(b.key), locale, { numeric: true }) * factor;
    });
  }, [findRequest, foundObjects, list.data, sort, ascending, locale]);

  const folders = useMemo(() => {
    if (findRequest) return [];
    const unique = new Set(list.data?.pages.flatMap((page) => page.prefixes) ?? []);
    return [...unique].sort((a, b) =>
      basename(a).localeCompare(basename(b), locale, { numeric: true }),
    );
  }, [findRequest, list.data, locale]);

  const selected = objects.filter((item) => selection.has(item.key));
  const canWrite = Boolean(profile && !profile.readOnly);

  const navigate = useCallback((nextPrefix: string) => {
    setPrefix(nextPrefix);
    setFindRequest(null);
    setSearch("");
    setSelection(new Set());
  }, []);

  const refresh = useCallback(() => {
    void queryClient.invalidateQueries({ queryKey: ["buckets", profileId] });
    void queryClient.invalidateQueries({ queryKey: ["objects", profileId, bucket, prefix] });
    if (findRequest) void queryClient.invalidateQueries({ queryKey: ["find"] });
  }, [queryClient, profileId, bucket, prefix, findRequest]);

  const tasks = useQuery({
    queryKey: ["transfers"],
    queryFn: api.Tasks,
    enabled: desktop,
    refetchInterval: transfersOpen ? 1000 : 2500,
  });

  const previousStatuses = useRef<Map<string, string>>(new Map());
  useEffect(() => {
    if (!tasks.data) return;
    let finished = false;
    for (const task of tasks.data) {
      const previous = previousStatuses.current.get(task.id);
      if (
        previous &&
        previous !== task.status &&
        (task.status === "completed" || task.status === "failed" || task.status === "cancelled")
      ) {
        finished = true;
      }
      previousStatuses.current.set(task.id, task.status);
    }
    if (finished) void queryClient.invalidateQueries({ queryKey: ["objects"] });
  }, [tasks.data, queryClient]);

  useEffect(() => {
    if (!desktop) return;
    let dispose: (() => void) | undefined;
    let cancelled = false;
    void onFilesDropped((paths) => {
      const current = profiles.data?.find((item) => item.id === profileId);
      if (!current || current.readOnly || !bucket) return;
      void api
        .UploadPaths(current.id, bucket, prefix, paths)
        .then((results) => {
          enqueuePlans(results.flatMap((result) => (result.plan ? [result.plan] : [])));
          notify(t("uploadQueued", { count: results.filter((result) => result.task).length }));
          void queryClient.invalidateQueries({ queryKey: ["transfers"] });
        })
        .catch((error) => notify(errorMessage(error)));
    }).then((off) => {
      if (cancelled) off();
      else dispose = off;
    });
    return () => {
      cancelled = true;
      dispose?.();
    };
  }, [desktop, profiles.data, profileId, bucket, prefix, notify, queryClient, t, enqueuePlans]);

  const runSearch = () => {
    if (!profile || !bucket) return;
    const term = search.trim();
    if (!term) {
      setFindRequest(null);
      return;
    }
    setFindRequest({
      profileId: profile.id,
      bucket,
      prefix,
      contains: term,
      before: "",
      minSize: 0,
      maxSize: 0,
      limit: 1000,
      cursor: "",
    });
    setSelection(new Set());
  };

  const onUpload = async (directory: boolean) => {
    if (!profile || !bucket || !canWrite) return;
    try {
      const paths = await api.PickFiles(directory);
      if (!paths.length) return;
      const results = await api.UploadPaths(profile.id, bucket, prefix, paths);
      enqueuePlans(results.flatMap((result) => (result.plan ? [result.plan] : [])));
      notify(t("uploadQueued", { count: results.filter((result) => result.task).length }));
      void queryClient.invalidateQueries({ queryKey: ["transfers"] });
    } catch (error) {
      notify(errorMessage(error));
    }
  };

  const onDownload = async (object: ObjectInfo) => {
    try {
      const filename = await api.PickDownload(basename(object.key));
      if (!filename) return;
      await api.StartTransfer({
        direction: "download",
        object,
        localPath: filename,
        expectedSha256: "",
      });
      notify(t("downloadStarted"));
      void queryClient.invalidateQueries({ queryKey: ["transfers"] });
      setTransfersOpen(true);
    } catch (error) {
      notify(errorMessage(error));
    }
  };

  const createPlan = useMutation({
    mutationFn: (input: {
      operation: string;
      sources: ObjectInfo[];
      target?: { profileId: string; bucket: string; key: string };
    }) =>
      api.CreatePlan({
        operation: input.operation,
        sources: input.sources.map((item) => ({
          profileId: item.profileId,
          bucket: item.bucket,
          key: item.key,
        })),
        target: input.target ?? { profileId: "", bucket: "", key: "" },
        localPath: "",
      }),
    onSuccess: (value) => enqueuePlans([value]),
    onError: (error) => notify(errorMessage(error)),
  });

  const execute = async () => {
    if (!plan) return;
    try {
      const result = await api.ExecutePlan(plan.id, true);
      setPlan(result);
      void queryClient.invalidateQueries({ queryKey: ["objects"] });
      void queryClient.invalidateQueries({ queryKey: ["transfers"] });
      setSelection(new Set());
    } catch (error) {
      throw new Error(errorMessage(error));
    }
  };

  const removeProfile = async () => {
    if (!removing) return;
    await api.RemoveProfile(removing.id, true);
    setRemoving(null);
    await queryClient.invalidateQueries({ queryKey: ["profiles"] });
    notify(t("connectionRemoved"));
  };

  const crumbs = breadcrumbs(prefix);

  return (
    <div className="app-frame">
      <TitleBar />
      <div className="app-shell">
        <aside className="sidebar">
          <div className="sidebar-section">
            <div className="sidebar-heading">
              <span>{t("connections")}</span>
              <IconButton
                label={t("addConnection")}
                onClick={() => setConnection({})}
                disabled={!desktop}
              >
                <Plus size={16} />
              </IconButton>
            </div>
            {!desktop ? <p className="sidebar-note">{t("desktopHint")}</p> : null}
            {profiles.isPending && desktop ? <Spinner /> : null}
            {profiles.error ? (
              <ErrorBanner
                message={errorMessage(profiles.error)}
                onRetry={() => void profiles.refetch()}
              />
            ) : null}
            <div className="connections">
              {profiles.data?.map((item) => (
                <div
                  key={item.id}
                  className={`connection-row ${item.id === profileId ? "active" : ""}`}
                >
                  <button
                    type="button"
                    className="connection-main"
                    onClick={() => setProfileId(item.id)}
                  >
                    <span className="connection-icon">
                      <Cloud size={17} />
                    </span>
                    <span className="connection-copy">
                      <strong>{item.name}</strong>
                      <small>{item.bucket || t("allBuckets")}</small>
                    </span>
                    {item.readOnly ? <LockKeyhole size={13} /> : null}
                  </button>
                  <IconButton
                    label={t("manageConnection")}
                    onClick={() => setConnection({ profile: item })}
                  >
                    <MoreHorizontal size={16} />
                  </IconButton>
                </div>
              ))}
            </div>
            {desktop && profiles.data?.length === 0 ? (
              <button type="button" className="sidebar-cta" onClick={() => setConnection({})}>
                <Plus size={16} />
                {t("addConnection")}
              </button>
            ) : null}
          </div>
          {profile ? (
            <div className="sidebar-section buckets-section">
              <div className="sidebar-heading">
                <span>{t("buckets")}</span>
                <IconButton label={t("refreshHint")} onClick={refresh}>
                  <RefreshCw size={14} />
                </IconButton>
              </div>
              {buckets.isPending ? <Spinner /> : null}
              {buckets.error ? <ErrorBanner message={errorMessage(buckets.error)} /> : null}
              {buckets.data?.length === 0 ? <p className="sidebar-note">{t("noBuckets")}</p> : null}
              {buckets.data?.map((item: Bucket) => (
                <button
                  key={item.name}
                  type="button"
                  className={`bucket-row ${bucket === item.name ? "active" : ""}`}
                  onClick={() => setBucket(item.name)}
                >
                  <HardDrive size={16} />
                  <span>{item.name}</span>
                </button>
              ))}
            </div>
          ) : null}
          <div className="sidebar-footer">
            <button type="button" className="footer-link" onClick={() => setTransfersOpen(true)}>
              <ArrowUpDown size={16} />
              <span>{t("transfers")}</span>
              {tasks.data?.some((task) => task.status === "running" || task.status === "queued") ? (
                <span className="pulse-dot" />
              ) : null}
            </button>
            <button type="button" className="footer-link" onClick={() => setSettings(true)}>
              <Settings size={16} />
              <span>{t("settings")}</span>
            </button>
          </div>
        </aside>

        <main className="workspace">
          {!desktop ? (
            <div className="browser-banner">
              <LockKeyhole size={16} />
              {t("desktopHint")}
            </div>
          ) : null}
          <header className="topbar">
            <div className="topbar-title">
              <p className="eyebrow">{profile?.name ?? t("workspace")}</p>
              <h1>{bucket || t("objects")}</h1>
            </div>
            <div className="topbar-actions">
              <div className="searchbox">
                <Search size={16} />
                <input
                  value={search}
                  onChange={(event) => setSearch(event.target.value)}
                  onKeyDown={(event) => {
                    if (event.key === "Enter") runSearch();
                    if (event.key === "Escape") {
                      setSearch("");
                      setFindRequest(null);
                    }
                  }}
                  placeholder={t("search")}
                  aria-label={t("searchLabel")}
                  disabled={!bucket}
                />
                {search ? (
                  <IconButton
                    label={t("clearSearch")}
                    onClick={() => {
                      setSearch("");
                      setFindRequest(null);
                    }}
                  >
                    <X size={14} />
                  </IconButton>
                ) : null}
              </div>
              <IconButton label={t("refresh")} onClick={refresh} disabled={!bucket}>
                <RefreshCw size={17} />
              </IconButton>
              <Button
                variant="secondary"
                onClick={() => void onUpload(false)}
                disabled={!canWrite || !bucket}
              >
                <Upload size={16} />
                {t("upload")}
              </Button>
              <Button
                variant="secondary"
                onClick={() => void onUpload(true)}
                disabled={!canWrite || !bucket}
              >
                <FolderUp size={16} />
                {t("uploadFolder")}
              </Button>
            </div>
          </header>

          <div className="contextbar">
            <nav className="breadcrumbs" aria-label={t("currentFolder")}>
              <button type="button" onClick={() => navigate("")}>
                {bucket || t("objects")}
              </button>
              {crumbs.map((crumb) => (
                <span key={crumb.prefix} className="crumb">
                  <ChevronRight size={13} />
                  <button type="button" onClick={() => navigate(crumb.prefix)}>
                    {crumb.name}
                  </button>
                </span>
              ))}
            </nav>
            <div className="view-actions">
              <span className="muted small">
                {t("loadedCount", { count: objects.length + folders.length })}
              </span>
              <IconButton
                label={t("listView")}
                onClick={() => ui.setView("list")}
                aria-pressed={ui.view === "list"}
              >
                <List size={17} />
              </IconButton>
              <IconButton
                label={t("gridView")}
                onClick={() => ui.setView("grid")}
                aria-pressed={ui.view === "grid"}
              >
                <Grid3X3 size={17} />
              </IconButton>
            </div>
          </div>

          {pendingCount > 0 && !plan ? (
            <div className="plan-toast">
              <AlertTriangle size={16} />
              <span>{t("pendingPlans", { count: pendingCount })}</span>
            </div>
          ) : null}

          <section
            className="content"
            data-file-drop-target=""
            aria-busy={list.isPending || found.isPending}
          >
            {!profile ? (
              <Welcome desktop={desktop} onAdd={() => setConnection({})} />
            ) : !bucket ? (
              <div className="empty-folder">
                <span className="empty-icon">
                  <HardDrive size={28} />
                </span>
                <h2>{t("selectBucket")}</h2>
                <p>{buckets.data?.length === 0 ? t("noBuckets") : t("selectBucketHint")}</p>
                {buckets.data?.length === 0 ? (
                  <Button variant="secondary" onClick={refresh}>
                    <RefreshCw size={16} />
                    {t("refresh")}
                  </Button>
                ) : null}
              </div>
            ) : list.error || found.error ? (
              <div className="state-card">
                <ErrorBanner
                  message={errorMessage(list.error ?? found.error)}
                  onRetry={() => {
                    if (findRequest) void found.refetch();
                    else void list.refetch();
                  }}
                />
              </div>
            ) : list.isPending || found.isPending ? (
              <div className="state-card">
                <Spinner label={t("loading")} />
              </div>
            ) : objects.length === 0 && folders.length === 0 ? (
              <EmptyFolder
                search={Boolean(findRequest)}
                readOnly={!canWrite}
                onUpload={() => void onUpload(false)}
                drop={desktop}
              />
            ) : (
              <ObjectTable
                view={ui.view}
                folders={folders}
                objects={objects}
                selected={selection}
                onToggle={(object, checked) =>
                  setSelection((current) => {
                    const next = new Set(current);
                    if (checked) next.add(object.key);
                    else next.delete(object.key);
                    return next;
                  })
                }
                onToggleAll={(checked) =>
                  setSelection(checked ? new Set(objects.map((object) => object.key)) : new Set())
                }
                onOpenFolder={navigate}
                onOpenPreview={setPreview}
                onSort={(value) => {
                  if (sort === value) setAscending(!ascending);
                  else {
                    setSort(value);
                    setAscending(true);
                  }
                }}
                sort={sort}
                ascending={ascending}
                locale={locale}
                loading={list.isFetchingNextPage}
              />
            )}
            {findRequest ? (
              <div className="search-footer">
                <span>{t("searchScanned", { count: searchScanned })}</span>
                {searchComplete ? (
                  <span className="tag">{t("endOfFolder")}</span>
                ) : (
                  <Button
                    variant="secondary"
                    onClick={() => void found.fetchNextPage()}
                    disabled={found.isFetchingNextPage}
                  >
                    {found.isFetchingNextPage ? <LoaderCircle className="spin" size={15} /> : null}
                    {t("loadMore")}
                  </Button>
                )}
              </div>
            ) : list.hasNextPage ? (
              <div className="search-footer">
                <Button
                  variant="secondary"
                  onClick={() => void list.fetchNextPage()}
                  disabled={list.isFetchingNextPage}
                >
                  {list.isFetchingNextPage ? <LoaderCircle className="spin" size={15} /> : null}
                  {t("loadMore")}
                </Button>
              </div>
            ) : null}
          </section>

          {selected.length > 0 ? (
            <div className="selection-bar" role="toolbar">
              <span className="selection-count">{t("selected", { count: selected.length })}</span>
              <Button variant="ghost" onClick={() => setSelection(new Set())}>
                {t("clearSelection")}
              </Button>
              <div className="spacer" />
              {selected.length === 1 ? (
                <Button variant="secondary" onClick={() => void onDownload(selected[0])}>
                  <Download size={15} />
                  {t("download")}
                </Button>
              ) : null}
              {selected.length === 1 ? (
                <Button
                  variant="secondary"
                  onClick={() => {
                    setSharing(selected[0]);
                    setShareLink(null);
                  }}
                >
                  <Share2 size={15} />
                  {t("share")}
                </Button>
              ) : null}
              <Button
                variant="secondary"
                disabled={!canWrite || selection.size > 1000}
                onClick={() => createPlan.mutate({ operation: "copy", sources: selected })}
              >
                <Copy size={15} />
                {t("copy")}
              </Button>
              <Button
                variant="secondary"
                disabled={!canWrite || selection.size > 1000}
                onClick={() => setMode({ kind: "move", objects: selected })}
              >
                <FolderDown size={15} />
                {t("move")}
              </Button>
              <Button
                variant="danger"
                disabled={!canWrite || selection.size > 1000}
                onClick={() => createPlan.mutate({ operation: "delete", sources: selected })}
              >
                <Trash2 size={15} />
                {t("delete")}
              </Button>
              {selection.size > 1000 ? (
                <span className="muted small">{t("selectionLimit")}</span>
              ) : null}
            </div>
          ) : null}
        </main>

        {transfersOpen ? (
          <TransferPanel
            tasks={tasks.data ?? []}
            onClose={() => setTransfersOpen(false)}
            onChanged={() => void queryClient.invalidateQueries({ queryKey: ["transfers"] })}
            notify={notify}
          />
        ) : null}

        {connection ? (
          <ConnectionModal
            profile={connection.profile}
            onRemove={(target) => {
              setConnection(null);
              setRemoving(target);
            }}
            onClose={() => setConnection(null)}
            onSaved={(saved) => {
              setConnection(null);
              setProfileId(saved.id);
              void queryClient.invalidateQueries({ queryKey: ["profiles"] });
              notify(t("connectionSaved"));
            }}
          />
        ) : null}

        {removing ? (
          <ConfirmDialog
            title={t("removeConnection")}
            message={t("removeHint", { name: removing.name })}
            confirmLabel={t("remove")}
            danger
            onClose={() => setRemoving(null)}
            onConfirm={removeProfile}
          />
        ) : null}

        {sharing ? (
          <ShareModal
            object={sharing}
            profile={profile}
            link={shareLink}
            setLink={setShareLink}
            notify={notify}
            onClose={() => {
              setSharing(null);
              setShareLink(null);
            }}
          />
        ) : null}

        {preview ? (
          <PreviewModal
            object={preview}
            onClose={() => setPreview(null)}
            onDownload={() => void onDownload(preview)}
          />
        ) : null}

        {plan ? (
          <PlanModal
            plan={plan}
            onClose={closePlan}
            onExecute={() => void runExecute(execute)}
            busy={executeState.busy}
            error={executeState.error}
            pending={pendingCount}
          />
        ) : null}

        {mode ? (
          <DestinationModal
            mode={mode}
            current={{ profileId, bucket, key: prefix }}
            onClose={() => setMode(null)}
            onCreate={(target) => {
              setMode(null);
              createPlan.mutate({ operation: mode.kind, sources: mode.objects, target });
            }}
          />
        ) : null}

        {settings ? (
          <SettingsModal
            onClose={() => setSettings(false)}
            profile={profile}
            bucket={bucket}
            prefix={prefix}
            notify={notify}
          />
        ) : null}

        {toast ? (
          <div className="toast" role="status">
            {toast}
          </div>
        ) : null}
      </div>
    </div>
  );
}

function Welcome({ desktop, onAdd }: { desktop: boolean; onAdd: () => void }) {
  const { t } = useTranslation();
  return (
    <div className="welcome">
      <p className="eyebrow">{t("welcomeEyebrow")}</p>
      <h2>{t("emptyTitle")}</h2>
      <p className="welcome-lead">{t("emptyDescription")}</p>
      <div className="welcome-actions">
        <Button variant="primary" onClick={onAdd} disabled={!desktop}>
          <Plus size={17} />
          {t("addConnection")}
        </Button>
        <span className="welcome-privacy">
          <LockKeyhole size={14} />
          {t("welcomePrivacy")}
        </span>
      </div>
      <div className="feature-grid">
        <article>
          <span className="feature-icon">
            <Search size={19} />
          </span>
          <h3>{t("welcomeBrowse")}</h3>
          <p>{t("welcomeBrowseHint")}</p>
        </article>
        <article>
          <span className="feature-icon">
            <Share2 size={19} />
          </span>
          <h3>{t("welcomeShare")}</h3>
          <p>{t("welcomeShareHint")}</p>
        </article>
        <article>
          <span className="feature-icon">
            <ArrowUpDown size={19} />
          </span>
          <h3>{t("welcomeTransfer")}</h3>
          <p>{t("welcomeTransferHint")}</p>
        </article>
      </div>
      <p className="welcome-footer">{t("welcomeFooter")}</p>
    </div>
  );
}

function EmptyFolder({
  search,
  readOnly,
  onUpload,
  drop,
}: {
  search: boolean;
  readOnly: boolean;
  onUpload: () => void;
  drop: boolean;
}) {
  const { t } = useTranslation();
  return (
    <div className="empty-folder">
      <span className="empty-icon">
        <Folder size={28} />
      </span>
      <h2>{search ? t("noSearchResults") : readOnly ? t("emptyReadOnly") : t("emptyBucket")}</h2>
      <p>{search ? t("noSearchHint") : readOnly ? t("readOnlyActions") : t("emptyBucketHint")}</p>
      {!search && !readOnly ? (
        <Button variant="primary" onClick={onUpload}>
          <Upload size={16} />
          {t("upload")}
        </Button>
      ) : null}
      {drop && !readOnly ? <p className="drop-hint">{t("dropHint")}</p> : null}
    </div>
  );
}

function ObjectTable({
  view,
  folders,
  objects,
  selected,
  onToggle,
  onToggleAll,
  onOpenFolder,
  onOpenPreview,
  onSort,
  sort,
  ascending,
  locale,
  loading,
}: {
  view: "list" | "grid";
  folders: string[];
  objects: ObjectInfo[];
  selected: Set<string>;
  onToggle: (object: ObjectInfo, checked: boolean) => void;
  onToggleAll: (checked: boolean) => void;
  onOpenFolder: (prefix: string) => void;
  onOpenPreview: (object: ObjectInfo) => void;
  onSort: (value: "name" | "size" | "modified") => void;
  sort: string;
  ascending: boolean;
  locale: string;
  loading: boolean;
}) {
  const { t } = useTranslation();
  const scrollRef = useRef<HTMLDivElement>(null);
  const items = useMemo(
    () => [
      ...folders.map((value) => ({ kind: "folder" as const, value })),
      ...objects.map((value) => ({ kind: "object" as const, value })),
    ],
    [folders, objects],
  );
  const virtualizer = useVirtualizer({
    count: items.length,
    getScrollElement: () => scrollRef.current,
    estimateSize: () => 52,
    overscan: 8,
  });
  const allSelected = objects.length > 0 && objects.every((object) => selected.has(object.key));

  if (view === "grid") {
    return (
      <div className="object-grid">
        {items.map((item) =>
          item.kind === "folder" ? (
            <button
              type="button"
              key={item.value}
              className="object-card folder-card"
              onClick={() => onOpenFolder(item.value)}
            >
              <span className="object-card-icon">
                <Folder size={26} />
              </span>
              <span className="object-card-name">{basename(item.value)}</span>
              <span className="object-card-meta">{t("folder")}</span>
            </button>
          ) : (
            <div
              key={item.value.key}
              className={`object-card ${selected.has(item.value.key) ? "selected" : ""}`}
            >
              <label className="object-card-check">
                <input
                  type="checkbox"
                  checked={selected.has(item.value.key)}
                  onChange={(event) => onToggle(item.value, event.target.checked)}
                  aria-label={t("selectObject", { name: basename(item.value.key) })}
                />
              </label>
              <button
                type="button"
                className="object-card-open"
                onClick={() => onOpenPreview(item.value)}
              >
                <span className="object-card-icon">
                  <FileIcon object={item.value} />
                </span>
                <span className="object-card-name">{basename(item.value.key)}</span>
                <span className="object-card-meta">{formatBytes(item.value.size, locale)}</span>
              </button>
            </div>
          ),
        )}
      </div>
    );
  }
  return (
    <div className="table-wrap" ref={scrollRef}>
      <table className="object-table">
        <thead>
          <tr>
            <th className="check-col">
              <input
                type="checkbox"
                checked={allSelected}
                onChange={(event) => onToggleAll(event.target.checked)}
                aria-label={t("selectAll")}
              />
            </th>
            <th>
              <button type="button" onClick={() => onSort("name")} className="sort-button">
                {t("name")}
                {sort === "name" ? (
                  <ChevronUp size={13} className={ascending ? "" : "flip"} />
                ) : (
                  <ArrowUpDown size={12} />
                )}
              </button>
            </th>
            <th>
              <button type="button" onClick={() => onSort("size")} className="sort-button">
                {t("size")}
                {sort === "size" ? (
                  <ChevronUp size={13} className={ascending ? "" : "flip"} />
                ) : null}
              </button>
            </th>
            <th>
              <button type="button" onClick={() => onSort("modified")} className="sort-button">
                {t("modified")}
                {sort === "modified" ? (
                  <ChevronUp size={13} className={ascending ? "" : "flip"} />
                ) : null}
              </button>
            </th>
            <th className="type-col">{t("type")}</th>
          </tr>
        </thead>
        <tbody style={{ height: virtualizer.getTotalSize() }}>
          {virtualizer.getVirtualItems().map((virtualRow) => {
            const item = items[virtualRow.index];
            if (item.kind === "folder") {
              return (
                <tr
                  key={item.value}
                  className="virtual-row folder-row"
                  style={{ transform: `translateY(${virtualRow.start}px)` }}
                >
                  <td />
                  <td colSpan={4}>
                    <button
                      type="button"
                      className="object-name folder-name"
                      onClick={() => onOpenFolder(item.value)}
                    >
                      <Folder size={17} />
                      <span>{basename(item.value)}</span>
                      <ChevronRight size={14} className="folder-chevron" />
                    </button>
                  </td>
                </tr>
              );
            }
            const object = item.value;
            return (
              <tr
                key={object.key}
                className={`virtual-row ${selected.has(object.key) ? "selected" : ""}`}
                style={{ transform: `translateY(${virtualRow.start}px)` }}
              >
                <td className="check-col">
                  <input
                    type="checkbox"
                    checked={selected.has(object.key)}
                    onChange={(event) => onToggle(object, event.target.checked)}
                    aria-label={t("selectObject", { name: basename(object.key) })}
                  />
                </td>
                <td>
                  <button
                    type="button"
                    className="object-name"
                    onClick={() => onOpenPreview(object)}
                  >
                    <FileIcon object={object} />
                    <span>{basename(object.key)}</span>
                  </button>
                </td>
                <td className="muted">{formatBytes(object.size, locale)}</td>
                <td className="muted">{formatDate(object.lastModified, locale)}</td>
                <td className="muted type-col">{displayType(object.key, object.contentType)}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
      {loading ? (
        <div className="table-loading">
          <LoaderCircle className="spin" size={16} />
          {t("loading")}
        </div>
      ) : null}
    </div>
  );
}

function FileIcon({ object }: { object: ObjectInfo }) {
  const kind = previewKind(object.key, object.contentType);
  if (kind === "image") return <ImageIcon size={17} />;
  if (kind === "video") return <Film size={17} />;
  if (kind === "audio") return <Music size={17} />;
  if (kind === "text") return <FileText size={17} />;
  return <File size={17} />;
}

function PreviewModal({
  object,
  onClose,
  onDownload,
}: {
  object: ObjectInfo;
  onClose: () => void;
  onDownload: () => void;
}) {
  const { t, i18n } = useTranslation();
  const locale = i18n.language === "zh" ? "zh-CN" : "en-US";
  const kind = previewKind(object.key, object.contentType);
  const [link, setLink] = useState<ShareLink | null>(null);
  const [text, setText] = useState<TextPreview | null>(null);
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      if (kind === "text") setText(await api.ReadText(object));
      else if (kind !== "file") setLink(await api.Share(object, 3600, false));
    } catch (value) {
      setError(errorMessage(value));
    } finally {
      setLoading(false);
    }
  }, [kind, object]);
  useEffect(() => {
    void load();
  }, [load]);

  const body: ReactNode = (() => {
    if (kind === "file")
      return (
        <div className="preview-placeholder">
          <File size={38} />
          <p>{t("previewUnavailable")}</p>
          <small>{t("previewDownload")}</small>
        </div>
      );
    if (loading)
      return (
        <div className="preview-placeholder">
          <Spinner label={t("previewLoading")} />
        </div>
      );
    if (error)
      return (
        <div className="preview-placeholder">
          <AlertTriangle size={28} />
          <p>{t("previewError")}</p>
          <small>{error}</small>
          <Button variant="secondary" onClick={() => void load()}>
            {t("reloadPreview")}
          </Button>
        </div>
      );
    if (kind === "text" && text)
      return (
        <div className="text-preview">
          <pre>{text.text}</pre>
          {text.truncated ? <p className="muted small">{t("textTruncated")}</p> : null}
        </div>
      );
    if (!link) return null;
    if (Object.keys(link.headers ?? {}).length)
      return (
        <div className="preview-placeholder">
          <LockKeyhole size={28} />
          <p>{t("previewHeaders")}</p>
        </div>
      );
    if (kind === "image")
      return (
        <div className="media-preview">
          <img src={link.url} alt={basename(object.key)} />
        </div>
      );
    if (kind === "video")
      return (
        <div className="media-preview">
          {/* biome-ignore lint/a11y/useMediaCaption: remote object previews have no caption track */}
          <video src={link.url} controls />
        </div>
      );
    if (kind === "audio")
      return (
        <div className="preview-placeholder audio-preview">
          <Music size={38} />
          {/* biome-ignore lint/a11y/useMediaCaption: remote object previews have no caption track */}
          <audio src={link.url} controls />
        </div>
      );
    if (kind === "pdf")
      return <iframe className="pdf-preview" src={link.url} title={basename(object.key)} />;
    return null;
  })();

  return (
    <Modal
      wide
      title={basename(object.key)}
      subtitle={formatBytes(object.size, locale)}
      onClose={onClose}
      footer={
        <>
          <Button onClick={onDownload}>
            <Download size={16} />
            {t("download")}
          </Button>
          <Button onClick={onClose}>{t("close")}</Button>
        </>
      }
    >
      <div className="preview-meta">
        <span className="mono break-all">{object.key}</span>
        <span className="tag">{displayType(object.key, object.contentType)}</span>
      </div>
      {body}
      {kind !== "text" && kind !== "file" ? (
        <p className="muted small preview-privacy">{t("previewPrivacy")}</p>
      ) : null}
    </Modal>
  );
}

function ShareModal({
  object,
  profile,
  link,
  setLink,
  notify,
  onClose,
}: {
  object: ObjectInfo;
  profile: Profile | null;
  link: ShareLink | null;
  setLink: (link: ShareLink | null) => void;
  notify: (value: string) => void;
  onClose: () => void;
}) {
  const { t, i18n } = useTranslation();
  const locale = i18n.language === "zh" ? "zh-CN" : "en-US";
  const [seconds, setSeconds] = useState(3600);
  const [publicLink, setPublicLink] = useState(false);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState("");
  const create = async () => {
    setBusy(true);
    setError("");
    try {
      setLink(await api.Share(object, seconds, publicLink));
    } catch (value) {
      setError(errorMessage(value));
    } finally {
      setBusy(false);
    }
  };
  const copy = async () => {
    if (!link) return;
    await api.CopyText(link.url);
    notify(t("copied"));
  };
  return (
    <Modal
      title={t("share")}
      subtitle={object.key}
      onClose={onClose}
      footer={
        <>
          <Button onClick={onClose}>{t("close")}</Button>
          {link ? (
            <Button variant="primary" onClick={() => void copy()}>
              <Clipboard size={16} />
              {t("copyLink")}
            </Button>
          ) : (
            <Button variant="primary" onClick={() => void create()} disabled={busy}>
              {busy ? <LoaderCircle className="spin" size={16} /> : <Share2 size={16} />}
              {t("generateLink")}
            </Button>
          )}
        </>
      }
    >
      <div className="share-form">
        <fieldset className="segmented-field">
          <legend>{t("linkExpiry")}</legend>
          <div className="segmented">
            <button
              type="button"
              aria-pressed={!publicLink && seconds === 3600}
              onClick={() => {
                setPublicLink(false);
                setSeconds(3600);
                setLink(null);
              }}
            >
              {t("hour")}
            </button>
            <button
              type="button"
              aria-pressed={!publicLink && seconds === 604800}
              onClick={() => {
                setPublicLink(false);
                setSeconds(604800);
                setLink(null);
              }}
            >
              {t("week")}
            </button>
            <button
              type="button"
              aria-pressed={publicLink}
              onClick={() => {
                setPublicLink(true);
                setLink(null);
              }}
              disabled={!profile?.publicUrl}
            >
              {t("publicLink")}
            </button>
          </div>
        </fieldset>
        {publicLink ? (
          <p className="field-hint">
            {t("publicLinkHint")} {t("publicDomainScope")}
          </p>
        ) : (
          <p className="field-hint">{t("shareHint")}</p>
        )}
        {!profile?.publicUrl ? <p className="field-hint">{t("noPublicDomain")}</p> : null}
        {link ? (
          <div className="link-result">
            <div className="link-value mono">{link.url}</div>
            <div className="link-meta">
              {link.public
                ? t("permanentPublic")
                : t("expires", { date: formatDate(link.expiresAt, locale) })}
            </div>
            {link.headers && Object.keys(link.headers).length ? (
              <div className="headers">
                <strong>{t("requiredHeaders")}</strong>
                {Object.entries(link.headers).map(([key, value]) => (
                  <code key={key}>
                    {key}: {value}
                  </code>
                ))}
              </div>
            ) : null}
          </div>
        ) : null}
        {error ? <ErrorBanner message={error} /> : null}
      </div>
    </Modal>
  );
}

function PlanModal({
  plan,
  onClose,
  onExecute,
  busy,
  error,
  pending,
}: {
  plan: OperationPlan;
  onClose: () => void;
  onExecute: () => void;
  busy: boolean;
  error: string;
  pending: number;
}) {
  const { t, i18n } = useTranslation();
  const locale = i18n.language === "zh" ? "zh-CN" : "en-US";
  const label =
    plan.operation === "delete"
      ? t("delete")
      : plan.operation === "move"
        ? t("move")
        : plan.operation === "copy"
          ? t("copy")
          : t("upload");
  const warning =
    plan.operation === "delete"
      ? t("deleteWarning")
      : plan.operation === "move"
        ? t("moveWarning")
        : plan.operation === "copy"
          ? t("copyWarning")
          : t("uploadWarning");
  const expired = Date.parse(plan.expiresAt) < Date.now();
  const canExecute = plan.status === "pending" && !expired;
  const done =
    plan.status === "completed" || plan.status === "partial" || plan.status === "running";
  return (
    <Modal
      wide
      title={canExecute ? t("reviewPlan") : t("planResult")}
      subtitle={`${label} · ${t("exactItems", { count: plan.items.length })}`}
      onClose={onClose}
      busy={busy}
      footer={
        canExecute ? (
          <>
            <Button onClick={onClose} disabled={busy}>
              {t("cancel")}
            </Button>
            <Button variant="danger" onClick={onExecute} disabled={busy}>
              {busy ? <LoaderCircle className="spin" size={16} /> : <Check size={16} />}
              {t("executePlan", { operation: label })}
            </Button>
          </>
        ) : (
          <Button onClick={onClose}>{t("dismiss")}</Button>
        )
      }
    >
      <div className="plan-summary">
        <div>
          <small>{t("exactTotal")}</small>
          <strong>{exactBytes(plan.bytes, locale)}</strong>
        </div>
        <div>
          <small>{t("items")}</small>
          <strong>{plan.items.length}</strong>
        </div>
        <div>
          <small>{t("overwrite")}</small>
          <strong>{plan.items.filter((item) => item.existing).length}</strong>
        </div>
        <div>
          <small>{t("planStatus")}</small>
          <strong>{t(plan.status)}</strong>
        </div>
      </div>
      {plan.warning ? (
        <div className="notice warning">
          <AlertTriangle size={17} />
          <span>{plan.warning}</span>
        </div>
      ) : null}
      <div className="notice">
        <AlertTriangle size={17} />
        <span>{warning}</span>
      </div>
      {expired && plan.status === "pending" ? (
        <div className="notice danger">
          <AlertTriangle size={17} />
          <span>{t("expiredPlan")}</span>
        </div>
      ) : null}
      <div className="plan-table-wrap">
        <table className="plan-table">
          <thead>
            <tr>
              <th>{t("source")}</th>
              <th>{t("target")}</th>
              <th>{t("size")}</th>
              <th>{t("overwrite")}</th>
            </tr>
          </thead>
          <tbody>
            {plan.items.map((item) => (
              <tr key={`${item.source.key}|${item.target?.key ?? ""}|${item.localPath ?? ""}`}>
                <td className="mono break-all">{item.source.key || item.localPath}</td>
                <td className="mono break-all">{item.target?.key || "—"}</td>
                <td>{formatBytes(item.source.size || item.localSize || 0, locale)}</td>
                <td>
                  {item.existing ? (
                    <span className="tag warn">{t("existingObject")}</span>
                  ) : (
                    <span className="tag">{t("newObject")}</span>
                  )}
                </td>
              </tr>
            ))}
          </tbody>
        </table>
      </div>
      {done ? (
        <div className="plan-results">
          <h3>{t("resultDetails")}</h3>
          {plan.results.length === 0 ? (
            <p className="muted">{t("working")}</p>
          ) : (
            plan.results.map((result) => (
              <div
                key={`${result.source.key}|${result.target?.key ?? ""}|${result.status}|${result.error?.code ?? ""}`}
                className={`result-row ${result.error ? "failed" : ""}`}
              >
                {result.error ? <AlertTriangle size={15} /> : <Check size={15} />}
                <span className="mono break-all">{result.source.key || result.target?.key}</span>
                <span>{result.error ? result.error.message : t(result.status)}</span>
              </div>
            ))
          )}
        </div>
      ) : null}
      {canExecute ? (
        <p className="muted small">
          {t("planExpires", { date: formatDate(plan.expiresAt, locale) })} · {t("planHint")}
        </p>
      ) : null}
      {pending > 1 && canExecute ? (
        <p className="muted small">{t("pendingPlans", { count: pending - 1 })}</p>
      ) : null}
      {error ? <ErrorBanner message={error} /> : null}
    </Modal>
  );
}

function DestinationModal({
  mode,
  current,
  onClose,
  onCreate,
}: {
  mode: NonNullable<Mode>;
  current: { profileId: string; bucket: string; key: string };
  onClose: () => void;
  onCreate: (target: { profileId: string; bucket: string; key: string }) => void;
}) {
  const { t } = useTranslation();
  const [key, setKey] = useState(mode.objects.length === 1 ? mode.objects[0].key : current.key);
  return (
    <Modal
      title={mode.kind === "move" ? t("move") : t("copy")}
      subtitle={t("exactItems", { count: mode.objects.length })}
      onClose={onClose}
      footer={
        <>
          <Button onClick={onClose}>{t("cancel")}</Button>
          <Button
            variant="primary"
            disabled={!key.trim()}
            onClick={() =>
              onCreate({ profileId: current.profileId, bucket: current.bucket, key: key.trim() })
            }
          >
            {t("createPlan")}
          </Button>
        </>
      }
    >
      <label className="field">
        <span>{t("destinationKey")}</span>
        <input
          value={key}
          onChange={(event) => setKey(event.target.value)}
          className="mono"
          spellCheck={false}
        />
        <small>{t("renameHint")}</small>
      </label>
      {mode.objects.length > 1 ? <p className="field-hint">{t("multipleDestination")}</p> : null}
      <p className="muted small">{current.bucket}</p>
    </Modal>
  );
}

function TransferPanel({
  tasks,
  onClose,
  onChanged,
  notify,
}: {
  tasks: TransferTask[];
  onClose: () => void;
  onChanged: () => void;
  notify: (value: string) => void;
}) {
  const { t, i18n } = useTranslation();
  const locale = i18n.language === "zh" ? "zh-CN" : "en-US";
  const act = async (action: () => Promise<unknown>, message: string) => {
    try {
      await action();
      onChanged();
      notify(message);
    } catch (error) {
      notify(errorMessage(error));
    }
  };
  return (
    <aside className="transfer-panel">
      <header>
        <div>
          <p className="eyebrow">{t("transfers")}</p>
          <h2>
            {t("activeTransfers", {
              count: tasks.filter((task) => task.status === "running" || task.status === "queued")
                .length,
            })}
          </h2>
        </div>
        <IconButton label={t("collapseTransfers")} onClick={onClose}>
          <X size={18} />
        </IconButton>
      </header>
      <div className="transfer-list">
        {tasks.length === 0 ? (
          <div className="transfer-empty">
            <ArrowUpDown size={26} />
            <p>{t("noTransfers")}</p>
            <small>{t("noTransfersHint")}</small>
          </div>
        ) : null}
        {tasks.map((task) => (
          <article key={task.id} className={`transfer-row ${task.status}`}>
            <div className="transfer-top">
              <span className="transfer-name">
                {task.direction === "upload" ? <Upload size={15} /> : <Download size={15} />}
                <span className="mono break-all">{task.object.key}</span>
              </span>
              <span className="tag">{t(task.status)}</span>
            </div>
            <div className="progress-track">
              <span style={{ width: `${progress(task.completed, task.size)}%` }} />
            </div>
            <div className="transfer-meta">
              <span>
                {formatBytes(task.completed, locale)} / {formatBytes(task.size, locale)}
              </span>
              <span>
                {task.bytesPerSecond > 0
                  ? `${formatBytes(task.bytesPerSecond, locale)}/s`
                  : t("unknown")}
              </span>
            </div>
            {task.error ? <p className="transfer-error">{task.error.message}</p> : null}
            {task.status === "running" || task.status === "queued" ? (
              <div className="transfer-actions">
                <Button
                  variant="ghost"
                  onClick={() => void act(() => api.PauseTransfer(task.id), t("transferPaused"))}
                >
                  <Pause size={14} />
                  {t("pause")}
                </Button>
                <Button
                  variant="ghost"
                  onClick={() =>
                    void act(() => api.CancelTransfer(task.id), t("transferCancelled"))
                  }
                >
                  <X size={14} />
                  {t("cancelTransfer")}
                </Button>
              </div>
            ) : null}
            {task.status === "paused" || task.status === "failed" ? (
              <div className="transfer-actions">
                <Button
                  variant="ghost"
                  onClick={() => void act(() => api.ResumeTransfer(task.id), t("transferResumed"))}
                >
                  <Play size={14} />
                  {t("resume")}
                </Button>
                <Button
                  variant="ghost"
                  onClick={() =>
                    void act(() => api.CancelTransfer(task.id), t("transferCancelled"))
                  }
                >
                  <X size={14} />
                  {t("cancelTransfer")}
                </Button>
              </div>
            ) : null}
          </article>
        ))}
      </div>
    </aside>
  );
}

function SettingsModal({
  onClose,
  profile,
  bucket,
  prefix,
  notify,
}: {
  onClose: () => void;
  profile: Profile | null;
  bucket: string;
  prefix: string;
  notify: (value: string) => void;
}) {
  const { t, i18n } = useTranslation();
  const locale = i18n.language === "zh" ? "zh-CN" : "en-US";
  const ui = useUI();
  const [rate, setRate] = useState(ui.rate);
  const [usage, setUsage] = useState<Usage | null>(null);
  const [scanning, setScanning] = useState(false);
  const [scanError, setScanError] = useState("");
  const requestID = useRef("");

  useEffect(
    () => () => {
      if (requestID.current) void api.CancelRequest(requestID.current);
    },
    [],
  );

  const applyRate = async () => {
    if (!Number.isFinite(rate) || rate < 0) {
      notify(t("rateInvalid"));
      return;
    }
    ui.setRate(rate);
    try {
      await api.SetRate(Math.round(rate * 1024 * 1024));
      notify(t("rateApplied"));
    } catch (error) {
      notify(errorMessage(error));
    }
  };

  const scan = async () => {
    if (!profile || !bucket) return;
    const id = crypto.randomUUID();
    requestID.current = id;
    setScanning(true);
    setScanError("");
    try {
      setUsage(await api.Usage(profile.id, bucket, prefix, id));
    } catch (error) {
      setScanError(errorMessage(error));
    } finally {
      setScanning(false);
      requestID.current = "";
    }
  };

  const cancelScan = async () => {
    if (requestID.current) await api.CancelRequest(requestID.current);
    setScanning(false);
    notify(t("scanCancelled"));
  };

  return (
    <Modal
      title={t("settings")}
      subtitle={t("preferencesHint")}
      onClose={onClose}
      footer={
        <Button variant="primary" onClick={onClose}>
          {t("close")}
        </Button>
      }
    >
      <div className="settings-grid">
        <section>
          <h3>
            <Languages size={17} />
            {t("language")}
          </h3>
          <div className="segmented">
            <button
              type="button"
              aria-pressed={ui.language === "en"}
              onClick={() => ui.setLanguage("en")}
            >
              English
            </button>
            <button
              type="button"
              aria-pressed={ui.language === "zh"}
              onClick={() => ui.setLanguage("zh")}
            >
              简体中文
            </button>
          </div>
        </section>
        <section>
          <h3>
            <Sun size={17} />
            {t("appearance")}
          </h3>
          <div className="segmented">
            <button
              type="button"
              aria-pressed={ui.theme === "light"}
              onClick={() => ui.setTheme("light")}
            >
              <Sun size={14} />
              {t("light")}
            </button>
            <button
              type="button"
              aria-pressed={ui.theme === "dark"}
              onClick={() => ui.setTheme("dark")}
            >
              <Moon size={14} />
              {t("dark")}
            </button>
            <button
              type="button"
              aria-pressed={ui.theme === "system"}
              onClick={() => ui.setTheme("system")}
            >
              {t("system")}
            </button>
          </div>
        </section>
        <section>
          <h3>
            <Gauge size={17} />
            {t("transferLimit")}
          </h3>
          <div className="inline-field">
            <input
              type="number"
              min={0}
              step={0.1}
              value={rate}
              onChange={(event) => setRate(Number(event.target.value))}
              aria-label={t("transferLimit")}
            />
            <span>{t("rateUnit")}</span>
            <Button variant="secondary" onClick={() => void applyRate()}>
              {t("applyLimit")}
            </Button>
          </div>
          <p className="field-hint">{t("rateHint")}</p>
        </section>
        <section className="usage-section">
          <h3>
            <HardDrive size={17} />
            {t("usage")}
          </h3>
          <p className="muted small">
            {bucket || "—"} · {prefix || "/"}
          </p>
          {usage ? (
            <div className="usage-stats">
              <div>
                <small>{t("objectCount")}</small>
                <strong>{usage.objects}</strong>
              </div>
              <div>
                <small>{t("totalStorage")}</small>
                <strong>{formatBytes(usage.bytes, locale)}</strong>
              </div>
              <div>
                <small>{t("scannedAt", { date: formatDate(usage.scannedAt, locale) })}</small>
                <strong>{usage.complete ? t("completed") : t("scanIncomplete")}</strong>
              </div>
            </div>
          ) : (
            <p className="muted">{t("noUsage")}</p>
          )}
          {usage && Object.keys(usage.types).length ? (
            <div className="type-list">
              {Object.entries(usage.types)
                .sort((a, b) => (b[1] ?? 0) - (a[1] ?? 0))
                .slice(0, 8)
                .map(([kind, bytes]) => (
                  <div key={kind}>
                    <span className="mono">{kind}</span>
                    <span>{formatBytes(bytes ?? 0, locale)}</span>
                  </div>
                ))}
            </div>
          ) : null}
          <div className="inline-actions">
            {scanning ? (
              <Button variant="secondary" onClick={() => void cancelScan()}>
                {t("cancelScan")}
              </Button>
            ) : (
              <Button
                variant="secondary"
                onClick={() => void scan()}
                disabled={!profile || !bucket}
              >
                <RefreshCw size={15} />
                {t("scanUsage")}
              </Button>
            )}
            {scanning ? <span className="muted small">{t("scanRunning")}</span> : null}
          </div>
          <p className="field-hint">{t("usageHint")}</p>
          {scanError ? <ErrorBanner message={scanError} /> : null}
        </section>
        <section>
          <h3>
            <Globe size={17} />
            {t("mcp")}
          </h3>
          <p className="muted small">{t("mcpHint")}</p>
          <pre className="code-block">{`{
  "mcpServers": {
    "bucky": {
      "command": "bucky",
      "args": ["mcp"]
    }
  }
}`}</pre>
          <p className="field-hint">
            {t("mcpPath")} {t("mcpConsent")}
          </p>
        </section>
      </div>
    </Modal>
  );
}
