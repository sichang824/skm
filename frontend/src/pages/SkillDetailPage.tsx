import { useEffect, useState } from "react";
import { Link, useParams } from "react-router-dom";
import { Badge } from "../components/ui/badge";
import { Button } from "../components/ui/button";
import { Card, CardContent, CardDescription, CardHeader, CardTitle } from "../components/ui/card";
import { api, type FileNode, type Skill } from "../lib/api";

export function SkillDetailPage() {
  const { zid = "" } = useParams();
  const [skill, setSkill] = useState<Skill | null>(null);
  const [files, setFiles] = useState<FileNode[]>([]);
  const [selectedPath, setSelectedPath] = useState("SKILL.md");
  const [content, setContent] = useState("");
  const [loading, setLoading] = useState(true);
  const [previewError, setPreviewError] = useState("");
  const [error, setError] = useState("");

  useEffect(() => {
    let active = true;
    setLoading(true);
    setError("");

    async function loadSkill() {
      try {
        const [skillData, fileTree] = await Promise.all([
          api.getSkill(zid),
          api.getSkillFiles(zid),
        ]);
        if (!active) {
          return;
        }
        setSkill(skillData);
        setFiles(fileTree);
        setSelectedPath(findFirstFilePath(fileTree) ?? "SKILL.md");
      } catch (loadError) {
        if (!active) {
          return;
        }
        setError(loadError instanceof Error ? loadError.message : "Failed to load skill");
      } finally {
        if (active) {
          setLoading(false);
        }
      }
    }

    if (zid) {
      void loadSkill();
    }
    return () => {
      active = false;
    };
  }, [zid]);

  useEffect(() => {
    let active = true;
    if (!zid || !selectedPath) {
      return () => {
        active = false;
      };
    }

    async function loadContent() {
      try {
        const file = await api.getSkillFileContent(zid, selectedPath);
        if (!active) {
          return;
        }
        setContent(file.content);
        setPreviewError("");
      } catch (loadError) {
        if (!active) {
          return;
        }
        setContent("");
        setPreviewError(loadError instanceof Error ? loadError.message : "Failed to load file");
      }
    }

    void loadContent();
    return () => {
      active = false;
    };
  }, [selectedPath, zid]);

  if (loading) {
    return <div className="mx-auto max-w-7xl px-6 py-8 text-sm text-muted-foreground">加载中…</div>;
  }

  if (error || !skill) {
    return <div className="mx-auto max-w-7xl px-6 py-8 text-sm text-destructive">{error || "Skill not found"}</div>;
  }

  return (
    <div className="mx-auto flex max-w-7xl flex-col gap-6 px-6 py-8">
      <div className="flex flex-wrap items-center justify-between gap-4">
        <div className="space-y-3">
          <Button asChild variant="outline" size="sm">
            <Link to="/skills">返回列表</Link>
          </Button>
          <div className="space-y-2">
            <div className="flex flex-wrap gap-2">
              <Badge variant="outline">{skill.provider?.name ?? "Unknown provider"}</Badge>
              <Badge variant={skill.status === "ready" ? "secondary" : "destructive"}>{skill.status}</Badge>
              {skill.isConflict ? <Badge variant="destructive">{skill.conflictKinds.join(" / ")}</Badge> : null}
            </div>
            <h1 className="text-4xl font-semibold tracking-tight">{skill.name}</h1>
            <p className="max-w-4xl text-sm leading-6 text-muted-foreground">{skill.summary || skill.bodyMarkdown || "暂无摘要"}</p>
          </div>
        </div>
      </div>

      <div className="grid gap-6 lg:grid-cols-[1.2fr_0.8fr]">
        <Card className="border-border/70 bg-white/80">
          <CardHeader>
            <CardTitle>文档预览</CardTitle>
            <CardDescription>{selectedPath || "请选择文件"}</CardDescription>
          </CardHeader>
          <CardContent>
            {previewError ? (
              <div className="rounded-xl border border-destructive/20 bg-destructive/5 px-4 py-3 text-sm text-destructive">
                {previewError}
              </div>
            ) : (
              <pre className="max-h-[720px] overflow-auto rounded-2xl bg-[#1e252a] p-5 font-mono text-sm leading-6 text-[#f5efe4] shadow-inner">
                {content}
              </pre>
            )}
          </CardContent>
        </Card>

        <div className="space-y-6">
          <Card className="border-border/70 bg-white/82">
            <CardHeader>
              <CardTitle>元数据</CardTitle>
            </CardHeader>
            <CardContent className="space-y-3 text-sm text-muted-foreground">
              <MetaRow label="Category" value={skill.category || "-"} />
              <MetaRow label="Directory" value={skill.directoryName} />
              <MetaRow label="Root Path" value={skill.rootPath} />
              <MetaRow label="Scan Time" value={formatDateTime(skill.lastScannedAt)} />
              <MetaRow label="Content Hash" value={skill.contentHash || "-"} />
            </CardContent>
          </Card>

          <Card className="border-border/70 bg-white/82">
            <CardHeader>
              <CardTitle>Tags & Issues</CardTitle>
            </CardHeader>
            <CardContent className="space-y-4">
              <div className="flex flex-wrap gap-2">
                {skill.tags.length > 0 ? skill.tags.map((tag) => <Badge key={tag} variant="outline">#{tag}</Badge>) : <span className="text-sm text-muted-foreground">未声明 tags</span>}
              </div>
              <div className="flex flex-wrap gap-2">
                {skill.issueCodes.length > 0 ? skill.issueCodes.map((code) => <Badge key={code} variant="destructive">{code}</Badge>) : <Badge variant="secondary">No open issues</Badge>}
              </div>
            </CardContent>
          </Card>

          <Card className="border-border/70 bg-white/82">
            <CardHeader>
              <CardTitle>文件树</CardTitle>
              <CardDescription>技能目录递归浏览。只预览文本文件。</CardDescription>
            </CardHeader>
            <CardContent className="space-y-1">
              <FileTree nodes={files} selectedPath={selectedPath} onSelect={setSelectedPath} />
            </CardContent>
          </Card>

          <Card className="border-border/70 bg-white/82">
            <CardHeader>
              <CardTitle>Frontmatter</CardTitle>
            </CardHeader>
            <CardContent>
              <pre className="overflow-auto rounded-2xl bg-secondary/60 p-4 text-xs leading-6 text-foreground">
                {JSON.stringify(skill.frontmatter ?? {}, null, 2)}
              </pre>
            </CardContent>
          </Card>
        </div>
      </div>
    </div>
  );
}

function FileTree({
  nodes,
  selectedPath,
  onSelect,
  depth = 0,
}: {
  nodes: FileNode[];
  selectedPath: string;
  onSelect: (path: string) => void;
  depth?: number;
}) {
  return (
    <div className="space-y-1">
      {nodes.map((node) => (
        <div key={node.path || node.name}>
          {node.isDir ? (
            <div>
              <div className="rounded-lg px-3 py-2 text-xs font-semibold uppercase tracking-[0.18em] text-muted-foreground" style={{ marginLeft: depth * 12 }}>
                {node.name}
              </div>
              <FileTree nodes={node.children ?? []} selectedPath={selectedPath} onSelect={onSelect} depth={depth + 1} />
            </div>
          ) : (
            <button
              type="button"
              onClick={() => onSelect(node.path)}
              className={`w-full rounded-lg px-3 py-2 text-left text-sm transition ${selectedPath === node.path ? "bg-primary text-primary-foreground" : "bg-secondary/40 text-foreground hover:bg-secondary"}`}
              style={{ marginLeft: depth * 12 }}
            >
              {node.path}
            </button>
          )}
        </div>
      ))}
    </div>
  );
}

function findFirstFilePath(nodes: FileNode[]): string | null {
  for (const node of nodes) {
    if (node.isDir) {
      const childPath = findFirstFilePath(node.children ?? []);
      if (childPath) {
        return childPath;
      }
      continue;
    }
    return node.path;
  }
  return null;
}

function MetaRow({ label, value }: { label: string; value: string }) {
  return (
    <div className="grid gap-1 rounded-xl border border-border/60 bg-background/80 px-3 py-2">
      <span className="text-xs uppercase tracking-[0.18em] text-muted-foreground">{label}</span>
      <span className="break-all text-foreground">{value}</span>
    </div>
  );
}

function formatDateTime(value?: string) {
  if (!value) {
    return "未扫描";
  }
  return new Intl.DateTimeFormat("zh-CN", {
    year: "numeric",
    month: "2-digit",
    day: "2-digit",
    hour: "2-digit",
    minute: "2-digit",
  }).format(new Date(value));
}