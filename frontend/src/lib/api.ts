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
  icon?: string;
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
  commands?: SkillCommand[];
  issueCodes: string[];
  conflictKinds: string[];
  isConflict: boolean;
  isEffective: boolean;
  provider?: Provider;
  relation?: SkillRelation;
  relatedSkills?: Skill[];
}

export interface SkillCommand {
  name: string;
  description?: string;
  line: string;
  confirm?: string;
  timeoutSeconds?: number;
  env?: string[];
  inputVia?: string;
  hasInputSchema?: boolean;
}

export interface CommandsView {
  skillZid: string;
  skillName: string;
  skillRoot: string;
  source: string;
  runtimeEnv?: string[];
  setup?: string;
  commands: SkillCommand[];
  note?: string;
}

export interface SkillExecPayload {
  command: string;
  args?: string[];
  input?: string;
  env?: string[];
  assumeYes?: boolean;
  timeoutSeconds?: number;
  isolate?: boolean;
  pin?: string;
  dryRun?: boolean;
}

export interface ExecSetupInfo {
  command: string;
  ran?: boolean;
  skipped?: boolean;
  exitCode?: number;
  timedOut?: boolean;
  durationMs?: number;
  output?: string;
}

export interface ExecDepsInfo {
  node?: string;
  python?: string;
  ran?: boolean;
  skipped?: boolean;
  exitCode?: number;
  timedOut?: boolean;
  durationMs?: number;
  output?: string;
}

export interface ExecPlan {
  workDir: string;
  mode: string;
  sourceDir?: string;
  cacheReused?: boolean;
  materialized?: boolean;
  pin?: string;
  commandLine: string;
  args?: string[];
  inputVia?: string;
  inputBytes?: number;
  envAdditions?: string[];
  timeoutSeconds?: number;
  confirm?: string;
  setup?: string;
  setupSkipped?: boolean;
  deps?: string[];
  depsSkipped?: boolean;
}

export interface SkillExecResult {
  ok: boolean;
  exitCode: number;
  timedOut: boolean;
  dryRun?: boolean;
  aborted?: string;
  skillZid: string;
  skillName: string;
  command: string;
  workDir: string;
  durationMs: number;
  stdout?: string;
  stderr?: string;
  deps?: ExecDepsInfo;
  setup?: ExecSetupInfo;
  plan?: ExecPlan;
}

export interface ExecRecord {
  zid: string;
  skillZid: string;
  skillName: string;
  command: string;
  trigger: string;
  who?: string;
  workDir?: string;
  mode?: string;
  pin?: string;
  sourceHash?: string;
  status: "completed" | "failed" | "timeout" | "setup_failed" | "deps_failed" | "rejected";
  exitCode: number;
  timedOut: boolean;
  reason?: string;
  args?: string[];
  envKeys?: string[];
  inputVia?: string;
  durationMs: number;
  startedAt: string;
}

export interface SkillRelation {
  mode: "from" | "to";
  fromPath?: string;
  directories?: string[];
	include?: string[];
	exclude?: string[];
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

export interface SkillAttachInput {
  targetProviderZid: string;
  mode: "move" | "attach";
}

export interface SkillAttachScanJob {
  providerZid: string;
  job: ScanJob;
}

export interface SkillAttachResult {
  skillZid: string;
  mode: "move" | "attach";
  sourceProvider: Provider;
  targetProvider: Provider;
  sourcePath: string;
  targetPath: string;
  jobs: SkillAttachScanJob[];
}

export interface SkillDeleteResult {
  skillZid: string;
  provider: Provider;
  deletedPath: string;
  deleted: boolean;
  forced: boolean;
  deleteMode: string;
  copyCount?: number;
  job?: ScanJob;
}

export interface SkillDeleteInput {
  force?: boolean;
}

export interface SkillSyncResult {
  skillZid: string;
  provider: Provider;
  sourcePath: string;
  targetPath: string;
  synced: boolean;
  job?: ScanJob;
}

export interface SkillCopySyncEntry {
  skillZid?: string;
  targetPath: string;
  synced: boolean;
}

export interface SkillSyncCopiesResult {
  skillZid: string;
  provider: Provider;
  sourcePath: string;
  copies: SkillCopySyncEntry[];
  synced: boolean;
  scannedProviderZids?: string[];
  job?: ScanJob;
}

export interface SkillRemoveRelationResult {
  skillZid: string;
  provider: Provider;
  rootPath: string;
  removedMode: "from" | "to";
  clearedPaths?: string[];
  scannedProviderZids?: string[];
  job?: ScanJob;
}

export interface DesktopCLIStatus {
  available: boolean;
  installed: boolean;
  sourcePath?: string;
  installedPath: string;
}

export interface DesktopCLIInstallResult {
  sourcePath: string;
  installedPath: string;
  replaced: boolean;
}

export interface RevealInFinderResult {
  path: string;
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
  icon: string;
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
  grouped?: boolean;
}

export interface IssueQuery {
  view?: string;
  provider?: string;
  severity?: string;
  code?: string;
}

const API_BASE_URL = (import.meta.env.VITE_API_BASE_URL ?? "").replace(/\/$/, "");

// ApiRequestError carries the HTTP status so callers can react to specific
// failures (e.g. 409 = exec confirm gate needs user approval).
export class ApiRequestError extends Error {
  status: number;

  constructor(status: number, message: string) {
    super(message);
    this.name = "ApiRequestError";
    this.status = status;
  }
}

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
    throw new ApiRequestError(response.status, payload.message || "Request failed");
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
  deleteSkill: (zid: string, input: SkillDeleteInput = {}) => request<SkillDeleteResult>(`/api/skills/${zid}`, {
    method: "DELETE",
    body: JSON.stringify(input),
  }),
  syncSkill: (zid: string) => request<SkillSyncResult>(`/api/skills/${zid}/sync`, { method: "POST" }),
  syncSkillCopies: (zid: string) => request<SkillSyncCopiesResult>(`/api/skills/${zid}/sync-copies`, { method: "POST" }),
  removeSkillRelation: (zid: string) =>
    request<SkillRemoveRelationResult>(`/api/skills/${zid}/relation/remove`, { method: "POST" }),
  attachSkill: (zid: string, input: SkillAttachInput) =>
    request<SkillAttachResult>(`/api/skills/${zid}/attach`, {
      method: "POST",
      body: JSON.stringify(input),
    }),
  getSkillFiles: (zid: string) => request<FileNode[]>(`/api/skills/${zid}/files`),
  getSkillFileContent: (zid: string, path: string) =>
    request<FileContent>(`/api/skills/${zid}/file-content${toQueryString({ path })}`),
  getSkillCommands: (zid: string) => request<CommandsView>(`/api/skills/${zid}/commands`),
  execSkillCommand: (zid: string, payload: SkillExecPayload) =>
    request<SkillExecResult>(`/api/skills/${zid}/exec`, {
      method: "POST",
      body: JSON.stringify(payload),
    }),
  getExecs: (query: { skill?: string; limit?: number } = {}) =>
    request<ExecRecord[]>(`/api/execs${toQueryString(query)}`),
  getIssues: (query: IssueQuery = {}) =>
    request<ScanIssue[]>(`/api/issues${toQueryString(query)}`),
  getConflicts: () => request<ConflictGroup[]>("/api/conflicts"),
  getScanJobs: () => request<ScanJob[]>("/api/scan-jobs"),
  getDesktopCLIStatus: () => request<DesktopCLIStatus>("/api/desktop/cli"),
  installDesktopCLI: () => request<DesktopCLIInstallResult>("/api/desktop/cli/install", { method: "POST" }),
  revealInFinder: (path: string) =>
    request<RevealInFinderResult>("/api/desktop/reveal", {
      method: "POST",
      body: JSON.stringify({ path }),
    }),
};