import { useEffect, useMemo, useState, type FormEvent } from "react";
import { Pen, RotateCw, Trash2 } from "lucide-react";
import { useOutletContext } from "react-router-dom";
import { toast } from "sonner";
import type { ShellOutletContext } from "../components/skm/ConsoleShell";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { Input } from "../components/ui/input";
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
      }
      await load();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Delete failed");
    }
  }

  async function handleToggleEnabled(provider: Provider) {
    try {
      await api.updateProvider(provider.zid, providerToInput({ ...provider, enabled: !provider.enabled }));
      toast.success(`${provider.name} 已${provider.enabled ? "停用" : "启用"}`);
      await load();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "Update failed");
    }
  }

  function startEdit(provider: Provider) {
    setEditingProviderZid(provider.zid);
    setForm(providerToInput(provider));
  }

  return (
    <div className="mx-auto flex max-w-7xl flex-col gap-6 px-6 py-8">
      <div className="grid gap-6 xl:grid-cols-[0.86fr_1.14fr]">
        <Card className="border-border/70 bg-white/82">
          <CardHeader>
            <CardTitle>{editingProviderZid ? "编辑 Provider" : "新增 Provider"}</CardTitle>
            <CardDescription>支持新增、编辑、启停、删除和单独扫描。默认递归扫描。</CardDescription>
          </CardHeader>
          <CardContent>
            <form className="grid gap-3" onSubmit={handleSubmit}>
              <Input placeholder="名称" value={form.name} onChange={(event) => setForm({ ...form, name: event.target.value })} />
              <Input placeholder="类型，如 workspace / repo" value={form.type} onChange={(event) => setForm({ ...form, type: event.target.value })} />
              <Input placeholder="根目录绝对路径" value={form.rootPath} onChange={(event) => setForm({ ...form, rootPath: event.target.value })} />
              <div className="grid gap-3 md:grid-cols-2">
                <Input type="number" placeholder="优先级" value={String(form.priority)} onChange={(event) => setForm({ ...form, priority: Number(event.target.value) || 0 })} />
                <select value={form.scanMode} onChange={(event) => setForm({ ...form, scanMode: event.target.value })} className="h-9 rounded-md border border-input bg-background px-3 text-sm">
                  <option value="recursive">recursive</option>
                  <option value="shallow">shallow</option>
                </select>
              </div>
              <textarea
                className="min-h-28 rounded-md border border-input bg-background px-3 py-2 text-sm"
                placeholder="描述"
                value={form.description}
                onChange={(event) => setForm({ ...form, description: event.target.value })}
              />
              <label className="flex items-center gap-2 text-sm text-muted-foreground">
                <input type="checkbox" checked={form.enabled} onChange={(event) => setForm({ ...form, enabled: event.target.checked })} />
                启用后参与扫描与冲突计算
              </label>
              <div className="flex gap-3">
                <Button type="submit" disabled={submitting}>{submitting ? "提交中…" : editingProviderZid ? "保存变更" : "创建 Provider"}</Button>
                {editingProviderZid ? <Button type="button" variant="outline" onClick={() => { setEditingProviderZid(null); setForm(DEFAULT_PROVIDER); }}>取消编辑</Button> : null}
              </div>
            </form>
          </CardContent>
        </Card>

        <Card className="border-border/70 bg-white/82">
          <CardHeader>
            <CardTitle>Provider 状态</CardTitle>
            <CardDescription>显示每个 Provider 的递归扫描模式、技能数量和 latest issue 计数。</CardDescription>
          </CardHeader>
          <CardContent className="grid gap-4">
            {error ? <p className="text-sm text-destructive">{error}</p> : null}
            {loading ? (
              <div className="text-sm text-muted-foreground">加载中…</div>
            ) : (
              providers.map((provider) => (
                <div key={provider.zid} className="group rounded-xl border border-slate-200 bg-white p-5 shadow-sm">
                  <div className="flex flex-col gap-3 lg:flex-row lg:items-start lg:justify-between">
                    <div className="space-y-2">
                      <div className="flex flex-wrap items-center gap-2">
                        <h2 className="text-lg font-semibold">{provider.name}</h2>
                        <Badge variant={provider.enabled ? "secondary" : "outline"}>{provider.enabled ? "enabled" : "disabled"}</Badge>
                        <Badge variant="outline">{provider.scanMode}</Badge>
                        <Badge variant="outline">priority {provider.priority}</Badge>
                      </div>
                      <p className="text-sm text-muted-foreground">{provider.rootPath}</p>
                      {provider.description ? <p className="text-sm text-muted-foreground">{provider.description}</p> : null}
                    </div>
                    <div className="flex items-center gap-2 opacity-100 transition-opacity lg:opacity-0 lg:group-hover:opacity-100">
                      <Button size="sm" variant="outline" onClick={() => void handleScan(provider)}><RotateCw className="h-4 w-4" />重扫</Button>
                      <Button size="sm" variant="outline" onClick={() => startEdit(provider)}><Pen className="h-4 w-4" />编辑</Button>
                      <Button size="sm" variant="outline" onClick={() => void handleDelete(provider)}><Trash2 className="h-4 w-4" />删除</Button>
                    </div>
                  </div>
                  <div className="mt-4 grid gap-3 sm:grid-cols-3">
                    <InlineMetric label="Skills" value={skillCountByProvider.get(provider.zid) ?? 0} />
                    <InlineMetric label="Latest Issues" value={issueCountByProvider.get(provider.zid) ?? 0} />
                    <InlineMetric label="Last Status" value={provider.lastScanStatus || "never"} />
                  </div>
                  <div className="mt-3 flex items-center gap-2 text-sm text-slate-500">
                    <input id={`enabled-${provider.zid}`} type="checkbox" checked={provider.enabled} onChange={() => void handleToggleEnabled(provider)} />
                    <label htmlFor={`enabled-${provider.zid}`}>启用 Provider</label>
                  </div>
                  {provider.lastScanSummary ? (
                    <div className="mt-3 rounded-xl bg-secondary/60 px-3 py-2 text-xs text-muted-foreground">
                      {provider.lastScanSummary}
                    </div>
                  ) : null}
                </div>
              ))
            )}
          </CardContent>
        </Card>
      </div>
    </div>
  );
}

function InlineMetric({ label, value }: { label: string; value: number | string }) {
  return (
    <div className="rounded-xl border border-border/60 bg-white/70 px-3 py-3">
      <div className="text-xs uppercase tracking-[0.16em] text-muted-foreground">{label}</div>
      <div className="mt-2 text-2xl font-semibold text-foreground">{value}</div>
    </div>
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