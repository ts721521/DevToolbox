# 协作流程

## 1. 分支

```
main                 始终可发布
└── feat/<desc>      新功能
└── fix/<desc>       修复
└── docs/<desc>      文档
└── chore/<desc>     CI / 依赖
└── release/vX.Y.Z   发版准备
```

不要在 `main` 上直接提交功能（仓库初始化与紧急热修除外）。

## 2. Conventional Commits

```
<type>(<scope>): <subject>
```

类型：`feat` `fix` `docs` `style` `refactor` `test` `chore` `perf` `ci` `build` `revert`

- `feat` → MINOR
- `fix` / `perf` → PATCH
- 破坏性变更在 footer 写 `BREAKING CHANGE:` → MAJOR

subject 可用中文，不加句号，说「为什么」而不是「改了文件」。

## 3. Pull Request

标题与 commit 同规范。描述使用仓库模板。CI 必须绿（汇总检查名 **CI**）。

合入 `main`：

- 禁止直接 push；GitHub ruleset 强制走 PR，且只允许 squash / rebase。
- 用户在对话里明确同意后（「合并」「同意合并」「merge」「可以合」），**任何 AI 都必须执行合并**，不要让用户去网页点 Merge：
  1. 确认已有指向 `main` 的 PR（没有则先开）。
  2. 单 commit：`gh pr merge --squash --auto`
  3. 多 commit：`gh pr merge --rebase --auto`
  4. CI 已绿会立刻合；未绿则由 GitHub auto-merge 在检查通过后合。
- 未获同意：禁止 merge、禁止 `--auto`。
- 禁止 `git merge --no-ff` / `gh pr merge --merge`。

## 4. Review

CODEOWNERS 为 `@ts721521`（提示，**不**作为合并门禁）。合并前确认测试与文档（CHANGELOG 用户可见变更必改）。GitHub **不**要求 Approve，同意发生在对话里。
