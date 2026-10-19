import { CopyIcon, ExternalLinkIcon, LoaderCircleIcon } from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { copyText } from "@/lib/actions";
import { api, type Settings } from "@/lib/api";
import { CLOUDFLARE_CLIENTS_DOCS, CLOUDFLARE_DASHBOARD } from "@/lib/cloudflare";
import { errorMessage } from "@/lib/errors";
import { Block, Group } from "./parts";

/** Configures the OAuth client that “Sign in with Cloudflare” uses. */
export function CloudflareCard() {
  const { t } = useTranslation();
  const [settings, setSettings] = useState<Settings | null>(null);
  const [clientId, setClientId] = useState("");
  const [saving, setSaving] = useState(false);

  useEffect(() => {
    let live = true;
    api
      .Settings()
      .then((value) => {
        if (!live) return;
        setSettings(value);
        setClientId(value.cloudflareClientId);
      })
      .catch(() => {});
    return () => {
      live = false;
    };
  }, []);

  const save = async () => {
    if (!settings) return;
    setSaving(true);
    try {
      const saved = await api.SaveSettings({
        ...settings,
        cloudflareClientId: clientId.trim(),
      });
      setSettings(saved);
      setClientId(saved.cloudflareClientId);
      toast.success(t("settings.cloudflareSaved"));
    } catch (cause) {
      toast.error(errorMessage(cause));
    } finally {
      setSaving(false);
    }
  };

  return (
    <Group>
      <Block title={t("settings.cloudflare")} description={t("settings.cloudflareDesc")}>
        <form
          className="flex items-center gap-2"
          onSubmit={(event) => {
            event.preventDefault();
            void save();
          }}
        >
          <Input
            value={clientId}
            disabled={!settings}
            onChange={(event) => setClientId(event.target.value)}
            placeholder={t("settings.cloudflareClientId")}
            spellCheck={false}
            autoComplete="off"
            className="font-mono text-[12px]"
          />
          <Button
            type="submit"
            variant="secondary"
            size="sm"
            disabled={!settings || saving}
            className="flex-none"
          >
            {saving ? <LoaderCircleIcon className="size-3.5 animate-spin" /> : null}
            {t("common.save")}
          </Button>
        </form>
        <span className="text-[10.5px] leading-relaxed text-faint">
          {t("settings.cloudflareClientIdHint")}
        </span>

        <div className="flex items-center gap-2 rounded-lg bg-field px-2.5 py-2">
          <div className="flex min-w-0 flex-col">
            <span className="text-[10.5px] text-faint">{t("settings.cloudflareRedirect")}</span>
            <span className="truncate font-mono text-[11.5px]">
              {settings?.cloudflareRedirectUri ?? ""}
            </span>
          </div>
          <span className="flex-1" />
          <Button
            type="button"
            variant="ghost"
            size="icon-sm"
            aria-label={t("common.copy")}
            onClick={() => void copyText(settings?.cloudflareRedirectUri ?? "")}
          >
            <CopyIcon className="size-3.5" />
          </Button>
        </div>
        <span className="text-[10.5px] leading-relaxed text-faint">
          {t("settings.cloudflareRedirectHint")}
        </span>

        <div className="flex flex-col gap-1.5">
          <span className="text-[11.5px] font-medium text-muted-foreground">
            {t("settings.cloudflareScopes")}
          </span>
          <div className="flex flex-wrap gap-1.5">
            {(settings?.cloudflareScopes ?? []).map((scope) => (
              <span
                key={scope}
                className="rounded bg-field px-1.5 py-0.5 font-mono text-[10.5px] text-muted-foreground"
              >
                {scope}
              </span>
            ))}
          </div>
          <span className="text-[10.5px] leading-relaxed text-faint">
            {t("settings.cloudflareScopesHint")}
          </span>
        </div>

        <p className="rounded-lg bg-field px-3 py-2 text-[11px] leading-relaxed text-muted-foreground">
          {t("settings.cloudflareSetup")}
        </p>

        <div className="flex items-center gap-3">
          <a
            href={CLOUDFLARE_DASHBOARD}
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-1 text-[11.5px] text-primary-text underline-offset-4 hover:underline"
          >
            <ExternalLinkIcon className="size-3" />
            {t("settings.cloudflareClients")}
          </a>
          <a
            href={CLOUDFLARE_CLIENTS_DOCS}
            target="_blank"
            rel="noreferrer"
            className="flex items-center gap-1 text-[11.5px] text-primary-text underline-offset-4 hover:underline"
          >
            <ExternalLinkIcon className="size-3" />
            {t("settings.cloudflareDocs")}
          </a>
        </div>
      </Block>
    </Group>
  );
}
