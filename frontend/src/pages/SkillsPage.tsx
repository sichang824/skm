import { useDeferredValue, useEffect, useMemo, useState } from "react";
import { ArrowRightLeft, FolderInput, FolderTree, Link2, Search } from "lucide-react";
import { useNavigate, useOutletContext, useParams } from "react-router-dom";
import { toast } from "sonner";
import { SkillDetailDialog } from "../components/skm/SkillDetailDialog";
import type { ShellOutletContext } from "../components/skm/ConsoleShell";
import { Accordion, AccordionContent, AccordionItem, AccordionTrigger } from "../components/ui/accordion";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "../components/ui/dialog";
import { api, type Provider, type Skill } from "../lib/api";

type ProviderAttachMode = "move" | "link";

export function SkillsPage() {
  const [providers, setProviders] = useState<Provider[]>([]);
  const [skills, setSkills] = useState<Skill[]>([]);
  const [search, setSearch] = useState("");
  const [selectedProviderZid, setSelectedProviderZid] = useState("");
  const [status, setStatus] = useState("");
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [draggingSkillZid, setDraggingSkillZid] = useState<string | null>(null);
  const [dropTargetProviderZid, setDropTargetProviderZid] = useState<string | null>(null);
  const [pendingDragSkill, setPendingDragSkill] = useState<Skill | null>(null);
  const [pendingDropProvider, setPendingDropProvider] = useState<Provider | null>(null);
  const [attachMode, setAttachMode] = useState<ProviderAttachMode>("move");
  const [attachDialogOpen, setAttachDialogOpen] = useState(false);
  const [attachSubmitting, setAttachSubmitting] = useState(false);
  const { refreshKey } = useOutletContext<ShellOutletContext>();
  const { zid } = useParams();
  const navigate = useNavigate();

  const deferredQuery = useDeferredValue(search);

  async function loadSkills() {
    setLoading(true);
    setError("");
    try {
      const [skillData, providerData] = await Promise.all([
        api.getSkills({ sort: "lastScanned" }),
        api.getProviders(),
      ]);
      setSkills(skillData);
      setProviders(providerData);
    } catch (loadError) {
      setError(loadError instanceof Error ? loadError.message : "Failed to load skills");
    } finally {
      setLoading(false);
    }
  }

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

  const selectedProvider = useMemo(
    () => providers.find((item) => item.zid === selectedProviderZid) ?? null,
    [providers, selectedProviderZid],
  );

  useEffect(() => {
    let active = true;

    async function loadSkillsSafe() {
      setLoading(true);
      setError("");
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

    void loadSkillsSafe();
    return () => {
      active = false;
    };
  }, [refreshKey]);

  useEffect(() => {
    if (selectedProviderZid && !providers.some((item) => item.zid === selectedProviderZid)) {
      setSelectedProviderZid("");
    }
  }, [providers, selectedProviderZid]);

  const filteredSkills = useMemo(() => {
    return skills.filter((skill) => {
      const matchesSearch = deferredQuery.trim() === ""
        ? true
        : [skill.name, skill.summary, skill.provider?.name, skill.category, skill.directoryName]
          .filter(Boolean)
          .some((value) => String(value).toLowerCase().includes(deferredQuery.toLowerCase()));
      const matchesProvider = selectedProviderZid ? skill.provider?.zid === selectedProviderZid : true;
      const matchesStatus = status === ""
        ? true
        : status === "Valid"
          ? skill.status === "ready" && !skill.isConflict
          : status === "Conflict"
            ? skill.isConflict
            : skill.status === "invalid";
      return matchesSearch && matchesProvider && matchesStatus;
    });
  }, [deferredQuery, selectedProviderZid, skills, status]);

  function resetDragState() {
    setDraggingSkillZid(null);
    setDropTargetProviderZid(null);
  }

  function handleSkillDragStart(skill: Skill) {
    setDraggingSkillZid(skill.zid);
  }

  function handleSkillDragEnd() {
    resetDragState();
  }

  function handleProviderDragOver(event: React.DragEvent<HTMLButtonElement>, providerItem: Provider) {
    if (!draggingSkillZid) {
      return;
    }
    event.preventDefault();
    if (dropTargetProviderZid !== providerItem.zid) {
      setDropTargetProviderZid(providerItem.zid);
    }
  }

  function handleProviderDragLeave(providerItem: Provider) {
    if (dropTargetProviderZid === providerItem.zid) {
      setDropTargetProviderZid(null);
    }
  }

  function handleProviderDrop(event: React.DragEvent<HTMLButtonElement>, providerItem: Provider) {
    event.preventDefault();
    const draggedSkill = skills.find((item) => item.zid === draggingSkillZid);
    resetDragState();
    if (!draggedSkill) {
      return;
    }
    if (draggedSkill.provider?.zid === providerItem.zid) {
      toast.info(`${draggedSkill.name} 已经属于 ${providerItem.name}`);
      return;
    }
    setPendingDragSkill(draggedSkill);
    setPendingDropProvider(providerItem);
    setAttachMode("move");
    setAttachDialogOpen(true);
  }

  function closeAttachDialog(open: boolean) {
    if (attachSubmitting && !open) {
      return;
    }
    setAttachDialogOpen(open);
    if (!open) {
      setPendingDragSkill(null);
      setPendingDropProvider(null);
      setAttachMode("move");
    }
  }

  async function handleConfirmAttach() {
    if (!pendingDragSkill || !pendingDropProvider) {
      return;
    }
    setAttachSubmitting(true);
    try {
      await api.attachSkill(pendingDragSkill.zid, {
        targetProviderZid: pendingDropProvider.zid,
        mode: attachMode,
      });
      await loadSkills();
      toast.success(`${pendingDragSkill.name} 已${attachMode === "move" ? "移动到" : "关联到"} ${pendingDropProvider.name}`);
      closeAttachDialog(false);
    } catch (submitError) {
      toast.error(submitError instanceof Error ? submitError.message : "操作失败");
    } finally {
      setAttachSubmitting(false);
    }
  }

  return (
    <div className="relative flex h-full min-h-0 flex-col gap-4">
      <div className="flex flex-wrap items-center justify-between gap-3">
        <div>
          <h2 className="skm-section-title">Skills 目录</h2>
          <p className="mt-1 text-sm text-slate-500">
            {selectedProvider ? `${selectedProvider.name} · ${skillCountByProvider.get(selectedProvider.zid) ?? 0} skills` : `全部 Providers · ${skills.length} skills`}
          </p>
        </div>

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
          <select value={status} onChange={(event) => setStatus(event.target.value)} className="rounded-md border border-slate-300 bg-white px-3 py-2 text-sm text-slate-700 outline-none transition-all focus:border-blue-500">
            <option value="">全部状态</option>
            <option value="Valid">正常</option>
            <option value="Conflict">冲突</option>
            <option value="Error">异常</option>
          </select>
        </div>
      </div>

      {error ? <p className="px-4 py-3 text-sm text-red-600">{error}</p> : null}

      <div className="grid min-h-0 flex-1 gap-4 overflow-visible lg:grid-cols-[280px_minmax(0,1fr)]">
        <aside className="skm-card min-h-0 overflow-visible">
          <div className="border-b border-slate-200 px-4 py-4">
            <div className="flex items-center gap-2 text-sm font-semibold text-slate-800">
              <FolderTree className="h-4 w-4 text-blue-600" />
              <span>Providers</span>
            </div>
            <p className="mt-1 text-xs text-slate-500">按 Provider 查看 Skills 分布与数量</p>
          </div>
          <div className="max-h-full overflow-auto px-4 py-3">
            <Accordion type="single" collapsible defaultValue="providers" className="w-full">
              <AccordionItem value="providers" className="border-b-0">
                <AccordionTrigger className="py-3 text-sm font-semibold text-slate-700 hover:no-underline">
                  <div className="flex flex-1 items-center justify-between gap-3 pr-2">
                    <span>Providers 列表</span>
                    <span className="rounded-full bg-slate-100 px-2 py-0.5 text-xs font-medium text-slate-500">{providers.length}</span>
                  </div>
                </AccordionTrigger>
                <AccordionContent className="pb-0">
                  <div className="space-y-2 px-1 py-1">
                    <button
                      type="button"
                      onClick={() => setSelectedProviderZid("")}
                      className={`flex w-full items-center justify-between rounded-xl px-3 py-2 text-left transition ${selectedProviderZid === "" ? "bg-blue-50 text-blue-700 ring-1 ring-blue-200" : "text-slate-600 hover:bg-slate-50 hover:text-slate-800"}`}
                    >
                      <span className="truncate text-sm font-medium">全部 Skills</span>
                      <span className={`rounded-full px-2 py-0.5 text-xs ${selectedProviderZid === "" ? "bg-blue-100 text-blue-700" : "bg-slate-100 text-slate-500"}`}>{skills.length}</span>
                    </button>
                    {providers.map((item) => {
                      const isActive = item.zid === selectedProviderZid;
                      const isDropTarget = item.zid === dropTargetProviderZid;
                      return (
                        <button
                          key={item.zid}
                          type="button"
                          onClick={() => setSelectedProviderZid(item.zid)}
                          onDragOver={(event) => handleProviderDragOver(event, item)}
                          onDragLeave={() => handleProviderDragLeave(item)}
                          onDrop={(event) => handleProviderDrop(event, item)}
                          className={`flex w-full items-center justify-between rounded-xl px-3 py-2 text-left transition ${isDropTarget ? "bg-emerald-50 text-emerald-700 ring-2 ring-emerald-200" : isActive ? "bg-blue-50 text-blue-700 ring-1 ring-blue-200" : "text-slate-600 hover:bg-slate-50 hover:text-slate-800"}`}
                        >
                          <div className="min-w-0">
                            <div className="truncate text-sm font-medium">{item.name}</div>
                            <div className={`truncate text-xs ${isDropTarget ? "text-emerald-500" : "text-slate-400"}`}>{isDropTarget ? "释放以选择迁移方式" : item.type}</div>
                          </div>
                          <span className={`ml-3 rounded-full px-2 py-0.5 text-xs ${isDropTarget ? "bg-emerald-100 text-emerald-700" : isActive ? "bg-blue-100 text-blue-700" : "bg-slate-100 text-slate-500"}`}>{skillCountByProvider.get(item.zid) ?? 0}</span>
                        </button>
                      );
                    })}
                  </div>
                </AccordionContent>
              </AccordionItem>
            </Accordion>
          </div>
        </aside>

        <div className="skm-card min-h-0 overflow-hidden">
          <div className="border-b border-slate-200 px-4 py-3">
            <div className="flex flex-wrap items-center justify-between gap-2">
              <div>
                <h3 className="text-sm font-semibold text-slate-800">{selectedProvider ? `${selectedProvider.name} 的 Skills` : "全部 Skills"}</h3>
                <p className="mt-1 text-xs text-slate-500">
                  {filteredSkills.length} 条结果
                  {status ? ` · 状态 ${status}` : ""}
                  {deferredQuery.trim() ? ` · 搜索 “${deferredQuery.trim()}”` : ""}
                </p>
              </div>
            </div>
          </div>
          <div className="h-full overflow-auto">
            <table className="w-full text-left text-sm whitespace-nowrap">
              <thead className="sticky top-0 z-10 border-b border-slate-200 bg-slate-50 text-slate-600">
                <tr>
                  <th className="px-4 py-2 font-medium">Skill 名称</th>
                  <th className="px-4 py-2 font-medium">Provider</th>
                  <th className="px-4 py-2 font-medium">分类</th>
                  <th className="w-24 px-4 py-2 font-medium">状态</th>
                  <th className="w-20 px-4 py-2 text-right font-medium">操作</th>
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
                  <tr
                    key={skill.zid}
                    draggable
                    onDragStart={() => handleSkillDragStart(skill)}
                    onDragEnd={handleSkillDragEnd}
                    className={`group cursor-pointer transition-colors hover:bg-blue-50 ${draggingSkillZid === skill.zid ? "bg-blue-50/70 opacity-70" : ""}`}
                    onClick={() => navigate(`/skills/${skill.zid}`)}
                  >
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
        </div>
      </div>

      {zid ? (
        <div className="absolute inset-4 z-20">
          <SkillDetailDialog zid={zid} open={Boolean(zid)} onOpenChange={(open) => { if (!open) { navigate("/skills"); } }} onDeleted={() => { void loadSkills(); }} />
        </div>
      ) : null}

      <Dialog open={attachDialogOpen} onOpenChange={closeAttachDialog}>
        <DialogContent className="max-w-2xl rounded-2xl border-slate-200 bg-white p-0 shadow-[0_24px_90px_rgba(15,23,42,0.16)]" showCloseButton={false}>
          <div className="border-b border-slate-200 px-6 py-5">
            <DialogHeader className="gap-2 text-left">
              <DialogTitle className="text-xl font-semibold text-slate-900">拖拽操作确认</DialogTitle>
              <DialogDescription className="text-sm text-slate-500">
                {pendingDragSkill && pendingDropProvider
                  ? `已将 ${pendingDragSkill.name} 拖到 ${pendingDropProvider.name}。请选择要执行的目录处理方式。`
                  : "请选择要执行的目录处理方式。"}
              </DialogDescription>
            </DialogHeader>
            {pendingDragSkill && pendingDropProvider ? (
              <div className="mt-4 flex flex-wrap items-center gap-3 text-sm text-slate-600">
                <span className="rounded-full bg-slate-100 px-3 py-1 font-medium text-slate-700">{pendingDragSkill.name}</span>
                <ArrowRightLeft className="h-4 w-4 text-slate-400" />
                <span className="rounded-full bg-blue-50 px-3 py-1 font-medium text-blue-700">{pendingDropProvider.name}</span>
              </div>
            ) : null}
          </div>

          <div className="grid gap-4 px-6 py-6 md:grid-cols-2">
            <ActionModeCard
              title="移动"
              description="将 Skill 目录整体迁移到目标 Provider 根目录，适合明确变更归属。"
              icon={FolderInput}
              selected={attachMode === "move"}
              accent="blue"
              onSelect={() => setAttachMode("move")}
            />
            <ActionModeCard
              title="关联"
              description="在目标 Provider 下建立目录链接，保留原目录位置，适合共享复用。"
              icon={Link2}
              selected={attachMode === "link"}
              accent="emerald"
              onSelect={() => setAttachMode("link")}
            />
          </div>

          <DialogFooter className="border-t border-slate-200 px-6 py-4 sm:justify-between">
            <button
              type="button"
              onClick={() => closeAttachDialog(false)}
              disabled={attachSubmitting}
              className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-600 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              取消
            </button>
            <button
              type="button"
              onClick={handleConfirmAttach}
              disabled={attachSubmitting}
              className="rounded-lg bg-slate-900 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-slate-800 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {attachSubmitting ? "处理中…" : `确认${attachMode === "move" ? "移动" : "关联"}`}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function ActionModeCard({
  title,
  description,
  icon: Icon,
  selected,
  accent,
  onSelect,
}: {
  title: string;
  description: string;
  icon: typeof FolderInput;
  selected: boolean;
  accent: "blue" | "emerald";
  onSelect: () => void;
}) {
  const selectedClass = accent === "blue"
    ? "border-blue-200 bg-blue-50 shadow-[0_16px_32px_rgba(37,99,235,0.12)]"
    : "border-emerald-200 bg-emerald-50 shadow-[0_16px_32px_rgba(16,185,129,0.12)]";
  const iconClass = accent === "blue" ? "bg-blue-100 text-blue-700" : "bg-emerald-100 text-emerald-700";

  return (
    <button
      type="button"
      onClick={onSelect}
      className={`rounded-2xl border p-5 text-left transition ${selected ? selectedClass : "border-slate-200 bg-white hover:border-slate-300 hover:bg-slate-50"}`}
    >
      <div className="flex items-start justify-between gap-4">
        <div>
          <div className="text-base font-semibold text-slate-900">{title}</div>
          <p className="mt-2 text-sm leading-6 text-slate-500">{description}</p>
        </div>
        <span className={`inline-flex h-10 w-10 shrink-0 items-center justify-center rounded-xl ${iconClass}`}>
          <Icon className="h-5 w-5" />
        </span>
      </div>
      <div className="mt-4 text-xs font-medium text-slate-400">
        {selected ? "已选择该方式" : "点击选择该方式"}
      </div>
    </button>
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