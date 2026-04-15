import { useEffect, useMemo, useState } from "react";
import { Boxes, FolderTree, Gauge, Radar, Settings, ShieldAlert } from "lucide-react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { api } from "../../lib/api";

type ShellOutletContext = {
  refreshKey: number;
};

const navItems = [
  { to: "/dashboard", label: "Dashboard", icon: Gauge },
  { to: "/skills", label: "Skills 目录", icon: Boxes },
  { to: "/providers", label: "Providers", icon: FolderTree },
  { to: "/issues", label: "异常检测", icon: ShieldAlert },
];

const titleMap: Record<string, string> = {
  "/dashboard": "系统大盘",
  "/skills": "本地 Skills 库",
  "/providers": "本地 Provider 配置",
  "/issues": "冲突与格式异常",
};

export function ConsoleShell() {
  const location = useLocation();
  const navigate = useNavigate();
  const isDesktopShell = typeof window !== "undefined" && "runtime" in window;
  const [issueCount, setIssueCount] = useState(0);
  const [isScanning, setIsScanning] = useState(false);
  const [refreshKey, setRefreshKey] = useState(0);

  useEffect(() => {
    let active = true;

    async function loadSummary() {
      try {
        const dashboard = await api.getDashboard();
        if (!active) {
          return;
        }
        setIssueCount(dashboard.issueCount);
      } catch {
        if (!active) {
          return;
        }
        setIssueCount(0);
      }
    }

    void loadSummary();
    return () => {
      active = false;
    };
  }, [refreshKey]);

  const currentTitle = useMemo(() => {
    if (location.pathname.startsWith("/skills/")) {
      return "Skills / Detail";
    }
    return titleMap[location.pathname] ?? "SKM";
  }, [location.pathname]);

  async function handleScanAll() {
    if (isScanning) {
      return;
    }
    setIsScanning(true);
    try {
      const result = await api.scanAll();
      toast.success(`全量扫描完成，共执行 ${result.jobs.length} 个 Provider 任务`);
      setRefreshKey((value) => value + 1);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "全量扫描失败");
    } finally {
      setIsScanning(false);
    }
  }

  return (
    <div className="flex h-screen overflow-hidden bg-[var(--app-shell)] text-slate-700">
      <aside className="flex w-56 shrink-0 flex-col justify-between border-r border-slate-200 bg-[var(--app-sidebar)] backdrop-blur-xl">
        <div>
          <div
            className={`skm-drag-region flex border-b border-slate-200 bg-white px-4 ${isDesktopShell ? "h-24 items-center pt-10" : "h-14 items-center"}`}
          >
            <div className="mr-3 flex h-9 w-9 items-center justify-center rounded-xl bg-blue-600 text-white shadow-sm shadow-blue-200/80">
              <Boxes className="h-4 w-4" />
            </div>
            <div>
              <div className="text-lg font-bold tracking-tight text-slate-900">SKM</div>
              <div className="text-[11px] uppercase tracking-[0.22em] text-slate-400">Skills Manager</div>
            </div>
          </div>

          <div className="p-3">
            <nav className="space-y-1">
              {navItems.map((item) => {
                const Icon = item.icon;

                return (
                  <NavLink
                    key={item.to}
                    to={item.to}
                    className={({ isActive }) => `flex items-center justify-between rounded-md px-3 py-2.5 text-sm transition-colors ${isActive ? "bg-blue-50 font-medium text-blue-700" : "text-slate-600 hover:bg-slate-100 hover:text-slate-900"}`}
                  >
                    <span className="flex items-center gap-3">
                      <Icon className={`h-4 w-4 ${item.to === "/issues" && issueCount > 0 ? "text-amber-500" : ""}`} />
                      {item.label}
                    </span>
                    {item.to === "/issues" && issueCount > 0 ? (
                      <span className="rounded-full bg-amber-100 px-1.5 py-0.5 text-[10px] font-semibold text-amber-700">
                        {issueCount}
                      </span>
                    ) : null}
                  </NavLink>
                );
              })}
            </nav>

            <div className="mt-6 rounded-xl border border-slate-200 bg-white/80 p-3 shadow-sm">
              <div className="mb-2 text-[11px] font-semibold uppercase tracking-[0.18em] text-slate-400">运行状态</div>
              <div className="flex items-center gap-2 text-sm text-slate-600">
                <span className="inline-block h-2 w-2 rounded-full bg-green-500" />
                Provider watch 已连接
              </div>
              <button
                type="button"
                onClick={() => void handleScanAll()}
                className="mt-3 flex w-full items-center justify-center gap-2 rounded-lg bg-blue-600 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700"
              >
                <Radar className={`h-4 w-4 ${isScanning ? "animate-spin" : ""}`} />
                全局扫描
              </button>
              <button
                type="button"
                onClick={() => navigate("/issues")}
                className="mt-2 flex w-full items-center justify-center gap-2 rounded-lg border border-slate-200 bg-white px-3 py-2 text-sm font-medium text-slate-600 transition-colors hover:bg-slate-50"
              >
                <ShieldAlert className="h-4 w-4" />
                查看异常
              </button>
            </div>
          </div>
        </div>

        <div className="flex items-center justify-between border-t border-slate-200 px-4 py-3 text-xs text-slate-400">
          <span>v2.0.0</span>
          <button type="button" className="rounded p-1 transition-colors hover:bg-slate-200 hover:text-slate-600" title="Settings">
            <Settings className="h-4 w-4" />
          </button>
        </div>
      </aside>

      <main className="relative flex min-w-0 flex-1 flex-col overflow-hidden">
        <header className={`skm-drag-region z-10 flex h-14 items-center justify-between border-b border-slate-200 bg-white px-6 ${isDesktopShell ? "pl-24" : ""}`}>
          <h1 className="text-sm font-medium text-slate-700">{currentTitle}</h1>
          <div className="skm-no-drag flex items-center gap-3">
            <span className="flex items-center gap-2 text-xs text-slate-400">
              <span className="inline-block h-2 w-2 rounded-full bg-green-500" />
              监听中
            </span>
            <div className="h-4 w-px bg-slate-200" />
            <button
              type="button"
              onClick={() => void handleScanAll()}
              className="inline-flex items-center gap-2 rounded-md border border-slate-200 bg-white px-3 py-1.5 text-xs font-medium text-slate-700 shadow-sm transition-colors hover:bg-slate-50"
            >
              <Radar className={`h-3.5 w-3.5 text-blue-600 ${isScanning ? "animate-spin" : ""}`} />
              全局扫描
            </button>
          </div>
        </header>

        <div className="min-h-0 flex-1 overflow-y-auto bg-transparent p-6">
          <Outlet context={{ refreshKey } satisfies ShellOutletContext} />
        </div>

        {isScanning ? (
          <div className="absolute right-6 bottom-6 z-50 flex items-center rounded-lg bg-slate-800 px-4 py-3 text-white shadow-xl">
            <Radar className="mr-3 h-4 w-4 animate-spin text-blue-300" />
            <div>
              <p className="text-sm font-medium">正在扫描本地目录...</p>
              <p className="text-xs text-slate-300">索引会在扫描完成后自动刷新</p>
            </div>
          </div>
        ) : null}
      </main>
    </div>
  );
}

export type { ShellOutletContext };