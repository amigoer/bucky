import { useQuery } from "@tanstack/react-query";
import { CopyIcon, LinkIcon, LoaderCircleIcon } from "lucide-react";
import { useMemo, useState } from "react";
import { useTranslation } from "react-i18next";
import { Button } from "@/components/ui/button";
import { Dialog, DialogContent, DialogFooter, DialogHeader } from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Segmented } from "@/components/ui/segmented";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { copyText, expandEntries } from "@/lib/actions";
import { api, type ObjectRef } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { basename } from "@/lib/format";
import { type ShareDraft, useNav } from "@/lib/nav";
import { useProfiles } from "@/lib/queries";

const DURATIONS = [
  { value: "900", label: "d15m" },
  { value: "3600", label: "d1h" },
  { value: "86400", label: "d1d" },
  { value: "604800", label: "d7d" },
] as const;

const VISIBLE = 20;

export function ShareDialog() {
  const draft = useNav((s) => s.share);
  return draft ? <ShareDialogInner draft={draft} /> : null;
}

function ShareDialogInner({ draft }: { draft: ShareDraft }) {
  const { t } = useTranslation();
  const close = useNav((s) => s.openShare);
  const { data: profiles = [] } = useProfiles();
  const profile = profiles.find((p) => p.id === draft.location.profileId);
  const canPublic = !!profile?.publicUrl;
  const [mode, setMode] = useState<"temporary" | "public">("temporary");
  const [seconds, setSeconds] = useState("3600");
  const publicMode = mode === "public";
  const objectCount = draft.entries.filter((entry) => !entry.folder).length;
  const folderCount = draft.entries.length - objectCount;

  const links = useQuery({
    queryKey: ["share", draft, mode, publicMode ? "public" : seconds],
    queryFn: async () => {
      const refs: ObjectRef[] = [];
      for (const entry of draft.entries) {
        if (entry.folder) {
          refs.push(...(await expandEntries(draft.location, [entry])));
        } else {
          refs.push({
            profileId: draft.location.profileId,
            bucket: draft.location.bucket,
            key: entry.key,
          });
        }
      }
      const results = await Promise.all(
        refs.map((ref) => api.Share(ref, Number(seconds), publicMode)),
      );
      return refs.map((ref, index) => ({ ref, link: results[index] }));
    },
    staleTime: publicMode ? Infinity : 50 * 60_000,
    retry: false,
  });

  const items = links.data ?? [];
  const allUrls = useMemo(() => items.map((item) => item.link.url).join("\n"), [items]);
  const headers = useMemo(() => {
    const map = new Map<string, string>();
    for (const item of items) {
      for (const [key, value] of Object.entries(item.link.headers ?? {})) {
        if (typeof value === "string") map.set(key, value);
      }
    }
    return [...map.entries()];
  }, [items]);

  const multiple = draft.entries.length > 1 || folderCount > 0;

  return (
    <Dialog open onOpenChange={(open) => (!open ? close(null) : undefined)}>
      <DialogContent className="w-[540px]">
        <DialogHeader
          icon={<LinkIcon />}
          title={t("menu.share")}
          description={t("share.objects", { count: multiple ? items.length || objectCount : 1 })}
        />

        <div className="flex flex-col gap-3 px-4.5 pb-4">
          <div className="flex items-center gap-2">
            <Segmented<"temporary" | "public">
              value={mode}
              onValueChange={setMode}
              className="flex-1"
              options={[
                { value: "temporary", label: t("share.temporary") },
                {
                  value: "public",
                  label: t("share.public"),
                  title: canPublic ? undefined : t("share.noPublic"),
                  disabled: !canPublic,
                },
              ]}
            />
            <Select value={seconds} onValueChange={setSeconds} disabled={publicMode}>
              <SelectTrigger className="w-[106px]">
                <SelectValue />
              </SelectTrigger>
              <SelectContent>
                {DURATIONS.map((duration) => (
                  <SelectItem key={duration.value} value={duration.value}>
                    {t(`share.${duration.label}`)}
                  </SelectItem>
                ))}
              </SelectContent>
            </Select>
          </div>

          {!canPublic ? (
            <p className="rounded-md bg-muted px-2.5 py-1.5 text-[11.5px] text-muted-foreground">
              {t("share.noPublic")}
            </p>
          ) : null}
          {publicMode && canPublic ? (
            <p className="rounded-md bg-warning-soft px-2.5 py-1.5 text-[11.5px] text-warning-text">
              {t("share.publicWarn")}
            </p>
          ) : null}
          {folderCount > 0 ? (
            <p className="text-[11.5px] text-muted-foreground">
              {t("share.folder", { count: items.length })}
            </p>
          ) : multiple ? (
            <p className="text-[11.5px] text-muted-foreground">
              {t("share.multi", { count: items.length })}
            </p>
          ) : null}

          {links.isPending ? (
            <div className="flex h-24 items-center justify-center gap-2 text-[12px] text-muted-foreground">
              <LoaderCircleIcon className="size-4 animate-spin" />
              {t("share.preparing")}
            </div>
          ) : links.error ? (
            <p className="rounded-md bg-destructive-soft px-3 py-2 text-[12px] text-destructive">
              {errorMessage(links.error)}
            </p>
          ) : multiple ? (
            <div className="max-h-[260px] overflow-y-auto rounded-lg border">
              {items.slice(0, VISIBLE).map((item) => (
                <div
                  key={item.ref.key}
                  className="flex items-center gap-2 border-b px-2.5 py-1.5 last:border-b-0"
                >
                  <span className="min-w-0 flex-1 truncate text-[11.5px]" title={item.ref.key}>
                    {item.ref.key.slice(draft.location.prefix.length) || basename(item.ref.key)}
                  </span>
                  <Button
                    variant="ghost"
                    size="icon-sm"
                    aria-label={t("common.copy")}
                    onClick={() => void copyText(item.link.url)}
                  >
                    <CopyIcon className="size-3" />
                  </Button>
                </div>
              ))}
              {items.length > VISIBLE ? (
                <div className="px-2.5 py-1.5 text-[11px] text-faint">
                  {t("share.more", { count: items.length - VISIBLE })}
                </div>
              ) : null}
            </div>
          ) : (
            <div className="flex items-center gap-2">
              <Input
                readOnly
                value={items[0]?.link.url ?? ""}
                className="font-mono text-[11.5px]"
              />
              <Button
                variant="secondary"
                size="sm"
                onClick={() => void copyText(items[0]?.link.url ?? "")}
              >
                <CopyIcon className="size-3.5" />
                {t("common.copy")}
              </Button>
            </div>
          )}

          {items[0]?.link.expiresAt && !publicMode ? (
            <p className="text-[11px] text-faint">
              {t("share.expiresAt", {
                time: new Intl.DateTimeFormat(undefined, {
                  dateStyle: "medium",
                  timeStyle: "short",
                }).format(new Date(items[0].link.expiresAt)),
              })}
            </p>
          ) : null}

          {headers.length > 0 ? (
            <div className="rounded-md bg-warning-soft px-2.5 py-2 text-[11.5px] text-warning-text">
              <p className="font-semibold">{t("share.headers")}</p>
              {headers.map(([key, value]) => (
                <p key={key} className="truncate font-mono text-[10.5px]">
                  {key}: {value}
                </p>
              ))}
            </div>
          ) : null}
        </div>

        <DialogFooter className="justify-between">
          <span className="text-[10.5px] text-faint">
            {publicMode ? t("share.footPublic") : t("share.footTemporary")}
          </span>
          <div className="flex gap-2">
            <Button
              variant="secondary"
              size="sm"
              disabled={items.length === 0}
              onClick={() => void copyText(allUrls)}
            >
              <CopyIcon className="size-3.5" />
              {t("common.copyAll")}
            </Button>
            <Button size="sm" onClick={() => close(null)}>
              {t("common.done")}
            </Button>
          </div>
        </DialogFooter>
      </DialogContent>
    </Dialog>
  );
}
