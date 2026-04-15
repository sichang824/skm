export interface ApiResponse<T> {
  code: number;
  message: string;
  data: T;
}

export interface DashboardSummary {
  providerCount: number;
  enabledProviderCount: number;
  skillCount: number;
  conflictCount: number;
  issueCount: number;
  recentScanCount: number;
}

export interface Provider {
  zid: string;
  name: string;
  type: string;
  rootPath: string;
  enabled: boolean;
  priority: number;
  scanMode: string;
  description?: string;
  lastScannedAt?: string;
  lastScanStatus: string;
  lastScanSummary?: string;
}

export interface Skill {
  zid: string;
  name: string;
  slug: string;
  directoryName: string;
  rootPath: string;
  skillMdPath?: string;
  category?: string;
  tags: string[];
  summary?: string;
  status: string;
  contentHash?: string;
  lastModifiedAt?: string;
  lastScannedAt: string;
  rawMarkdown?: string;
  bodyMarkdown?: string;
  frontmatter?: Record<string, unknown>;
  issueCodes: string[];
  conflictKinds: string[];
  isConflict: boolean;
  isEffective: boolean;
  provider?: Provider;
}

export interface FileNode {
  name: string;
  path: string;
  isDir: boolean;
  size?: number;
  modifiedAt?: string;
  children?: FileNode[];
}

export interface FileContent {
  path: string;
  content: string;
}

export interface ScanIssue {
  zid: string;
  code: string;
  severity: string;
  message: string;
  rootPath: string;
  relativePath?: string;
  createdAt: string;
  details?: Record<string, unknown>;
  provider?: Provider;
  skill?: Skill;
}

export interface ScanJob {
  zid: string;
  status: string;
  startedAt: string;
  finishedAt?: string;
  addedCount: number;
  removedCount: number;
  changedCount: number;
  invalidCount: number;
  conflictCount: number;
  logLines: string[];
  provider?: Provider;
}

export interface ScanRunResult {
  jobs: ScanJob[];
}

export interface ConflictGroup {
  kind: string;
  key: string;
  effectiveSkillZid?: string;
  skills: Skill[];
}

export interface ProviderInput {
  name: string;
  type: string;
  rootPath: string;
  enabled: boolean;
  priority: number;
  scanMode: string;
  description: string;
}

export interface SkillQuery {
  q?: string;
  provider?: string;
  status?: string;
  sort?: string;
  conflict?: boolean;
}

export interface IssueQuery {
  view?: string;
  provider?: string;
  severity?: string;
  code?: string;
}

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? "").replace(/\/$/, "");

async function request<T>(path: string, init?: RequestInit): Promise<T> {
  const response = await fetch(`${API_BASE_URL}${path}`, {
    headers: {
      "Content-Type": "application/json",
      ...(init?.headers ?? {}),
    },
    ...init,
  });

  const payload = (await response.json()) as ApiResponse<T>;
  if (!response.ok || payload.code !== 0) {
    throw new Error(payload.message || "Request failed");
  }
  return payload.data;
}

function toQueryString<T extends object>(params: T) {
  const query = new URLSearchParams();
  for (const [key, value] of Object.entries(params)) {
    if (value === undefined || value === "") {
      continue;
    }
    query.set(key, String(value));
  }
  const encoded = query.toString();
  return encoded ? `?${encoded}` : "";
}

export const api = {
  getDashboard: () => request<DashboardSummary>("/api/dashboard"),
  getProviders: () => request<Provider[]>("/api/providers"),
  getProvider: (zid: string) => request<Provider>(`/api/providers/${zid}`),
  createProvider: (input: ProviderInput) =>
    request<Provider>("/api/providers", {
      method: "POST",
      body: JSON.stringify(input),
    }),
  updateProvider: (zid: string, input: ProviderInput) =>
    request<Provider>(`/api/providers/${zid}`, {
      method: "PUT",
      body: JSON.stringify(input),
    }),
  deleteProvider: (zid: string) =>
    request<{ deleted: boolean }>(`/api/providers/${zid}`, {
      method: "DELETE",
    }),
  scanProvider: (zid: string) =>
    request<ScanJob>(`/api/providers/${zid}/scan`, { method: "POST" }),
  scanAll: () => request<ScanRunResult>("/api/scan", { method: "POST" }),
  getSkills: (query: SkillQuery = {}) =>
    request<Skill[]>(`/api/skills${toQueryString(query)}`),
  getSkill: (zid: string) => request<Skill>(`/api/skills/${zid}`),
  getSkillFiles: (zid: string) => request<FileNode[]>(`/api/skills/${zid}/files`),
  getSkillFileContent: (zid: string, path: string) =>
    request<FileContent>(`/api/skills/${zid}/file-content${toQueryString({ path })}`),
  getIssues: (query: IssueQuery = {}) =>
    request<ScanIssue[]>(`/api/issues${toQueryString(query)}`),
  getConflicts: () => request<ConflictGroup[]>("/api/conflicts"),
  getScanJobs: () => request<ScanJob[]>("/api/scan-jobs"),
};