import {
  AlertTriangleIcon,
  ArrowLeftIcon,
  CheckIcon,
  ExternalLinkIcon,
  InfoIcon,
  KeyRoundIcon,
  LoaderCircleIcon,
} from "lucide-react";
import { type ReactNode, useState } from "react";
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
import { Kbd } from "@/components/ui/kbd";
import { Segmented } from "@/components/ui/segmented";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { api, type Bucket, type Profile, type ProfileInput } from "@/lib/api";
import { CLOUDFLARE_DASHBOARD } from "@/lib/cloudflare";
import { errorMessage } from "@/lib/errors";
import { useNav } from "@/lib/nav";
import { keys } from "@/lib/queries";
import { queryClient } from "@/lib/query";
import { cn } from "@/lib/utils";
import { Field, LeftColumn, Note, Panel, RightColumn, StepBody, StepRow, SwitchRow } from "./parts";

const JURISDICTIONS = ["default", "eu", "fedramp"] as const;
type Jurisdiction = (typeof JURISDICTIONS)[number];
type Source = "keychain" | "environment";

/** The fallback path: type the S3 keys yourself, for people who mint tokens by hand. */
export function ManualFlow({
  profile,
  onBack,
  onSaved,
}: {
  profile?: Profile;
  onBack?: () => void;
  onSaved: (saved: Profile) => void;
}) {
  const { t } = useTranslation();
  const close = useNav((s) => s.openConnection);
  const forgetProfile = useNav((s) => s.forgetProfile);
  const existing = profile;

  const [name, setName] = useState(existing?.name ?? "");
  const [accountId, setAccountId] = useState(existing?.accountId ?? "");
  const [source, setSource] = useState<Source>(
    (existing?.credentialSource as Source) || "keychain",
  );
  const [accessKey, setAccessKey] = useState("");
  const [secret, setSecret] = useState("");
  const [bucket, setBucket] = useState(existing?.bucket ?? "");
  const [jurisdiction, setJurisdiction] = useState<Jurisdiction>(
    (existing?.jurisdiction as Jurisdiction) || "default",
  );
  const [publicUrl, setPublicUrl] = useState(existing?.publicUrl ?? "");
  const [readOnly, setReadOnly] = useState(existing?.readOnly ?? false);
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
      id: existing?.id ?? "",
      name: name.trim(),
      providerType: "r2",
      accountId: accountId.trim().toLowerCase(),
      jurisdiction,
      bucket: bucket.trim(),
      publicUrl: publicUrl.trim(),
      readOnly,
      credentialSource: source,
      createdAt: existing?.createdAt ?? "",
      updatedAt: existing?.updatedAt ?? "",
    },
    accessKeyId: source === "keychain" ? accessKey : "",
    secretAccessKey: source === "keychain" ? secret : "",
  });

  const signature = JSON.stringify({
    name,
    accountId,
    source,
    accessKey,
    secret,
    bucket,
    jurisdiction,
    publicUrl,
    readOnly,
  });
  const shownTest = testResult?.signature === signature ? testResult : null;

  const validate = (): string | null => {
    if (!name.trim()) return t("connection.nameRequired");
    if (!/^[0-9a-f]{32}$/i.test(accountId.trim())) return t("connection.accountIdError");
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
    if (source === "keychain") {
      const entered = accessKey || secret;
      const incomplete = !accessKey !== !secret;
      if (incomplete) return t("connection.keysBoth");
      if (!existing && !entered) return t("connection.keysBoth");
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
      toast.success(t(existing ? "connection.saved" : "connection.added"));
      onSaved(saved);
    } catch (cause) {
      setError(errorMessage(cause));
    } finally {
      setSaving(false);
    }
  };

  const remove = async () => {
    if (!existing) return;
    setDeleting(true);
    setError(null);
    try {
      await api.RemoveProfile(existing.id, true);
      await queryClient.invalidateQueries({ queryKey: keys.profiles });
      forgetProfile(existing.id);
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
              {onBack ? (
                <Button variant="ghost" size="sm" className="self-start" onClick={onBack}>
                  <ArrowLeftIcon className="size-3.5" />
                  {t("connection.cloudflare.backToSignIn")}
                </Button>
              ) : null}
              <h1 className="text-[28px] leading-[1.12] font-[650] tracking-[-0.022em]">
                {existing ? t("connection.titleEdit") : t("connection.manual.title")}
              </h1>
              <p className="text-[13.5px] leading-relaxed text-muted-foreground">
                {t("connection.manual.lead")}
              </p>
            </div>

            <Field label={t("connection.name")}>
              <Input
                value={name}
                autoFocus={!existing}
                onChange={(event) => setName(event.target.value)}
                placeholder="Acme assets"
                maxLength={100}
              />
            </Field>

            <div className="grid grid-cols-[minmax(0,1fr)_140px] gap-3">
              <Field label={t("connection.accountId")}>
                <Input
                  value={accountId}
                  onChange={(event) => setAccountId(event.target.value)}
                  placeholder="0123456789abcdef0123456789abcdef"
                  className="font-mono text-[12px]"
                />
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

            <Field label={t("connection.credentialSource")}>
              <Segmented<Source>
                value={source}
                onValueChange={setSource}
                options={[
                  { value: "keychain", label: t("connection.keychain") },
                  { value: "environment", label: t("connection.environment") },
                ]}
              />
            </Field>

            {source === "keychain" ? (
              <div className="grid grid-cols-2 gap-3">
                <Field label={t("connection.accessKeyId")}>
                  <Input
                    value={accessKey}
                    onChange={(event) => setAccessKey(event.target.value)}
                    placeholder={existing ? t("connection.savedInKeychain") : ""}
                    autoComplete="off"
                    className="font-mono text-[12px]"
                  />
                </Field>
                <Field label={t("connection.secretAccessKey")}>
                  <Input
                    type="password"
                    value={secret}
                    onChange={(event) => setSecret(event.target.value)}
                    placeholder={existing ? t("connection.savedInKeychain") : ""}
                    autoComplete="new-password"
                    className="font-mono text-[12px]"
                  />
                </Field>
              </div>
            ) : (
              <Note>
                {t("connection.environmentHint")} <Kbd>BUCKY_R2_ACCESS_KEY_ID</Kbd>{" "}
                <Kbd>BUCKY_R2_SECRET_ACCESS_KEY</Kbd>
              </Note>
            )}

            <Field label={t("connection.bucket")} hint={t("connection.bucketHint")}>
              <Input
                value={bucket}
                onChange={(event) => setBucket(event.target.value)}
                placeholder={t("common.optional")}
              />
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
              id="manual-read-only"
              title={t("connection.readOnly")}
              description={t("connection.readOnlyHint")}
              checked={readOnly}
              onCheckedChange={setReadOnly}
            />

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
                  {shownTest.ok && shownTest.buckets && shownTest.buckets.length > 0 ? (
                    <div className="mt-1.5 flex flex-wrap gap-1">
                      {shownTest.buckets.slice(0, 8).map((entry) => (
                        <span
                          key={entry.name}
                          className="rounded bg-white/45 px-1.5 py-0.5 font-mono text-[10.5px]"
                        >
                          {entry.name}
                        </span>
                      ))}
                    </div>
                  ) : null}
                </div>
              </div>
            ) : null}

            {error ? (
              <p className="rounded-xl bg-destructive-soft px-3.5 py-2.5 text-[11.5px] text-destructive">
                {error}
              </p>
            ) : null}

            {existing ? (
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
            ) : null}
          </StepBody>
        </form>
      </LeftColumn>

      <RightColumn>
        {existing ? (
          <Panel title={t("connection.manual.facts")}>
            <dl className="flex flex-col gap-2.5">
              <Fact label={t("connection.provider")} value="Cloudflare R2" />
              <Fact label={t("connection.accountId")} value={existing.accountId} mono />
              <Fact label={t("connection.bucket")} value={existing.bucket || "—"} />
              <Fact
                label={t("connection.credentialSource")}
                value={t(
                  existing.credentialSource === "environment"
                    ? "connection.environment"
                    : "connection.keychain",
                )}
              />
            </dl>
          </Panel>
        ) : (
          <Panel title={t("connection.manual.asideTitle")}>
            <div className="flex flex-col gap-3.5">
              <StepRow index={1} state="todo" title={t("connection.manual.asideStep1")} />
              <StepRow index={2} state="todo" title={t("connection.manual.asideStep2")} />
              <StepRow index={3} state="todo" title={t("connection.manual.asideStep3")} />
            </div>
            <a
              href={CLOUDFLARE_DASHBOARD}
              target="_blank"
              rel="noreferrer"
              className="flex items-center gap-1 self-start text-[11.5px] text-primary-text underline-offset-4 hover:underline"
            >
              {t("settings.cloudflareClients")}
              <ExternalLinkIcon className="size-3" />
            </a>
          </Panel>
        )}
        <Note icon={<KeyRoundIcon />}>{t("connection.manual.asideNote")}</Note>
        <Note icon={<InfoIcon />}>{t("connection.keychainHint")}</Note>
      </RightColumn>

      <AlertDialog open={confirmDelete} onOpenChange={setConfirmDelete}>
        <AlertDialogContent>
          <AlertDialogTitle>
            {t("connection.deleteTitle", { name: existing?.name ?? "" })}
          </AlertDialogTitle>
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

function Fact({ label, value, mono }: { label: ReactNode; value: ReactNode; mono?: boolean }) {
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
