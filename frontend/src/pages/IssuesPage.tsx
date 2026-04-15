import { useEffect, useState } from "react";
import { useOutletContext } from "react-router-dom";
import type { ShellOutletContext } from "../components/skm/ConsoleShell";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { api, type ConflictGroup, type ScanIssue } from "../lib/api";

export function IssuesPage() {
  const [issues, setIssues] = useState<ScanIssue[]>([]);
  const [conflicts, setConflicts] = useState<ConflictGroup[]>([]);
  const [error, setError] = useState("");
  const { refreshKey } = useOutletContext<ShellOutletContext>();

  useEffect(() => {
    let active = true;

    async function load() {
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
      } catch (loadError) {
        if (!active) {
          return;
        }
        setError(loadError instanceof Error ? loadError.message : "Failed to load issues");
      }
    }

    void load();
    return () => {
      active = false;
    };
  }, [refreshKey]);

  return (
    <div className="space-y-6">
      {error ? <p className="text-sm text-red-600">{error}</p> : null}

      <Card className="overflow-hidden rounded-xl border-slate-200 shadow-sm">
        <div className="border-b border-slate-200 bg-red-50/60 p-4">
          <h3 className="text-base font-medium text-red-800">发现 {issues.length} 个需要关注的异常</h3>
        </div>
        <CardContent className="p-0">
          <ul className="divide-y divide-slate-100">
            {issues.map((issue) => (
              <li key={issue.zid} className="flex items-start p-4 hover:bg-slate-50">
                <div className="mt-0.5 mr-4 text-red-500">●</div>
                <div className="flex-1">
                  <h4 className="text-sm font-medium text-slate-900">{issue.message}</h4>
                  <p className="mt-1 text-xs text-slate-500">{issue.provider?.name ?? "Unknown provider"} · {issue.rootPath}</p>
                  {issue.details ? (
                    <div className="mt-2 rounded border border-slate-200 bg-slate-100/60 p-2 text-sm text-slate-700">
                      {JSON.stringify(issue.details)}
                    </div>
                  ) : null}
                </div>
                <div className="ml-4">
                  <span className={`rounded-md px-2 py-1 text-xs font-medium ${issue.severity === "error" ? "bg-red-100 text-red-700" : "bg-yellow-100 text-yellow-700"}`}>{issue.code}</span>
                </div>
              </li>
            ))}
            {issues.length === 0 ? <li className="p-6 text-sm text-slate-500">暂无异常项</li> : null}
          </ul>
        </CardContent>
      </Card>

      <Card className="rounded-xl border-slate-200 shadow-sm">
        <CardHeader>
          <CardTitle className="text-base">冲突组</CardTitle>
        </CardHeader>
        <CardContent className="space-y-4">
          {conflicts.length === 0 ? <p className="text-sm text-slate-500">暂无冲突组</p> : conflicts.map((group) => (
            <div key={`${group.kind}:${group.key}`} className="rounded-xl border border-slate-200 bg-white p-4">
              <div className="flex flex-wrap items-center justify-between gap-2">
                <div>
                  <p className="text-sm font-semibold text-slate-900">{group.kind}</p>
                  <p className="text-xs text-slate-500">{group.key}</p>
                </div>
                <span className="rounded-md bg-indigo-50 px-2 py-1 text-xs font-medium text-indigo-700">生效项 {group.effectiveSkillZid}</span>
              </div>
              <div className="mt-3 space-y-2">
                {group.skills.map((skill) => (
                  <div key={skill.zid} className="rounded-lg bg-slate-50 px-3 py-2 text-sm text-slate-700">
                    {skill.name} · {skill.provider?.name ?? "Unknown provider"} · {skill.rootPath}
                  </div>
                ))}
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}