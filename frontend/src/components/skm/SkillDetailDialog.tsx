import { useEffect, useMemo, useState } from "react";
import { Copy, ExternalLink, FileText, FolderOpen, Link2Off, Play, RefreshCw, Terminal, Trash2, X } from "lucide-react";
import { toast } from "sonner";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "../ui/dialog";
import { api, ApiRequestError, type CommandsView, type ExecRecord, type FileNode, type Skill, type SkillExecResult } from "../../lib/api";

const RELATION_SOURCE_PREVIEW = "__relation_from__";
const RELATION_OUTPUT_PREVIEW = "__relation_to__";
const COMMANDS_PREVIEW = "__commands__";

type SkillDetailDialogProps = {
  zid: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
  onSynced?: () => void;
};

export function SkillDetailDialog({ zid, open, onOpenChange, onDeleted, onSynced }: SkillDetailDialogProps) {
  const [skill, setSkill] = useState<Skill | null>(null);
  const [files, setFiles] = useState<FileNode[]>([]);
  const [selectedPath, setSelectedPath] = useState("SKILL.md");
  const [content, setContent] = useState("");
  const [loading, setLoading] = useState(false);
  const [previewError, setPreviewError] = useState("");
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [removeRelationDialogOpen, setRemoveRelationDialogOpen] = useState(false);
  const [removeRelationError, setRemoveRelationError] = useState("");
  const [deleting, setDeleting] = useState(false);
  const [removingRelation, setRemovingRelation] = useState(false);
  const [syncing, setSyncing] = useState(false);

  async function loadSkillDetail(skillZid: string, active: () => boolean) {
    setLoading(true);
    try {
      const [skillData, fileTree] = await Promise.all([
        api.getSkill(skillZid),
        api.getSkillFiles(skillZid),
      ]);
      if (!active()) {
        return;
      }
      setSkill(skillData);
      setFiles(fileTree);
      setSelectedPath((currentPath) => {
        if (currentPath === RELATION_SOURCE_PREVIEW && skillData.relation?.mode === "from") {
          return currentPath;
        }
        if (currentPath === RELATION_OUTPUT_PREVIEW && skillData.relation?.mode === "to") {
          return currentPath;
        }
        if (currentPath === COMMANDS_PREVIEW && (skillData.commands?.length ?? 0) > 0) {
          return currentPath;
        }
        if (currentPath && findFilePath(fileTree, currentPath)) {
          return currentPath;
        }
        return findFirstFilePath(fileTree) ?? "SKILL.md";
      });
    } catch (error) {
      if (!active()) {
        return;
      }
      toast.error(error instanceof Error ? error.message : "加载技能详情失败");
    } finally {
      if (active()) {
        setLoading(false);
      }
    }
  }

  useEffect(() => {
    let active = true;
    if (!open || !zid) {
      return () => {
        active = false;
      };
    }
    setLoading(true);
    const skillZid = zid;

    void loadSkillDetail(skillZid, () => active);
    return () => {
      active = false;
    };
  }, [open, zid]);

  useEffect(() => {
    let active = true;
    if (!open || !zid || !selectedPath) {
      return () => {
        active = false;
      };
    }
    const skillZid = zid;

    if (selectedPath === RELATION_SOURCE_PREVIEW || selectedPath === RELATION_OUTPUT_PREVIEW || selectedPath === COMMANDS_PREVIEW) {
      setContent("");
      setPreviewError("");
      return () => {
        active = false;
      };
    }

    async function loadContent() {
      try {
        const file = await api.getSkillFileContent(skillZid, selectedPath);
        if (!active) {
          return;
        }
        setContent(file.content);
        setPreviewError("");
      } catch (error) {
        if (!active) {
          return;
        }
        setContent("");
        setPreviewError(error instanceof Error ? error.message : "文件预览失败");
      }
    }

    void loadContent();
    return () => {
      active = false;
    };
  }, [open, selectedPath, zid]);

  const displaySummary = useMemo(() => {
    if (!skill) {
      return "";
    }
    return skill.summary || skill.bodyMarkdown || "暂无摘要";
  }, [skill]);

  async function copyText(value: string) {
    try {
      await navigator.clipboard.writeText(value);
      toast.success("路径已复制");
    } catch {
      toast.error("复制失败");
    }
  }

  async function handleRevealInFinder() {
    if (!skill?.rootPath) {
      return;
    }
    try {
      await api.revealInFinder(skill.rootPath);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "无法在 Finder 中显示");
    }
  }

  async function handleDeleteSkill() {
    if (!zid || !skill) {
      return;
    }
    const attachedCopyCount = skill.relation?.mode === "to" ? (skill.relation.directories?.length ?? 0) : 0;
    const forceDelete = attachedCopyCount > 0;
    setDeleting(true);
    try {
      const result = await api.deleteSkill(zid, { force: forceDelete });
      const successMessage = result.deleteMode === "attached-copy"
        ? `${skill.name} 已删除，关联来源已清理`
        : result.forced
          ? `${skill.name} 已强制删除，现有副本未联动清理`
          : `${skill.name} 已删除`;
      toast.success(successMessage);
      setDeleteDialogOpen(false);
      onDeleted?.();
      onOpenChange(false);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "删除 Skill 失败");
    } finally {
      setDeleting(false);
    }
  }

  async function handleSyncSkill() {
    if (!zid || !skill || skill.relation?.mode !== "from") {
      return;
    }
    setSyncing(true);
    try {
      await api.syncSkill(zid);
      toast.success(`${skill.name} 已从关联来源同步`);
      await loadSkillDetail(zid, () => true);
      onSynced?.();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "同步 Skill 失败");
    } finally {
      setSyncing(false);
    }
  }

  async function handleSyncSkillCopies() {
    if (!zid || !skill || skill.relation?.mode !== "to") {
      return;
    }
    const copyCount = skill.relation.directories?.length ?? 0;
    if (copyCount === 0) {
      toast.error("当前 Skill 没有关联副本目录");
      return;
    }
    setSyncing(true);
    try {
      const result = await api.syncSkillCopies(zid);
      toast.success(`${skill.name} 已同步到 ${result.copies.length} 个关联副本`);
      await loadSkillDetail(zid, () => true);
      onSynced?.();
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "同步关联副本失败");
    } finally {
      setSyncing(false);
    }
  }

  async function handleRemoveRelation() {
    if (!zid || !skill?.relation) {
      return;
    }
    setRemovingRelation(true);
    setRemoveRelationError("");
    try {
      const result = await api.removeSkillRelation(zid);
      const successMessage = result.removedMode === "from"
        ? `${skill.name} 的关联副本标记已移除`
        : `${skill.name} 的关联副本已清理，.to 规则已保留`;
      setRemoveRelationDialogOpen(false);
      setSelectedPath("SKILL.md");
      toast.success(successMessage);
      await loadSkillDetail(zid, () => true);
      onSynced?.();
    } catch (error) {
      const message = error instanceof Error ? error.message : "移除关联失败";
      setRemoveRelationError(message);
      toast.error(message);
    } finally {
      setRemovingRelation(false);
    }
  }

  if (!open || !zid) {
    return null;
  }

  const previewPath = selectedPath || "SKILL.md";
  const isRelationSourcePreview = previewPath === RELATION_SOURCE_PREVIEW;
  const isRelationOutputPreview = previewPath === RELATION_OUTPUT_PREVIEW;
  const isCommandsPreview = previewPath === COMMANDS_PREVIEW;
  const isRelationPreview = isRelationSourcePreview || isRelationOutputPreview;
  const isSkillMarkdown = !isRelationPreview && !isCommandsPreview && /(^|\/)SKILL\.md$/i.test(previewPath);
  const previewTitle = isRelationSourcePreview
    ? "关联来源"
    : isRelationOutputPreview
      ? "关联输出"
      : isCommandsPreview
        ? "可执行命令"
        : previewPath;
  const attachedCopyCount = skill?.relation?.mode === "to" ? (skill.relation.directories?.length ?? 0) : 0;
  const isAttachedCopy = skill?.relation?.mode === "from";
  const hasRelation = skill?.relation?.mode === "from" || skill?.relation?.mode === "to";
  const issueBadge = skill?.status === "invalid" ? "异常" : skill?.isConflict ? "存在冲突" : "Frontmatter Parsed";
  const issueBadgeClass = skill?.status === "invalid"
    ? "bg-red-50 text-red-700"
    : skill?.isConflict
      ? "bg-amber-50 text-amber-700"
      : "bg-green-50 text-green-700";

  return (
    <div className="flex h-full flex-col overflow-hidden rounded-2xl border border-slate-200 bg-white shadow-[0_24px_90px_rgba(15,23,42,0.18)] animate-in fade-in zoom-in-95 duration-200">
      <div className="flex items-start justify-between border-b border-slate-200 bg-slate-50 px-4 py-3">
        <div className="min-w-0">
          <div className="mb-1 flex items-center gap-2">
            <h2 className="text-lg font-bold text-slate-800">
              <span className="mr-2 text-blue-600">◈</span>
              {skill?.name ?? "Skill Detail"}
            </h2>
            <span className={`rounded border px-2 py-0.5 text-xs ${issueBadgeClass}`}>{issueBadge}</span>
          </div>
          <div className="max-w-xl truncate font-mono text-xs text-slate-500" title={skill?.rootPath ?? ""}>{skill?.rootPath ?? "加载中..."}</div>
        </div>
        <div className="flex items-center gap-2">
          <button type="button" onClick={() => skill ? void copyText(skill.rootPath) : undefined} className="inline-flex items-center gap-1 rounded border border-slate-200 bg-white px-2 py-1 text-xs text-slate-600 shadow-sm transition-colors hover:text-blue-600">
            <FolderOpen className="h-3.5 w-3.5" />
            复制目录
          </button>
          <button type="button" onClick={() => void handleRevealInFinder()} className="inline-flex items-center gap-1 rounded border border-slate-200 bg-white px-2 py-1 text-xs text-slate-600 shadow-sm transition-colors hover:text-blue-600" disabled={!skill}>
            <ExternalLink className="h-3.5 w-3.5" />
            Reveal in Finder
          </button>
          {hasRelation ? (
            <button type="button" onClick={() => { setRemoveRelationError(""); setRemoveRelationDialogOpen(true); }} className="inline-flex items-center gap-1 rounded border border-amber-200 bg-amber-50 px-2 py-1 text-xs text-amber-800 shadow-sm transition-colors hover:bg-amber-100" disabled={!skill || removingRelation}>
              <Link2Off className="h-3.5 w-3.5" />
              移除关联
            </button>
          ) : null}
          <button type="button" onClick={() => setDeleteDialogOpen(true)} className="inline-flex items-center gap-1 rounded border border-red-200 bg-red-50 px-2 py-1 text-xs text-red-700 shadow-sm transition-colors hover:bg-red-100" disabled={!skill || deleting}>
            <Trash2 className="h-3.5 w-3.5" />
            删除 Skill
          </button>
          <button type="button" onClick={() => onOpenChange(false)} className="inline-flex h-7 w-7 items-center justify-center rounded text-slate-500 transition-colors hover:bg-slate-200 hover:text-slate-700" title="关闭">
            <X className="h-4 w-4" />
          </button>
        </div>
      </div>

      <div className="flex min-h-0 flex-1 overflow-hidden">
        <aside className="flex w-64 shrink-0 flex-col border-r border-slate-200 bg-slate-50">
          <div className="border-b border-slate-200 p-3">
            <h3 className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">属性</h3>
            <div className="space-y-2 text-sm">
              <div className="flex justify-between gap-3">
                <span className="text-slate-500">Provider</span>
                <span className="text-right font-medium text-slate-700">{skill?.provider?.name ?? "Unknown"}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span className="text-slate-500">分类</span>
                <span className="text-right font-medium text-slate-700">{skill?.category ?? "Uncategorized"}</span>
              </div>
              <div className="flex justify-between gap-3">
                <span className="text-slate-500">状态</span>
                <span className="text-right font-medium text-slate-700">{skill?.status ?? "unknown"}</span>
              </div>
              {skill?.relation?.mode === "from" ? (
                <div className={`rounded border px-2 py-2 text-xs ${selectedPath === RELATION_SOURCE_PREVIEW ? "border-emerald-300 bg-emerald-100 text-emerald-800" : "border-emerald-100 bg-emerald-50 text-emerald-700"}`}>
                  <button
                    type="button"
                    onClick={() => setSelectedPath(RELATION_SOURCE_PREVIEW)}
                    className="block min-w-0 w-full cursor-pointer text-left"
                  >
                    <div className="font-medium">关联来源</div>
                    <div className="mt-1 break-all font-mono text-[11px]">{skill.relation.fromPath}</div>
                  </button>
                </div>
              ) : null}
              {skill?.relation?.mode === "to" ? (
                <button
                  type="button"
                  onClick={() => setSelectedPath(RELATION_OUTPUT_PREVIEW)}
                  className={`block w-full cursor-pointer rounded border px-2 py-2 text-left text-xs ${selectedPath === RELATION_OUTPUT_PREVIEW ? "border-blue-300 bg-blue-100 text-blue-800" : "border-blue-100 bg-blue-50 text-blue-700"}`}
                >
                  <div className="font-medium">关联输出</div>
                  <div className="mt-1">{`${skill.relation.directories?.length ?? 0} 个目录，${skill.relation.include?.length ?? 0} 条包含规则，${skill.relation.exclude?.length ?? 0} 条排除规则`}</div>
                </button>
              ) : null}
              {(skill?.commands?.length ?? 0) > 0 ? (
                <button
                  type="button"
                  onClick={() => setSelectedPath(COMMANDS_PREVIEW)}
                  className={`block w-full cursor-pointer rounded border px-2 py-2 text-left text-xs ${selectedPath === COMMANDS_PREVIEW ? "border-amber-300 bg-amber-100 text-amber-800" : "border-amber-100 bg-amber-50 text-amber-700"}`}
                >
                  <div className="flex items-center gap-1 font-medium"><Terminal className="h-3.5 w-3.5" /> 可执行命令</div>
                  <div className="mt-1">{`${skill?.commands?.length ?? 0} 条命令，点击运行或预览`}</div>
                </button>
              ) : null}
              {skill?.tags.length ? (
                <div>
                  <div className="mb-1 text-slate-500">标签</div>
                  <div className="flex flex-wrap gap-1">
                    {skill.tags.map((tag) => <span key={tag} className="rounded border border-blue-100 bg-blue-50 px-1.5 py-0.5 text-[11px] text-blue-700">{tag}</span>)}
                  </div>
                </div>
              ) : null}
            </div>
            <button type="button" onClick={() => skill ? void copyText(skill.skillMdPath ?? `${skill.rootPath}/SKILL.md`) : undefined} className="mt-3 inline-flex items-center gap-2 rounded border border-slate-200 bg-white px-2 py-1 text-xs text-slate-600 hover:text-blue-600">
              <Copy className="h-3.5 w-3.5" />
              复制 SKILL.md
            </button>
          </div>

          <div className="flex-1 overflow-y-auto p-3">
            <h3 className="mb-2 text-xs font-semibold uppercase tracking-[0.18em] text-slate-500">目录结构</h3>
            <div className="space-y-1">
              {loading ? <p className="text-xs text-slate-500">加载中…</p> : <FileTree nodes={files} selectedPath={selectedPath} onSelect={setSelectedPath} />}
            </div>
          </div>
        </aside>

        <section className="flex min-w-0 flex-1 flex-col bg-white">
          <div className="flex items-center gap-4 border-b border-slate-200 bg-slate-100 px-3 py-1.5 font-mono text-xs text-slate-600">
            <span className="flex items-center gap-1"><FileText className="h-3.5 w-3.5 text-blue-500" /> {previewTitle}</span>
            <span className="text-slate-300">|</span>
            <span className={issueBadgeClass}>✓ {issueBadge}</span>
          </div>

          <div className="flex-1 overflow-y-auto p-6">
            {previewError ? <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">{previewError}</div> : null}
            {loading || !skill ? <p className="text-sm text-slate-500">加载中…</p> : null}
            {!loading && skill ? (
              isCommandsPreview ? (
                <SkillCommandsPanel zid={zid} />
              ) : isRelationSourcePreview ? (
                <div className="space-y-4 text-sm text-slate-700">
                  <div className="rounded-xl border border-emerald-100 bg-emerald-50 p-4">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="text-xs font-semibold uppercase tracking-[0.18em] text-emerald-600">关联来源</div>
                        <div className="mt-3 break-all font-mono text-sm text-emerald-900">{skill.relation?.fromPath}</div>
                      </div>
                      <button
                        type="button"
                        onClick={() => void handleSyncSkill()}
                        disabled={syncing}
                        className="inline-flex shrink-0 cursor-pointer items-center gap-1 rounded border border-emerald-200 bg-white px-3 py-1.5 text-xs font-medium text-emerald-700 transition-colors hover:bg-emerald-100 disabled:cursor-not-allowed disabled:opacity-60"
                      >
                        <RefreshCw className={`h-3.5 w-3.5 ${syncing ? "animate-spin" : ""}`} />
                        {syncing ? "同步中" : "同步"}
                      </button>
                    </div>
                  </div>
                  <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
                    <div className="text-sm font-medium text-slate-800">说明</div>
                    <p className="mt-2 leading-6 text-slate-600">当前 Skill 是一个关联副本。点击这里的“同步”按钮后，会从来源目录覆盖同步到当前目录。</p>
                  </div>
                </div>
              ) : isRelationOutputPreview ? (
                <div className="space-y-4 text-sm text-slate-700">
                  <div className="rounded-xl border border-blue-100 bg-blue-50 p-4">
                    <div className="flex items-start justify-between gap-3">
                      <div>
                        <div className="text-xs font-semibold uppercase tracking-[0.18em] text-blue-600">关联输出概览</div>
                        <div className="mt-3 text-sm text-blue-900">{`${skill.relation?.directories?.length ?? 0} 个目录，${skill.relation?.include?.length ?? 0} 条包含规则，${skill.relation?.exclude?.length ?? 0} 条排除规则`}</div>
                      </div>
                      <button
                        type="button"
                        onClick={() => void handleSyncSkillCopies()}
                        disabled={syncing || (skill.relation?.directories?.length ?? 0) === 0}
                        className="inline-flex shrink-0 cursor-pointer items-center gap-1 rounded border border-blue-200 bg-white px-3 py-1.5 text-xs font-medium text-blue-700 transition-colors hover:bg-blue-100 disabled:cursor-not-allowed disabled:opacity-60"
                      >
                        <RefreshCw className={`h-3.5 w-3.5 ${syncing ? "animate-spin" : ""}`} />
                        {syncing ? "同步中" : "同步全部副本"}
                      </button>
                    </div>
                  </div>
                  <div className="grid gap-4 lg:grid-cols-3">
                    <div className="rounded-xl border border-slate-200 bg-white p-4">
                      <div className="mb-3 text-sm font-medium text-slate-800">包含规则</div>
                      {(skill.relation?.include?.length ?? 0) > 0 ? (
                        <ul className="space-y-2 text-sm text-slate-600">
                          {(skill.relation?.include ?? []).map((pattern) => <li key={pattern} className="break-all font-mono text-xs">{pattern}</li>)}
                        </ul>
                      ) : (
                        <p className="text-sm text-slate-500">默认包含全部文件</p>
                      )}
                    </div>
                    <div className="rounded-xl border border-slate-200 bg-white p-4">
                      <div className="mb-3 text-sm font-medium text-slate-800">排除规则</div>
                      {(skill.relation?.exclude?.length ?? 0) > 0 ? (
                        <ul className="space-y-2 text-sm text-slate-600">
                          {(skill.relation?.exclude ?? []).map((pattern) => <li key={pattern} className="break-all font-mono text-xs">{pattern}</li>)}
                        </ul>
                      ) : (
                        <p className="text-sm text-slate-500">暂无排除规则</p>
                      )}
                    </div>
                    <div className="rounded-xl border border-slate-200 bg-white p-4">
                      <div className="mb-3 text-sm font-medium text-slate-800">关联目录</div>
                      {(skill.relation?.directories?.length ?? 0) > 0 ? (
                        <ul className="space-y-2 text-sm text-slate-600">
                          {(skill.relation?.directories ?? []).map((directory) => <li key={directory} className="break-all font-mono text-xs">{directory}</li>)}
                        </ul>
                      ) : (
                        <p className="text-sm text-slate-500">暂无关联目录</p>
                      )}
                    </div>
                  </div>
                  <div className="rounded-xl border border-slate-200 bg-slate-50 p-4">
                    <div className="text-sm font-medium text-slate-800">说明</div>
                    <p className="mt-2 leading-6 text-slate-600">当前 Skill 是关联源。点击“同步全部副本”后，会按 `.to` 规则把来源目录覆盖同步到每一个关联副本目录。</p>
                  </div>
                </div>
              ) : isSkillMarkdown ? (
                <div className="skm-prose max-w-none text-sm">
                  <h1>{skill.name}</h1>
                  <p>{displaySummary}</p>

                  <h2>Frontmatter</h2>
                  <pre><code>{formatFrontmatter(skill)}</code></pre>

                  {skill.bodyMarkdown ? (
                    <>
                      <h2>Instructions</h2>
                      <pre><code>{skill.bodyMarkdown}</code></pre>
                    </>
                  ) : null}

                  {skill.issueCodes.length > 0 ? (
                    <>
                      <h2>Issue Codes</h2>
                      <ul>
                        {skill.issueCodes.map((code) => <li key={code}>{code}</li>)}
                      </ul>
                    </>
                  ) : null}
                </div>
              ) : (
                <pre className="overflow-x-auto rounded-lg bg-slate-900 p-4 text-sm leading-6 whitespace-pre-wrap text-slate-100">{content}</pre>
              )
            ) : null}
          </div>
        </section>
      </div>

      <Dialog open={deleteDialogOpen} onOpenChange={(nextOpen) => { if (!deleting) { setDeleteDialogOpen(nextOpen); } }}>
        <DialogContent className="max-w-md rounded-2xl border-red-100 bg-white p-0 shadow-[0_24px_90px_rgba(15,23,42,0.16)]" showCloseButton={false}>
          <div className="px-6 py-5">
            <DialogHeader className="gap-2 text-left">
              <DialogTitle className="text-xl font-semibold text-slate-900">确认删除 Skill</DialogTitle>
              <DialogDescription className="text-sm leading-6 text-red-600">
                {attachedCopyCount > 0
                  ? `当前源 Skill 存在 ${attachedCopyCount} 个副本。若继续强制删除，只会删除源目录，不会逐个清理副本中的 .from。`
                  : isAttachedCopy
                    ? "该操作会删除当前关联副本目录，并同步清理来源 Skill 的 .to 目录记录。"
                    : "该操作会直接删除 Skill 目录。"}
                {skill ? ` 删除后将移除 ${skill.name} 对应目录：${skill.rootPath}` : ""}
              </DialogDescription>
            </DialogHeader>
          </div>
          <DialogFooter className="border-t border-slate-200 px-6 py-4 sm:justify-between">
            <button
              type="button"
              onClick={() => setDeleteDialogOpen(false)}
              disabled={deleting}
              className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-600 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              取消
            </button>
            <button
              type="button"
              onClick={() => void handleDeleteSkill()}
              disabled={deleting}
              className="rounded-lg bg-red-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-red-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {deleting ? "删除中…" : attachedCopyCount > 0 ? "强制删除源 Skill" : "确认删除目录"}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>

      <Dialog open={removeRelationDialogOpen} onOpenChange={(nextOpen) => { if (!removingRelation) { setRemoveRelationDialogOpen(nextOpen); } }}>
        <DialogContent className="max-w-md rounded-2xl border-amber-100 bg-white p-0 shadow-[0_24px_90px_rgba(15,23,42,0.16)]" showCloseButton={false}>
          <div className="px-6 py-5">
            <DialogHeader className="gap-2 text-left">
              <DialogTitle className="text-xl font-semibold text-slate-900">确认移除关联</DialogTitle>
              <DialogDescription className="text-sm leading-6 text-amber-700">
                {isAttachedCopy
                  ? "将删除当前目录的 .from，并从来源 Skill 的 .to 中移除该目录记录。.to 的包含/排除规则会保留，便于后续再次关联。"
                  : attachedCopyCount > 0
                    ? `将清理 ${attachedCopyCount} 个关联副本目录中的 .from，并清空 .to 的目录列表。包含/排除规则会保留，便于后续再次关联。`
                    : "将清空 .to 的目录列表。包含/排除规则会保留，便于后续再次关联。"}
              </DialogDescription>
              {removeRelationError ? (
                <p className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-sm text-red-700">{removeRelationError}</p>
              ) : null}
            </DialogHeader>
          </div>
          <DialogFooter className="border-t border-slate-200 px-6 py-4 sm:justify-between">
            <button
              type="button"
              onClick={() => setRemoveRelationDialogOpen(false)}
              disabled={removingRelation}
              className="rounded-lg border border-slate-200 px-4 py-2 text-sm font-medium text-slate-600 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-50"
            >
              取消
            </button>
            <button
              type="button"
              onClick={() => void handleRemoveRelation()}
              disabled={removingRelation}
              className="rounded-lg bg-amber-600 px-4 py-2 text-sm font-medium text-white transition-colors hover:bg-amber-700 disabled:cursor-not-allowed disabled:opacity-60"
            >
              {removingRelation ? "移除中…" : "确认移除关联"}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
    </div>
  );
}

function SkillCommandsPanel({ zid }: { zid: string }) {
  const [view, setView] = useState<CommandsView | null>(null);
  const [loadError, setLoadError] = useState("");
  const [selectedCommand, setSelectedCommand] = useState("");
  const [inputJson, setInputJson] = useState("");
  const [envText, setEnvText] = useState("");
  const [pin, setPin] = useState("");
  const [assumeYes, setAssumeYes] = useState(false);
  const [isolate, setIsolate] = useState(false);
  const [running, setRunning] = useState(false);
  const [result, setResult] = useState<SkillExecResult | null>(null);
  const [error, setError] = useState("");
  const [confirmPrompt, setConfirmPrompt] = useState("");
  const [historyKey, setHistoryKey] = useState(0);

  useEffect(() => {
    let active = true;
    setLoadError("");
    api.getSkillCommands(zid)
      .then((data) => {
        if (!active) {
          return;
        }
        setView(data);
        setSelectedCommand((current) => current || data.commands[0]?.name || "");
      })
      .catch((panelError) => {
        if (active) {
          setLoadError(panelError instanceof Error ? panelError.message : "加载命令清单失败");
        }
      });
    return () => {
      active = false;
    };
  }, [zid]);

  const command = useMemo(
    () => view?.commands.find((entry) => entry.name === selectedCommand) ?? null,
    [view, selectedCommand],
  );

  function resetRunState() {
    setResult(null);
    setError("");
    setConfirmPrompt("");
  }

  async function runCommand(dryRun: boolean) {
    if (!command) {
      return;
    }
    const envEntries = envText
      .split("\n")
      .map((line) => line.trim())
      .filter((line) => line.length > 0);
    const invalidEnv = envEntries.find((entry) => !entry.includes("="));
    if (invalidEnv) {
      setError(`环境变量格式错误：${invalidEnv}（应为 KEY=VAL）`);
      return;
    }
    setRunning(true);
    resetRunState();
    try {
      const execResult = await api.execSkillCommand(zid, {
        command: command.name,
        input: inputJson.trim() ? inputJson : undefined,
        env: envEntries.length > 0 ? envEntries : undefined,
        assumeYes: assumeYes || undefined,
        isolate: isolate || undefined,
        pin: pin.trim() || undefined,
        dryRun: dryRun || undefined,
      });
      setResult(execResult);
    } catch (execError) {
      if (execError instanceof ApiRequestError && execError.status === 409 && command.confirm) {
        setConfirmPrompt(execError.message);
        setAssumeYes(false);
      } else {
        setError(execError instanceof Error ? execError.message : "执行失败");
      }
    } finally {
      setRunning(false);
      setHistoryKey((current) => current + 1);
    }
  }

  if (loadError) {
    return <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">{loadError}</div>;
  }
  if (!view) {
    return <p className="text-sm text-slate-500">加载中…</p>;
  }

  return (
    <div className="space-y-4 text-sm">
      <div className="rounded-xl border border-amber-100 bg-amber-50 p-4">
        <div className="text-xs font-semibold uppercase tracking-[0.18em] text-amber-600">可执行命令（package.json）</div>
        <div className="mt-2 text-xs leading-5 text-amber-800">
          命令白名单来自 manifest，skm 在启动前完成 env / input / confirm 预检。
          {view.setup ? ` 首次执行会自动运行 setup 命令 ${view.setup}（幂等）。` : ""}
          {view.runtimeEnv?.length ? ` Skill 级必需环境变量：${view.runtimeEnv.join("、")}。` : ""}
          {view.source === "catalog" ? " 当前展示 catalog 快照（package.json 无法实时读取）。" : ""}
        </div>
        {view.note ? <div className="mt-2 text-xs text-amber-700">{view.note}</div> : null}
      </div>

      {view.commands.length === 0 ? (
        <div className="rounded-xl border border-slate-200 bg-slate-50 p-4 text-slate-600">该 Skill 没有声明可执行命令。</div>
      ) : (
        <>
          <div className="grid gap-3">
            {view.commands.map((entry) => (
              <button
                key={entry.name}
                type="button"
                onClick={() => {
                  setSelectedCommand(entry.name);
                  resetRunState();
                }}
                className={`cursor-pointer rounded-xl border p-3 text-left transition-colors ${selectedCommand === entry.name ? "border-amber-300 bg-amber-50" : "border-slate-200 bg-white hover:border-slate-300"}`}
              >
                <div className="flex items-center gap-2">
                  <span className="font-mono text-sm font-semibold text-slate-800">{entry.name}</span>
                  {entry.confirm ? <span className="rounded-full bg-red-50 px-2 py-0.5 text-[11px] font-medium text-red-700">confirm</span> : null}
                  {entry.inputVia ? <span className="rounded-full bg-blue-50 px-2 py-0.5 text-[11px] font-medium text-blue-700">input:{entry.inputVia}</span> : null}
                  {entry.timeoutSeconds ? <span className="rounded-full bg-slate-100 px-2 py-0.5 text-[11px] font-medium text-slate-500">{entry.timeoutSeconds}s</span> : null}
                </div>
                {entry.description ? <div className="mt-1 text-xs text-slate-600">{entry.description}</div> : null}
                {entry.env?.length ? <div className="mt-1 text-[11px] text-slate-500">必需 env：{entry.env.join("、")}</div> : null}
              </button>
            ))}
          </div>

          {command ? (
            <div className="rounded-xl border border-slate-200 bg-white p-4">
              <div className="mb-3 flex items-center gap-2 text-sm font-medium text-slate-800">
                <Play className="h-4 w-4 text-amber-600" />
                运行 <span className="font-mono">{command.name}</span>
              </div>
              <div className="space-y-3">
                {command.inputVia ? (
                  <label className="block">
                    <span className="mb-1 block text-xs font-medium text-slate-600">结构化输入（JSON，经 {command.inputVia} 投递，按 schema 校验）</span>
                    <textarea
                      value={inputJson}
                      onChange={(event) => setInputJson(event.target.value)}
                      rows={3}
                      spellCheck={false}
                      placeholder='{"key": "value"}'
                      className="w-full rounded-lg border border-slate-200 bg-slate-50 p-2 font-mono text-xs focus:border-amber-400 focus:outline-none"
                    />
                  </label>
                ) : (
                  <p className="text-xs text-slate-500">该命令未声明结构化输入；位置参数请通过 CLI 的 `-- args` 传递。</p>
                )}
                <label className="block">
                  <span className="mb-1 block text-xs font-medium text-slate-600">环境变量注入（每行 KEY=VAL）</span>
                  <textarea
                    value={envText}
                    onChange={(event) => setEnvText(event.target.value)}
                    rows={2}
                    spellCheck={false}
                    placeholder={"API_KEY=xxx"}
                    className="w-full rounded-lg border border-slate-200 bg-slate-50 p-2 font-mono text-xs focus:border-amber-400 focus:outline-none"
                  />
                </label>
                <label className="block">
                  <span className="mb-1 block text-xs font-medium text-slate-600">版本 pin（可选，source hash 前缀 ≥8 位 hex，留空跑最新）</span>
                  <input
                    value={pin}
                    onChange={(event) => setPin(event.target.value)}
                    spellCheck={false}
                    placeholder={"如 3fa2c1d8（见下方执行历史的 HASH）"}
                    className="w-full rounded-lg border border-slate-200 bg-slate-50 p-2 font-mono text-xs focus:border-amber-400 focus:outline-none"
                  />
                </label>
                <div className="flex flex-wrap items-center gap-4 text-xs text-slate-600">
                  <label className="flex cursor-pointer items-center gap-1.5">
                    <input type="checkbox" checked={isolate} onChange={(event) => setIsolate(event.target.checked)} className="h-3.5 w-3.5 accent-amber-600" />
                    隔离执行（缓存副本，--isolate）
                  </label>
                  {command.confirm ? (
                    <label className="flex cursor-pointer items-center gap-1.5">
                      <input type="checkbox" checked={assumeYes} onChange={(event) => setAssumeYes(event.target.checked)} className="h-3.5 w-3.5 accent-red-600" />
                      我已确认（--yes）
                    </label>
                  ) : null}
                </div>
                {command.confirm ? (
                  <div className="rounded-lg border border-red-100 bg-red-50 px-3 py-2 text-xs text-red-700">⚠️ {command.confirm}</div>
                ) : null}
                {confirmPrompt ? (
                  <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700">{confirmPrompt} 勾选“我已确认”后重试。</div>
                ) : null}
                <div className="flex items-center gap-2">
                  <button
                    type="button"
                    onClick={() => void runCommand(false)}
                    disabled={running}
                    className="cursor-pointer rounded-lg bg-amber-600 px-4 py-1.5 text-xs font-medium text-white transition-colors hover:bg-amber-700 disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    {running ? "执行中…" : "执行"}
                  </button>
                  <button
                    type="button"
                    onClick={() => void runCommand(true)}
                    disabled={running}
                    className="cursor-pointer rounded-lg border border-slate-200 bg-white px-4 py-1.5 text-xs font-medium text-slate-600 transition-colors hover:bg-slate-50 disabled:cursor-not-allowed disabled:opacity-60"
                  >
                    预览（dry-run）
                  </button>
                </div>
                {error ? <div className="rounded-lg border border-red-200 bg-red-50 px-3 py-2 text-xs text-red-700">{error}</div> : null}
                {result ? <ExecResultView result={result} /> : null}
              </div>
            </div>
          ) : null}
          <ExecHistory zid={zid} refreshKey={historyKey} />
        </>
      )}
    </div>
  );
}

function ExecResultView({ result }: { result: SkillExecResult }) {
  const statusText = result.dryRun
    ? "预览（未执行）"
    : result.aborted
      ? `已中止：${result.aborted}`
      : result.ok
        ? "成功"
        : `退出码 ${result.exitCode}${result.timedOut ? "（超时）" : ""}`;
  const statusClass = result.dryRun
    ? "border-slate-200 bg-slate-50 text-slate-600"
    : result.ok
      ? "border-green-200 bg-green-50 text-green-700"
      : "border-red-200 bg-red-50 text-red-700";
  return (
    <div className="space-y-2">
      <div className={`rounded-lg border px-3 py-2 text-xs ${statusClass}`}>
        {statusText} · {result.command} · {result.durationMs}ms
        <div className="mt-1 break-all font-mono text-[11px] opacity-80">{result.workDir}</div>
        {result.deps ? (
          <div className="mt-1 text-[11px] opacity-80">
            deps: {[result.deps.node, result.deps.python].filter(Boolean).join(" + ") || "（无托管安装）"}
            {result.deps.ran ? `（已执行 ${result.deps.durationMs}ms）` : result.deps.skipped ? "（已跳过，安装标记有效）" : ""}
          </div>
        ) : null}
        {result.setup ? (
          <div className="mt-1 text-[11px] opacity-80">
            setup: {result.setup.command}（{result.setup.ran ? `已执行 ${result.setup.durationMs}ms` : result.setup.skipped ? "已跳过（完成标记有效）" : "未执行"}）
          </div>
        ) : null}
        {result.plan && result.dryRun ? (
          <>
            {result.plan.pin ? <div className="mt-1 font-mono text-[11px] opacity-80">pin: {result.plan.pin}</div> : null}
            {result.plan.deps?.length ? (
              <div className="mt-1 text-[11px] opacity-80">deps 计划：{result.plan.deps.join("；")}</div>
            ) : result.plan.depsSkipped ? (
              <div className="mt-1 text-[11px] opacity-80">deps 已最新，将跳过</div>
            ) : null}
            <div className="mt-1 break-all font-mono text-[11px] opacity-80">$ {result.plan.commandLine}</div>
          </>
        ) : null}
      </div>
      {result.stdout ? (
        <div>
          <div className="mb-1 text-[11px] font-medium uppercase tracking-wide text-slate-400">stdout</div>
          <pre className="max-h-64 overflow-auto whitespace-pre-wrap rounded-lg bg-slate-900 p-3 font-mono text-xs leading-5 text-slate-100">{result.stdout}</pre>
        </div>
      ) : null}
      {result.stderr ? (
        <div>
          <div className="mb-1 text-[11px] font-medium uppercase tracking-wide text-slate-400">stderr</div>
          <pre className="max-h-64 overflow-auto whitespace-pre-wrap rounded-lg bg-slate-900 p-3 font-mono text-xs leading-5 text-red-200">{result.stderr}</pre>
        </div>
      ) : null}
    </div>
  );
}

const EXEC_STATUS_BADGES: Record<ExecRecord["status"], { label: string; className: string }> = {
  completed: { label: "成功", className: "bg-green-50 text-green-700" },
  failed: { label: "失败", className: "bg-red-50 text-red-700" },
  timeout: { label: "超时", className: "bg-red-50 text-red-700" },
  setup_failed: { label: "setup 失败", className: "bg-amber-50 text-amber-700" },
  deps_failed: { label: "deps 失败", className: "bg-amber-50 text-amber-700" },
  rejected: { label: "被拒绝", className: "bg-slate-100 text-slate-500" },
};

function ExecHistory({ zid, refreshKey }: { zid: string; refreshKey: number }) {
  const [records, setRecords] = useState<ExecRecord[]>([]);
  const [loading, setLoading] = useState(true);
  const [loadError, setLoadError] = useState("");

  useEffect(() => {
    let active = true;
    setLoading(true);
    setLoadError("");
    api
      .getExecs({ skill: zid, limit: 10 })
      .then((data) => {
        if (active) {
          setRecords(data);
        }
      })
      .catch((historyError) => {
        if (active) {
          setLoadError(historyError instanceof Error ? historyError.message : "加载执行历史失败");
        }
      })
      .finally(() => {
        if (active) {
          setLoading(false);
        }
      });
    return () => {
      active = false;
    };
  }, [zid, refreshKey]);

  function copyHash(hash: string) {
    navigator.clipboard
      .writeText(hash)
      .then(() => toast.success("已复制 source hash，可用作 --pin"))
      .catch(() => toast.error("复制失败"));
  }

  return (
    <div className="rounded-xl border border-slate-200 bg-white p-4">
      <div className="mb-2 flex items-center gap-2 text-sm font-medium text-slate-800">
        <Terminal className="h-4 w-4 text-slate-500" />
        执行历史
        <span className="text-xs font-normal text-slate-400">（HASH 可复制，用作 --pin 重放该版本）</span>
      </div>
      {loadError ? <div className="text-xs text-red-600">{loadError}</div> : null}
      {loading ? <p className="text-xs text-slate-500">加载中…</p> : records.length === 0 ? (
        <p className="text-xs text-slate-500">还没有执行记录。</p>
      ) : (
        <ul className="divide-y divide-slate-100">
          {records.map((record) => {
            const badge = EXEC_STATUS_BADGES[record.status] ?? EXEC_STATUS_BADGES.rejected;
            return (
              <li key={record.zid} className="flex flex-wrap items-center gap-x-3 gap-y-1 py-2 text-xs">
                <span className="text-slate-400">{new Date(record.startedAt).toLocaleString()}</span>
                <span className="font-mono font-medium text-slate-700">{record.command}</span>
                <span className={`rounded-full px-2 py-0.5 text-[11px] font-medium ${badge.className}`}>{badge.label}</span>
                {record.status !== "rejected" ? <span className="text-slate-500">exit {record.exitCode}</span> : null}
                <span className="text-slate-500">{record.durationMs}ms</span>
                <span className="text-slate-400">{record.trigger}</span>
                {record.pin ? <span className="rounded-full bg-blue-50 px-2 py-0.5 font-mono text-[11px] text-blue-700">pin:{record.pin}</span> : null}
                {record.sourceHash ? (
                  <button
                    type="button"
                    onClick={() => copyHash(record.sourceHash ?? "")}
                    title="复制完整 source hash，可用作 --pin"
                    className="flex cursor-pointer items-center gap-1 rounded-md bg-slate-50 px-2 py-0.5 font-mono text-[11px] text-slate-600 transition-colors hover:bg-slate-100"
                  >
                    {record.sourceHash.slice(0, 12)}
                    <Copy className="h-3 w-3" />
                  </button>
                ) : null}
                {record.reason ? <span className="basis-full text-[11px] text-slate-400">{record.reason}</span> : null}
              </li>
            );
          })}
        </ul>
      )}
    </div>
  );
}

function FileTree({ nodes, selectedPath, onSelect, depth = 0 }: { nodes: FileNode[]; selectedPath: string; onSelect: (path: string) => void; depth?: number }) {
  return (
    <ul className="space-y-1">
      {nodes.map((node) => (
        <li key={node.path || node.name}>
          {node.isDir ? (
            <div>
              <div className="flex items-center rounded px-2 py-1 text-sm font-medium text-slate-700 hover:bg-slate-100" style={{ paddingLeft: depth * 16 }}>
                <FolderOpen className="mr-2 h-4 w-4 text-amber-400" />
                {node.name}
              </div>
              <FileTree nodes={node.children ?? []} selectedPath={selectedPath} onSelect={onSelect} depth={depth + 1} />
            </div>
          ) : (
            <button
              type="button"
              onClick={() => onSelect(node.path)}
              className={`flex w-full items-center rounded px-2 py-1 text-left text-sm ${selectedPath === node.path ? "bg-blue-50 font-medium text-blue-700" : "text-slate-600 hover:bg-slate-100"}`}
              style={{ paddingLeft: depth * 16 + 16 }}
            >
              <FileText className="mr-2 h-4 w-4" />
              {node.name}
            </button>
          )}
        </li>
      ))}
    </ul>
  );
}

function findFirstFilePath(nodes: FileNode[]): string | null {
  for (const node of nodes) {
    if (node.isDir) {
      const child = findFirstFilePath(node.children ?? []);
      if (child) {
        return child;
      }
      continue;
    }
    return node.path;
  }
  return null;
}

function findFilePath(nodes: FileNode[], targetPath: string): boolean {
  for (const node of nodes) {
    if (node.isDir) {
      if (findFilePath(node.children ?? [], targetPath)) {
        return true;
      }
      continue;
    }
    if (node.path === targetPath) {
      return true;
    }
  }
  return false;
}

function formatFrontmatter(skill: Skill) {
  const frontmatter = skill.frontmatter ?? {};
  const entries = Object.entries(frontmatter);
  if (entries.length === 0) {
    return `name: ${skill.name}\ncategory: ${skill.category ?? "Uncategorized"}\nsummary: ${skill.summary ?? ""}`;
  }
  return entries.map(([key, value]) => `${key}: ${typeof value === "string" ? value : JSON.stringify(value)}`).join("\n");
}