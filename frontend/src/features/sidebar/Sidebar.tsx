import {
  ChevronRightIcon,
  DatabaseIcon,
  LockIcon,
  PieChartIcon,
  PlusIcon,
  SlidersHorizontalIcon,
  TriangleAlertIcon,
  ZapIcon,
} from "lucide-react";
import { useTranslation } from "react-i18next";
import { Brand } from "@/components/brand";
import { ConnTile } from "@/components/conn-tile";
import { SidebarToggle } from "@/components/page-header";
import { isMac } from "@/components/platform";
import { Badge } from "@/components/ui/badge";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { Tip } from "@/components/ui/tooltip";
import type { Profile } from "@/lib/api";
import { errorMessage, isCredentialError } from "@/lib/errors";
import { useNav } from "@/lib/nav";
import { useBuckets, useProfiles } from "@/lib/queries";
import { usePrefs } from "@/lib/store";
import { cn } from "@/lib/utils";
import { TransferTray } from "./TransferTray";

export function Sidebar() {
  const { t } = useTranslation();
  const hidden = usePrefs((s) => s.sidebarHidden);
  const { data: profiles = [] } = useProfiles();
  const openConnection = useNav((s) => s.openConnection);
  const openQuickUpload = useNav((s) => s.openQuickUpload);
  const setPage = useNav((s) => s.setPage);
  const page = useNav((s) => s.page);

  return (
    <aside
      className={cn(
        "flex-none overflow-hidden bg-sidebar transition-[width] duration-300 ease-mac",
        hidden ? "w-0" : "w-60 border-r",
      )}
    >
      <div
        className={cn(
          "flex h-full w-60 flex-col transition-opacity duration-200",
          hidden && "opacity-0",
        )}
      >
        <div
          className={cn(
            "drag-region flex h-[var(--titlebar-height)] flex-none items-center gap-2 pr-2.5",
            isMac ? "pl-[82px]" : "pl-4",
          )}
        >
          <Brand />
          <div className="ml-auto">
            <SidebarToggle />
          </div>
        </div>

        <nav className="flex min-h-0 flex-1 flex-col gap-2 overflow-y-auto px-2.5 pt-1.5 pb-2.5">
          <div className="flex items-center justify-between px-2 py-0.5 text-[11px] font-semibold text-faint">
            <span>{t("sidebar.connections")}</span>
            <Tip label={t("sidebar.addConnection")}>
              <Button
                variant="ghost"
                size="icon-sm"
                aria-label={t("sidebar.addConnection")}
                onClick={() => openConnection({})}
              >
                <PlusIcon className="size-3" strokeWidth={2.4} />
              </Button>
            </Tip>
          </div>
          {profiles.map((profile) => (
            <ConnectionGroup key={profile.id} profile={profile} />
          ))}
        </nav>

        <div className="flex flex-none flex-col gap-2 border-t px-2.5 pt-2 pb-2.5">
          <TransferTray />
          <div className="flex gap-1.5">
            <Tip label={`${t("sidebar.quickUpload")} ⌥⌘U`}>
              <button
                type="button"
                onClick={() => openQuickUpload({})}
                className="flex h-8 min-w-0 flex-1 items-center gap-2 rounded-lg bg-primary-soft px-2.5 font-semibold text-primary-text transition-[background,transform] duration-150 hover:bg-primary-line active:scale-[0.98]"
              >
                <ZapIcon className="size-3.75 flex-none" />
                <span className="flex-1 truncate text-left">{t("sidebar.quickUpload")}</span>
              </button>
            </Tip>
            <Tip label={t("sidebar.usage")}>
              <Button
                variant="ghost"
                size="icon-lg"
                aria-label={t("sidebar.usage")}
                className={cn(page === "usage" && "bg-hover text-foreground")}
                onClick={() => setPage("usage")}
              >
                <PieChartIcon className="size-3.75" />
              </Button>
            </Tip>
            <Tip label={`${t("sidebar.settings")} ⌘,`}>
              <Button
                variant="ghost"
                size="icon-lg"
                aria-label={t("sidebar.settings")}
                className={cn(page === "settings" && "bg-hover text-foreground")}
                onClick={() => setPage("settings")}
              >
                <SlidersHorizontalIcon className="size-3.75" />
              </Button>
            </Tip>
          </div>
        </div>
      </div>
    </aside>
  );
}

function ConnectionGroup({ profile }: { profile: Profile }) {
  const { t } = useTranslation();
  const collapsed = usePrefs((s) => s.collapsed[profile.id] ?? false);
  const setPrefs = usePrefs((s) => s.set);
  const location = useNav((s) => s.location);
  const page = useNav((s) => s.page);
  const navigate = useNav((s) => s.navigate);
  const openConnection = useNav((s) => s.openConnection);
  const buckets = useBuckets(profile.id);
  const bad = buckets.isError && isCredentialError(buckets.error);
  const open = !collapsed;

  const toggle = () =>
    setPrefs({ collapsed: { ...usePrefs.getState().collapsed, [profile.id]: open } });

  return (
    <div className="flex flex-col">
      <button
        type="button"
        title={t("sidebar.editHint")}
        onClick={toggle}
        onDoubleClick={(e) => {
          e.preventDefault();
          openConnection({ id: profile.id });
        }}
        className="flex h-7.5 items-center gap-2.25 rounded-md pr-1.5 pl-2 text-left transition-colors hover:bg-hover"
      >
        <ConnTile />
        <span className="min-w-0 flex-1 truncate font-semibold">{profile.name}</span>
        {profile.readOnly ? (
          <Tip label={t("sidebar.readOnly")}>
            <Badge variant="warning" className="px-1">
              <LockIcon />
            </Badge>
          </Tip>
        ) : null}
        {bad ? (
          <Badge variant="destructive">
            <TriangleAlertIcon />
            {t("sidebar.attention")}
          </Badge>
        ) : null}
        {!open && !profile.readOnly && !bad && buckets.data ? (
          <span className="text-[11.5px] tabular text-faint">{buckets.data.length}</span>
        ) : null}
        <ChevronRightIcon
          className={cn(
            "size-2.5 flex-none text-faint transition-transform duration-200 ease-mac",
            open && "rotate-90",
          )}
          strokeWidth={3}
        />
      </button>
      <div
        className="grid transition-[grid-template-rows] duration-250 ease-mac"
        style={{ gridTemplateRows: open ? "1fr" : "0fr" }}
      >
        <div className="min-h-0 overflow-hidden">
          <div className="my-0.5 ml-4.25 flex flex-col gap-px border-l border-border-strong pl-2.25">
            {buckets.isPending ? (
              <span className="flex h-7 items-center gap-2 px-2 text-xs text-faint">
                <Spinner className="size-2.5" />
              </span>
            ) : buckets.isError ? (
              <button
                type="button"
                title={errorMessage(buckets.error)}
                onClick={() => (bad ? openConnection({ id: profile.id }) : void buckets.refetch())}
                className="flex h-7 items-center gap-2 truncate rounded-md px-2 text-left text-xs text-destructive hover:bg-hover"
              >
                <TriangleAlertIcon className="size-3 flex-none" />
                <span className="truncate">{t("sidebar.bucketsFailed")}</span>
              </button>
            ) : buckets.data && buckets.data.length === 0 ? (
              <span className="flex h-7 items-center px-2 text-xs text-faint">
                {t("sidebar.noBuckets")}
              </span>
            ) : (
              buckets.data?.map((bucket) => {
                const active =
                  page === "browse" &&
                  location?.profileId === profile.id &&
                  location.bucket === bucket.name;
                return (
                  <button
                    type="button"
                    key={bucket.name}
                    onClick={() =>
                      navigate({ profileId: profile.id, bucket: bucket.name, prefix: "" })
                    }
                    className={cn(
                      "flex h-7 items-center gap-2 rounded-md px-2 text-left transition-colors",
                      active ? "bg-primary-soft font-semibold text-primary-text" : "hover:bg-hover",
                    )}
                  >
                    <DatabaseIcon
                      className={cn("size-3.5 flex-none", active ? "text-primary" : "text-faint")}
                    />
                    <span className="truncate">{bucket.name}</span>
                  </button>
                );
              })
            )}
          </div>
        </div>
      </div>
    </div>
  );
}
