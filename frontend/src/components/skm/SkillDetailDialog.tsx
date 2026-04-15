import { useEffect, useMemo, useState } from "react";
import { Copy, ExternalLink, FileText, FolderOpen, X } from "lucide-react";
import { toast } from "sonner";
import { Badge } from "../ui/badge";
import { Button } from "../ui/button";
import { Dialog, DialogContent } from "../ui/dialog";
import { api, type FileNode, type Skill } from "../../lib/api";

type SkillDetailDialogProps = {
  zid: string | null;
  open: boolean;
  onOpenChange: (open: boolean) => void;
};

export function SkillDetailDialog({ zid, open, onOpenChange }: SkillDetailDialogProps) {
  const [skill, setSkill] = useState<Skill | null>(null);
  const [files, setFiles] = useState<FileNode[]>([]);
  const [selectedPath, setSelectedPath] = useState("SKILL.md");
  const [content, setContent] = useState("");
  const [activeTab, setActiveTab] = useState<"rendered" | "raw" | "frontmatter">("rendered");
  const [loading, setLoading] = useState(false);
  const [previewError, setPreviewError] = useState("");

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

  return (
    <Dialog open={open} onOpenChange={onOpenChange}>
      <DialogContent className="h-[85vh] max-w-6xl overflow-hidden rounded-2xl border-0 bg-white p-0 shadow-2xl" showCloseButton={false}>
        <div className="flex h-full flex-col overflow-hidden">
          <div className="flex items-center justify-between border-b border-slate-200 bg-slate-50 px-6 py-4">
            <div className="flex min-w-0 items-center">
              <div className="mr-4 flex h-10 w-10 items-center justify-center rounded-lg bg-indigo-100 text-indigo-600">
                <FileText className="h-5 w-5" />
              </div>
              <div className="min-w-0">
                <h2 className="truncate text-lg font-bold text-slate-900">{skill?.name ?? "Skill Detail"}</h2>
                <p className="truncate text-xs text-slate-500">{skill?.rootPath ?? "加载中..."}</p>
              </div>
            </div>
            <button onClick={() => onOpenChange(false)} className="rounded-full bg-slate-200/60 p-2 text-slate-400 transition-colors hover:bg-slate-200 hover:text-slate-600">
              <X className="h-4 w-4" />
            </button>
          </div>

          <div className="flex min-h-0 flex-1 overflow-hidden">
            <div className="flex w-[34%] min-w-[320px] flex-col overflow-y-auto border-r border-slate-200 bg-slate-50/60">
              <div className="space-y-4 border-b border-slate-200 p-5">
                <div>
                  <label className="mb-1 block text-xs font-semibold uppercase tracking-wider text-slate-500">Provider 来源</label>
                  <div className="text-sm font-medium text-slate-800">{skill?.provider?.name ?? "Unknown provider"}</div>
                </div>
                <div>
                  <label className="mb-1 block text-xs font-semibold uppercase tracking-wider text-slate-500">分类 & 标签</label>
                  <div className="mt-1 flex flex-wrap gap-2">
                    {skill?.category ? <span className="rounded bg-slate-200 px-2 py-0.5 text-xs text-slate-700">{skill.category}</span> : null}
                    {skill?.tags.map((tag) => <span key={tag} className="rounded border border-blue-100 bg-blue-50 px-2 py-0.5 text-xs text-blue-600">{tag}</span>)}
                  </div>
                </div>
                <div>
                  <label className="mb-1 block text-xs font-semibold uppercase tracking-wider text-slate-500">状态</label>
                  <div className="text-sm">
                    {skill?.status === "ready" && !skill.isConflict ? <span className="font-medium text-green-600">结构规范</span> : null}
                    {skill?.isConflict ? <span className="font-medium text-yellow-600">存在冲突</span> : null}
                    {skill?.status === "invalid" ? <span className="font-medium text-red-600">解析异常</span> : null}
                  </div>
                </div>
                <div className="flex flex-wrap gap-2">
                  <Button variant="outline" size="sm" onClick={() => skill ? void copyText(skill.rootPath) : undefined}>
                    <Copy className="h-4 w-4" />复制路径
                  </Button>
                  <Button variant="outline" size="sm" onClick={() => skill ? void copyText(skill.skillMdPath ?? `${skill.rootPath}/SKILL.md`) : undefined}>
                    <ExternalLink className="h-4 w-4" />复制 SKILL.md
                  </Button>
                </div>
              </div>

              <div className="flex-1 p-5">
                <h3 className="mb-3 flex items-center justify-between text-sm font-semibold text-slate-800">
                  目录结构
                  <button className="text-xs font-normal text-indigo-600 hover:underline" onClick={() => skill ? void copyText(skill.rootPath) : undefined}>打开目录</button>
                </h3>
                <div className="rounded-lg border border-slate-200 bg-white p-3 font-mono text-sm text-slate-600 shadow-inner">
                  {loading ? <p className="text-xs text-slate-500">加载中…</p> : <FileTree nodes={files} selectedPath={selectedPath} onSelect={setSelectedPath} />}
                </div>
              </div>
            </div>

            <div className="flex min-w-0 flex-1 flex-col bg-white">
              <div className="flex border-b border-slate-200 bg-slate-50">
                <TabButton active={activeTab === "rendered"} onClick={() => setActiveTab("rendered")}>预览渲染</TabButton>
                <TabButton active={activeTab === "raw"} onClick={() => setActiveTab("raw")}>原始 Markdown</TabButton>
                <TabButton active={activeTab === "frontmatter"} onClick={() => setActiveTab("frontmatter")}>Frontmatter 解析</TabButton>
              </div>

              <div className="flex-1 overflow-y-auto p-8">
                {previewError ? <div className="rounded-lg border border-red-200 bg-red-50 p-4 text-sm text-red-700">{previewError}</div> : null}
                {loading || !skill ? <p className="text-sm text-slate-500">加载中…</p> : null}
                {!loading && skill && activeTab === "rendered" ? (
                  <div className="prose prose-slate max-w-none">
                    <h1>{skill.name}</h1>
                    <p>{displaySummary}</p>
                    <pre className="overflow-x-auto rounded-lg bg-slate-900 p-4 text-slate-100">{content}</pre>
                    {skill.isConflict ? (
                      <div className="rounded-lg border-l-4 border-yellow-400 bg-yellow-50 p-4 text-sm text-yellow-800">
                        当前 skill 存在冲突，系统会依据 Provider 优先级和最近扫描结果确定生效对象。
                      </div>
                    ) : null}
                  </div>
                ) : null}
                {!loading && skill && activeTab === "raw" ? (
                  <pre className="overflow-x-auto rounded-lg bg-slate-900 p-4 text-sm leading-6 whitespace-pre-wrap text-slate-100">{skill.rawMarkdown || content}</pre>
                ) : null}
                {!loading && skill && activeTab === "frontmatter" ? (
                  <div className="space-y-4">
                    <div className="flex flex-wrap gap-2">
                      {skill.issueCodes.length > 0 ? skill.issueCodes.map((code) => <Badge key={code} variant="destructive">{code}</Badge>) : <Badge variant="secondary">No frontmatter issues</Badge>}
                    </div>
                    <pre className="overflow-x-auto rounded-lg bg-slate-100 p-4 text-sm leading-6 text-slate-700">{JSON.stringify(skill.frontmatter ?? {}, null, 2)}</pre>
                  </div>
                ) : null}
              </div>
            </div>
          </div>
        </div>
      </DialogContent>
    </Dialog>
  );
}

function FileTree({ nodes, selectedPath, onSelect, depth = 0 }: { nodes: FileNode[]; selectedPath: string; onSelect: (path: string) => void; depth?: number }) {
  return (
    <ul className="space-y-1.5">
      {nodes.map((node) => (
        <li key={node.path || node.name}>
          {node.isDir ? (
            <div>
              <div className="flex items-center font-medium text-slate-800" style={{ paddingLeft: depth * 16 }}>
                <FolderOpen className="mr-2 h-4 w-4 text-blue-400" />
                {node.name}
              </div>
              <FileTree nodes={node.children ?? []} selectedPath={selectedPath} onSelect={onSelect} depth={depth + 1} />
            </div>
          ) : (
            <button
              type="button"
              onClick={() => onSelect(node.path)}
              className={`flex w-full items-center rounded px-1 py-1 text-left ${selectedPath === node.path ? "font-medium text-indigo-600" : "text-slate-600 hover:bg-slate-50"}`}
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

function TabButton({ active, onClick, children }: { active: boolean; onClick: () => void; children: string }) {
  return <button onClick={onClick} className={`border-b-2 px-6 py-3 text-sm font-medium ${active ? "border-indigo-600 bg-white text-indigo-600" : "border-transparent text-slate-500 hover:text-slate-700"}`}>{children}</button>;
}