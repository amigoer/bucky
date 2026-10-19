import { PieChartIcon, ScanSearchIcon, XIcon } from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { EmptyState } from "@/components/empty-state";
import { PageHeader, PageTitle } from "@/components/page-header";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Segmented } from "@/components/ui/segmented";
import { Spinner } from "@/components/ui/spinner";
import { api, events, type Usage } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { formatBytes, formatDate } from "@/lib/format";
import { CHART_COLOR, type Kind, kindOfExtension } from "@/lib/kinds";
import { useNav } from "@/lib/nav";
import { cn } from "@/lib/utils";

type Scope = "bucket" | "prefix";

export function UsagePage() {
  const { t, i18n } = useTranslation();
  const location = useNav((s) => s.location);
  const [scope, setScope] = useState<Scope>("prefix");
  const [usage, setUsage] = useState<Usage | null>(null);
  const [scanning, setScanning] = useState(false);
  const [scanned, setScanned] = useState(0);
  const [error, setError] = useState<unknown>(null);
  const requestRef = useRef<string | null>(null);

  // biome-ignore lint/correctness/useExhaustiveDependencies: reset the scan when the location changes
  useEffect(() => {
    setScope(location?.prefix ? "prefix" : "bucket");
    setUsage(null);
    setError(null);
  }, [location?.profileId, location?.bucket, location?.prefix]);

  useEffect(() => {
    return events.onUsageProgress((progress) => {
      if (progress.requestId === requestRef.current) setScanned(progress.usage.objects);
    });
  }, []);

  useEffect(() => {
    return () => {
      if (requestRef.current) void api.CancelRequest(requestRef.current).catch(() => {});
    };
  }, []);

  const prefix = location ? (scope === "prefix" ? location.prefix : "") : "";

  const scan = async () => {
    if (!location) return;
    const id = crypto.randomUUID();
    requestRef.current = id;
    setScanning(true);
    setScanned(0);
    setUsage(null);
    setError(null);
    try {
      const result = await api.Usage(location.profileId, location.bucket, prefix, id);
      setUsage(result);
    } catch (cause) {
      setError(cause);
    } finally {
      setScanning(false);
      requestRef.current = null;
    }
  };

  const cancel = async () => {
    const id = requestRef.current;
    if (!id) return;
    try {
      await api.CancelRequest(id);
    } catch (cause) {
      setError(cause);
    }
  };

  const byKind = useMemo(() => {
    if (!usage) return [];
    const totals = new Map<Kind, number>();
    for (const [ext, bytes] of Object.entries(usage.types ?? {})) {
      const kind = kindOfExtension(ext);
      totals.set(kind, (totals.get(kind) ?? 0) + (bytes ?? 0));
    }
    return [...totals.entries()].sort((a, b) => b[1] - a[1]);
  }, [usage]);

  const extensions = useMemo(() => {
    if (!usage) return [];
    return Object.entries(usage.types ?? {})
      .map(([ext, bytes]) => [ext, bytes ?? 0] as const)
      .sort((a, b) => b[1] - a[1])
      .slice(0, 24);
  }, [usage]);
  const maxExtension = extensions[0]?.[1] ?? 0;

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader>
        <PageTitle>{t("usage.title")}</PageTitle>
      </PageHeader>
      <div className="min-h-0 flex-1 overflow-y-auto">
        <div className="mx-auto flex w-full max-w-[920px] flex-col gap-4 px-6 py-5">
          {!location ? (
            <div className="grid place-items-center py-16">
              <EmptyState icon={<PieChartIcon />} title={t("usage.noBucket")} />
            </div>
          ) : (
            <>
              <div className="flex flex-wrap items-center gap-2">
                <Segmented<Scope>
                  value={scope}
                  onValueChange={(next) => {
                    setScope(next);
                    setUsage(null);
                    setError(null);
                  }}
                  className="w-[280px]"
                  options={[
                    {
                      value: "bucket",
                      label: t("usage.scopeBucket", { bucket: location.bucket }),
                      disabled: scanning,
                    },
                    {
                      value: "prefix",
                      label: t("usage.scopePrefix"),
                      disabled: !location.prefix || scanning,
                    },
                  ]}
                />
                <span className="ml-auto flex items-center gap-2">
                  {scanning ? (
                    <Button variant="secondary" size="sm" onClick={() => void cancel()}>
                      <XIcon className="size-3.5" />
                      {t("usage.cancel")}
                    </Button>
                  ) : (
                    <Button
                      size="sm"
                      variant={usage ? "secondary" : "default"}
                      onClick={() => void scan()}
                    >
                      <ScanSearchIcon className="size-3.5" />
                      {usage ? t("usage.rescan") : t("usage.start")}
                    </Button>
                  )}
                </span>
              </div>

              <p className="text-[11px] text-faint">{t("usage.note")}</p>

              {scanning ? (
                <div className="flex items-center gap-3 rounded-xl bg-card px-4 py-3.5 shadow-card">
                  <Spinner className="size-4" />
                  <div className="flex min-w-0 flex-1 flex-col gap-0.5">
                    <span className="text-[12px] font-medium">{t("usage.scanning")}</span>
                    <span className="tabular text-[11px] text-muted-foreground">
                      {t("usage.progress", { count: scanned })}
                    </span>
                  </div>
                  <span className="flex-none truncate font-mono text-[10.5px] text-faint">
                    {prefix || location.bucket}
                  </span>
                </div>
              ) : error ? (
                <div className="flex flex-col gap-2 rounded-xl bg-destructive-soft px-4 py-3 text-destructive">
                  <span className="text-[12px] font-semibold">{t("states.errorTitle")}</span>
                  <span className="text-[11.5px]">
                    {t("usage.failed", { message: errorMessage(error) })}
                  </span>
                  <Button
                    variant="secondary"
                    size="sm"
                    className="mt-1 self-start"
                    onClick={() => void scan()}
                  >
                    {t("common.retry")}
                  </Button>
                </div>
              ) : !usage ? (
                <div className="grid place-items-center py-14">
                  <EmptyState
                    icon={<PieChartIcon />}
                    title={t("usage.idleTitle")}
                    description={t("usage.idleDesc")}
                    action={
                      <Button size="sm" onClick={() => void scan()}>
                        {t("usage.start")}
                      </Button>
                    }
                  />
                </div>
              ) : (
                <>
                  {!usage.complete ? (
                    <p className="rounded-lg bg-warning-soft px-3 py-2 text-[11.5px] text-warning-text">
                      {t("usage.partial")}
                    </p>
                  ) : null}
                  <div className="grid grid-cols-2 gap-3 sm:grid-cols-4">
                    <Stat
                      label={t("usage.objects")}
                      value={new Intl.NumberFormat(i18n.language).format(usage.objects)}
                    />
                    <Stat label={t("usage.size")} value={formatBytes(usage.bytes, i18n.language)} />
                    <Stat
                      label={t("usage.scannedAt")}
                      value={formatDate(usage.scannedAt, i18n.language)}
                    />
                    <div className="flex flex-col gap-1 rounded-xl bg-card px-3.5 py-2.5 shadow-card">
                      <span className="text-[10.5px] font-medium text-faint">
                        {t("usage.completeness")}
                      </span>
                      <Badge
                        variant={usage.complete ? "success" : "warning"}
                        className="mt-0.5 h-5 w-fit"
                      >
                        {usage.complete ? t("usage.complete") : t("usage.incomplete")}
                      </Badge>
                    </div>
                  </div>

                  {usage.objects === 0 ? (
                    <div className="grid place-items-center py-10">
                      <EmptyState icon={<PieChartIcon />} title={t("usage.noObjects")} />
                    </div>
                  ) : (
                    <>
                      <section className="flex flex-col gap-3 rounded-xl bg-card p-4 shadow-card">
                        <h2 className="text-[12px] font-semibold">{t("usage.byType")}</h2>
                        <div className="flex h-2.5 w-full overflow-hidden rounded-full bg-muted">
                          {byKind.map(([kind, bytes]) => (
                            <span
                              key={kind}
                              style={{
                                width: `${(bytes / Math.max(usage.bytes, 1)) * 100}%`,
                                background: CHART_COLOR[kind],
                              }}
                            />
                          ))}
                        </div>
                        <div className="grid grid-cols-2 gap-x-6 gap-y-1.5 sm:grid-cols-3">
                          {byKind.map(([kind, bytes]) => (
                            <div key={kind} className="flex items-center gap-2 text-[11.5px]">
                              <span
                                className="size-2 flex-none rounded-[2px]"
                                style={{ background: CHART_COLOR[kind] }}
                              />
                              <span className="min-w-0 flex-1 truncate">{t(`kinds.${kind}`)}</span>
                              <span className="flex-none tabular text-faint">
                                {formatBytes(bytes, i18n.language)}
                              </span>
                              <span className="w-9 flex-none text-right tabular text-faint">
                                {((bytes / Math.max(usage.bytes, 1)) * 100).toFixed(0)}%
                              </span>
                            </div>
                          ))}
                        </div>
                      </section>

                      <section className="flex flex-col gap-2 rounded-xl bg-card p-4 shadow-card">
                        <div className="flex items-baseline justify-between">
                          <h2 className="text-[12px] font-semibold">{t("usage.byExtension")}</h2>
                          <span className="text-[10.5px] text-faint">
                            {Object.keys(usage.types ?? {}).length}
                          </span>
                        </div>
                        <div className="flex flex-col">
                          {extensions.map(([ext, bytes]) => (
                            <div
                              key={ext}
                              className="flex items-center gap-2.5 border-b py-1.5 text-[11.5px] last:border-b-0"
                            >
                              <span className="w-16 flex-none truncate font-mono text-[10.5px] text-muted-foreground">
                                {ext === "(none)" ? "—" : ext}
                              </span>
                              <div className="h-1.5 min-w-0 flex-1 overflow-hidden rounded-full bg-muted">
                                <div
                                  className="h-full rounded-full bg-primary"
                                  style={{
                                    width: `${maxExtension > 0 ? (bytes / maxExtension) * 100 : 0}%`,
                                  }}
                                />
                              </div>
                              <span className="w-12 flex-none text-right tabular text-faint">
                                {new Intl.NumberFormat(i18n.language).format(
                                  usage.counts?.[ext] ?? 0,
                                )}
                              </span>
                              <span className="w-20 flex-none text-right tabular text-muted-foreground">
                                {formatBytes(bytes, i18n.language)}
                              </span>
                            </div>
                          ))}
                        </div>
                      </section>
                    </>
                  )}
                </>
              )}
            </>
          )}
        </div>
      </div>
    </div>
  );
}

function Stat({ label, value }: { label: string; value: string }) {
  return (
    <div className={cn("flex flex-col gap-1 rounded-xl bg-card px-3.5 py-2.5 shadow-card")}>
      <span className="text-[10.5px] font-medium text-faint">{label}</span>
      <span className="truncate tabular text-[15px] font-semibold" title={value}>
        {value}
      </span>
    </div>
  );
}
