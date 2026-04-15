import { useEffect, useState } from "react";
import { RotateCw } from "lucide-react";
import { useOutletContext } from "react-router-dom";
import { toast } from "sonner";
import type { ShellOutletContext } from "../components/skm/ConsoleShell";
import { api, type ConflictGroup, type ScanIssue } from "../lib/api";

export function IssuesPage() {
  const [issues, setIssues] = useState<ScanIssue[]>([]);
  const [conflicts, setConflicts] = useState<ConflictGroup[]>([]);
  const [error, setError] = useState("");
  const [refreshing, setRefreshing] = useState(false);
  const { refreshKey } = useOutletContext<ShellOutletContext>();

  async function load() {
    const [issueData, conflictData] = await Promise.all([
      api.getIssues({ view: "latest" }),
      api.getConflicts(),
    ]);
    setIssues(issueData);
    setConflicts(conflictData);
    setError("");
  }

  useEffect(() => {
    let active = true;

    async function loadSafe() {
      try {
        const [issueData, conflictData] = await Promise.all([
          api.getIssues({ view: "latest" }),
          api.getConflicts(),
        ]);
        if (!active) {
          return;
        }
        setIssues(issueData);
        setConflicts(conflictData);
        setError("");
      } catch (loadError) {
        if (!active) {
          return;
        }
        setError(loadError instanceof Error ? loadError.message : "Failed to load issues");
      }
    }

    void loadSafe();
    return () => {
      active = false;
    };
  }, [refreshKey]);

  async function handleRefresh() {
    setRefreshing(true);
    try {
      await load();
      toast.success("异常与冲突数据已刷新");
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "Failed to load issues");
    } finally {
      setRefreshing(false);
    }
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="skm-section-title">异常与冲突检测</h2>
        <button type="button" onClick={() => void handleRefresh()} className="inline-flex items-center gap-2 rounded-md border border-slate-300 bg-white px-3 py-1.5 text-sm text-slate-700 transition-colors hover:bg-slate-50">
          <RotateCw className={`h-4 w-4 ${refreshing ? "animate-spin" : ""}`} />
          重新检测
        </button>
      </div>

      {error ? <p className="text-sm text-red-600">{error}</p> : null}

      <section className="skm-card overflow-hidden">
        <div className="border-b border-slate-200 bg-slate-50 px-4 py-3">
          <h3 className="text-sm font-semibold text-slate-700">最新异常列表</h3>
        </div>
        <table className="w-full text-left text-sm">
          <thead className="border-b border-slate-200 bg-slate-50 text-slate-600">
            <tr>
              <th className="px-4 py-2 font-medium w-24">类型</th>
              <th className="px-4 py-2 font-medium">目标对象/路径</th>
              <th className="px-4 py-2 font-medium">问题描述</th>
              <th className="px-4 py-2 font-medium text-right w-24">严重级别</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {issues.map((issue) => (
              <tr key={issue.zid} className="hover:bg-slate-50 transition-colors">
                <td className="px-4 py-3">
                  <span className={`rounded border px-2 py-0.5 text-xs ${issue.severity === "error" ? "border-red-200 bg-red-50 text-red-700" : "border-amber-200 bg-amber-50 text-amber-700"}`}>
                    {issue.severity === "error" ? "Error" : "Warning"}
                  </span>
                </td>
                <td className="px-4 py-3 font-mono text-xs text-slate-700">{issue.relativePath || issue.rootPath}</td>
                <td className="px-4 py-3">
                  <div className="font-medium text-slate-800">{issue.message}</div>
                  <div className="mt-0.5 text-xs text-slate-500">{issue.provider?.name ?? "Unknown provider"} · {issue.code}</div>
                  {issue.details ? <div className="mt-1 text-xs text-slate-500">{JSON.stringify(issue.details)}</div> : null}
                </td>
                <td className="px-4 py-3 text-right text-xs text-slate-500">{issue.code}</td>
              </tr>
            ))}
            {issues.length === 0 ? (
              <tr>
                <td colSpan={4} className="px-4 py-10 text-center text-slate-500">当前没有异常或冲突</td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </section>

      <section className="skm-card p-4">
        <div className="mb-4 flex items-center justify-between">
          <h3 className="text-sm font-semibold text-slate-700">冲突组</h3>
          <span className="text-xs text-slate-400">{conflicts.length} groups</span>
        </div>
        <div className="space-y-3">
          {conflicts.map((group) => (
            <div key={`${group.kind}:${group.key}`} className="rounded-lg border border-slate-200 bg-slate-50 p-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <div className="font-medium text-slate-800">{group.key}</div>
                  <div className="text-xs text-slate-500">{group.kind}</div>
                </div>
                <span className="rounded bg-blue-50 px-2 py-1 text-xs text-blue-700">当前生效 {group.effectiveSkillZid ?? "unknown"}</span>
              </div>
              <div className="mt-3 space-y-2">
                {group.skills.map((skill) => (
                  <div key={skill.zid} className="rounded border border-slate-200 bg-white px-3 py-2 text-sm text-slate-700">
                    {skill.name} · {skill.provider?.name ?? "Unknown provider"} · {skill.rootPath}
                  </div>
                ))}
              </div>
            </div>
          ))}
          {conflicts.length === 0 ? <p className="text-sm text-slate-500">暂无冲突组</p> : null}
        </div>
      </section>
    </div>
  );
}