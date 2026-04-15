import { useDeferredValue, useEffect, useMemo, useState } from "react";
import { Search } from "lucide-react";
import { useNavigate, useOutletContext, useParams } from "react-router-dom";
import { SkillDetailDialog } from "../components/skm/SkillDetailDialog";
import type { ShellOutletContext } from "../components/skm/ConsoleShell";
import { api, type Provider, type Skill } from "../lib/api";

export function SkillsPage() {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [skills, setSkills] = useState<Skill[]>([]);
  const [search, setSearch] = useState("");
  const [provider, setProvider] = useState("");
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const { refreshKey } = useOutletContext<ShellOutletContext>();
  const { zid } = useParams();
  const navigate = useNavigate();

  const deferredQuery = useDeferredValue(search);

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

  const filteredSkills = useMemo(() => {
    return skills.filter((skill) => {
      const matchesSearch = deferredQuery.trim() === ""
        ? true
        : [skill.name, skill.summary, skill.provider?.name, skill.category, skill.directoryName]
          .filter(Boolean)
          .some((value) => String(value).toLowerCase().includes(deferredQuery.toLowerCase()));
      const matchesProvider = provider ? skill.provider?.zid === provider : true;
      const matchesStatus = status === ""
        ? true
        : status === "Valid"
          ? skill.status === "ready" && !skill.isConflict
          : status === "Conflict"
            ? skill.isConflict
            : skill.status === "invalid";
      return matchesSearch && matchesProvider && matchesStatus;
    });
  }, [deferredQuery, provider, skills, status]);

  return (
    <div className="relative flex h-full min-h-[40rem] flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <h2 className="skm-section-title">Skills 目录</h2>

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
          <select value={provider} onChange={(event) => setProvider(event.target.value)} className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-700 outline-none transition-all focus:border-blue-500">
            <option value="">所有 Providers</option>
            {providers.map((item) => <option key={item.zid} value={item.zid}>{item.name}</option>)}
          </select>
          <select value={status} onChange={(event) => setStatus(event.target.value)} className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-700 outline-none transition-all focus:border-blue-500">
            <option value="">全部状态</option>
            <option value="Valid">正常</option>
            <option value="Conflict">冲突</option>
            <option value="Error">异常</option>
          </select>
        </div>
      </div>

      {error ? <p className="px-4 py-3 text-sm text-red-600">{error}</p> : null}

      <div className="skm-card flex-1 overflow-hidden">
        <table className="w-full text-left text-sm whitespace-nowrap">
          <thead className="sticky top-0 z-10 border-b border-slate-200 bg-slate-50 text-slate-600">
            <tr>
              <th className="px-4 py-2 font-medium">Skill 名称</th>
              <th className="px-4 py-2 font-medium">Provider</th>
              <th className="px-4 py-2 font-medium">分类</th>
              <th className="px-4 py-2 font-medium w-24">状态</th>
              <th className="px-4 py-2 font-medium text-right w-20">操作</th>
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