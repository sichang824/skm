import { useEffect, useMemo, useState } from "react";
import { Boxes, Bug, FolderTree, TriangleAlert } from "lucide-react";
import { useOutletContext } from "react-router-dom";
import type { ShellOutletContext } from "../components/skm/ConsoleShell";
import { Card, CardContent, CardHeader, CardTitle } from "../components/ui/card";
import { api, type DashboardSummary, type Provider, type ScanIssue, type ScanJob, type Skill } from "../lib/api";

const EMPTY_DASHBOARD: DashboardSummary = {
  providerCount: 0,
  enabledProviderCount: 0,
  skillCount: 0,
  conflictCount: 0,
  issueCount: 0,
  recentScanCount: 0,
};

export function DashboardPage() {
  const [dashboard, setDashboard] = useState<DashboardSummary>(EMPTY_DASHBOARD);
  const [providers, setProviders] = useState<Provider[]>([]);
  const [skills, setSkills] = useState<Skill[]>([]);
  const [issues, setIssues] = useState<ScanIssue[]>([]);
  const [jobs, setJobs] = useState<ScanJob[]>([]);
  const [error, setError] = useState("");
  const { refreshKey } = useOutletContext<ShellOutletContext>();

  useEffect(() => {
    let active = true;

    async function load() {
      try {
        const [dashboardData, providerData, skillData, issueData, jobData] = await Promise.all([
          api.getDashboard(),
          api.getProviders(),
          api.getSkills({ sort: "lastScanned" }),
          api.getIssues({ view: "latest" }),
          api.getScanJobs(),
        ]);
        if (!active) {
          return;
        }
        setDashboard(dashboardData);
        setProviders(providerData);
        setSkills(skillData);
        setIssues(issueData.slice(0, 5));
        setJobs(jobData.slice(0, 5));
      } catch (loadError) {
        if (!active) {
          return;
        }
        setError(loadError instanceof Error ? loadError.message : "Failed to load dashboard");
      }
    }

    void load();
    return () => {
      active = false;
    };
  }, [refreshKey]);

  const providerDistribution = useMemo(() => {
    return providers.map((provider) => {
      const skillCount = skills.filter((skill) => skill.provider?.zid === provider.zid).length;
      return { provider, skillCount };
    });
  }, [providers, skills]);

  return (
    <div className="space-y-6">
      {error ? <p className="text-sm text-red-600">{error}</p> : null}

      <div className="grid grid-cols-1 gap-6 md:grid-cols-2 xl:grid-cols-4">
        <StatCard title="总 Skills" value={dashboard.skillCount} icon={Boxes} tone="blue" />
        <StatCard title="Providers" value={dashboard.providerCount} icon={FolderTree} tone="indigo" />
        <StatCard title="冲突项" value={dashboard.conflictCount} icon={TriangleAlert} tone="yellow" />
        <StatCard title="异常目录" value={dashboard.issueCount} icon={Bug} tone="red" />
      </div>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-2">
        <Card className="rounded-xl border-slate-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-base">最近扫描日志</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {jobs.length === 0 ? <EmptyText text="暂无扫描记录" /> : jobs.map((job) => (
              <div key={job.zid} className="flex border-b border-slate-100 pb-4 last:border-0">
                <div className="mt-1 mr-4 text-sm">
                  <span className={job.status === "completed" ? "text-green-500" : "text-yellow-500"}>●</span>
                </div>
                <div>
                  <p className="text-sm font-medium text-slate-800">{job.provider?.name ?? "System"} 扫描 {job.status}</p>
                  <p className="mt-0.5 text-xs text-slate-500">added={job.addedCount} removed={job.removedCount} changed={job.changedCount} invalid={job.invalidCount} conflicts={job.conflictCount}</p>
                </div>
                <div className="ml-auto text-xs text-slate-400">{formatRelative(job.startedAt)}</div>
              </div>
            ))}
          </CardContent>
        </Card>

        <Card className="rounded-xl border-slate-200 shadow-sm">
          <CardHeader>
            <CardTitle className="text-base">Skills 来源分布</CardTitle>
          </CardHeader>
          <CardContent className="space-y-4">
            {providerDistribution.length === 0 ? <EmptyText text="暂无 Provider 数据" /> : providerDistribution.map(({ provider, skillCount }) => (
              <div key={provider.zid}>
                <div className="mb-1 flex justify-between text-sm">
                  <span className="font-medium text-slate-600">{provider.name}</span>
                  <span className="text-slate-500">{skillCount} 个</span>
                </div>
                <div className="h-2 w-full rounded-full bg-slate-100">
                  <div className="h-2 rounded-full bg-indigo-600" style={{ width: `${dashboard.skillCount === 0 ? 0 : (skillCount / dashboard.skillCount) * 100}%` }} />
                </div>
              </div>
            ))}
          </CardContent>
        </Card>
      </div>

      <Card className="rounded-xl border-slate-200 shadow-sm">
        <CardHeader>
          <CardTitle className="text-base">最新异常</CardTitle>
        </CardHeader>
        <CardContent className="space-y-3">
          {issues.length === 0 ? <EmptyText text="当前 latest issue 视图为空" /> : issues.map((issue) => (
            <div key={issue.zid} className="rounded-lg border border-slate-200 bg-slate-50 p-4">
              <div className="flex items-start justify-between gap-3">
                <div>
                  <p className="text-sm font-medium text-slate-900">{issue.message}</p>
                  <p className="mt-1 text-xs text-slate-500">{issue.provider?.name ?? "Unknown provider"} · {issue.code} · {issue.rootPath}</p>
                </div>
                <span className={`rounded-md px-2 py-1 text-xs font-medium ${issue.severity === "error" ? "bg-red-100 text-red-700" : "bg-yellow-100 text-yellow-700"}`}>{issue.severity}</span>
              </div>
            </div>
          ))}
        </CardContent>
      </Card>
    </div>
  );
}

function StatCard({ title, value, icon: Icon, tone }: { title: string; value: number; icon: typeof Boxes; tone: "blue" | "indigo" | "yellow" | "red" }) {
  const toneClass = {
    blue: "bg-blue-50 text-blue-600",
    indigo: "bg-indigo-50 text-indigo-600",
    yellow: "bg-yellow-50 text-yellow-600",
    red: "bg-red-50 text-red-600",
  }[tone];

  return (
    <div className="flex items-center rounded-xl border border-slate-200 bg-white p-6 shadow-sm">
      <div className={`mr-4 flex h-12 w-12 items-center justify-center rounded-lg text-xl ${toneClass}`}>
        <Icon className="h-5 w-5" />
      </div>
      <div>
        <p className="text-sm font-medium text-slate-500">{title}</p>
        <p className="text-2xl font-bold text-slate-800">{value}</p>
      </div>
    </div>
  );
}

function EmptyText({ text }: { text: string }) {
  return <p className="text-sm text-slate-500">{text}</p>;
}

function formatRelative(value: string) {
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}