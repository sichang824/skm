import { useEffect, useMemo, useState } from "react";
import { Copy, FileText, FolderOpen, Trash2, X } from "lucide-react";
import { toast } from "sonner";
import { Dialog, DialogContent, DialogDescription, DialogFooter, DialogHeader, DialogTitle } from "../ui/dialog";
import { api, type FileNode, type Skill } from "../../lib/api";

type SkillDetailDialogProps = {
  zid: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
  onDeleted?: () => void;
};

export function SkillDetailDialog({ zid, open, onOpenChange, onDeleted }: SkillDetailDialogProps) {
  const [skill, setSkill] = useState<Skill | null>(null);
  const [files, setFiles] = useState<FileNode[]>([]);
  const [selectedPath, setSelectedPath] = useState("SKILL.md");
  const [content, setContent] = useState("");
  const [loading, setLoading] = useState(false);
  const [previewError, setPreviewError] = useState("");
  const [deleteDialogOpen, setDeleteDialogOpen] = useState(false);
  const [deleting, setDeleting] = useState(false);

  useEffect(() => {
    let active = true;
    if (!open || !zid) {
      return () => {
        active = false;
      };
    }
    setLoading(true);
    const skillZid = zid;

    async function load() {
      try {
        const [skillData, fileTree] = await Promise.all([
          api.getSkill(skillZid),
          api.getSkillFiles(skillZid),
        ]);
        if (!active) {
          return;
        }
        setSkill(skillData);
        setFiles(fileTree);
        setSelectedPath(findFirstFilePath(fileTree) ?? "SKILL.md");
      } catch (error) {
        if (!active) {
          return;
        }
        toast.error(error instanceof Error ? error.message : "加载技能详情失败");
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    void load();
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

  async function handleDeleteSkill() {
    if (!zid || !skill) {
      return;
    }
    setDeleting(true);
    try {
      await api.deleteSkill(zid);
      toast.success(`${skill.name} 已删除`);
      setDeleteDialogOpen(false);
      onDeleted?.();
      onOpenChange(false);
    } catch (error) {
      toast.error(error instanceof Error ? error.message : "删除 Skill 失败");
    } finally {
      setDeleting(false);
    }
  }

  if (!open || !zid) {
    return null;
  }

  const previewPath = selectedPath || "SKILL.md";
  const isSkillMarkdown = /(^|\/)SKILL\.md$/i.test(previewPath);
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
            <span className="flex items-center gap-1"><FileText className="h-3.5 w-3.5 text-blue-500" /> {previewPath}</span>
            <span className="text-slate-300">|</span>
            <span className={issueBadgeClass}>✓ {issueBadge}</span>
          </div>

          <div className="flex-1 overflow-y-auto p-6">
            {previewError ? <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">{previewError}</div> : null}
            {loading || !skill ? <p className="text-sm text-slate-500">加载中…</p> : null}
            {!loading && skill ? (
              isSkillMarkdown ? (
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
                该操作会直接删除 Skill 目录。
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
              {deleting ? "删除中…" : "确认删除目录"}
            </button>
          </DialogFooter>
        </DialogContent>
      </Dialog>
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

function formatFrontmatter(skill: Skill) {
  const frontmatter = skill.frontmatter ?? {};
  const entries = Object.entries(frontmatter);
  if (entries.length === 0) {
    return `name: ${skill.name}\ncategory: ${skill.category ?? "Uncategorized"}\nsummary: ${skill.summary ?? ""}`;
  }
  return entries.map(([key, value]) => `${key}: ${typeof value === "string" ? value : JSON.stringify(value)}`).join("\n");
}