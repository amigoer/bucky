import {
  AlertTriangleIcon,
  CheckIcon,
  ExternalLinkIcon,
  KeyRoundIcon,
  LoaderCircleIcon,
  RefreshCwIcon,
  ShieldCheckIcon,
  UnplugIcon,
} from "lucide-react";
import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import {
  AlertDialog,
  AlertDialogAction,
  AlertDialogCancel,
  AlertDialogContent,
  AlertDialogDescription,
  AlertDialogFooter,
  AlertDialogTitle,
} from "@/components/ui/alert-dialog";
import { Button } from "@/components/ui/button";
import { Input } from "@/components/ui/input";
import { Label } from "@/components/ui/label";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { Tip } from "@/components/ui/tooltip";
import { copyText } from "@/lib/actions";
import { api, type Bucket, type CloudflareStatus, type Profile } from "@/lib/api";
import { CLOUDFLARE_CLIENTS_DOCS, includesCloudflareAccount } from "@/lib/cloudflare";
import { errorCode, errorMessage } from "@/lib/errors";
import i18n from "@/lib/i18n";
import { useNav } from "@/lib/nav";
import { keys } from "@/lib/queries";
import { queryClient } from "@/lib/query";
import { useCloudflareSignIn } from "@/lib/use-cloudflare-sign-in";
import { cn } from "@/lib/utils";

/** The new-connection flow: authorize in the browser, then pick a bucket. */
export function CloudflareSignInPanel({ onConnected }: { onConnected: (p: Profile) => void }) {
  const { t } = useTranslation();
  const openSettings = useNav((s) => s.openSettings);
  const closeDialog = useNav((s) => s.openConnection);
  const {
    state,
    starting,
    start: startSignIn,
    cancel,
    fail,
    selectAccount,
  } = useCloudflareSignIn();
  const [connecting, setConnecting] = useState(false);
  const [name, setName] = useState("");
  const [bucket, setBucket] = useState("");
  const [readOnly, setReadOnly] = useState(false);
  const [buckets, setBuckets] = useState<Bucket[]>([]);
  const [loadingBuckets, setLoadingBuckets] = useState(false);

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

  const start = async () => {
    setBucket("");
    try {
      await startSignIn();
    } catch (cause) {
      if (errorCode(cause) === "configuration") {
        // The connection cannot start without a client ID; take the user there.
        closeDialog(null);
        openSettings("integrations");
        return;
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

  return (
    <div className="flex flex-col gap-3.5">
      {state.phase === "idle" ? (
        <>
          <div className="rounded-lg bg-field px-3 py-2.5 text-[11.5px] leading-relaxed text-muted-foreground">
            {t("connection.cloudflare.signInDesc")}
          </div>
          <div>
            <Button type="button" size="lg" disabled={starting} onClick={() => void start()}>
              <KeyRoundIcon className="size-3.5" />
              {t("connection.cloudflare.signIn")}
            </Button>
          </div>
        </>
      ) : null}

      {state.phase === "waiting" ? (
        <div className="flex flex-col gap-3 rounded-lg border px-3 py-3">
          <span className="flex items-center gap-2 text-[12.5px] font-medium">
            <Spinner />
            {t("connection.cloudflare.waiting")}
          </span>
          <span className="text-[11.5px] leading-relaxed text-muted-foreground">
            {t("connection.cloudflare.waitingHint")}
          </span>
          <span className="break-all rounded bg-field px-2 py-1.5 font-mono text-[10.5px] text-muted-foreground">
            {state.authUrl}
          </span>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="secondary"
              size="sm"
              onClick={() => void api.OpenExternal(state.authUrl).catch(() => {})}
            >
              <ExternalLinkIcon className="size-3.5" />
              {t("connection.cloudflare.openBrowser")}
            </Button>
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => void copyText(state.authUrl, true)}
            >
              {t("common.copy")}
            </Button>
            <span className="flex-1" />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              onClick={() => void cancel().catch(() => {})}
            >
              {t("connection.cloudflare.cancel")}
            </Button>
          </div>
        </div>
      ) : null}

      {state.phase === "ready" || connecting ? (
        <>
          {state.accounts.length > 1 ? (
            <Field label={t("connection.cloudflare.account")}>
              <Select
                value={state.accountId}
                onValueChange={(value) => {
                  setBucket("");
                  selectAccount(value);
                }}
              >
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {state.accounts.map((account) => (
                    <SelectItem key={account.id} value={account.id}>
                      {account.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            </Field>
          ) : null}

          <Field label={t("connection.name")}>
            <Input
              value={name}
              autoFocus
              onChange={(event) => setName(event.target.value)}
              placeholder="Acme assets"
              maxLength={100}
            />
          </Field>

          <Field
            label={t("connection.cloudflare.bucket")}
            hint={t("connection.cloudflare.bucketHint")}
          >
            {loadingBuckets ? (
              <span className="flex items-center gap-2 text-[12px] text-muted-foreground">
                <Spinner />
                {t("connection.cloudflare.loadingBuckets")}
              </span>
            ) : buckets.length === 0 ? (
              <span className="text-[12px] text-muted-foreground">
                {t("connection.cloudflare.noBuckets")}
              </span>
            ) : (
              <Select value={bucket} onValueChange={setBucket}>
                <SelectTrigger>
                  <SelectValue />
                </SelectTrigger>
                <SelectContent>
                  {buckets.map((entry) => (
                    <SelectItem key={entry.name} value={entry.name}>
                      {entry.name}
                    </SelectItem>
                  ))}
                </SelectContent>
              </Select>
            )}
          </Field>

          <div className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2.5">
            <div className="flex min-w-0 flex-col">
              <Label htmlFor="cf-read-only" className="text-[12.5px]">
                {t("connection.cloudflare.readOnly")}
              </Label>
              <span className="text-[11px] leading-relaxed text-muted-foreground">
                {t("connection.cloudflare.readOnlyHint")}
              </span>
            </div>
            <Switch
              id="cf-read-only"
              checked={readOnly}
              onCheckedChange={setReadOnly}
              className="flex-none"
            />
          </div>

          <div className="flex items-center gap-2">
            <Button
              type="button"
              size="lg"
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
            <span className="flex-1" />
            <Button
              type="button"
              variant="ghost"
              size="sm"
              disabled={connecting}
              onClick={() => void cancel().catch(() => {})}
            >
              {t("connection.cloudflare.cancel")}
            </Button>
          </div>
          <span className="text-[10.5px] leading-relaxed text-faint">
            {t("connection.cloudflare.tokenHint")}
          </span>
        </>
      ) : null}

      {state.phase === "error" ? (
        <div className="flex flex-col gap-3">
          <p className="flex items-start gap-2 rounded-lg bg-destructive-soft px-3 py-2 text-[11.5px] leading-relaxed text-destructive">
            <AlertTriangleIcon className="mt-0.5 size-3.5 flex-none" />
            <span className="min-w-0">{state.error}</span>
          </p>
          <div className="flex items-center gap-2">
            <Button
              type="button"
              variant="secondary"
              size="sm"
              disabled={starting}
              onClick={() => void start()}
            >
              {t("connection.cloudflare.retry")}
            </Button>
            <a
              href={CLOUDFLARE_CLIENTS_DOCS}
              target="_blank"
              rel="noreferrer"
              className="text-[11.5px] text-primary-text underline-offset-4 hover:underline"
            >
              {t("connection.cloudflare.docs")}
            </a>
          </div>
        </div>
      ) : null}
    </div>
  );
}

/** The stored authorization of an existing connection, with its two actions. */
export function CloudflareStatusBlock({ profile }: { profile: Profile }) {
  const { t } = useTranslation();
  const [status, setStatus] = useState<CloudflareStatus | null>(null);
  const [busy, setBusy] = useState(false);
  const [confirming, setConfirming] = useState(false);
  const {
    state: signIn,
    starting,
    start: startSignIn,
    cancel: cancelSignIn,
    fail,
    isCurrent,
  } = useCloudflareSignIn();
  const signingIn = starting || signIn.phase === "waiting" || signIn.phase === "ready";
  const profileRef = useRef(profile);
  profileRef.current = profile;
  const connectingSession = useRef("");

  const load = async () => {
    try {
      setStatus(await api.CloudflareStatus(profile.id));
    } catch {
      setStatus(null);
    }
  };

  useEffect(() => {
    let live = true;
    api
      .CloudflareStatus(profile.id)
      .then((value) => live && setStatus(value))
      .catch(() => live && setStatus(null));
    return () => {
      live = false;
    };
  }, [profile.id]);

  const signInAgain = () => startSignIn().catch(() => {});

  useEffect(() => {
    if (signIn.phase !== "ready" || connectingSession.current === signIn.sessionId) return;
    const current = profileRef.current;
    if (!includesCloudflareAccount(signIn.accounts, current.accountId)) {
      fail(i18n.t("connection.cloudflare.accountMismatch"));
      return;
    }
    connectingSession.current = signIn.sessionId;
    api
      .ConnectCloudflare({
        sessionId: signIn.sessionId,
        profileId: current.id,
        name: current.name,
        accountId: current.accountId,
        bucket: current.bucket,
        jurisdiction: current.jurisdiction,
        publicUrl: current.publicUrl,
        readOnly: current.readOnly,
      })
      .then(async () => {
        await queryClient.invalidateQueries({ queryKey: keys.profiles });
        if (!isCurrent(signIn.sessionId)) return;
        setStatus(await api.CloudflareStatus(current.id).catch(() => null));
        toast.success(i18n.t("connection.cloudflare.reauthorized"));
        await cancelSignIn();
      })
      .catch((cause) => {
        if (isCurrent(signIn.sessionId)) fail(errorMessage(cause));
      });
  }, [signIn.phase, signIn.sessionId, signIn.accounts, fail, cancelSignIn, isCurrent]);

  const reauthorize = async () => {
    setBusy(true);
    try {
      await api.ReauthorizeCloudflare(profile.id);
      await queryClient.invalidateQueries({ queryKey: keys.profiles });
      await load();
      toast.success(t("connection.cloudflare.reauthorized"));
    } catch (cause) {
      toast.error(errorMessage(cause));
    } finally {
      setBusy(false);
    }
  };

  const disconnect = async () => {
    setBusy(true);
    try {
      await api.DisconnectCloudflare(profile.id, true);
      await load();
      setConfirming(false);
      toast.success(t("connection.cloudflare.disconnected"));
    } catch (cause) {
      toast.error(errorMessage(cause));
      setConfirming(false);
    } finally {
      setBusy(false);
    }
  };

  const authorized = status?.authorized ?? false;

  return (
    <div className="flex flex-col gap-2.5 rounded-lg border px-3 py-2.5">
      <div className="flex items-center gap-2">
        <span
          className={cn(
            "grid size-5 flex-none place-items-center rounded-full",
            authorized ? "bg-success-soft text-success-text" : "bg-muted text-muted-foreground",
          )}
        >
          {authorized ? (
            <CheckIcon className="size-3" strokeWidth={3} />
          ) : (
            <UnplugIcon className="size-3" />
          )}
        </span>
        <div className="flex min-w-0 flex-col">
          <span className="text-[12.5px] font-medium">
            {authorized
              ? t("connection.cloudflare.authorized")
              : t("connection.cloudflare.notAuthorized")}
          </span>
          <span className="truncate text-[11px] text-muted-foreground">
            {authorized
              ? t("connection.cloudflare.authorizedNote", {
                  token: status?.tokenName ?? "",
                })
              : t("connection.cloudflare.notAuthorizedNote")}
          </span>
        </div>
        <span className="flex-1" />
        <Tip label={t("connection.cloudflare.reauthorize")}>
          <Button
            type="button"
            variant="secondary"
            size="sm"
            disabled={busy || signingIn}
            onClick={() => void reauthorize()}
          >
            {busy ? (
              <LoaderCircleIcon className="size-3.5 animate-spin" />
            ) : (
              <RefreshCwIcon className="size-3.5" />
            )}
            {t("connection.cloudflare.reauthorize")}
          </Button>
        </Tip>
        <Button
          type="button"
          variant="ghost"
          size="sm"
          disabled={busy || signingIn}
          onClick={() => void signInAgain()}
        >
          <ExternalLinkIcon className="size-3.5" />
          {t("connection.cloudflare.signInAgain")}
        </Button>
        {authorized ? (
          <Button
            type="button"
            variant="ghost-destructive"
            size="sm"
            disabled={busy || signingIn}
            onClick={() => setConfirming(true)}
          >
            {t("connection.cloudflare.disconnect")}
          </Button>
        ) : null}
      </div>

      {signingIn ? (
        <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
          <Spinner />
          {signIn.phase === "ready"
            ? t("connection.cloudflare.connecting")
            : t("connection.cloudflare.waiting")}
          <span className="flex-1" />
          <Button
            type="button"
            variant="ghost"
            size="sm"
            disabled={signIn.phase === "ready"}
            onClick={() => void cancelSignIn().catch(() => {})}
          >
            {t("connection.cloudflare.cancel")}
          </Button>
        </div>
      ) : null}

      {signIn.phase === "error" ? (
        <p role="alert" className="text-[11.5px] text-destructive">
          {signIn.error}
        </p>
      ) : null}

      <AlertDialog open={confirming} onOpenChange={setConfirming}>
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("connection.cloudflare.disconnectTitle", { name: profile.name })}
          </AlertDialogTitle>
          <AlertDialogDescription>
            {t("connection.cloudflare.disconnectDesc")}
          </AlertDialogDescription>
          <AlertDialogFooter>
            <AlertDialogCancel asChild>
              <Button variant="secondary" size="sm">
                {t("common.cancel")}
              </Button>
            </AlertDialogCancel>
            <AlertDialogAction asChild>
              <Button
                variant="destructive"
                size="sm"
                disabled={busy}
                onClick={(event) => {
                  event.preventDefault();
                  void disconnect();
                }}
              >
                {t("connection.cloudflare.disconnect")}
              </Button>
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </div>
  );
}

function Field({
  label,
  hint,
  children,
}: {
  label: string;
  hint?: string;
  children: React.ReactNode;
}) {
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <span className="text-[11.5px] font-medium text-muted-foreground">{label}</span>
      {children}
      {hint ? <span className="text-[10.5px] leading-relaxed text-faint">{hint}</span> : null}
    </div>
  );
}
