import { FolderOpenIcon } from "lucide-react";
import { useTranslation } from "react-i18next";
import { Brand } from "@/components/brand";
import { EmptyState } from "@/components/empty-state";
import { PageHeader } from "@/components/page-header";
import { Button } from "@/components/ui/button";
import { Spinner } from "@/components/ui/spinner";
import { useNav } from "@/lib/nav";
import { useProfiles } from "@/lib/queries";

export function Welcome() {
  const { t } = useTranslation();
  const openConnection = useNav((s) => s.openConnection);
  const { data: profiles = [], isPending } = useProfiles();

  if (isPending) {
    return (
      <div className="grid h-full place-items-center">
        <Spinner className="size-5" />
      </div>
    );
  }

  if (profiles.length > 0) {
    return (
      <div className="flex min-h-0 flex-1 flex-col">
        <PageHeader />
        <div className="grid flex-1 place-items-center pb-12">
          <EmptyState icon={<FolderOpenIcon />} tone="accent" title={t("welcome.pick")} />
        </div>
      </div>
    );
  }

  return (
    <div className="flex min-h-0 flex-1 flex-col">
      <PageHeader />
      <div className="relative grid flex-1 place-items-center overflow-hidden pb-16">
        <div
          aria-hidden
          className="pointer-events-none absolute top-1/4 left-1/2 size-[520px] -translate-x-1/2 rounded-full bg-primary-tint blur-3xl"
        />
        <div className="relative flex max-w-[440px] flex-col items-center gap-3 px-8 text-center animate-fade">
          <Brand className="mb-1 scale-[1.35]" />
          <h1 className="mt-3 text-[22px] font-bold tracking-[-0.02em]">{t("welcome.title")}</h1>
          <p className="text-[12.5px] leading-relaxed text-muted-foreground">{t("welcome.desc")}</p>
          <Button className="mt-3" onClick={() => openConnection({})}>
            {t("welcome.add")}
          </Button>
        </div>
      </div>
    </div>
  );
}
