import { useDeferredValue, useEffect, useMemo, useState } from "react";
import { useNavigate, useOutletContext, useParams } from "react-router-dom";
import { SkillDetailDialog } from "../components/skm/SkillDetailDialog";
import type { ShellOutletContext } from "../components/skm/ConsoleShell";
import { api, type Provider, type Skill } from "../lib/api";

export function SkillsPage() {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [skills, setSkills] = useState<Skill[]>([]);
  const [provider, setProvider] = useState("");
  const [status, setStatus] = useState("");
  const [sort, setSort] = useState("lastScanned");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const { globalSearch, refreshKey } = useOutletContext<ShellOutletContext>();
  const { zid } = useParams();
  const navigate = useNavigate();

  const deferredQuery = useDeferredValue(globalSearch);

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");

    async function loadSkills() {
      try {
        const [skillData, providerData] = await Promise.all([
          api.getSkills({ sort }),
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
  }, [refreshKey, sort]);

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
    <div className="flex h-full flex-col rounded-xl border border-slate-200 bg-white shadow-sm">
      <div className="flex flex-wrap items-center justify-between gap-4 border-b border-slate-200 p-4">
        <div className="flex items-center space-x-2">
          <select value={provider} onChange={(event) => setProvider(event.target.value)} className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm text-slate-600 focus:border-indigo-500 focus:outline-none">
            <option value="">全部 Provider</option>
            {providers.map((item) => <option key={item.zid} value={item.zid}>{item.name}</option>)}
          </select>
          <select value={status} onChange={(event) => setStatus(event.target.value)} className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm text-slate-600 focus:border-indigo-500 focus:outline-none">
            <option value="">全部状态</option>
            <option value="Valid">正常 (Valid)</option>
            <option value="Conflict">冲突 (Conflict)</option>
            <option value="Error">异常 (Error)</option>
          </select>
          <select value={sort} onChange={(event) => setSort(event.target.value)} className="rounded-lg border border-slate-200 bg-white px-3 py-1.5 text-sm text-slate-600 focus:border-indigo-500 focus:outline-none">
            <option value="name">按名称</option>
            <option value="provider">按 Provider</option>
            <option value="status">按状态</option>
            <option value="lastScanned">按扫描时间</option>
          </select>
        </div>
        <div className="text-sm text-slate-500">共找到 {filteredSkills.length} 个结果</div>
      </div>

      {error ? <p className="px-4 py-3 text-sm text-red-600">{error}</p> : null}

      <div className="flex-1 overflow-auto">
        <table className="w-full border-collapse text-left">
          <thead>
            <tr className="sticky top-0 z-10 border-b border-slate-200 bg-slate-50 text-xs uppercase tracking-wider text-slate-500">
              <th className="px-6 py-3 font-medium">名称 / 简介</th>
              <th className="px-6 py-3 font-medium">Provider</th>
              <th className="px-6 py-3 font-medium">分类</th>
              <th className="px-6 py-3 font-medium">状态</th>
              <th className="px-6 py-3 text-right font-medium">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100 text-sm">
            {loading ? (
              Array.from({ length: 6 }).map((_, index) => (
                <tr key={index}>
                  <td colSpan={5} className="px-6 py-6 text-slate-400">加载中…</td>
                </tr>
              ))
            ) : filteredSkills.map((skill) => (
              <tr key={skill.zid} className="group transition-colors hover:bg-slate-50">
                <td className="px-6 py-4">
                  <div className="flex items-center">
                    <div className="mr-3 flex h-8 w-8 items-center justify-center rounded bg-slate-100 text-slate-500">◆</div>
                    <div>
                      <button onClick={() => navigate(`/skills/${skill.zid}`)} className="cursor-pointer font-medium text-slate-900 transition-colors group-hover:text-indigo-600">{skill.name}</button>
                      <div className="mt-0.5 w-64 truncate text-xs text-slate-500" title={skill.summary}>{skill.summary || "无摘要"}</div>
                    </div>
                  </div>
                </td>
                <td className="px-6 py-4">
                  <span className="inline-flex items-center rounded-md bg-slate-100 px-2 py-1 text-xs font-medium text-slate-700">{skill.provider?.name ?? "Unknown"}</span>
                </td>
                <td className="px-6 py-4 text-slate-600">{skill.category || "Uncategorized"}</td>
                <td className="px-6 py-4">{renderSkillStatus(skill)}</td>
                <td className="px-6 py-4 text-right">
                  <button onClick={() => navigate(`/skills/${skill.zid}`)} className="text-sm font-medium text-indigo-600 hover:text-indigo-700">查看详情</button>
                </td>
              </tr>
            ))}
            {!loading && filteredSkills.length === 0 ? (
              <tr>
                <td colSpan={5} className="px-6 py-12 text-center text-slate-500">未找到符合条件的 Skills</td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </div>

      <SkillDetailDialog zid={zid ?? null} open={Boolean(zid)} onOpenChange={(open) => { if (!open) { navigate("/skills"); } }} />
    </div>
  );
}

function renderSkillStatus(skill: Skill) {
  if (skill.status === "invalid") {
    return <span className="inline-flex items-center rounded-md border border-red-200 bg-red-50 px-2 py-1 text-xs font-medium text-red-700">异常</span>;
  }
  if (skill.isConflict) {
    return <span className="inline-flex items-center rounded-md border border-yellow-200 bg-yellow-50 px-2 py-1 text-xs font-medium text-yellow-700">冲突</span>;
  }
  return <span className="inline-flex items-center rounded-md border border-green-200 bg-green-50 px-2 py-1 text-xs font-medium text-green-700">正常</span>;
}