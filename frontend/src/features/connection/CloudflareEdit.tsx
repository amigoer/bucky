import {
  AlertTriangleIcon,
  CheckIcon,
  ExternalLinkIcon,
  LoaderCircleIcon,
  RefreshCwIcon,
  ShieldCheckIcon,
  UnplugIcon,
} from "lucide-react";
import { type ReactNode, useEffect, useRef, useState } from "react";
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
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Spinner } from "@/components/ui/spinner";
import {
  api,
  type Bucket,
  type CloudflareStatus,
  type Profile,
  type ProfileInput,
} from "@/lib/api";
import { includesCloudflareAccount } from "@/lib/cloudflare";
import { errorMessage } from "@/lib/errors";
import i18n from "@/lib/i18n";
import { useNav } from "@/lib/nav";
import { keys } from "@/lib/queries";
import { queryClient } from "@/lib/query";
import { useCloudflareSignIn } from "@/lib/use-cloudflare-sign-in";
import { cn } from "@/lib/utils";
import {
  Field,
  LeftColumn,
  Panel,
  ProviderHeading,
  RightColumn,
  StepBody,
  SwitchRow,
} from "./parts";

const JURISDICTIONS = ["default", "eu", "fedramp"] as const;
type Jurisdiction = (typeof JURISDICTIONS)[number];

/** Editing a connection whose S3 credentials come from a Cloudflare authorization. */
export function CloudflareEditFlow({
  profile,
  onSaved,
}: {
  profile: Profile;
  onSaved: (saved: Profile) => void;
}) {
  const { t } = useTranslation();
  const close = useNav((s) => s.openConnection);
  const forgetProfile = useNav((s) => s.forgetProfile);
  const [name, setName] = useState(profile.name);
  const [publicUrl, setPublicUrl] = useState(profile.publicUrl);
  const [jurisdiction, setJurisdiction] = useState<Jurisdiction>(
    (profile.jurisdiction as Jurisdiction) || "default",
  );
  const [readOnly, setReadOnly] = useState(profile.readOnly);
  const [testing, setTesting] = useState(false);
  const [saving, setSaving] = useState(false);
  const [deleting, setDeleting] = useState(false);
  const [confirmDelete, setConfirmDelete] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [testResult, setTestResult] = useState<{
    signature: string;
    ok: boolean;
    buckets?: Bucket[];
    message?: string;
  } | null>(null);

  const wire = (): ProfileInput => ({
    profile: {
      id: profile.id,
      name: name.trim(),
      providerType: "r2",
      accountId: profile.accountId,
      jurisdiction,
      bucket: profile.bucket,
      publicUrl: publicUrl.trim(),
      readOnly,
      credentialSource: "cloudflare",
      createdAt: profile.createdAt,
      updatedAt: profile.updatedAt,
    },
    // A Cloudflare connection keeps the credentials the sign-in flow minted.
    accessKeyId: "",
    secretAccessKey: "",
  });

  const signature = JSON.stringify({ name, publicUrl, jurisdiction, readOnly });
  const shownTest = testResult?.signature === signature ? testResult : null;

  const validate = (): string | null => {
    if (!name.trim()) return t("connection.nameRequired");
    if (publicUrl.trim()) {
      try {
        const url = new URL(publicUrl.trim());
        if (
          url.protocol !== "https:" ||
          !url.hostname ||
          url.username ||
          url.password ||
          url.search ||
          url.hash
        ) {
          return t("connection.publicUrlError");
        }
      } catch {
        return t("connection.publicUrlError");
      }
    }
    return null;
  };

  const test = async () => {
    const invalid = validate();
    if (invalid) {
      setError(invalid);
      return;
    }
    setError(null);
    setTesting(true);
    try {
      const buckets = await api.TestDraft(wire());
      setTestResult({ signature, ok: true, buckets });
    } catch (cause) {
      setTestResult({ signature, ok: false, message: errorMessage(cause) });
    } finally {
      setTesting(false);
    }
  };

  const save = async () => {
    const invalid = validate();
    if (invalid) {
      setError(invalid);
      return;
    }
    setError(null);
    setSaving(true);
    try {
      const saved = await api.SaveProfile(wire());
      await queryClient.invalidateQueries({ queryKey: keys.profiles });
      await queryClient.invalidateQueries({ queryKey: keys.buckets(saved.id) });
      toast.success(t("connection.saved"));
      onSaved(saved);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    setDeleting(true);
    setError(null);
    try {
      await api.RemoveProfile(profile.id, true);
      await queryClient.invalidateQueries({ queryKey: keys.profiles });
      forgetProfile(profile.id);
      setConfirmDelete(false);
      close(null);
      toast.success(t("connection.deleted"));
    } catch (cause) {
      setError(errorMessage(cause));
      setConfirmDelete(false);
    } finally {
      setDeleting(false);
    }
  };

  return (
    <>
      <LeftColumn>
        <form
          className="flex min-h-0 flex-1 flex-col"
          onSubmit={(event) => {
            event.preventDefault();
            void save();
          }}
        >
          <StepBody
            footer={
              <>
                <Button
                  variant="secondary"
                  size="sm"
                  disabled={testing || saving}
                  onClick={() => void test()}
                >
                  {testing ? <LoaderCircleIcon className="size-3.5 animate-spin" /> : null}
                  {testing ? t("connection.testing") : t("connection.test")}
                </Button>
                <span className="flex-1" />
                <Button variant="secondary" size="sm" disabled={saving} onClick={() => close(null)}>
                  {t("common.cancel")}
                </Button>
                <Button type="submit" size="sm" disabled={saving || testing}>
                  {saving ? <LoaderCircleIcon className="size-3.5 animate-spin" /> : null}
                  {t("common.save")}
                </Button>
              </>
            }
          >
            <div className="flex flex-col gap-3">
              <ProviderHeading />
              <h1 className="text-[28px] leading-[1.12] font-[650] tracking-[-0.022em]">
                {t("connection.titleEdit")}
              </h1>
              <p className="text-[13.5px] leading-relaxed text-muted-foreground">
                {t("connection.cloudflare.editLead")}
              </p>
            </div>

            <Field label={t("connection.name")}>
              <Input
                value={name}
                autoFocus
                onChange={(event) => setName(event.target.value)}
                maxLength={100}
              />
            </Field>

            <div className="grid grid-cols-[minmax(0,1fr)_140px] gap-3">
              <Field label={t("connection.accountId")}>
                <Input value={profile.accountId} disabled className="font-mono text-[12px]" />
              </Field>
              <Field label={t("connection.jurisdiction")}>
                <Select
                  value={jurisdiction}
                  onValueChange={(value) => setJurisdiction(value as Jurisdiction)}
                >
                  <SelectTrigger>
                    <SelectValue />
                  </SelectTrigger>
                  <SelectContent>
                    {JURISDICTIONS.map((value) => (
                      <SelectItem key={value} value={value}>
                        {value === "default" ? "Default" : value.toUpperCase()}
                      </SelectItem>
                    ))}
                  </SelectContent>
                </Select>
              </Field>
            </div>

            <Field label={t("connection.bucket")}>
              <Input value={profile.bucket} disabled />
            </Field>

            <Field label={t("connection.publicUrl")} hint={t("connection.publicUrlHint")}>
              <Input
                value={publicUrl}
                onChange={(event) => setPublicUrl(event.target.value)}
                placeholder="https://cdn.example.com"
                className="font-mono text-[12px]"
              />
            </Field>

            <SwitchRow
              id="cf-edit-read-only"
              title={t("connection.readOnly")}
              description={t("connection.readOnlyHint")}
              checked={readOnly}
              onCheckedChange={setReadOnly}
            />

            <p className="text-[11px] leading-relaxed text-faint">
              {t("connection.cloudflare.scopeChangeHint")}
            </p>

            {shownTest ? (
              <div
                className={cn(
                  "flex items-start gap-2 rounded-xl px-3.5 py-2.5 text-[11.5px] leading-relaxed",
                  shownTest.ok
                    ? "bg-success-soft text-success-text"
                    : "bg-destructive-soft text-destructive",
                )}
              >
                {shownTest.ok ? (
                  <CheckIcon className="mt-0.5 size-3.5 flex-none" />
                ) : (
                  <AlertTriangleIcon className="mt-0.5 size-3.5 flex-none" />
                )}
                <div className="min-w-0">
                  {shownTest.ok
                    ? t("connection.testOk", { count: shownTest.buckets?.length ?? 0 })
                    : t("connection.testFail", { message: shownTest.message ?? "" })}
                </div>
              </div>
            ) : null}

            {error ? (
              <p className="rounded-xl bg-destructive-soft px-3.5 py-2.5 text-[11.5px] text-destructive">
                {error}
              </p>
            ) : null}

            <div className="mt-2 flex items-center justify-between gap-3 border-t pt-5">
              <div className="flex min-w-0 flex-col">
                <span className="text-[12.5px] font-medium">{t("connection.delete")}</span>
                <span className="text-[11px] leading-relaxed text-muted-foreground">
                  {t("connection.deleteDesc")}
                </span>
              </div>
              <Button
                variant="ghost-destructive"
                size="sm"
                className="flex-none"
                onClick={() => setConfirmDelete(true)}
              >
                {t("connection.delete")}
              </Button>
            </div>
          </StepBody>
        </form>
      </LeftColumn>

      <RightColumn>
        <CloudflareAuthorization profile={profile} />
        <Panel title={t("connection.manual.facts")}>
          <dl className="flex flex-col gap-2.5">
            <FactRow label={t("connection.provider")} value="Cloudflare R2" />
            <FactRow label={t("connection.accountId")} value={profile.accountId} mono />
            <FactRow label={t("connection.bucket")} value={profile.bucket || "—"} />
            <FactRow
              label={t("connection.jurisdiction")}
              value={jurisdiction === "default" ? "Default" : jurisdiction.toUpperCase()}
            />
          </dl>
        </Panel>
      </RightColumn>

      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogTitle>{t("connection.deleteTitle", { name: profile.name })}</AlertDialogTitle>
          <AlertDialogDescription>{t("connection.deleteDesc")}</AlertDialogDescription>
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
                disabled={deleting}
                onClick={(event) => {
                  event.preventDefault();
                  void remove();
                }}
              >
                {deleting ? <LoaderCircleIcon className="size-3.5 animate-spin" /> : null}
                {t("connection.delete")}
              </Button>
            </AlertDialogAction>
          </AlertDialogFooter>
        </AlertDialogContent>
      </AlertDialog>
    </>
  );
}

function FactRow({ label, value, mono }: { label: ReactNode; value: ReactNode; mono?: boolean }) {
  return (
    <div className="flex items-baseline gap-3">
      <dt className="w-24 flex-none text-[11.5px] text-muted-foreground">{label}</dt>
      <dd
        className={cn(
          "min-w-0 flex-1 truncate text-right text-[11.5px] text-foreground",
          mono && "font-mono text-[11px]",
        )}
      >
        {value}
      </dd>
    </div>
  );
}

/** The stored authorization of an existing connection, shown beside the edit form. */
export function CloudflareAuthorization({ profile }: { profile: Profile }) {
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
    <>
      <Panel title={t("connection.cloudflare.authorization")}>
        <div className="flex items-center gap-2.5">
          <span
            className={cn(
              "grid size-9 flex-none place-items-center rounded-[10px]",
              authorized ? "bg-success-soft text-success-text" : "bg-muted text-muted-foreground",
            )}
          >
            {authorized ? (
              <ShieldCheckIcon className="size-4" />
            ) : (
              <UnplugIcon className="size-4" />
            )}
          </span>
          <div className="flex min-w-0 flex-col">
            <span className="text-[12.5px] font-semibold">
              {authorized
                ? t("connection.cloudflare.authorized")
                : t("connection.cloudflare.notAuthorized")}
            </span>
            <span className="truncate text-[11px] text-muted-foreground">
              {authorized
                ? t("connection.cloudflare.authorizedNote", { token: status?.tokenName ?? "" })
                : t("connection.cloudflare.notAuthorizedNote")}
            </span>
          </div>
        </div>

        <div className="flex flex-wrap items-center gap-2">
          <Button
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
          <Button
            variant="secondary"
            size="sm"
            disabled={busy || signingIn}
            onClick={() => void signInAgain()}
          >
            <ExternalLinkIcon className="size-3.5" />
            {t("connection.cloudflare.signInAgain")}
          </Button>
          {authorized ? (
            <Button
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
      </Panel>

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
    </>
  );
}
