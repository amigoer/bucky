import { CheckIcon, CircleAlertIcon, InfoIcon } from "lucide-react";
import { Toaster as Sonner, type ToasterProps } from "sonner";

function Toaster(props: ToasterProps) {
  return (
    <Sonner
      position="bottom-center"
      offset={22}
      gap={8}
      visibleToasts={3}
      icons={{
        success: <CheckIcon className="size-3.5 text-[oklch(0.8_0.13_150)]" strokeWidth={2.6} />,
        error: <CircleAlertIcon className="size-3.5 text-[oklch(0.76_0.14_27)]" />,
        info: <InfoIcon className="size-3.5 text-white/70" />,
        warning: <CircleAlertIcon className="size-3.5 text-[oklch(0.83_0.12_80)]" />,
      }}
      toastOptions={{
        unstyled: true,
        classNames: {
          toast:
            "flex min-h-9 w-fit max-w-[min(520px,calc(100vw-48px))] items-center gap-2 rounded-xl bg-hud px-3.5 py-2 text-[12.5px] text-[#f5f5f7] shadow-[0_10px_30px_rgb(0_0_0/0.25)] backdrop-blur-xl",
          title: "leading-snug",
          description: "text-[11.5px] text-white/60",
          icon: "flex",
          actionButton:
            "ml-2 h-6 rounded-md bg-white/12 px-2.5 text-xs font-medium text-white hover:bg-white/18",
          cancelButton: "ml-1 h-6 rounded-md px-2 text-xs text-white/60",
        },
      }}
      {...props}
    />
  );
}

export { Toaster };
