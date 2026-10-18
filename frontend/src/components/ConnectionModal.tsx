import {
  ArrowLeft,
  ArrowRight,
  Check,
  Cloud,
  KeyRound,
  LoaderCircle,
  LockKeyhole,
  ShieldCheck,
} from "lucide-react";
import { type FormEvent, useId, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { api, errorMessage, isDesktop, type Profile, type ProfileInput } from "../lib/api";
import { Button, ErrorBanner, Modal, useAsyncState } from "./ui";

export function ConnectionModal({
  profile,
  onClose,
  onSaved,
  onRemove,
}: {
  profile?: Profile;
  onClose: () => void;
  onSaved: (profile: Profile) => void;
  onRemove?: (profile: Profile) => void;
}) {
  const { t } = useTranslation();
  const id = useId();
  const [form, setForm] = useState({
    name: profile?.name ?? "",
    accountId: profile?.accountId ?? "",
    jurisdiction: profile?.jurisdiction ?? "default",
    bucket: profile?.bucket ?? "",
    publicUrl: profile?.publicUrl ?? "",
    readOnly: profile?.readOnly ?? false,
    credentialSource: profile?.credentialSource ?? "keychain",
  });
  const [accessKey, setAccessKey] = useState("");
  const [secretKey, setSecretKey] = useState("");
  const [validation, setValidation] = useState("");
  const [reviewing, setReviewing] = useState(false);
  const [state, run] = useAsyncState();
  const pending = useRef<ProfileInput | null>(null);
  const [testing, setTesting] = useState(false);
  const [testState, setTestState] = useState<{ ok: boolean; text: string } | null>(null);
  const test = async () => {
    if (!profile) return;
    setTesting(true);
    setTestState(null);
    try {
      const buckets = await api.TestConnection(profile.id);
      setTestState({ ok: true, text: t("testSuccess", { count: buckets.length }) });
    } catch (error) {
      setTestState({ ok: false, text: errorMessage(error) });
    } finally {
      setTesting(false);
    }
  };
  const clear = () => {
    pending.current = null;
    setAccessKey("");
    setSecretKey("");
  };
  const close = () => {
    clear();
    onClose();
  };
  const review = (event: FormEvent) => {
    event.preventDefault();
    setValidation("");
    if (!/^[a-fA-F0-9]{32}$/.test(form.accountId.trim())) {
      setValidation(t("invalidAccount"));
      return;
    }
    if (form.publicUrl) {
      try {
        const url = new URL(form.publicUrl);
        if (url.protocol !== "https:" || url.username || url.password || url.search || url.hash)
          throw new Error();
      } catch {
        setValidation(t("invalidPublicUrl"));
        return;
      }
    }
    if (
      form.credentialSource === "keychain" &&
      ((!profile && (!accessKey || !secretKey)) ||
        Boolean(accessKey) !== Boolean(secretKey) ||
        (profile?.credentialSource !== "keychain" && (!accessKey || !secretKey)))
    ) {
      setValidation(t("credentialsRequired"));
      return;
    }
    pending.current = {
      profile: {
        id: profile?.id ?? "",
        ...form,
        name: form.name.trim(),
        accountId: form.accountId.trim(),
        bucket: form.bucket.trim(),
        publicUrl: form.publicUrl.trim(),
        providerType: "r2",
        createdAt: profile?.createdAt ?? "0001-01-01T00:00:00Z",
        updatedAt: profile?.updatedAt ?? "0001-01-01T00:00:00Z",
      },
      accessKeyId: form.credentialSource === "keychain" ? accessKey : "",
      secretAccessKey: form.credentialSource === "keychain" ? secretKey : "",
    };
    setAccessKey("");
    setSecretKey("");
    setReviewing(true);
  };
  const save = async () => {
    const input = pending.current;
    if (!input) return;
    try {
      const result = await api.SaveProfile(input);
      onSaved(result);
      close();
    } finally {
      input.accessKeyId = "";
      input.secretAccessKey = "";
      pending.current = null;
    }
  };
  const change = (name: keyof typeof form, value: string | boolean) =>
    setForm((old) => ({ ...old, [name]: value }));
  return (
    <Modal
      title={
        reviewing ? t("connectionConfirmTitle") : profile ? t("editConnection") : t("newConnection")
      }
      subtitle={t("connectionSubtitle")}
      onClose={close}
      busy={state.busy}
      footer={
        reviewing ? (
          <>
            <Button
              disabled={state.busy}
              onClick={() => {
                clear();
                setReviewing(false);
              }}
            >
              <ArrowLeft size={16} />
              {t("back")}
            </Button>
            <Button
              variant="primary"
              disabled={state.busy || !isDesktop() || !pending.current}
              onClick={() => {
                void run(save);
              }}
            >
              {state.busy ? <LoaderCircle className="spin" size={16} /> : <Check size={16} />}
              {profile ? t("saveChanges") : t("saveConnection")}
            </Button>
          </>
        ) : (
          <>
            <Button onClick={close}>{t("cancel")}</Button>
            <Button variant="primary" type="submit" form={`${id}-form`} disabled={!isDesktop()}>
              {t("reviewConnection")}
              <ArrowRight size={16} />
            </Button>
          </>
        )
      }
    >
      {!isDesktop() && (
        <div className="notice">
          <LockKeyhole size={17} />
          <span>{t("desktopRequired")}</span>
        </div>
      )}
      {reviewing ? (
        <div className="connection-review">
          <div className="provider-card">
            <span className="provider-icon">
              <Cloud size={25} />
            </span>
            <div>
              <strong>{form.name}</strong>
              <span>Cloudflare R2</span>
            </div>
            <ShieldCheck size={20} />
          </div>
          <p>{t("reviewConnectionHint")}</p>
          <dl className="detail-list">
            <dt>{t("accountId")}</dt>
            <dd className="mono">{form.accountId}</dd>
            <dt>{t("jurisdiction")}</dt>
            <dd>{form.jurisdiction}</dd>
            <dt>{t("bucketScope")}</dt>
            <dd>{form.bucket || t("allBuckets")}</dd>
            <dt>{t("credentialSource")}</dt>
            <dd>{t(form.credentialSource === "keychain" ? "keychain" : "environment")}</dd>
            <dt>{t("readonly")}</dt>
            <dd>{form.readOnly ? t("readOnlyBadge") : "—"}</dd>
            {form.publicUrl && (
              <>
                <dt>{t("publicLink")}</dt>
                <dd className="break-all">{form.publicUrl}</dd>
              </>
            )}
          </dl>
          <div className="notice">
            <KeyRound size={17} />
            <span>{t("credentialsCleared")}</span>
          </div>
        </div>
      ) : (
        <form id={`${id}-form`} className="connection-form" onSubmit={review} autoComplete="off">
          <div className="provider-card">
            <span className="provider-icon">
              <Cloud size={25} />
            </span>
            <div>
              <strong>Cloudflare R2</strong>
              <span>S3-compatible object storage</span>
            </div>
            <span className="tag">R2</span>
          </div>
          <label className="field" htmlFor={`${id}-name`}>
            <span>{t("connectionName")}</span>
            <input
              id={`${id}-name`}
              value={form.name}
              onChange={(e) => change("name", e.target.value)}
              required
              maxLength={100}
              placeholder="My workspace"
            />
          </label>
          <label className="field" htmlFor={`${id}-account`}>
            <span>{t("accountId")}</span>
            <input
              id={`${id}-account`}
              className="mono"
              value={form.accountId}
              onChange={(e) => change("accountId", e.target.value)}
              required
              pattern="[a-fA-F0-9]{32}"
              maxLength={32}
              placeholder="0123456789abcdef0123456789abcdef"
              spellCheck={false}
            />
            <small>{t("accountHint")}</small>
          </label>
          <div className="field-row">
            <label className="field" htmlFor={`${id}-jurisdiction`}>
              <span>{t("jurisdiction")}</span>
              <select
                id={`${id}-jurisdiction`}
                value={form.jurisdiction}
                onChange={(e) => change("jurisdiction", e.target.value)}
              >
                <option value="default">{t("defaultJurisdiction")}</option>
                <option value="eu">{t("euJurisdiction")}</option>
                <option value="fedramp">{t("fedrampJurisdiction")}</option>
              </select>
            </label>
            <label className="field" htmlFor={`${id}-bucket`}>
              <span>{t("bucketOptional")}</span>
              <input
                id={`${id}-bucket`}
                value={form.bucket}
                onChange={(e) => change("bucket", e.target.value)}
                maxLength={63}
                placeholder="my-bucket"
                spellCheck={false}
              />
            </label>
          </div>
          <p className="field-hint">{t("bucketHint")}</p>
          <label className="field" htmlFor={`${id}-public`}>
            <span>{t("publicUrl")}</span>
            <input
              id={`${id}-public`}
              type="url"
              value={form.publicUrl}
              onChange={(e) => change("publicUrl", e.target.value)}
              placeholder="https://assets.example.com"
            />
            <small>
              {t("publicUrlHint")} {t("publicDomainScope")}
            </small>
          </label>
          <label className="checkbox-card" htmlFor={`${id}-readonly`}>
            <input
              id={`${id}-readonly`}
              type="checkbox"
              checked={form.readOnly}
              onChange={(e) => change("readOnly", e.target.checked)}
            />
            <span>
              <strong>{t("readonly")}</strong>
              <small>{t("readonlyHint")}</small>
            </span>
            <LockKeyhole size={17} />
          </label>
          <fieldset className="credential-fieldset">
            <legend>{t("credentialSource")}</legend>
            <div className="segmented">
              <button
                type="button"
                aria-pressed={form.credentialSource === "keychain"}
                onClick={() => change("credentialSource", "keychain")}
              >
                <KeyRound size={15} />
                {t("keychain")}
              </button>
              <button
                type="button"
                aria-pressed={form.credentialSource === "environment"}
                onClick={() => {
                  change("credentialSource", "environment");
                  clear();
                }}
              >
                {t("environment")}
              </button>
            </div>
            {form.credentialSource === "keychain" ? (
              <>
                <label className="field" htmlFor={`${id}-access`}>
                  <span>{t("accessKey")}</span>
                  <input
                    id={`${id}-access`}
                    value={accessKey}
                    onChange={(e) => setAccessKey(e.target.value)}
                    type="password"
                    autoComplete="new-password"
                    spellCheck={false}
                  />
                </label>
                <label className="field" htmlFor={`${id}-secret`}>
                  <span>{t("secretKey")}</span>
                  <input
                    id={`${id}-secret`}
                    value={secretKey}
                    onChange={(e) => setSecretKey(e.target.value)}
                    type="password"
                    autoComplete="new-password"
                    spellCheck={false}
                  />
                </label>
                <p className="field-hint">
                  {t("credentialsHint")}{" "}
                  {profile?.credentialSource === "keychain" && t("keepCredentials")}
                </p>
              </>
            ) : (
              <div className="environment-help">
                <p>{t("environmentHint")}</p>
                <code>
                  BUCKY_R2_ACCESS_KEY_ID
                  <br />
                  BUCKY_R2_SECRET_ACCESS_KEY
                  <br />
                  BUCKY_R2_SESSION_TOKEN
                </code>
              </div>
            )}
          </fieldset>
          {profile ? (
            <div className="connection-tools">
              <Button
                variant="secondary"
                onClick={() => void test()}
                disabled={testing || state.busy}
              >
                {testing ? <LoaderCircle className="spin" size={15} /> : <ShieldCheck size={15} />}
                {testing ? t("connecting") : t("testConnection")}
              </Button>
              {onRemove ? (
                <Button variant="ghost" onClick={() => onRemove(profile)} disabled={state.busy}>
                  {t("removeConnection")}
                </Button>
              ) : null}
              {testState ? (
                <span className={`inline-result ${testState.ok ? "ok" : "failed"}`}>
                  {testState.text}
                </span>
              ) : null}
            </div>
          ) : null}
        </form>
      )}
      {(validation || state.error) && <ErrorBanner message={validation || state.error} />}
    </Modal>
  );
}
