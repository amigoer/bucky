import type { TFunction } from "i18next";
import {
  ClipboardPasteIcon,
  ImagePlusIcon,
  LoaderCircleIcon,
  UploadIcon,
  ZapIcon,
} from "lucide-react";
import { useEffect, useMemo, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { toast } from "sonner";
import { EmptyState } from "@/components/empty-state";
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
import { Spinner } from "@/components/ui/spinner";
import { Switch } from "@/components/ui/switch";
import { copyText } from "@/lib/actions";
import { api, type OperationPlan } from "@/lib/api";
import { errorCode, errorMessage } from "@/lib/errors";
import { extension, formatBytes } from "@/lib/format";
import { useNav } from "@/lib/nav";
import { useBuckets, useProfiles } from "@/lib/queries";
import { type CopyFormat, usePrefs } from "@/lib/store";
import { markdownImage, PATH_TEMPLATES, renderTemplate, sanitizeName } from "@/lib/template";
import { cn } from "@/lib/utils";

const MAX_BYTES = 64 * 1024 * 1024;

interface Prepared {
  blob: Blob;
  url: string;
  name: string;
  ext: string;
  date: Date;
  hash: string;
}

export function QuickUpload() {
  const value = useNav((s) => s.quickUpload);
  if (!value) return null;
  return <QuickUploadInner initialFile={value.file} />;
}

function QuickUploadInner({ initialFile }: { initialFile?: File }) {
  const { t, i18n } = useTranslation();
  const close = useNav((s) => s.openQuickUpload);
  const openConnection = useNav((s) => s.openConnection);
  const queuePlans = useNav((s) => s.queuePlans);
  const { data: profiles = [] } = useProfiles();
  const quick = usePrefs((s) => s.quick);
  const setQuick = usePrefs((s) => s.setQuick);

  const writable = useMemo(
    () => profiles.filter((profile) => !profile.readOnly && profile.publicUrl),
    [profiles],
  );
  const [profileId, setProfileId] = useState(
    writable.some((profile) => profile.id === quick.profileId)
      ? quick.profileId
      : (writable[0]?.id ?? ""),
  );
  const profile = writable.find((item) => item.id === profileId);
  const { data: buckets = [] } = useBuckets(profile?.id, !!profile);
  const [bucket, setBucket] = useState(quick.bucket || profile?.bucket || "");
  const bucketNames = profile?.bucket ? [profile.bucket] : buckets.map((item) => item.name);

  useEffect(() => {
    if (!profile) return;
    const allowed = profile.bucket ? [profile.bucket] : buckets.map((item) => item.name);
    if (allowed.length > 0 && !allowed.includes(bucket)) setBucket(allowed[0]);
  }, [profile, buckets, bucket]);

  const [template, setTemplate] = useState(quick.template);
  const [compress, setCompress] = useState(quick.compress);
  const [webp, setWebp] = useState(quick.webp);
  const [format, setFormat] = useState<CopyFormat>(quick.format);
  const [source, setSource] = useState<File | null>(initialFile ?? null);
  const [image, setImage] = useState<Prepared | null>(null);
  const [preparing, setPreparing] = useState(false);
  const [uploading, setUploading] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [conflict, setConflict] = useState<{ plan: OperationPlan | null; key: string } | null>(
    null,
  );
  const [done, setDone] = useState<{ url: string; key: string } | null>(null);
  const inputRef = useRef<HTMLInputElement>(null);
  const objectUrl = useRef<string | null>(null);

  const choose = (file: File) => {
    setDone(null);
    setConflict(null);
    setError(null);
    setSource(file);
  };

  useEffect(() => {
    if (!source) return;
    let cancelled = false;
    const prepare = async () => {
      if (source.size > MAX_BYTES) {
        setImage(null);
        setError(t("quick.tooLarge"));
        return;
      }
      setPreparing(true);
      try {
        let blob: Blob = source;
        let ext = extension(source.name) || source.type.split("/")[1] || "png";
        const compressible = /^image\/(png|jpeg|webp)$/.test(source.type);
        if (compressible && (compress || webp)) {
          const bitmap = await createImageBitmap(source);
          const canvas = document.createElement("canvas");
          canvas.width = bitmap.width;
          canvas.height = bitmap.height;
          const context = canvas.getContext("2d");
          if (context) {
            context.drawImage(bitmap, 0, 0);
            const type = webp ? "image/webp" : "image/jpeg";
            const encoded = await new Promise<Blob | null>((resolve) =>
              canvas.toBlob(resolve, type, 0.86),
            );
            if (encoded) {
              blob = encoded;
              ext = webp ? "webp" : "jpg";
            }
          }
        }
        const digest = await crypto.subtle.digest("SHA-256", await blob.arrayBuffer());
        const hash = [...new Uint8Array(digest)]
          .map((byte) => byte.toString(16).padStart(2, "0"))
          .join("");
        if (cancelled) return;
        if (objectUrl.current) URL.revokeObjectURL(objectUrl.current);
        objectUrl.current = URL.createObjectURL(blob);
        setImage({
          blob,
          url: objectUrl.current,
          name: sanitizeName(source.name) || "image",
          ext,
          date: new Date(),
          hash,
        });
      } catch {
        if (!cancelled) {
          setImage(null);
          setError(t("quick.processing"));
        }
      } finally {
        if (!cancelled) setPreparing(false);
      }
    };
    void prepare();
    return () => {
      cancelled = true;
    };
  }, [source, compress, webp, t]);

  // biome-ignore lint/correctness/useExhaustiveDependencies: choose is stable enough for the window-level paste listener
  useEffect(() => {
    const onPaste = (event: ClipboardEvent) => {
      const item = [...(event.clipboardData?.items ?? [])].find((entry) =>
        entry.type.startsWith("image/"),
      );
      const file = item?.getAsFile();
      if (file) {
        event.preventDefault();
        choose(file);
      }
    };
    document.addEventListener("paste", onPaste);
    return () => document.removeEventListener("paste", onPaste);
  }, []);

  // biome-ignore lint/correctness/useExhaustiveDependencies: only react to a newly provided initial file
  useEffect(() => {
    if (initialFile) choose(initialFile);
  }, [initialFile]);

  useEffect(() => {
    return () => {
      if (objectUrl.current) URL.revokeObjectURL(objectUrl.current);
    };
  }, []);

  const key = useMemo(() => {
    if (!image) return "";
    return renderTemplate(PATH_TEMPLATES[template] ?? PATH_TEMPLATES[0], {
      date: image.date,
      hash: image.hash,
      name: image.name,
      ext: image.ext,
    });
  }, [image, template]);

  const publicUrl = (keyName: string) => {
    if (!profile) return "";
    const encoded = keyName.split("/").map(encodeURIComponent).join("/");
    return `${profile.publicUrl.replace(/\/$/, "")}/${encoded}`;
  };

  const finish = async (keyName: string) => {
    const url = publicUrl(keyName);
    setDone({ url, key: keyName });
    const content = format === "markdown" ? markdownImage(image?.name ?? "image", url) : url;
    await copyText(content, true);
    toast.success(t("quick.done"));
    setQuick({ format });
  };

  const upload = async (autoRename = false) => {
    if (!image || !profile || !bucket) return;
    setUploading(true);
    setError(null);
    setConflict(null);
    try {
      let keyName = key;
      if (autoRename) {
        for (let attempt = 1; attempt <= 8; attempt++) {
          const candidate = renderTemplate(
            PATH_TEMPLATES[template] ?? PATH_TEMPLATES[0],
            { date: image.date, hash: image.hash, name: image.name, ext: image.ext },
            `-${attempt}`,
          );
          let free = false;
          try {
            await api.Stat({ profileId, bucket, key: candidate });
          } catch (cause) {
            if (errorCode(cause) === "not_found") {
              free = true;
            } else {
              throw cause;
            }
          }
          if (free) {
            keyName = candidate;
            break;
          }
          if (attempt === 8) {
            setConflict({ plan: null, key: candidate });
            return;
          }
        }
      }
      const base64 = await new Promise<string>((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = () => resolve(String(reader.result).split(",")[1] ?? "");
        reader.onerror = () => reject(reader.error);
        reader.readAsDataURL(image.blob);
      });
      const result = await api.UploadData({ profileId, bucket, key: keyName }, base64);
      if (result.plan) {
        setConflict({ plan: result.plan, key: keyName });
        return;
      }
      await finish(keyName);
    } catch (cause) {
      if (errorCode(cause) === "conflict") {
        setError(t("quick.conflict"));
      } else {
        setError(t("quick.failed", { message: errorMessage(cause) }));
      }
    } finally {
      setUploading(false);
    }
  };

  const overwrite = () => {
    if (!conflict?.plan || !profile) return;
    queuePlans([
      {
        operation: "upload",
        location: { profileId: profile.id, bucket, prefix: "" },
        entries: [],
        plan: conflict.plan,
      },
    ]);
    close(null);
  };

  const again = () => {
    setDone(null);
    setSource(null);
    setImage(null);
    setConflict(null);
    setError(null);
    inputRef.current?.click();
  };

  const setCopyFormat = (next: CopyFormat) => {
    setFormat(next);
    setQuick({ format: next });
  };

  return (
    <Dialog open onOpenChange={(open) => (!open ? close(null) : undefined)}>
      <DialogContent className="w-[520px]">
        <DialogHeader icon={<ZapIcon />} title={t("quick.title")} />
        <div className="flex flex-col gap-3 px-4.5 pb-4">
          {writable.length === 0 ? (
            <EmptyState
              icon={<ImagePlusIcon />}
              title={t("quick.noConnection")}
              className="py-6"
              action={
                <Button
                  size="sm"
                  onClick={() => {
                    close(null);
                    openConnection({});
                  }}
                >
                  {t("welcome.add")}
                </Button>
              }
            />
          ) : (
            <>
              <input
                ref={inputRef}
                type="file"
                accept="image/*"
                className="hidden"
                onChange={(event) => {
                  const file = event.target.files?.[0];
                  event.target.value = "";
                  if (file) choose(file);
                }}
              />

              {done ? (
                <div className="flex flex-col gap-2.5">
                  <div className="grid h-[168px] place-items-center overflow-hidden rounded-xl bg-tile-image">
                    <img src={image?.url} alt="" className="max-h-full max-w-full object-contain" />
                  </div>
                  <Input readOnly value={done.url} className="font-mono text-[11.5px]" />
                  <div className="flex items-center gap-2">
                    <Button
                      variant="secondary"
                      size="sm"
                      onClick={() =>
                        void copyText(
                          format === "markdown"
                            ? markdownImage(image?.name ?? "image", done.url)
                            : done.url,
                        )
                      }
                    >
                      {t("quick.copyAs")} {format === "markdown" ? "Markdown" : "URL"}
                    </Button>
                    <span className="min-w-0 flex-1 truncate font-mono text-[10.5px] text-faint">
                      {done.key}
                    </span>
                    <Button variant="ghost" size="sm" onClick={again}>
                      {t("quick.again")}
                    </Button>
                  </div>
                </div>
              ) : (
                <>
                  <button
                    type="button"
                    onClick={() => inputRef.current?.click()}
                    className={cn(
                      "relative grid h-[168px] place-items-center overflow-hidden rounded-xl border-2 border-dashed transition-colors",
                      image
                        ? "border-transparent bg-tile-image"
                        : "border-border-strong bg-field hover:bg-hover",
                    )}
                  >
                    {preparing ? (
                      <Spinner className="size-5" />
                    ) : image ? (
                      <img
                        src={image.url}
                        alt=""
                        className="max-h-full max-w-full object-contain"
                      />
                    ) : (
                      <span className="flex flex-col items-center gap-1.5 text-muted-foreground">
                        <ClipboardPasteIcon className="size-7" />
                        <span className="text-[12px]">{t("quick.paste")}</span>
                        <span className="text-[11px] text-faint">{t("quick.choose")}</span>
                      </span>
                    )}
                  </button>

                  {image ? (
                    <div className="flex items-center gap-2 text-[11px] text-muted-foreground">
                      <span className="truncate">
                        {image.name}.{image.ext}
                      </span>
                      <span className="text-faint">·</span>
                      <span className="tabular">{formatBytes(image.blob.size, i18n.language)}</span>
                      <span className="ml-auto flex items-center gap-1.5">
                        <span>{t("quick.compress")}</span>
                        <Switch checked={compress} onCheckedChange={setCompress} />
                        <span className="ml-1">{t("quick.webp")}</span>
                        <Switch checked={webp} onCheckedChange={setWebp} />
                      </span>
                    </div>
                  ) : null}

                  <div className="grid grid-cols-[minmax(0,1fr)_minmax(0,1fr)] gap-2">
                    <div className="flex flex-col gap-1">
                      <span className="text-[11px] font-medium text-muted-foreground">
                        {t("quick.connection")}
                      </span>
                      <Select
                        value={profileId}
                        onValueChange={(value) => {
                          setProfileId(value);
                          setQuick({ profileId: value });
                          setBucket("");
                        }}
                      >
                        <SelectTrigger>
                          <SelectValue />
                        </SelectTrigger>
                        <SelectContent>
                          {writable.map((item) => (
                            <SelectItem key={item.id} value={item.id}>
                              {item.name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                    <div className="flex flex-col gap-1">
                      <span className="text-[11px] font-medium text-muted-foreground">
                        {t("quick.bucket")}
                      </span>
                      <Select
                        value={bucket}
                        onValueChange={(value) => {
                          setBucket(value);
                          setQuick({ bucket: value });
                        }}
                        disabled={bucketNames.length === 0}
                      >
                        <SelectTrigger>
                          <SelectValue placeholder={t("common.loading")} />
                        </SelectTrigger>
                        <SelectContent>
                          {bucketNames.map((name) => (
                            <SelectItem key={name} value={name}>
                              {name}
                            </SelectItem>
                          ))}
                        </SelectContent>
                      </Select>
                    </div>
                  </div>

                  <div className="flex flex-col gap-1">
                    <span className="text-[11px] font-medium text-muted-foreground">
                      {t("quick.template")}
                    </span>
                    <Select
                      value={String(template)}
                      onValueChange={(value) => {
                        const index = Number(value);
                        setTemplate(index);
                        setQuick({ template: index });
                      }}
                    >
                      <SelectTrigger>
                        <SelectValue />
                      </SelectTrigger>
                      <SelectContent>
                        {PATH_TEMPLATES.map((value, index) => (
                          <SelectItem key={value} value={String(index)}>
                            <span className="font-mono text-[11.5px]">{value}</span>
                          </SelectItem>
                        ))}
                      </SelectContent>
                    </Select>
                  </div>
                  {key ? (
                    <p className="truncate font-mono text-[10.5px] text-faint" title={key}>
                      {bucket}/{key}
                    </p>
                  ) : null}

                  <div className="flex items-center gap-2">
                    <span className="text-[11px] font-medium text-muted-foreground">
                      {t("quick.copyAs")}
                    </span>
                    <Segmented<CopyFormat>
                      value={format}
                      onValueChange={setCopyFormat}
                      options={[
                        { value: "markdown", label: "Markdown" },
                        { value: "url", label: "URL" },
                      ]}
                      className="w-[190px]"
                    />
                  </div>

                  {conflict ? (
                    <div className="flex flex-col gap-2 rounded-lg bg-warning-soft px-3 py-2.5 text-[11.5px] text-warning-text">
                      <span>{t("quick.conflict")}</span>
                      <div className="flex gap-2">
                        <Button
                          variant="secondary"
                          size="sm"
                          onClick={() => void upload(true)}
                          disabled={uploading}
                        >
                          {t("quick.rename")}
                        </Button>
                        <Button
                          variant="destructive"
                          size="sm"
                          onClick={overwrite}
                          disabled={!conflict.plan}
                        >
                          {t("quick.overwrite")}
                        </Button>
                      </div>
                    </div>
                  ) : null}

                  {error ? (
                    <p className="rounded-md bg-destructive-soft px-2.5 py-1.5 text-[11.5px] text-destructive">
                      {error}
                    </p>
                  ) : null}
                </>
              )}
            </>
          )}
        </div>
        {renderFooter({
          done: !!done,
          image: !!image,
          uploading,
          disabled: !profile || !bucket,
          close,
          upload,
          t,
        })}
      </DialogContent>
    </Dialog>
  );
}

function renderFooter({
  done,
  image,
  uploading,
  disabled,
  close,
  upload,
  t,
}: {
  done: boolean;
  image: boolean;
  uploading: boolean;
  disabled: boolean;
  close: (value: null) => void;
  upload: (autoRename?: boolean) => Promise<void>;
  t: TFunction;
}) {
  if (done) {
    return (
      <DialogFooter>
        <Button size="sm" onClick={() => close(null)}>
          {t("common.done")}
        </Button>
      </DialogFooter>
    );
  }
  return (
    <DialogFooter>
      <Button variant="secondary" size="sm" disabled={uploading} onClick={() => close(null)}>
        {t("common.cancel")}
      </Button>
      <Button
        size="sm"
        disabled={!image || disabled || uploading}
        onClick={() => void upload(false)}
      >
        {uploading ? (
          <LoaderCircleIcon className="size-3.5 animate-spin" />
        ) : (
          <UploadIcon className="size-3.5" />
        )}
        {uploading ? t("quick.uploading") : t("quick.upload")}
      </Button>
    </DialogFooter>
  );
}
