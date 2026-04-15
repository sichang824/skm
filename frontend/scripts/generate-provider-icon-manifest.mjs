import fs from "node:fs";
import path from "node:path";
import { createRequire } from "node:module";

const require = createRequire(import.meta.url);
const frontendRoot = process.cwd();
const thesvgEntry = require.resolve("thesvg");
const thesvgDistDir = path.dirname(thesvgEntry);
const outputPath = path.join(frontendRoot, "src/generated/provider-icon-catalog.json");
const loaderOutputPath = path.join(frontendRoot, "src/generated/provider-icon-loaders.ts");
const EXCLUDED_MODULE_NAMES = new Set(["svg"]);

function normalizeProviderIconKey(value) {
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

function readIconModules() {
  return fs.readdirSync(thesvgDistDir)
    .filter((fileName) => fileName.endsWith(".cjs") && fileName !== "index.cjs")
    .map((fileName) => ({
      fileName,
      moduleName: fileName.slice(0, -4),
      modulePath: path.join(thesvgDistDir, fileName),
    }))
    .filter((moduleInfo) => !EXCLUDED_MODULE_NAMES.has(moduleInfo.moduleName));
}

async function buildCatalog() {
  const entries = [];

  for (const moduleInfo of readIconModules()) {
    const iconModule = require(moduleInfo.modulePath);
    const icon = iconModule.default ?? iconModule;
    if (!icon || typeof icon.title !== "string" || typeof icon.svg !== "string") {
      continue;
    }

    const key = normalizeProviderIconKey(moduleInfo.moduleName);
    const categories = Array.isArray(icon.categories) ? icon.categories : [];
    entries.push({
      key,
      moduleName: moduleInfo.moduleName,
      title: icon.title,
      categories,
      searchText: `${key} ${moduleInfo.moduleName} ${icon.title} ${categories.join(" ")}`.toLowerCase(),
    });
  }

  entries.sort((left, right) => left.title.localeCompare(right.title, "en"));
  return entries;
}

async function main() {
  const catalog = await buildCatalog();
  fs.mkdirSync(path.dirname(outputPath), { recursive: true });
  fs.writeFileSync(outputPath, `${JSON.stringify(catalog, null, 2)}\n`);
  const loaderSource = `export const providerIconLoaders = {\n${catalog
    .map((entry) => `  ${JSON.stringify(entry.key)}: () => import(${JSON.stringify(`thesvg/${entry.moduleName}`)}),`)
    .join("\n")}\n} as const;\n`;
  fs.writeFileSync(loaderOutputPath, loaderSource);
  console.log(`wrote ${catalog.length} provider icons to ${path.relative(frontendRoot, outputPath)}`);
}

main().catch((error) => {
  console.error(error);
  process.exitCode = 1;
});