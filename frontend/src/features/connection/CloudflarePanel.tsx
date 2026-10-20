import {
  AlertTriangleIcon,
  CheckIcon,
  DatabaseIcon,
  ExternalLinkIcon,
  InfoIcon,
  LayoutGridIcon,
  LoaderCircleIcon,
  LockIcon,
  PieChartIcon,
  Share2Icon,
  ShieldCheckIcon,
} from "lucide-react";
import { useEffect, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { CloudflareMark } from "@/components/cloudflare-mark";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Spinner } from "@/components/ui/spinner";
import { copyText } from "@/lib/actions";
import { api, type Bucket, type Profile } from "@/lib/api";
import { CLOUDFLARE_CLIENTS_DOCS, CLOUDFLARE_DASHBOARD } from "@/lib/cloudflare";
import { errorCode, errorMessage } from "@/lib/errors";
import { formatDate } from "@/lib/format";
import i18n from "@/lib/i18n";
import { useNav } from "@/lib/nav";
import { keys } from "@/lib/queries";
import { queryClient } from "@/lib/query";
import { useCloudflareSignIn } from "@/lib/use-cloudflare-sign-in";
import { cn } from "@/lib/utils";
import {
  CapabilityRow,
  Field,
  LeftColumn,
  Note,
  Panel,
  ProviderHeading,
  RightColumn,
  StepBody,
  StepRow,
  SwitchRow,
} from "./parts";

/** The browser sign-in path: authorize, then limit the minted token to one bucket. */
export function CloudflareFlow({
  onUseKeys,
  onConnected,
}: {
  onUseKeys: () => void;
  onConnected: (profile: Profile) => void;
}) {
  const { t } = useTranslation();
  const close = useNav((s) => s.openConnection);
  const openSettings = useNav((s) => s.openSettings);
  const { state, starting, start, cancel, fail, selectAccount } = useCloudflareSignIn();
  const [connecting, setConnecting] = useState(false);
  const [name, setName] = useState("");
  const [bucket, setBucket] = useState("");
  const [readOnly, setReadOnly] = useState(false);
  const [buckets, setBuckets] = useState<Bucket[]>([]);
  const [loadingBuckets, setLoadingBuckets] = useState(false);
  const [needsSetup, setNeedsSetup] = useState(false);

  useEffect(() => {
    if (state.phase !== "ready" || !state.accountId) return;
    let live = true;
    setLoadingBuckets(true);
    setBuckets([]);
    api
      .CloudflareBuckets(state.sessionId, state.accountId)
      .then((list) => {
        if (!live) return;
        setBuckets(list);
        setBucket((current) => current || list[0]?.name || "");
      })
      .catch((cause) => live && fail(errorMessage(cause)))
      .finally(() => live && setLoadingBuckets(false));
    return () => {
      live = false;
    };
  }, [state.phase, state.sessionId, state.accountId, fail]);

  const startFlow = async () => {
    setBucket("");
    setNeedsSetup(false);
    try {
      await start();
    } catch (cause) {
      // Nothing can be authorized without a client ID, so the aside offers settings.
      if (errorCode(cause) === "configuration") {
        setNeedsSetup(true);
        // The shared message for this code describes a saved profile, not a missing client.
        fail(t("connection.cloudflare.setupRequired"));
      }
    }
  };

  const connect = async () => {
    if (connecting || !name.trim() || !bucket) return;
    setConnecting(true);
    try {
      const saved = await api.ConnectCloudflare({
        sessionId: state.sessionId,
        profileId: "",
        name: name.trim(),
        accountId: state.accountId,
        bucket,
        jurisdiction: "default",
        publicUrl: "",
        readOnly,
      });
      await queryClient.invalidateQueries({ queryKey: keys.profiles });
      toast.success(t("connection.cloudflare.connected", { bucket }));
      onConnected(saved);
    } catch (cause) {
      fail(errorMessage(cause));
    } finally {
      setConnecting(false);
    }
  };

  const account = state.accounts.find((entry) => entry.id === state.accountId);

  return (
    <>
      <LeftColumn>
        <StepBody>
          <ProviderHeading />

          {state.phase === "idle" ? (
            <>
              <Heading
                title={t("connection.flow.title")}
                lead={t("connection.cloudflare.signInDesc")}
              />
              <Note icon={<InfoIcon />} tone="warn">
                {t("connection.cloudflare.prereq")}{" "}
                <a
                  href={CLOUDFLARE_CLIENTS_DOCS}
                  target="_blank"
                  rel="noreferrer"
                  className="underline underline-offset-2"
                >
                  {t("connection.cloudflare.docs")}
                </a>
              </Note>
              <div className="flex flex-col gap-2">
                <Button
                  size="lg"
                  className="h-10 w-full text-[14px]"
                  disabled={starting}
                  onClick={() => void startFlow()}
                >
                  {starting ? <LoaderCircleIcon className="size-3.5 animate-spin" /> : null}
                  {t("connection.cloudflare.continue")}
                </Button>
                <Button
                  variant="secondary"
                  size="lg"
                  className="h-10 w-full text-[14px]"
                  onClick={onUseKeys}
                >
                  {t("connection.cloudflare.useKeys")}
                </Button>
              </div>
            </>
          ) : null}

          {state.phase === "waiting" ? (
            <>
              <Heading
                title={t("connection.cloudflare.waiting")}
                lead={t("connection.cloudflare.waitingHint")}
              />
              <div className="flex flex-col gap-1.5">
                <span className="text-[11.5px] font-medium text-muted-foreground">
                  {t("connection.cloudflare.authUrl")}
                </span>
                <span className="break-all rounded-lg bg-field px-2.5 py-2 font-mono text-[10.5px] leading-relaxed text-muted-foreground">
                  {state.authUrl}
                </span>
              </div>
              <div className="flex items-center gap-2">
                <Button onClick={() => void api.OpenExternal(state.authUrl).catch(() => {})}>
                  <ExternalLinkIcon className="size-3.5" />
                  {t("connection.cloudflare.openBrowser")}
                </Button>
                <Button variant="secondary" onClick={() => void copyText(state.authUrl)}>
                  {t("common.copy")}
                </Button>
                <span className="flex-1" />
                <Button variant="ghost" onClick={() => void cancel().catch(() => {})}>
                  {t("connection.cloudflare.cancel")}
                </Button>
              </div>
            </>
          ) : null}

          {state.phase === "ready" || connecting ? (
            <>
              <Heading
                title={t("connection.cloudflare.readyTitle")}
                lead={t("connection.cloudflare.readyLead", { account: account?.name ?? "" })}
              />
              <Field label={t("connection.name")}>
                <Input
                  value={name}
                  autoFocus
                  onChange={(event) => setName(event.target.value)}
                  placeholder="Acme assets"
                  maxLength={100}
                />
              </Field>
              <SwitchRow
                id="cf-read-only"
                title={t("connection.cloudflare.readOnly")}
                description={t("connection.cloudflare.readOnlyHint")}
                checked={readOnly}
                onCheckedChange={setReadOnly}
              />
              <div className="flex flex-col gap-2">
                <Button
                  size="lg"
                  className="h-10 w-full text-[14px]"
                  disabled={connecting || !name.trim() || !bucket}
                  onClick={() => void connect()}
                >
                  {connecting ? (
                    <LoaderCircleIcon className="size-3.5 animate-spin" />
                  ) : (
                    <ShieldCheckIcon className="size-3.5" />
                  )}
                  {connecting
                    ? t("connection.cloudflare.connecting")
                    : t("connection.cloudflare.connect")}
                </Button>
                <Button
                  variant="ghost"
                  size="sm"
                  disabled={connecting}
                  onClick={() => void cancel().catch(() => {})}
                >
                  {t("connection.cloudflare.cancel")}
                </Button>
              </div>
              <p className="text-[11px] leading-relaxed text-faint">
                {t("connection.cloudflare.tokenHint")}
              </p>
            </>
          ) : null}

          {state.phase === "error" ? (
            <>
              <Heading
                title={t("connection.cloudflare.errorTitle")}
                lead={t("connection.cloudflare.errorLead")}
              />
              <p className="flex items-start gap-2 rounded-xl bg-destructive-soft px-3.5 py-2.5 text-[11.5px] leading-relaxed text-destructive">
                <AlertTriangleIcon className="mt-0.5 size-3.5 flex-none" />
                <span className="min-w-0">{state.error}</span>
              </p>
              <div className="flex items-center gap-2">
                <Button disabled={starting} onClick={() => void startFlow()}>
                  {starting ? <LoaderCircleIcon className="size-3.5 animate-spin" /> : null}
                  {t("connection.cloudflare.retry")}
                </Button>
                <Button variant="ghost" onClick={onUseKeys}>
                  {t("connection.cloudflare.useKeys")}
                </Button>
              </div>
            </>
          ) : null}
        </StepBody>
      </LeftColumn>

      <RightColumn>
        {state.phase === "idle" ? <IntroAside /> : null}
        {state.phase === "waiting" ? <ProgressAside /> : null}
        {state.phase === "ready" || connecting ? (
          <>
            <Panel title={t("connection.cloudflare.account")}>
              <div className="flex flex-col gap-0.5">
                {state.accounts.map((entry) => {
                  const active = entry.id === state.accountId;
                  const selectable = state.accounts.length > 1;
                  return (
                    <button
                      key={entry.id}
                      type="button"
                      disabled={!selectable}
                      onClick={() => {
                        setBucket("");
                        selectAccount(entry.id);
                      }}
                      className={cn(
                        "flex items-center gap-2.5 rounded-lg px-2.5 py-2 text-left transition-colors",
                        selectable && !active && "hover:bg-hover",
                        active && "bg-primary-soft text-primary-text",
                      )}
                    >
                      <CloudflareMark size={20} />
                      <span className="flex min-w-0 flex-1 flex-col">
                        <span className="truncate text-[12.5px] font-semibold">{entry.name}</span>
                        <span className="truncate font-mono text-[10.5px] text-faint">
                          {entry.id}
                        </span>
                      </span>
                      {active ? (
                        <CheckIcon className="size-3.5 flex-none" strokeWidth={2.6} />
                      ) : null}
                    </button>
                  );
                })}
              </div>
            </Panel>

            <Panel
              title={t("connection.cloudflare.bucket")}
              action={
                loadingBuckets ? (
                  <Spinner />
                ) : (
                  <span className="text-[11px] tabular text-faint">
                    {t("connection.cloudflare.bucketCount", { count: buckets.length })}
                  </span>
                )
              }
            >
              {loadingBuckets ? (
                <div className="flex flex-col gap-1">
                  {[0, 1, 2].map((row) => (
                    <span key={row} className="h-9 animate-pulse rounded-lg bg-field" />
                  ))}
                </div>
              ) : buckets.length === 0 ? (
                <p className="text-[11.5px] leading-relaxed text-muted-foreground">
                  {t("connection.cloudflare.noBuckets")}
                </p>
              ) : (
                <div className="flex flex-col gap-0.5">
                  {buckets.map((entry) => {
                    const active = entry.name === bucket;
                    return (
                      <button
                        key={entry.name}
                        type="button"
                        onClick={() => setBucket(entry.name)}
                        className={cn(
                          "flex w-full items-center gap-2.5 rounded-lg px-2.5 py-2 text-left transition-colors",
                          active ? "bg-primary-soft text-primary-text" : "hover:bg-hover",
                        )}
                      >
                        <DatabaseIcon
                          className={cn(
                            "size-3.5 flex-none",
                            active ? "text-primary" : "text-faint",
                          )}
                        />
                        <span className="flex min-w-0 flex-1 flex-col">
                          <span className="truncate text-[12.5px] font-medium">{entry.name}</span>
                          <span className="truncate text-[10.5px] text-faint">
                            {formatDate(entry.createdAt, i18n.language)}
                          </span>
                        </span>
                        {active ? (
                          <CheckIcon className="size-3.5 flex-none" strokeWidth={2.6} />
                        ) : null}
                      </button>
                    );
                  })}
                </div>
              )}
            </Panel>

            <Note icon={<LockIcon />}>{t("connection.cloudflare.bucketHint")}</Note>
          </>
        ) : null}
        {state.phase === "error" ? (
          <Panel title={t("connection.cloudflare.errorAside")}>
            <ul className="flex flex-col gap-2.5 text-[11.5px] leading-relaxed text-muted-foreground">
              <AsideReason>{t("connection.cloudflare.errorReason1")}</AsideReason>
              <AsideReason>{t("connection.cloudflare.errorReason2")}</AsideReason>
            </ul>
            <a
              href={CLOUDFLARE_CLIENTS_DOCS}
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1 text-[11.5px] text-primary-text underline-offset-4 hover:underline"
            >
              {t("connection.cloudflare.docs")}
              <ExternalLinkIcon className="size-3" />
            </a>
            {needsSetup ? (
              <Button
                variant="secondary"
                size="sm"
                className="self-start"
                onClick={() => {
                  close(null);
                  openSettings("integrations");
                }}
              >
                {t("connection.cloudflare.openSettings")}
              </Button>
            ) : null}
          </Panel>
        ) : null}
      </RightColumn>
    </>
  );
}

function Heading({ title, lead }: { title: string; lead: string }) {
  return (
    <div className="flex flex-col gap-2">
      <h1 className="text-[28px] leading-[1.12] font-[650] tracking-[-0.022em]">{title}</h1>
      <p className="text-[13.5px] leading-relaxed text-muted-foreground">{lead}</p>
    </div>
  );
}

function AsideReason({ children }: { children: React.ReactNode }) {
  return (
    <li className="flex gap-2.5">
      <span className="mt-1.75 size-1 flex-none rounded-full bg-faint" />
      <span className="min-w-0">{children}</span>
    </li>
  );
}

function IntroAside() {
  const { t } = useTranslation();
  return (
    <>
      <Panel title={t("connection.cloudflare.asideTitle")}>
        <div className="flex flex-col gap-3.5">
          <CapabilityRow
            icon={<LayoutGridIcon />}
            tone="accent"
            title={t("connection.cloudflare.featureBrowse")}
            description={t("connection.cloudflare.featureBrowseDesc")}
          />
          <CapabilityRow
            icon={<PieChartIcon />}
            tone="success"
            title={t("connection.cloudflare.featureUsage")}
            description={t("connection.cloudflare.featureUsageDesc")}
          />
          <CapabilityRow
            icon={<Share2Icon />}
            tone="warn"
            title={t("connection.cloudflare.featureShare")}
            description={t("connection.cloudflare.featureShareDesc")}
          />
        </div>
      </Panel>
      <Note icon={<LockIcon />}>{t("connection.cloudflare.keychainNote")}</Note>
      <a
        href={CLOUDFLARE_DASHBOARD}
        target="_blank"
        rel="noreferrer"
        className="flex items-center gap-1.5 self-start px-1 text-[11.5px] text-muted-foreground underline-offset-4 hover:text-foreground hover:underline"
      >
        {t("connection.cloudflare.dashboard")}
        <ExternalLinkIcon className="size-3" />
      </a>
    </>
  );
}

function ProgressAside() {
  const { t } = useTranslation();
  return (
    <>
      <Panel title={t("connection.cloudflare.progress")}>
        <div className="flex flex-col gap-3.5">
          <StepRow index={1} state="active" title={t("connection.cloudflare.stepAuthorize")} />
          <StepRow index={2} state="todo" title={t("connection.cloudflare.stepMint")} />
          <StepRow index={3} state="todo" title={t("connection.cloudflare.stepVerify")} />
        </div>
      </Panel>
      <Note icon={<InfoIcon />}>{t("connection.cloudflare.waitingAside")}</Note>
    </>
  );
}
