# Contributing to SKM

感谢你愿意为 SKM 做贡献。

这个项目目前仍在快速演进，最有价值的贡献通常集中在下面几类：

- 扫描、索引、冲突识别相关的 bug 修复
- Provider、Skill、Issues 相关的交互和可视化改进
- 桌面端体验、构建流程和文档完善
- 测试补充、错误处理和开发体验优化

## Before You Start

在开始较大的改动前，建议先开一个 issue 对齐范围，尤其是这些情况：

- 会改动数据模型或 API 结构
- 会改变扫描规则或 attach 语义
- 会调整信息架构、路由或桌面端行为

如果只是小的 bug fix、文案修正或文档更新，通常可以直接提 PR。

## Local Setup

Prerequisites:

- Go 1.24+
- Node.js 20+
- pnpm 10+

Start the full stack:

```bash
make dev
```

Start with seeded providers:

```bash
make dev/seed
```

Run tests:

```bash
make test
```

Build artifacts:

```bash
make build
make app-build
```

## Contribution Guidelines

- Keep changes focused. Avoid mixing refactors, features, and docs cleanup in one PR.
- Prefer root-cause fixes over one-off patches.
- Preserve existing API and data shape unless the change explicitly requires a breaking update.
- Update documentation when commands, routes, configuration, or user-facing behavior changes.
- Add or update tests when fixing logic bugs or introducing new behavior.

## Pull Request Checklist

Before opening a PR, verify these items:

- The branch builds locally
- Relevant tests pass
- New behavior is documented
- Screenshots are attached for UI changes
- Migration or compatibility risks are described when applicable

## Review Notes

When writing a PR description, include:

- What changed
- Why it changed
- How to validate it
- Any follow-up work or known limitations

This makes review much faster and helps keep architectural decisions traceable.

## Code of Collaboration

- Be direct and specific in technical discussion
- Prefer reproducible examples over vague descriptions
- Optimize for maintainability, not just local success

## Thanks

Every issue report, doc fix, test case, and code change helps make SKM more useful. Thank you for spending time on it.
