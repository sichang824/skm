import type { Provider } from "./api";

type TheSvgIcon = {
  title: string;
  svg: string;
  hex?: string;
  categories?: string[];
  variants?: Record<string, string>;
};

export type ProviderIconCatalogEntry = {
  key: string;
  moduleName: string;
  title: string;
  categories: string[];
  searchText: string;
};

const DEFAULT_ICON_BY_PROVIDER_TYPE: Record<string, string> = {
  codex: "codex_openai",
  cursor: "cursor",
  gitee: "gitee",
  github: "github",
  gitlab: "gitlab",
  global: "github_copilot",
  local: "local",
  openai: "openai",
  repo: "github",
  workspace: "visual_studio_code",
  workbuddy: "github_copilot",
};

const RECOMMENDED_PROVIDER_ICON_KEYS = [
  "github_copilot",
  "cursor",
  "codex_openai",
  "openai",
  "claude_ai",
  "visual_studio_code",
  "github",
  "gitlab",
  "gitee",
  "docker",
  "kubernetes",
  "google_cloud",
  "amazon_web_services",
  "microsoft_azure",
  "local",
  "files",
  "wails",
];

let iconCatalogPromise: Promise<ProviderIconCatalogEntry[]> | null = null;
let iconLoaderMapPromise: Promise<Record<string, () => Promise<{ default?: TheSvgIcon }>>> | null = null;
const iconValueCache = new Map<string, Promise<TheSvgIcon | null>>();

export function normalizeProviderIconKey(value: string | undefined | null) {
  const trimmed = (value ?? "").trim().toLowerCase();
  if (!trimmed) {
    return "";
  }

  const normalized = trimmed
    .replace(/[^a-z0-9]+/g, "_")
    .replace(/^_+|_+$/g, "");

  if (!normalized) {
    return "";
  }

  if (/^\d/.test(normalized) && !normalized.startsWith("i_")) {
    return `i_${normalized}`;
  }

  return normalized;
}

export function getRecommendedProviderIconKeys() {
  return RECOMMENDED_PROVIDER_ICON_KEYS;
}

async function getProviderIconLoaders() {
  if (!iconLoaderMapPromise) {
    iconLoaderMapPromise = import("../generated/provider-icon-loaders").then(
      (module) => module.providerIconLoaders as Record<string, () => Promise<{ default?: TheSvgIcon }>>,
    );
  }

  return iconLoaderMapPromise;
}

export function getProviderIconKey(provider: Pick<Provider, "icon" | "type" | "name" | "rootPath"> | null | undefined) {
  if (!provider) {
    return "";
  }

  const explicit = normalizeProviderIconKey(provider.icon);
  if (explicit) {
    return explicit;
  }

  const providerType = normalizeProviderIconKey(provider.type);
  if (providerType && DEFAULT_ICON_BY_PROVIDER_TYPE[providerType]) {
    return DEFAULT_ICON_BY_PROVIDER_TYPE[providerType];
  }

  const hint = `${provider.name} ${provider.rootPath}`.toLowerCase();
  if (hint.includes("cursor")) {
    return "cursor";
  }
  if (hint.includes("codex")) {
    return "codex_openai";
  }
  if (hint.includes("claude")) {
    return "claude_ai";
  }
  if (hint.includes("openai")) {
    return "openai";
  }
  if (hint.includes("github")) {
    return "github";
  }
  if (hint.includes("gitlab")) {
    return "gitlab";
  }
  if (hint.includes("gitee")) {
    return "gitee";
  }
  if (hint.includes("workspace") || hint.includes("vscode") || hint.includes("code")) {
    return "visual_studio_code";
  }
  if (hint.includes("agent") || hint.includes("copilot")) {
    return "github_copilot";
  }

  return "";
}
export async function getProviderIconSvg(iconKey: string) {
  const normalizedKey = normalizeProviderIconKey(iconKey);
  if (!normalizedKey) {
    return null;
  }

  let iconPromise = iconValueCache.get(normalizedKey);
  if (!iconPromise) {
    iconPromise = getProviderIconLoaders()
      .then((loaders) => loaders[normalizedKey]?.())
      .then((module) => module?.default ?? null)
      .catch(() => null);
    iconValueCache.set(normalizedKey, iconPromise);
  }

  return iconPromise;
}

export async function getProviderIconCatalog() {
  if (!iconCatalogPromise) {
    iconCatalogPromise = import("../generated/provider-icon-catalog.json").then(
      (module) => module.default as ProviderIconCatalogEntry[],
    );
  }

  return iconCatalogPromise;
}