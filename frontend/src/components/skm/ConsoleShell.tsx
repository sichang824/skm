import { useEffect, useMemo, useState } from "react";
import { Bell, FolderTree, LayoutDashboard, RefreshCw, ShieldAlert, Sparkles, Boxes } from "lucide-react";
import { NavLink, Outlet, useLocation, useNavigate } from "react-router-dom";
import { toast } from "sonner";
import { api } from "../../lib/api";

type ShellOutletContext = {
  globalSearch: string;
  setGlobalSearch: (value: string) => void;
  refreshKey: number;
};

const navItems = [
  { to: "/dashboard", label: "仪表盘", icon: LayoutDashboard },
  { to: "/skills", label: "所有技能", icon: Boxes },
  { to: "/providers", label: "来源管理", icon: FolderTree },
  { to: "/issues", label: "异常与冲突", icon: ShieldAlert },
];

const titleMap: Record<string, string> = {
  "/dashboard": "仪表盘",
  "/skills": "所有技能 (Skills)",
  "/providers": "来源管理 (Providers)",
  "/issues": "异常与冲突",
};

export function ConsoleShell() {
  const location = useLocation();
  const navigate = useNavigate();
  const [globalSearch, setGlobalSearch] = useState("");
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
      return "技能详情";
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
    <div className="flex h-screen w-full overflow-hidden bg-slate-50 text-slate-800">
      <aside className="flex w-64 shrink-0 flex-col border-r border-slate-200 bg-slate-50 shadow-[8px_0_30px_rgba(15,23,42,0.04)]">
        <div className="flex h-16 items-center border-b border-slate-200 px-6">
          <div className="mr-3 flex h-9 w-9 items-center justify-center rounded-xl bg-indigo-600 text-white shadow-sm">
            <Sparkles className="h-4 w-4" />
          </div>
          <div className="flex items-center gap-2">
            <h1 className="text-xl font-bold tracking-tight text-slate-900">SKM</h1>
            <span className="rounded-full bg-slate-200 px-2 py-0.5 text-xs font-medium text-slate-600">v0.1</span>
          </div>
        </div>

        <div className="flex-1 overflow-y-auto p-4">
          <nav className="space-y-1">
            {navItems.map((item) => {
              const Icon = item.icon;
              return (
                <NavLink
                  key={item.to}
                  to={item.to}
                  className={({ isActive }) => `flex items-center rounded-lg px-3 py-2.5 text-sm font-medium transition-colors ${isActive ? "bg-indigo-600 text-white shadow-sm" : "text-slate-600 hover:bg-slate-200 hover:text-slate-900"}`}
                >
                  <Icon className="mr-2 h-4 w-4" />
                  {item.label}
                </NavLink>
              );
            })}
          </nav>

          <div className="mt-8">
            <h3 className="mb-2 px-3 text-xs font-semibold uppercase tracking-wider text-slate-400">快捷操作</h3>
            <button
              type="button"
              onClick={() => void handleScanAll()}
              className="flex w-full items-center rounded-lg px-3 py-2 text-sm font-medium text-slate-600 transition-colors hover:bg-slate-200"
            >
              <RefreshCw className={`mr-2 h-4 w-4 ${isScanning ? "animate-spin text-indigo-600" : ""}`} />
              全量扫描目录
            </button>
            <button
              type="button"
              onClick={() => navigate("/issues")}
              className="mt-2 flex w-full items-center rounded-lg px-3 py-2 text-sm font-medium text-slate-600 transition-colors hover:bg-slate-200"
            >
              <ShieldAlert className="mr-2 h-4 w-4" />
              查看异常与冲突
            </button>
          </div>
        </div>

        <div className="border-t border-slate-200 p-4">
          <div className="flex items-center">
            <div className="flex h-8 w-8 items-center justify-center rounded-full bg-indigo-100 font-bold text-indigo-600">U</div>
            <div className="ml-3">
              <p className="text-sm font-medium text-slate-700">Local User</p>
              <p className="text-xs text-slate-500">Read-only Mode</p>
            </div>
          </div>
        </div>
      </aside>

      <main className="relative flex min-w-0 flex-1 flex-col overflow-hidden">
        <header className="flex h-16 shrink-0 items-center justify-between border-b border-slate-200 bg-white px-8">
          <h2 className="text-xl font-semibold text-slate-800">{currentTitle}</h2>
          <div className="flex items-center gap-4">
            <div className="relative">
              <input
                value={globalSearch}
                onChange={(event) => setGlobalSearch(event.target.value)}
                placeholder="全局搜索 Skills..."
                className="w-72 rounded-lg border border-transparent bg-slate-100 py-2 pr-4 pl-4 text-sm outline-none transition-all focus:border-indigo-400 focus:bg-white focus:ring-2 focus:ring-indigo-100"
              />
            </div>
            <button type="button" className="relative rounded-lg p-2 text-slate-400 transition-colors hover:text-slate-600">
              <Bell className="h-5 w-5" />
              {issueCount > 0 ? <span className="absolute top-1 right-1 h-2 w-2 rounded-full border border-white bg-red-500" /> : null}
            </button>
          </div>
        </header>

        <div className="flex-1 overflow-y-auto bg-slate-50/60 p-8">
          <Outlet context={{ globalSearch, setGlobalSearch, refreshKey } satisfies ShellOutletContext} />
        </div>

        {isScanning ? (
          <div className="absolute right-6 bottom-6 z-50 flex items-center rounded-lg bg-slate-800 px-4 py-3 text-white shadow-xl">
            <RefreshCw className="mr-3 h-4 w-4 animate-spin text-indigo-300" />
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