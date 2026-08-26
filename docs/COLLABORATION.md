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

标题与 commit 同规范。描述使用仓库模板。CI（编译、测试、vet、lint、commitlint）必须绿。

## 4. Review

当前个人仓库 CODEOWNERS 为 `@ts721521`。合并前确认测试与文档（CHANGELOG 用户可见变更必改）。
