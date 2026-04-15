import { useEffect, useMemo, useState } from "react";
import { Boxes, Bug, FolderTree, Radar, TriangleAlert } from "lucide-react";
import { useOutletContext } from "react-router-dom";
import type { ShellOutletContext } from "../components/skm/ConsoleShell";
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

      <div className="grid grid-cols-1 gap-4 md:grid-cols-2 xl:grid-cols-4">
        <StatCard title="总 Skills" value={dashboard.skillCount} icon={Boxes} accent="text-slate-800" />
        <StatCard title="Providers" value={dashboard.providerCount} icon={FolderTree} accent="text-slate-800" />
        <StatCard title="冲突项" value={dashboard.conflictCount} icon={TriangleAlert} accent="text-amber-600" />
        <StatCard title="异常数" value={dashboard.issueCount} icon={Bug} accent="text-red-600" />
      </div>

      <section className="skm-card overflow-hidden">
        <div className="flex items-center justify-between border-b border-slate-200 bg-slate-50 px-4 py-3">
          <h2 className="text-sm font-semibold text-slate-700">最近扫描动态</h2>
          <span className="text-xs text-slate-500">{jobs[0] ? formatRelative(jobs[0].startedAt) : "暂无记录"}</span>
        </div>
        <div className="p-4">
          {jobs.length === 0 ? <EmptyText text="暂无扫描记录" /> : (
            <ul className="space-y-3 text-sm">
              {jobs.map((job) => (
                <li key={job.zid} className="flex items-start gap-3 text-slate-600">
                  <Radar className={`mt-0.5 h-4 w-4 ${job.status === "completed" ? "text-green-500" : "text-amber-500"}`} />
                  <div>
                    <span className="font-medium text-slate-800">{job.provider?.name ?? "System"}</span>
                    <span> 扫描完成。新增 {job.addedCount} 个，变更 {job.changedCount} 个，异常 {job.invalidCount} 个，冲突 {job.conflictCount} 个。</span>
                    <div className="mt-0.5 text-xs text-slate-500">开始于 {formatRelative(job.startedAt)} {job.finishedAt ? `· 结束于 ${formatRelative(job.finishedAt)}` : ""}</div>
                  </div>
                </li>
              ))}
            </ul>
          )}
        </div>
      </section>

      <div className="grid grid-cols-1 gap-6 lg:grid-cols-[1.1fr_0.9fr]">
        <section className="skm-card p-4">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-slate-700">最新异常</h2>
            <span className="text-xs text-slate-400">latest view</span>
          </div>
          <div className="space-y-3">
            {issues.length === 0 ? <EmptyText text="当前没有异常项" /> : issues.map((issue) => (
              <div key={issue.zid} className="rounded-lg border border-slate-200 bg-slate-50 p-4">
                <div className="flex items-start justify-between gap-3">
                  <div>
                    <p className="text-sm font-medium text-slate-900">{issue.message}</p>
                    <p className="mt-1 text-xs text-slate-500">{issue.provider?.name ?? "Unknown provider"} · {issue.code} · {issue.rootPath}</p>
                  </div>
                  <span className={`rounded-md px-2 py-1 text-xs font-medium ${issue.severity === "error" ? "bg-red-100 text-red-700" : "bg-amber-100 text-amber-700"}`}>{issue.severity}</span>
                </div>
              </div>
            ))}
          </div>
        </section>

        <section className="skm-card p-4">
          <div className="mb-4 flex items-center justify-between">
            <h2 className="text-sm font-semibold text-slate-700">Skills 来源分布</h2>
            <span className="text-xs text-slate-400">{dashboard.enabledProviderCount} 个启用中</span>
          </div>
          <div className="space-y-4">
            {providerDistribution.length === 0 ? <EmptyText text="暂无 Provider 数据" /> : providerDistribution.map(({ provider, skillCount }) => (
              <div key={provider.zid}>
                <div className="mb-1 flex justify-between text-sm">
                  <span className="font-medium text-slate-700">{provider.name}</span>
                  <span className="text-slate-500">{skillCount} 个</span>
                </div>
                <div className="h-2 w-full rounded-full bg-slate-100">
                  <div className="h-2 rounded-full bg-blue-600" style={{ width: `${dashboard.skillCount === 0 ? 0 : (skillCount / dashboard.skillCount) * 100}%` }} />
                </div>
              </div>
            ))}
          </div>
        </section>
      </div>
    </div>
  );
}

function StatCard({ title, value, icon: Icon, accent }: { title: string; value: number; icon: typeof Boxes; accent: string }) {
  return (
    <div className="skm-card p-4">
      <div className="mb-3 flex items-center justify-between">
        <div className="text-xs font-medium uppercase tracking-[0.16em] text-slate-400">{title}</div>
        <Icon className={`h-5 w-5 ${accent}`} />
      </div>
      <div className={`text-3xl font-bold ${accent}`}>{value}</div>
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