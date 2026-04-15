# SKM Product Requirements Document

## 1. 文档信息

- 产品名称：SKM（Skills Manager）
- 文档版本：v0.1
- 文档日期：2026-04-15
- 文档状态：Draft

## 2. 产品概述

SKM 是一个面向本地技能目录的统一管理系统，用于扫描、收集、解析、展示和管理多个 Provider 下的 skills。产品的核心目标不是替代文件系统，而是把分散在不同目录中的 skill 定义，提升为可搜索、可分类、可校验、可比较、可追踪的产品对象。

当前技能定义以 `SKILL.md` 为核心文件。一个 skill 通常由一个目录组成，目录中至少包含 `SKILL.md`，并可能包含 `assets`、`references`、`scripts`、`bin` 等辅助文件或子目录。

SKM 以本地只读管理为主，聚焦于扫描、展示、归类、异常识别和 Provider 管理，不以在线编辑或自动修改文件系统为目标。

## 3. 背景与问题

随着 skills 数量增加，用户会同时维护来自多个来源的 skill 目录，例如：

- 工作区内 skills 目录
- `~/.agents/skills`
- Cursor 对应目录
- Codex 对应目录
- Workbuddy 对应目录

现状问题包括：

- skills 分散在多个物理目录，缺乏统一视图
- 用户难以确认某个 skill 来自哪个 Provider
- 同名 skill 可能出现在多个 Provider 中，难以识别冲突
- 部分目录不规范，缺失 `SKILL.md` 或 frontmatter 不合法
- 用户无法快速查看 skill 的完整目录结构与关键文件
- Provider 的增删和路径调整缺乏统一管理入口

## 4. 产品目标

### 4.1 业务目标

1. 建立多 Provider 的统一 skill 注册表。
2. 提供面向本地目录的技能扫描与索引能力。
3. 提供 skill 基本信息、`SKILL.md` 内容和目录树的可视化查看能力。
4. 支持 Provider 的增加、减少、启用、禁用和重扫。
5. 支持重复、冲突、异常 skill 的识别与提示。

### 4.2 用户目标

1. 快速知道系统当前一共有哪些 skills。
2. 快速知道每个 skill 来自哪个 Provider、位于哪个目录。
3. 快速查看 skill 的 `SKILL.md` 和目录结构。
4. 快速发现重复 skill、异常目录和扫描问题。
5. 能按 Provider、分类、标签、状态进行筛选和搜索。

## 5. 非目标

以下内容不纳入本次开发范围：

1. 直接在产品内编辑或重命名 skill 文件。
2. 自动创建或自动修复 `SKILL.md`。
3. 云端同步、多用户协作、权限系统。
4. 执行 skill、运行 agent 或触发外部工作流。
5. 发布 marketplace 或远程仓库同步能力。

## 6. 目标用户

### 6.1 主要用户

- 维护本地 skills 目录的个人开发者
- 维护多个 AI 工具或 agent skill 集合的高级用户
- 需要统一管理工作区 skill 与全局 skill 的工程用户

### 6.2 次要用户

- 需要检查 skill 结构是否规范的维护者
- 希望整理多来源 skills 并建立目录规范的团队成员

## 7. 核心概念

### 7.1 Skill

Skill 是一个目录级对象，通常以 `SKILL.md` 为入口文件。SKM 仅将包含 `SKILL.md` 的目录认定为有效 skill。

### 7.2 Provider

Provider 是 skill 根目录的配置实体。每个 Provider 对应一个物理目录，系统从该目录扫描 skills。

典型 Provider 包括：

- `global`：例如 `~/.agents/skills`
- `workspace`：当前工作区中的 skills 根目录
- `cursor`
- `codex`
- `workbuddy`

Provider 在产品中是可配置对象，而不是硬编码枚举。系统可内置默认 Provider 模板，但路径应允许用户调整。

### 7.3 分类

分类是产品层面的业务归类，用于帮助用户管理 skills。分类不等同于物理目录路径。

### 7.4 路径

路径是 skill 在文件系统中的物理位置，包括：

- Provider 根目录
- skill 根目录
- `SKILL.md` 路径
- 目录内其他文件的相对路径

### 7.5 扫描

扫描是系统从 Provider 根目录发现 skill、解析 `SKILL.md`、更新索引并产出结果的过程。

## 8. 用户场景

### 8.1 浏览现有 skills

用户打开 SKM 后，希望看到所有已发现的 skills 列表，并能够按 Provider、分类、标签、状态快速筛选。

### 8.2 查看 skill 详情

用户点击某个 skill，希望查看：

- skill 基本信息
- `SKILL.md` 内容
- frontmatter 解析结果
- 目录树和关键文件
- 来源 Provider 和物理路径

### 8.3 新增 Provider

用户希望配置一个新的 Provider 根目录，例如某个 Cursor 或 Codex 目录，并验证路径是否可读、是否包含可扫描的 skills。

### 8.4 删除或停用 Provider

用户希望临时停用某个 Provider，或永久删除其配置，并让对应 skills 从当前索引中移除或标记为不可用。

### 8.5 扫描目录变化

用户新增或删除 skill 目录后，希望重新扫描，并看到新增、删除、变化、异常和冲突项。

### 8.6 识别重复与冲突

用户发现多个 Provider 中存在同名 skills，希望系统提示冲突来源，并显示当前优先级与生效对象。

## 9. 功能需求

### 9.1 Provider 管理

#### 9.1.1 Provider 列表

系统应提供 Provider 列表页，展示：

- Provider 名称
- 类型
- 根目录路径
- 是否启用
- 优先级
- 最近扫描时间
- 最近扫描状态
- 扫描结果摘要

#### 9.1.2 新增 Provider

用户应能够新增 Provider，至少填写：

- 名称
- 类型
- 根目录路径
- 是否启用
- 优先级

系统在保存前应校验：

- 路径是否存在
- 路径是否可读
- 路径是否与已有 Provider 重复或冲突
- 目录下是否存在潜在 skills

#### 9.1.3 编辑 Provider

用户应能够修改 Provider 的名称、路径、启用状态和优先级。

#### 9.1.4 删除 Provider

用户应能够删除 Provider。删除后，对应 skills 应从索引中移除，或在历史记录中标记来源已删除。

#### 9.1.5 启用与停用 Provider

用户应能够启用或停用 Provider。停用后，该 Provider 不参与扫描和生效计算。

### 9.2 Skill 扫描

#### 9.2.1 扫描入口

系统应支持：

- 扫描全部 Provider
- 扫描单个 Provider
- 手动重新扫描

#### 9.2.2 扫描规则

系统扫描时应遵循以下规则：

1. 只将包含 `SKILL.md` 的目录识别为 skill。
2. 忽略 `.git`、`.DS_Store`、`node_modules` 等非业务目录。
3. 默认扫描 Provider 根目录下的一级子目录，也允许后续扩展为递归策略。
4. 对 `SKILL.md` 执行 frontmatter 与 Markdown 内容解析。

#### 9.2.3 扫描结果

每次扫描应产出：

- 新增 skill 数
- 删除 skill 数
- 变更 skill 数
- 异常 skill 数
- 冲突 skill 数
- 扫描日志
- 扫描耗时

#### 9.2.4 自动扫描

系统应支持：

- 应用启动时自动扫描
- 用户手动触发扫描
- 定时扫描
- 文件系统变更监听

### 9.3 Skill 列表与管理

#### 9.3.1 Skill 列表

系统应提供统一的 skill 列表页，展示：

- skill 名称
- Provider
- 分类
- 标签
- 状态
- skill 根目录
- 最后扫描时间
- 是否存在冲突

#### 9.3.2 搜索与筛选

系统应支持：

- 按 skill 名称搜索
- 按 Provider 筛选
- 按分类筛选
- 按标签筛选
- 按状态筛选
- 按是否存在冲突筛选

#### 9.3.3 排序

系统应支持按以下字段排序：

- 名称
- 最近扫描时间
- Provider
- 状态

### 9.4 Skill 详情页

#### 9.4.1 基础信息

详情页应展示：

- skill 名称
- summary 或 description
- Provider
- skill 根目录路径
- `SKILL.md` 路径
- 分类
- 标签
- 状态
- 更新时间

#### 9.4.2 `SKILL.md` 查看

详情页应提供：

- 原始 Markdown 预览
- 渲染后的内容预览
- frontmatter 字段解析结果

#### 9.4.3 文件目录树

详情页应展示 skill 根目录下的文件树，至少支持：

- 展示目录和文件
- 展示相对路径
- 展示常见文件类型
- 点击查看文本文件内容

系统只要求支持文本文件内容预览，不要求支持二进制文件预览。

#### 9.4.4 原路径跳转

详情页应支持：

- 打开 skill 目录
- 打开 `SKILL.md`
- 复制路径

### 9.5 分类与标签

#### 9.5.1 分类模型

系统应支持为 skill 维护分类。分类不依赖物理路径，可由用户手动指定。

#### 9.5.2 标签模型

系统应支持为 skill 维护多个标签，用于补充检索与管理。

#### 9.5.3 自动分类

系统不要求从路径自动生成分类，可预留目录推断或规则归类能力。

### 9.6 异常识别与质量校验

系统应识别并展示以下异常：

- 目录存在但缺失 `SKILL.md`
- `SKILL.md` frontmatter 无法解析
- `SKILL.md` 缺失 `name`
- 目录名与 `name` 不一致
- skill 目录为空或结构异常
- 文件读取失败

系统应在扫描结果与异常列表中展示异常详情。

### 9.7 重复与冲突检测

系统应识别以下冲突：

- 同名 skill 在多个 Provider 中重复出现
- 同一路径被多个 Provider 重复指向
- 同名 skill 内容不一致

系统应展示：

- 冲突组
- 冲突来源
- Provider 优先级
- 当前生效 skill

### 9.8 扫描历史与日志

系统应保留扫描历史记录，至少包括：

- 开始时间
- 结束时间
- 扫描耗时
- Provider
- 扫描状态
- 新增、删除、变更、异常数量
- 关键日志

## 10. 数据模型建议

### 10.1 Provider

- `id`
- `name`
- `type`
- `rootPath`
- `enabled`
- `priority`
- `scanMode`
- `description`
- `lastScannedAt`
- `lastScanStatus`

### 10.2 Skill

- `id`
- `name`
- `slug`
- `providerId`
- `rootPath`
- `skillMdPath`
- `categoryId`
- `summary`
- `status`
- `contentHash`
- `lastModifiedAt`
- `lastScannedAt`

### 10.3 SkillTag

- `skillId`
- `tag`

### 10.4 SkillFile

- `id`
- `skillId`
- `relativePath`
- `fileType`
- `size`
- `modifiedAt`

### 10.5 ScanJob

- `id`
- `providerId`
- `startedAt`
- `finishedAt`
- `status`
- `addedCount`
- `removedCount`
- `changedCount`
- `invalidCount`
- `conflictCount`
- `log`

## 11. 页面建议

### 11.1 Dashboard

展示：

- skills 总数
- Providers 总数
- 冲突数
- 异常数
- 最近扫描结果

### 11.2 Skills 列表页

展示统一列表、搜索、筛选、排序和状态视图。

### 11.3 Skill 详情页

展示 skill 基础信息、`SKILL.md`、目录树、异常信息和冲突信息。

### 11.4 Providers 页面

展示 Provider 列表、增删改、启用停用、路径校验和单独扫描。

### 11.5 Scan Results / Issues 页面

展示最近扫描结果、异常目录、解析失败项、重复 skill 和冲突组。

## 12. 非功能需求

### 12.1 性能

- 支持数百个 skills 的扫描和列表展示
- 单次扫描失败不应阻断整体流程

### 12.2 稳定性

- 单个 `SKILL.md` 解析失败不应导致全量扫描失败
- 文件权限异常应被捕获并展示为错误项

### 12.3 兼容性

- 优先支持 macOS
- 后续考虑 Linux 和 Windows 路径兼容

### 12.4 安全与隐私

- 默认只读取本地目录，不上传文件内容
- 不在未经确认的前提下修改用户文件

### 12.5 可扩展性

- Provider 类型应可扩展
- 扫描策略应可扩展
- 校验规则应可扩展

## 13. 开发范围

### 13.1 本次开发包含

1. Provider 列表与增删改启停
2. 全量扫描与单 Provider 扫描
3. Skill 列表、搜索、筛选、排序
4. Skill 详情页：基础信息、`SKILL.md`、文件树
5. 异常识别与冲突识别
6. 扫描历史与结果展示
7. 打开原目录与打开 `SKILL.md`

### 13.2 本次开发不包含

1. 在线编辑 `SKILL.md`
2. 自动修复异常 skill
3. 文件系统实时监听
4. 多用户协作
5. 云端同步

## 14. 验收标准

### 14.1 Provider 管理

1. 用户可以新增一个 Provider 并成功通过路径校验。
2. 用户可以停用一个 Provider，停用后相关 skill 不再参与当前视图。
3. 用户可以删除一个 Provider，对应 skill 从当前索引中消失。

### 14.2 Skill 扫描

1. 系统能够识别包含 `SKILL.md` 的目录。
2. 系统能够产出新增、删除、变更、异常、冲突统计。
3. 单个 skill 解析失败时，其它 skills 仍能正常完成扫描。

### 14.3 Skill 查看

1. 用户可以在列表中搜索并筛选 skills。
2. 用户可以查看某个 skill 的 `SKILL.md` 和目录树。
3. 用户可以识别 skill 来源 Provider 和物理路径。

### 14.4 冲突与异常

1. 系统能够识别同名 skill 的重复来源。
2. 系统能够展示缺失 `SKILL.md` 和 frontmatter 解析失败等异常。

## 15. 待确认问题

1. 是否允许在 SKM 中直接编辑 `SKILL.md`。
2. 分类是否全部手动维护，还是允许从目录路径推断。
3. 同名 skill 出现在多个 Provider 时，生效规则是否完全由优先级决定。
4. Provider 是否允许指向任意目录，还是仅允许指定已知 Provider 类型。
5. skill 扫描是否默认一级目录，还是需要支持多层递归策略。
6. `SKILL.md` 的 frontmatter schema 是否需要强校验。
7. `workspace` 是否应作为默认内置 Provider。

## 16. 可选扩展方向

1. 增加实时文件变更监听。
2. 增加规则引擎与自动校验修复建议。
3. 增加在线编辑与保存能力。
4. 增加 skill 导入导出。
5. 增加团队协作、权限与共享视图。
