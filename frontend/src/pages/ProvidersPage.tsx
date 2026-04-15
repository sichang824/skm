import { useEffect, useMemo, useState, type FormEvent, type ReactNode } from "react";
import { Pen, Plus, RotateCw, Trash2, X } from "lucide-react";
import { useOutletContext } from "react-router-dom";
import { toast } from "sonner";
import type { ShellOutletContext } from "../components/skm/ConsoleShell";
import { api, type Provider, type ProviderInput, type ScanIssue, type Skill } from "../lib/api";

const DEFAULT_PROVIDER: ProviderInput = {
  name: "",
  type: "workspace",
  rootPath: "",
  enabled: true,
  priority: 100,
  scanMode: "recursive",
  description: "",
};

export function ProvidersPage() {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [skills, setSkills] = useState<Skill[]>([]);
  const [issues, setIssues] = useState<ScanIssue[]>([]);
  const [form, setForm] = useState<ProviderInput>(DEFAULT_PROVIDER);
  const [editingProviderZid, setEditingProviderZid] = useState<string | null>(null);
  const [showForm, setShowForm] = useState(false);
  const [loading, setLoading] = useState(true);
  const [submitting, setSubmitting] = useState(false);
  const [error, setError] = useState("");
  const { refreshKey } = useOutletContext<ShellOutletContext>();

  async function load() {
    setLoading(true);
    setError("");
    try {
      const [providerData, skillData, issueData] = await Promise.all([
        api.getProviders(),
        api.getSkills({ sort: "provider" }),
        api.getIssues({ view: "latest" }),
      ]);
      setProviders(providerData);
      setSkills(skillData);
      setIssues(issueData);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "Failed to load providers");
    } finally {
      setLoading(false);
    }
  }

  useEffect(() => {
    void load();
  }, [refreshKey]);

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

  const issueCountByProvider = useMemo(() => {
    const counts = new Map<string, number>();
    for (const issue of issues) {
      const key = issue.provider?.zid;
      if (!key) {
        continue;
      }
      counts.set(key, (counts.get(key) ?? 0) + 1);
    }
    return counts;
  }, [issues]);

  async function handleSubmit(event: FormEvent<HTMLFormElement>) {
    event.preventDefault();
    setSubmitting(true);
    try {
      if (editingProviderZid) {
        await api.updateProvider(editingProviderZid, form);
        toast.success("Provider updated");
      } else {
        await api.createProvider(form);
        toast.success("Provider created");
      }
      setForm(DEFAULT_PROVIDER);
      setEditingProviderZid(null);
      setShowForm(false);
      await load();
    } catch (submitError) {
      toast.error(submitError instanceof Error ? submitError.message : "Create failed");
    } finally {
      setSubmitting(false);
    }
  }

  async function handleScan(provider: Provider) {
    try {
      await api.scanProvider(provider.zid);
      toast.success(`Scanned ${provider.name}`);
      await load();
    } catch (scanError) {
      toast.error(scanError instanceof Error ? scanError.message : "Scan failed");
    }
  }

  async function handleDelete(provider: Provider) {
    if (!window.confirm(`确认删除 Provider ${provider.name}？`)) {
      return;
    }
    try {
      await api.deleteProvider(provider.zid);
      toast.success(`Deleted ${provider.name}`);
      if (editingProviderZid === provider.zid) {
        setEditingProviderZid(null);
        setForm(DEFAULT_PROVIDER);
        setShowForm(false);
      }
      await load();
    } catch (deleteError) {
      toast.error(deleteError instanceof Error ? deleteError.message : "Delete failed");
    }
  }

  async function handleToggleEnabled(provider: Provider) {
    try {
      await api.updateProvider(provider.zid, providerToInput({ ...provider, enabled: !provider.enabled }));
      toast.success(`${provider.name} 已${provider.enabled ? "停用" : "启用"}`);
      await load();
    } catch (toggleError) {
      toast.error(toggleError instanceof Error ? toggleError.message : "Update failed");
    }
  }

  function startEdit(provider: Provider) {
    setEditingProviderZid(provider.zid);
    setForm(providerToInput(provider));
    setShowForm(true);
  }

  return (
    <div className="space-y-4">
      <div className="flex items-center justify-between">
        <h2 className="skm-section-title">Providers 管理</h2>
        <button
          type="button"
          onClick={() => {
            setShowForm((value) => !value);
            if (showForm) {
              setEditingProviderZid(null);
              setForm(DEFAULT_PROVIDER);
            }
          }}
          className="inline-flex items-center gap-2 rounded-md bg-blue-600 px-3 py-1.5 text-sm font-medium text-white transition-colors hover:bg-blue-700"
        >
          <Plus className="h-4 w-4" />
          新增 Provider
        </button>
      </div>

      {showForm ? (
        <section className="skm-card p-4">
          <div className="mb-4 flex items-center justify-between">
            <div>
              <h3 className="text-sm font-semibold text-slate-800">{editingProviderZid ? "编辑 Provider" : "新增 Provider"}</h3>
              <p className="mt-1 text-xs text-slate-500">直接按 v2 数据模型维护 Provider，不保留旧流程。</p>
            </div>
            <button type="button" onClick={() => { setShowForm(false); setEditingProviderZid(null); setForm(DEFAULT_PROVIDER); }} className="rounded p-1 text-slate-400 hover:bg-slate-100 hover:text-slate-600">
              <X className="h-4 w-4" />
            </button>
          </div>
          <form className="grid gap-3 md:grid-cols-2" onSubmit={handleSubmit}>
            <input className="rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-blue-500" placeholder="名称" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
            <input className="rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-blue-500" placeholder="类型，如 workspace / repo" value={form.type} onChange={(event) => setForm({ ...form, type: event.target.value })} />
            <input className="md:col-span-2 rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-blue-500" placeholder="根目录绝对路径" value={form.rootPath} onChange={(event) => setForm({ ...form, rootPath: event.target.value })} />
            <input className="rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-blue-500" type="number" placeholder="优先级" value={String(form.priority)} onChange={(event) => setForm({ ...form, priority: Number(event.target.value) || 0 })} />
            <select value={form.scanMode} onChange={(event) => setForm({ ...form, scanMode: event.target.value })} className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-700 outline-none focus:border-blue-500">
              <option value="recursive">recursive</option>
              <option value="shallow">shallow</option>
            </select>
            <textarea
              className="md:col-span-2 min-h-28 rounded-md border border-slate-300 px-3 py-2 text-sm outline-none focus:border-blue-500"
              placeholder="描述"
              value={form.description}
              onChange={(event) => setForm({ ...form, description: event.target.value })}
            />
            <label className="flex items-center gap-2 text-sm text-slate-500">
              <input type="checkbox" checked={form.enabled} onChange={(event) => setForm({ ...form, enabled: event.target.checked })} />
              启用后参与扫描与冲突计算
            </label>
            <div className="flex items-center gap-3 md:justify-end">
              <button type="submit" disabled={submitting} className="rounded-md bg-blue-600 px-3 py-2 text-sm font-medium text-white transition-colors hover:bg-blue-700 disabled:cursor-not-allowed disabled:opacity-60">{submitting ? "提交中…" : editingProviderZid ? "保存变更" : "创建 Provider"}</button>
              <button type="button" className="rounded-md border border-slate-300 px-3 py-2 text-sm text-slate-600 transition-colors hover:bg-slate-50" onClick={() => { setShowForm(false); setEditingProviderZid(null); setForm(DEFAULT_PROVIDER); }}>取消</button>
            </div>
          </form>
        </section>
      ) : null}

      <section className="skm-card overflow-hidden">
        {error ? <p className="px-4 py-3 text-sm text-red-600">{error}</p> : null}
        <table className="w-full text-left text-sm whitespace-nowrap">
          <thead className="border-b border-slate-200 bg-slate-50 text-slate-600">
            <tr>
              <th className="px-4 py-2 font-medium w-12">启/停</th>
              <th className="px-4 py-2 font-medium">名称</th>
              <th className="px-4 py-2 font-medium">根目录路径</th>
              <th className="px-4 py-2 font-medium">类型</th>
              <th className="px-4 py-2 font-medium w-24">优先级</th>
              <th className="px-4 py-2 font-medium w-40">最近扫描</th>
              <th className="px-4 py-2 font-medium text-right w-28">操作</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-slate-100">
            {loading ? (
              <tr>
                <td colSpan={7} className="px-4 py-6 text-slate-400">加载中…</td>
              </tr>
            ) : providers.map((provider) => (
              <tr key={provider.zid} className="hover:bg-slate-50 transition-colors">
                <td className="px-4 py-3 text-center">
                  <input type="checkbox" checked={provider.enabled} onChange={() => void handleToggleEnabled(provider)} className="h-4 w-4 cursor-pointer rounded border-slate-300 text-blue-600" />
                </td>
                <td className="px-4 py-3">
                  <div className="font-medium text-slate-800">{provider.name}</div>
                  <div className="mt-0.5 text-xs text-slate-500">{skillCountByProvider.get(provider.zid) ?? 0} skills · {issueCountByProvider.get(provider.zid) ?? 0} issues</div>
                </td>
                <td className="px-4 py-3 font-mono text-xs text-slate-500">{provider.rootPath}</td>
                <td className="px-4 py-3"><span className="rounded border border-slate-200 bg-slate-50 px-2 py-0.5 text-xs text-slate-600">{provider.type}</span></td>
                <td className="px-4 py-3 text-center text-slate-700">{provider.priority}</td>
                <td className="px-4 py-3 text-xs text-slate-500">
                  <div>{provider.lastScanStatus || "never"}</div>
                  <div className="mt-0.5">{provider.lastScannedAt ? formatTime(provider.lastScannedAt) : "未扫描"}</div>
                </td>
                <td className="px-4 py-3 text-right">
                  <div className="inline-flex items-center gap-1">
                    <ActionIcon title="重新扫描" onClick={() => void handleScan(provider)}><RotateCw className="h-4 w-4" /></ActionIcon>
                    <ActionIcon title="编辑" onClick={() => startEdit(provider)}><Pen className="h-4 w-4" /></ActionIcon>
                    <ActionIcon title="删除" onClick={() => void handleDelete(provider)} danger><Trash2 className="h-4 w-4" /></ActionIcon>
                  </div>
                </td>
              </tr>
            ))}
            {!loading && providers.length === 0 ? (
              <tr>
                <td colSpan={7} className="px-4 py-10 text-center text-slate-500">当前没有 Provider</td>
              </tr>
            ) : null}
          </tbody>
        </table>
      </section>
    </div>
  );
}

function ActionIcon({ children, onClick, title, danger = false }: { children: ReactNode; onClick: () => void; title: string; danger?: boolean }) {
  return (
    <button type="button" onClick={onClick} title={title} className={`inline-flex h-7 w-7 items-center justify-center rounded transition-colors ${danger ? "text-red-500 hover:bg-red-50" : "text-slate-500 hover:bg-slate-100 hover:text-slate-800"}`}>
      {children}
    </button>
  );
}

function providerToInput(provider: Provider): ProviderInput {
  return {
    name: provider.name,
    type: provider.type,
    rootPath: provider.rootPath,
    enabled: provider.enabled,
    priority: provider.priority,
    scanMode: provider.scanMode,
    description: provider.description ?? "",
  };
}

function formatTime(value: string) {
  return new Intl.DateTimeFormat("zh-CN", {
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}