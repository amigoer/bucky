import { XIcon } from "lucide-react";
import { useState } from "react";
import { useTranslation } from "react-i18next";
import { Brand } from "@/components/brand";
import { isMac } from "@/components/platform";
import { Button } from "@/components/ui/button";
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { Tip } from "@/components/ui/tooltip";
import { WindowControls } from "@/components/window-controls";
import type { Profile } from "@/lib/api";
import { useNav } from "@/lib/nav";
import { useProfiles } from "@/lib/queries";
import { cn } from "@/lib/utils";
import { CloudflareEditFlow } from "./CloudflareEdit";
import { CloudflareFlow } from "./CloudflarePanel";
import { ManualFlow } from "./ManualPanel";

type Mode = "cloudflare" | "manual" | "edit";

/** Takes over the whole window: a focused flow reads better than a small dialog. */
export function ConnectionFlow() {
  const value = useNav((s) => s.connection);
  return value ? <Flow key={value.id ?? "new"} id={value.id} /> : null;
}

function Flow({ id }: { id?: string }) {
  const { t } = useTranslation();
  const close = useNav((s) => s.openConnection);
  const navigate = useNav((s) => s.navigate);
  const { data: profiles = [] } = useProfiles();
  const existing = id ? profiles.find((profile) => profile.id === id) : undefined;
  const [isNew] = useState(!existing);
  const [mode, setMode] = useState<Mode>(existing ? "edit" : "cloudflare");

  const done = (saved: Profile) => {
    close(null);
    // Only a freshly added connection moves the browser; an edit leaves it alone.
    if (isNew && saved.bucket) {
      navigate({ profileId: saved.id, bucket: saved.bucket, prefix: "" });
    }
  };

  const title = existing ? t("connection.titleEdit") : t("connection.flow.title");

  return (
    <Dialog open onOpenChange={(open) => (!open ? close(null) : undefined)}>
      <DialogContent placement="full">
        <DialogTitle className="sr-only">{title}</DialogTitle>
        <DialogDescription className="sr-only">{t("connection.flow.desc")}</DialogDescription>

        <header
          className={cn(
            "drag-region flex h-[var(--titlebar-height)] flex-none items-center gap-2 border-b bg-background pr-3",
            isMac ? "pl-[var(--titlebar-leading)]" : "pl-[52px]",
          )}
        >
          <Brand />
          <span aria-hidden className="mx-0.5 h-4 w-px flex-none rounded-full bg-border-strong" />
          <span className="flex-none text-[15px] font-[650]">{title}</span>
          <span className="flex-1" />
          <Tip label={t("connection.flow.close")}>
            <DialogClose asChild>
              <Button variant="ghost" size="icon" aria-label={t("connection.flow.close")}>
                <XIcon className="size-4" />
              </Button>
            </DialogClose>
          </Tip>
          {isMac ? null : <WindowControls />}
        </header>

        <div className="flex min-h-0 flex-1">
          {mode === "cloudflare" ? (
            <CloudflareFlow onUseKeys={() => setMode("manual")} onConnected={done} />
          ) : mode === "manual" ? (
            <ManualFlow onBack={() => setMode("cloudflare")} onSaved={done} />
          ) : existing?.credentialSource === "cloudflare" ? (
            <CloudflareEditFlow profile={existing} onSaved={done} />
          ) : (
            <ManualFlow profile={existing} onSaved={done} />
          )}
        </div>
      </DialogContent>
    </Dialog>
  );
}
