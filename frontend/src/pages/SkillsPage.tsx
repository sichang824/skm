import { useDeferredValue, useEffect, useMemo, useState } from "react";
import { FolderTree, Search } from "lucide-react";
import { useNavigate, useOutletContext, useParams } from "react-router-dom";
import { SkillDetailDialog } from "../components/skm/SkillDetailDialog";
import type { ShellOutletContext } from "../components/skm/ConsoleShell";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "../components/ui/accordion";
import { api, type Provider, type Skill } from "../lib/api";

export function SkillsPage() {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [skills, setSkills] = useState<Skill[]>([]);
  const [search, setSearch] = useState("");
  const [selectedProviderZid, setSelectedProviderZid] = useState("");
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const { refreshKey } = useOutletContext<ShellOutletContext>();
  const { zid } = useParams();
  const navigate = useNavigate();

  const deferredQuery = useDeferredValue(search);

  const skillCountByProvider = useMemo(() => {
    const counts = new Map<string, number>();
    for (const skill of skills) {
      const key = skill.provider?.zid;
      if (!key) {
        continue;
      }
      counts.set(key, (counts.get(key) ?? 0) + 1);
    }
    return counts;
  }, [skills]);

  const selectedProvider = useMemo(
    () => providers.find((item) => item.zid === selectedProviderZid) ?? null,
    [providers, selectedProviderZid],
  );

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");

    async function loadSkills() {
      try {
        const [skillData, providerData] = await Promise.all([
          api.getSkills({ sort: "lastScanned" }),
          api.getProviders(),
        ]);
        if (!active) {
          return;
        }
        setSkills(skillData);
        setProviders(providerData);
      } catch (loadError) {
        if (!active) {
          return;
        }
        setError(loadError instanceof Error ? loadError.message : "Failed to load skills");
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void loadSkills();
    return () => {
      active = false;
    };
  }, [refreshKey]);

  useEffect(() => {
    if (selectedProviderZid && !providers.some((item) => item.zid === selectedProviderZid)) {
      setSelectedProviderZid("");
    }
  }, [providers, selectedProviderZid]);

  const filteredSkills = useMemo(() => {
    return skills.filter((skill) => {
      const matchesSearch = deferredQuery.trim() === ""
        ? true
        : [skill.name, skill.summary, skill.provider?.name, skill.category, skill.directoryName]
          .filter(Boolean)
          .some((value) => String(value).toLowerCase().includes(deferredQuery.toLowerCase()));
      const matchesProvider = selectedProviderZid ? skill.provider?.zid === selectedProviderZid : true;
      const matchesStatus = status === ""
        ? true
        : status === "Valid"
          ? skill.status === "ready" && !skill.isConflict
          : status === "Conflict"
            ? skill.isConflict
            : skill.status === "invalid";
      return matchesSearch && matchesProvider && matchesStatus;
    });
  }, [deferredQuery, selectedProviderZid, skills, status]);

  return (
    <div className="relative flex h-full min-h-0 flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="skm-section-title">Skills 目录</h2>
          <p className="mt-1 text-sm text-slate-500">
            {selectedProvider ? `${selectedProvider.name} · ${skillCountByProvider.get(selectedProvider.zid) ?? 0} skills` : `全部 Providers · ${skills.length} skills`}
          </p>
        </div>

        <div className="flex flex-wrap gap-2">
          <div className="relative">
            <Search className="pointer-events-none absolute top-2.5 left-2.5 h-4 w-4 text-slate-400" />
            <input
              type="text"
              value={search}
              onChange={(event) => setSearch(event.target.value)}
              placeholder="搜索 Skill..."
              className="w-64 rounded-md border border-slate-300 bg-white py-2 pr-3 pl-8 text-sm text-slate-700 outline-none transition-all focus:border-blue-500 focus:ring-2 focus:ring-blue-100"
            />
          </div>
          <select value={status} onChange={(event) => setStatus(event.target.value)} className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-700 outline-none transition-all focus:border-blue-500">
            <option value="">全部状态</option>
            <option value="Valid">正常</option>
            <option value="Conflict">冲突</option>
            <option value="Error">异常</option>
          </select>
        </div>
      </div>

      {error ? <p className="px-4 py-3 text-sm text-red-600">{error}</p> : null}

      <div className="grid min-h-0 flex-1 gap-4 overflow-visible lg:grid-cols-[280px_minmax(0,1fr)]">
        <aside className="skm-card min-h-0 overflow-visible">
          <div className="border-b border-slate-200 px-4 py-4">
            <div className="flex items-center gap-2 text-sm font-semibold text-slate-800">
              <FolderTree className="h-4 w-4 text-blue-600" />
              <span>Providers</span>
            </div>
            <p className="mt-1 text-xs text-slate-500">按 Provider 查看 Skills 分布与数量</p>
          </div>
          <div className="max-h-full overflow-auto px-4 py-3">
            <Accordion type="single" collapsible defaultValue="providers" className="w-full">
              <AccordionItem value="providers" className="border-b-0">
                <AccordionTrigger className="py-3 text-sm font-semibold text-slate-700 hover:no-underline">
                  <div className="flex flex-1 items-center justify-between gap-3 pr-2">
                    <span>Providers 列表</span>
                    <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-500">{providers.length}</span>
                  </div>
                </AccordionTrigger>
                <AccordionContent className="pb-0">
                  <div className="space-y-2 px-1 py-1">
                    <button
                      type="button"
                      onClick={() => setSelectedProviderZid("")}
                      className={`flex w-full items-center justify-between rounded-xl px-3 py-2 text-left transition ${selectedProviderZid === "" ? "bg-blue-50 text-blue-700 ring-1 ring-blue-200" : "text-slate-600 hover:bg-slate-50 hover:text-slate-800"}`}
                    >
                      <span className="truncate text-sm font-medium">全部 Skills</span>
                      <span className={`rounded-full px-2 py-0.5 text-xs ${selectedProviderZid === "" ? "bg-blue-100 text-blue-700" : "bg-slate-100 text-slate-500"}`}>{skills.length}</span>
                    </button>
                    {providers.map((item) => {
                      const isActive = item.zid === selectedProviderZid;
                      return (
                        <button
                          key={item.zid}
                          type="button"
                          onClick={() => setSelectedProviderZid(item.zid)}
                          className={`flex w-full items-center justify-between rounded-xl px-3 py-2 text-left transition ${isActive ? "bg-blue-50 text-blue-700 ring-1 ring-blue-200" : "text-slate-600 hover:bg-slate-50 hover:text-slate-800"}`}
                        >
                          <div className="min-w-0">
                            <div className="truncate text-sm font-medium">{item.name}</div>
                            <div className="truncate text-xs text-slate-400">{item.type}</div>
                          </div>
                          <span className={`ml-3 rounded-full px-2 py-0.5 text-xs ${isActive ? "bg-blue-100 text-blue-700" : "bg-slate-100 text-slate-500"}`}>{skillCountByProvider.get(item.zid) ?? 0}</span>
                        </button>
                      );
                    })}
                  </div>
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          </div>
        </aside>

        <div className="skm-card min-h-0 overflow-hidden">
          <div className="border-b border-slate-200 px-4 py-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <h3 className="text-sm font-semibold text-slate-800">{selectedProvider ? `${selectedProvider.name} 的 Skills` : "全部 Skills"}</h3>
                <p className="mt-1 text-xs text-slate-500">
                  {filteredSkills.length} 条结果
                  {status ? ` · 状态 ${status}` : ""}
                  {deferredQuery.trim() ? ` · 搜索 “${deferredQuery.trim()}”` : ""}
                </p>
              </div>
            </div>
          </div>
          <div className="h-full overflow-auto">
            <table className="w-full text-left text-sm whitespace-nowrap">
              <thead className="sticky top-0 z-10 border-b border-slate-200 bg-slate-50 text-slate-600">
                <tr>
                  <th className="px-4 py-2 font-medium">Skill 名称</th>
                  <th className="px-4 py-2 font-medium">Provider</th>
                  <th className="px-4 py-2 font-medium">分类</th>
                  <th className="w-24 px-4 py-2 font-medium">状态</th>
                  <th className="w-20 px-4 py-2 text-right font-medium">操作</th>
                </tr>
              </thead>
              <tbody className="divide-y divide-slate-100">
                {loading ? (
                  Array.from({ length: 6 }).map((_, index) => (
                    <tr key={index}>
                      <td colSpan={5} className="px-4 py-6 text-slate-400">加载中…</td>
                    </tr>
                  ))
                ) : filteredSkills.map((skill) => (
                  <tr key={skill.zid} className="group cursor-pointer transition-colors hover:bg-blue-50" onClick={() => navigate(`/skills/${skill.zid}`)}>
                    <td className="px-4 py-3 font-medium text-slate-800">
                      <div className="flex items-center gap-2">
                        <span className="text-blue-600">◈</span>
                        <div>
                          <div>{skill.name}</div>
                          <div className="mt-0.5 max-w-md truncate text-xs font-normal text-slate-500">{skill.summary || skill.rootPath}</div>
                        </div>
                      </div>
                    </td>
                    <td className="px-4 py-3 text-slate-600">{skill.provider?.name ?? "Unknown"}</td>
                    <td className="px-4 py-3">
                      <span className="rounded bg-slate-100 px-1.5 py-0.5 text-xs text-slate-500">{skill.category || "Uncategorized"}</span>
                    </td>
                    <td className="px-4 py-3">{renderSkillStatus(skill)}</td>
                    <td className="px-4 py-3 text-right opacity-0 transition-opacity group-hover:opacity-100">
                      <button type="button" onClick={(event) => { event.stopPropagation(); navigate(`/skills/${skill.zid}`); }} className="text-sm font-medium text-blue-600 hover:text-blue-700">查看详情</button>
                    </td>
                  </tr>
                ))}
                {!loading && filteredSkills.length === 0 ? (
                  <tr>
                    <td colSpan={5} className="px-4 py-12 text-center text-slate-500">没有找到匹配的 Skills</td>
                  </tr>
                ) : null}
              </tbody>
            </table>
          </div>
        </div>
      </div>

      {zid ? (
        <div className="absolute inset-4 z-20">
          <SkillDetailDialog zid={zid} open={Boolean(zid)} onOpenChange={(open) => { if (!open) { navigate("/skills"); } }} />
        </div>
      ) : null}
    </div>
  );
}

function renderSkillStatus(skill: Skill) {
  if (skill.status === "invalid") {
    return <span className="inline-flex items-center rounded-md border border-red-200 bg-red-50 px-2 py-1 text-xs font-medium text-red-700">Error</span>;
  }
  if (skill.isConflict) {
    return <span className="inline-flex items-center rounded-md border border-amber-200 bg-amber-50 px-2 py-1 text-xs font-medium text-amber-700">Conflict</span>;
  }
  return <span className="inline-flex items-center rounded-md border border-green-200 bg-green-50 px-2 py-1 text-xs font-medium text-green-700">Active</span>;
}