import { useDeferredValue, useEffect, useMemo, useState } from "react";
import { LoaderCircle, Search, X } from "lucide-react";
import { ProviderIcon } from "./ProviderIcon";
import {
  getProviderIconCatalog,
  getRecommendedProviderIconKeys,
  normalizeProviderIconKey,
  type ProviderIconCatalogEntry,
} from "../../lib/provider-icons";

type ProviderIconPickerProps = {
  value: string;
  onChange: (value: string) => void;
};

export function ProviderIconPicker({ value, onChange }: ProviderIconPickerProps) {
  const [catalog, setCatalog] = useState<ProviderIconCatalogEntry[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [search, setSearch] = useState("");
  const deferredSearch = useDeferredValue(search);
  const normalizedValue = normalizeProviderIconKey(value);

  useEffect(() => {
    let active = true;

    async function loadCatalog() {
      setLoading(true);
      setError("");
      try {
        const icons = await getProviderIconCatalog();
        if (!active) {
          return;
        }
        setCatalog(icons);
      } catch (loadError) {
        if (!active) {
          return;
        }
        setError(loadError instanceof Error ? loadError.message : "图标库加载失败");
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void loadCatalog();
    return () => {
      active = false;
    };
  }, []);

  const recommendedIcons = useMemo(() => {
    const recommendedKeys = new Set(getRecommendedProviderIconKeys());
    return catalog.filter((icon) => recommendedKeys.has(icon.key)).slice(0, 12);
  }, [catalog]);

  const selectedIcon = useMemo(
    () => catalog.find((icon) => icon.key === normalizedValue) ?? null,
    [catalog, normalizedValue],
  );

  const visibleIcons = useMemo(() => {
    const query = deferredSearch.trim().toLowerCase();
    if (!query) {
      return recommendedIcons;
    }

    return catalog.filter((icon) => icon.searchText.includes(query)).slice(0, 24);
  }, [catalog, deferredSearch, recommendedIcons]);

  return (
    <div className="md:col-span-2 rounded-xl border border-slate-200 bg-slate-50/70 p-4">
      <div className="flex items-start justify-between gap-3">
        <div>
          <h4 className="text-sm font-semibold text-slate-800">Provider 图标</h4>
          <p className="mt-1 text-xs text-slate-500">基于 thesvg 品牌图标库选择，保存的是库内 icon key。</p>
        </div>
        {normalizedValue ? (
          <button
            type="button"
            onClick={() => onChange("")}
            className="inline-flex items-center gap-1 rounded-md border border-slate-200 bg-white px-2 py-1 text-xs text-slate-500 transition-colors hover:bg-slate-100 hover:text-slate-700"
          >
            <X className="h-3.5 w-3.5" />
            清空
          </button>
        ) : null}
      </div>

      <div className="mt-4 flex items-center gap-3 rounded-xl border border-slate-200 bg-white p-3">
        <ProviderIcon iconKey={normalizedValue} className="h-12 w-12 rounded-2xl p-2" title={selectedIcon?.title ?? "未选择图标"} />
        <div className="min-w-0 flex-1">
          <div className="truncate text-sm font-medium text-slate-800">{selectedIcon?.title ?? "未选择图标"}</div>
          <div className="mt-1 truncate text-xs text-slate-500">{normalizedValue || "可按名称、品牌或分类搜索"}</div>
        </div>
      </div>

      <label className="mt-4 block text-xs font-medium uppercase tracking-[0.16em] text-slate-400">搜索图标</label>
      <div className="relative mt-2">
        <Search className="pointer-events-none absolute top-1/2 left-3 h-4 w-4 -translate-y-1/2 text-slate-400" />
        <input
          value={search}
          onChange={(event) => setSearch(event.target.value)}
          placeholder="例如 github、cursor、openai、docker"
          className="w-full rounded-md border border-slate-300 bg-white py-2 pr-3 pl-9 text-sm text-slate-700 outline-none transition-all focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
        />
      </div>

      {loading ? (
        <div className="mt-4 flex items-center gap-2 rounded-xl border border-dashed border-slate-200 bg-white px-3 py-4 text-sm text-slate-500">
          <LoaderCircle className="h-4 w-4 animate-spin text-blue-600" />
          正在加载 thesvg 图标库…
        </div>
      ) : error ? (
        <div className="mt-4 rounded-xl border border-red-200 bg-red-50 px-3 py-4 text-sm text-red-600">{error}</div>
      ) : (
        <div className="mt-4 space-y-3">
          {!deferredSearch.trim() && recommendedIcons.length > 0 ? (
            <div>
              <div className="mb-2 text-xs font-medium uppercase tracking-[0.16em] text-slate-400">推荐图标</div>
              <div className="flex flex-wrap gap-2">
                {recommendedIcons.map((icon) => {
                  const selected = icon.key === normalizedValue;
                  return (
                    <button
                      key={icon.key}
                      type="button"
                      onClick={() => onChange(icon.key)}
                      className={`inline-flex items-center gap-2 rounded-full border px-3 py-1.5 text-xs transition ${selected ? "border-blue-200 bg-blue-50 text-blue-700" : "border-slate-200 bg-white text-slate-600 hover:bg-slate-50 hover:text-slate-800"}`}
                    >
                      <ProviderIcon iconKey={icon.key} className="h-5 w-5 rounded-md p-1" title={icon.title} />
                      <span>{icon.title}</span>
                    </button>
                  );
                })}
              </div>
            </div>
          ) : null}

          <div>
            <div className="mb-2 text-xs font-medium uppercase tracking-[0.16em] text-slate-400">{deferredSearch.trim() ? `搜索结果 ${visibleIcons.length}` : "图标候选"}</div>
            <div className="max-h-72 overflow-auto rounded-xl border border-slate-200 bg-white p-2">
              {visibleIcons.length === 0 ? (
                <div className="px-2 py-6 text-center text-sm text-slate-500">没有匹配的图标</div>
              ) : (
                <div className="grid grid-cols-1 gap-2 md:grid-cols-2 xl:grid-cols-3">
                  {visibleIcons.map((icon) => {
                    const selected = icon.key === normalizedValue;
                    return (
                      <button
                        key={icon.key}
                        type="button"
                        onClick={() => onChange(icon.key)}
                        className={`flex items-center gap-3 rounded-xl border p-3 text-left transition ${selected ? "border-blue-200 bg-blue-50" : "border-slate-200 bg-white hover:bg-slate-50"}`}
                      >
                        <ProviderIcon iconKey={icon.key} className="h-10 w-10 rounded-xl p-2" title={icon.title} />
                        <div className="min-w-0">
                          <div className="truncate text-sm font-medium text-slate-800">{icon.title}</div>
                          <div className="truncate text-xs text-slate-500">{icon.key}</div>
                        </div>
                      </button>
                    );
                  })}
                </div>
              )}
            </div>
          </div>
        </div>
      )}
    </div>
  );
}