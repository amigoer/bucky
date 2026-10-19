import { QueryClientProvider } from "@tanstack/react-query";
import { StrictMode } from "react";
import { createRoot } from "react-dom/client";
import App from "./App";
import "./lib/i18n";
import { queryClient } from "./lib/query";
import { usePrefs } from "./lib/store";
import { applyAppearance } from "./lib/theme";
import "./style.css";

// Apply the persisted theme before the first paint to avoid a light flash.
const { theme, accent } = usePrefs.getState();
applyAppearance(
  theme === "dark" ||
    (theme === "system" && window.matchMedia("(prefers-color-scheme: dark)").matches),
  accent,
);

const root = document.getElementById("root");
if (!root) throw new Error("root element not found");

createRoot(root).render(
  <StrictMode>
    <QueryClientProvider client={queryClient}>
      <App />
    </QueryClientProvider>
  </StrictMode>,
);
