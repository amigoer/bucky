import {
  AlertTriangleIcon,
  CheckIcon,
  CloudIcon,
  LoaderCircleIcon,
  LockIcon,
  ShieldCheckIcon,
  Trash2Icon,
  XIcon,
} from "lucide-react";
import { type ReactNode, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { ConnTile } from "@/components/conn-tile";
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
import {
  Dialog,
  DialogClose,
  DialogContent,
  DialogDescription,
  DialogTitle,
} from "@/components/ui/dialog";
import { Input } from "@/components/ui/input";
import { Kbd } from "@/components/ui/kbd";
import { Label } from "@/components/ui/label";
import { Segmented } from "@/components/ui/segmented";
import {
  Select,
  SelectContent,
  SelectItem,
  SelectTrigger,
  SelectValue,
} from "@/components/ui/select";
import { Switch } from "@/components/ui/switch";
import { api, type Bucket, type Profile, type ProfileInput } from "@/lib/api";
import { errorMessage } from "@/lib/errors";
import { useNav } from "@/lib/nav";
import { keys, useProfiles } from "@/lib/queries";
import { queryClient } from "@/lib/query";
import { cn } from "@/lib/utils";

const JURISDICTIONS = ["default", "eu", "fedramp"] as const;
type Jurisdiction = (typeof JURISDICTIONS)[number];
type Source = "keychain" | "environment";

export function ConnectionDialog() {
  const value = useNav((s) => s.connection);
  return value ? <ConnectionDialogInner key={value.id ?? "new"} id={value.id} /> : null;
}

function ConnectionDialogInner({ id }: { id?: string }) {
  const { t } = useTranslation();
  const close = useNav((s) => s.openConnection);
  const navigate = useNav((s) => s.navigate);
  const forgetProfile = useNav((s) => s.forgetProfile);
  const { data: profiles = [] } = useProfiles();
  const existing = id ? profiles.find((profile) => profile.id === id) : undefined;

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

  const profile = (): Profile => ({
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
  });

  const wire = (): ProfileInput => ({
    profile: profile(),
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
      close(null);
      if (!existing && saved.bucket) {
        navigate({ profileId: saved.id, bucket: saved.bucket, prefix: "" });
      }
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
    <Dialog open onOpenChange={(open) => (!open ? close(null) : undefined)}>
      <DialogContent className="h-[min(600px,calc(100vh-80px))] w-[min(820px,calc(100vw-80px))] p-0">
        <DialogTitle className="sr-only">
          {existing ? t("connection.titleEdit") : t("connection.titleNew")}
        </DialogTitle>
        <DialogDescription className="sr-only">{t("connection.r2Note")}</DialogDescription>
        <div className="flex h-full min-h-0">
          <aside className="flex w-[248px] flex-none flex-col gap-4 border-r bg-sidebar p-5">
            <div className="flex items-center gap-2.5">
              <ConnTile size={34} />
              <div className="flex flex-col">
                <span className="text-[13.5px] font-semibold">{t("connection.r2")}</span>
                <span className="text-[11px] text-faint">{t("connection.provider")}</span>
              </div>
            </div>
            <p className="text-[11.5px] leading-relaxed text-muted-foreground">
              {t("connection.r2Note")}
            </p>
            <ul className="flex flex-col gap-2.5 text-[11.5px] leading-relaxed text-muted-foreground">
              <Bullet icon={<LockIcon />}>{t("connection.keychainHint")}</Bullet>
              <Bullet icon={<ShieldCheckIcon />}>{t("connection.readOnlyHint")}</Bullet>
              <Bullet icon={<CloudIcon />}>{t("connection.bucketHint")}</Bullet>
            </ul>
          </aside>

          <form
            className="flex min-w-0 flex-1 flex-col"
            onSubmit={(event) => {
              event.preventDefault();
              void save();
            }}
          >
            <div className="flex h-12 flex-none items-center gap-2 border-b pr-2 pl-5">
              <span className="text-[14px] font-semibold">
                {existing ? t("connection.titleEdit") : t("connection.titleNew")}
              </span>
              <div className="ml-auto flex items-center gap-1">
                {existing ? (
                  <Button
                    type="button"
                    variant="ghost-destructive"
                    size="icon"
                    aria-label={t("connection.delete")}
                    onClick={() => setConfirmDelete(true)}
                  >
                    <Trash2Icon className="size-4" />
                  </Button>
                ) : null}
                <DialogClose asChild>
                  <Button variant="ghost" size="icon" aria-label={t("common.close")}>
                    <XIcon className="size-4" />
                  </Button>
                </DialogClose>
              </div>
            </div>

            <div className="flex min-h-0 flex-1 flex-col gap-3.5 overflow-y-auto px-5 py-4">
              <Field label={t("connection.name")}>
                <Input
                  value={name}
                  autoFocus
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
                    spellCheck={false}
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
                      spellCheck={false}
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
                <div className="rounded-lg bg-field px-3 py-2 text-[11.5px] leading-relaxed text-muted-foreground">
                  {t("connection.environmentHint")} <Kbd>BUCKY_R2_ACCESS_KEY_ID</Kbd>{" "}
                  <Kbd>BUCKY_R2_SECRET_ACCESS_KEY</Kbd>
                </div>
              )}

              <Field label={t("connection.bucket")} hint={t("connection.bucketHint")}>
                <Input
                  value={bucket}
                  onChange={(event) => setBucket(event.target.value)}
                  placeholder={t("common.optional")}
                  spellCheck={false}
                />
              </Field>

              <Field label={t("connection.publicUrl")} hint={t("connection.publicUrlHint")}>
                <Input
                  value={publicUrl}
                  onChange={(event) => setPublicUrl(event.target.value)}
                  placeholder="https://cdn.example.com"
                  spellCheck={false}
                  className="font-mono text-[12px]"
                />
              </Field>

              <div className="flex items-center justify-between gap-3 rounded-lg border px-3 py-2.5">
                <div className="flex min-w-0 flex-col">
                  <Label htmlFor="read-only" className="text-[12.5px]">
                    {t("connection.readOnly")}
                  </Label>
                  <span className="text-[11px] leading-relaxed text-muted-foreground">
                    {t("connection.readOnlyHint")}
                  </span>
                </div>
                <Switch
                  id="read-only"
                  checked={readOnly}
                  onCheckedChange={setReadOnly}
                  className="flex-none"
                />
              </div>

              {shownTest ? (
                <div
                  className={cn(
                    "flex items-start gap-2 rounded-lg px-3 py-2 text-[11.5px] leading-relaxed",
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
                      <div className="mt-1 flex flex-wrap gap-1">
                        {shownTest.buckets.slice(0, 8).map((entry) => (
                          <span
                            key={entry.name}
                            className="rounded bg-white/40 px-1.5 py-0.5 font-mono text-[10.5px]"
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
                <p className="rounded-lg bg-destructive-soft px-3 py-2 text-[11.5px] text-destructive">
                  {error}
                </p>
              ) : null}
            </div>

            <div className="flex flex-none items-center gap-2 border-t bg-sidebar px-4 py-2.5">
              <Button
                type="button"
                variant="secondary"
                size="sm"
                disabled={testing || saving}
                onClick={() => void test()}
              >
                {testing ? <LoaderCircleIcon className="size-3.5 animate-spin" /> : null}
                {testing ? t("connection.testing") : t("connection.test")}
              </Button>
              <span className="flex-1" />
              <Button
                type="button"
                variant="secondary"
                size="sm"
                disabled={saving}
                onClick={() => close(null)}
              >
                {t("common.cancel")}
              </Button>
              <Button type="submit" size="sm" disabled={saving || testing}>
                {saving ? <LoaderCircleIcon className="size-3.5 animate-spin" /> : null}
                {t("common.save")}
              </Button>
            </div>
          </form>
        </div>
      </DialogContent>

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
    </Dialog>
  );
}

function Field({ label, hint, children }: { label: string; hint?: string; children: ReactNode }) {
  return (
    <div className="flex min-w-0 flex-col gap-1.5">
      <span className="text-[11.5px] font-medium text-muted-foreground">{label}</span>
      {children}
      {hint ? <span className="text-[10.5px] leading-relaxed text-faint">{hint}</span> : null}
    </div>
  );
}

function Bullet({ icon, children }: { icon: ReactNode; children: ReactNode }) {
  return (
    <li className="flex gap-2">
      <span className="mt-0.5 grid size-4 flex-none place-items-center text-muted-foreground [&_svg]:size-3.5">
        {icon}
      </span>
      <span>{children}</span>
    </li>
  );
}
