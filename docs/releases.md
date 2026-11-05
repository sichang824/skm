# 发布维护说明

版本由 RustyTag 管理，Git 标签是唯一版本来源。CI 不会自行递增版本或创建标签；构建时通过 Go 链接参数注入标签，并更新构建用的 Wails 应用版本信息。

## 发布一个版本

1. 确认所有改动已经提交，并且 main 的 CI 全部通过。
2. 在 main 上运行 `rustytag show --json` 检查当前版本。
3. 用 `rustytag patch`、`rustytag minor` 或 `rustytag major` 创建新版本。首次公开发布用 `rustytag patch --version v0.1.0`。RustyTag 会生成 CHANGELOG、发布提交和 annotated tag。
4. 检查 `git status --short` 与 `git tag --points-at HEAD`，然后 `git push --follow-tags origin main`。
5. 在 Actions 中等待测试、macOS 和 Windows 构建通过。流水线先创建草稿，上传两个平台的 ZIP 和校验文件后再公开 Release，避免出现缺少平台安装包的发布。

RustyTag 仓库配置保存 `version_prefix` 和工具维护的版本元数据。不要将 GitHub Token 写入仓库；CI 通过内置的 `GITHUB_TOKEN` 发布。发布失败时可修复后创建新版本；网络等暂时问题可以重新运行原工作流，附件上传支持覆盖重试。

## 流水线

- `test`：Linux 上运行完整后端测试和前端测试。前端依赖使用锁文件安装。
- `build`：macOS Intel runner 上编译通用版（主程序和内置 CLI 均包含 amd64、arm64）；Windows runner 上启用 MinGW / CGO，编译桌面程序和 CLI。Windows 原生测试覆盖 SQLite 模型、ZID、文件锁、CLI 安装与超时进程树清理；完整的 POSIX 脚本夹具测试在 Linux 上运行。
- `release`：仅 `v*` 标签触发。只有前面的任务全部成功才上传附件并发布。非标签构建上传 Actions Artifacts，不创建 Release。

macOS 采用临时签名以保持包完整性，不包含 Apple Developer ID 签名或公证；Windows 程序未签名。若后续接入正式签名证书，可在打包步骤扩展，首次启动说明应同步更新。

## 本地构建

需要 Go 1.24+、Node.js 22.12+、pnpm 10.32.0，以及 Wails v2.12.0 对应平台工具链。

macOS：

```bash
make app-build VERSION="$(rustytag show --quiet)"
```

Windows（PowerShell，需 MinGW GCC 和 Git Bash 的 sh.exe 在 PATH）：

```powershell
$env:GOWORK = 'off'
$env:CGO_ENABLED = '1'
$version = rustytag show --quiet
go install github.com/wailsapp/wails/v2/cmd/wails@v2.12.0
wails build -platform windows/amd64 -webview2 embed -trimpath -ldflags "-X backend-go/app.Version=$version"
./scripts/package-windows.ps1 -Version $version
```

产物在 `dist/`。Windows 包保留 `SKM.exe` 与 `cli/skm.exe` 两个路径，避免 Windows 文件名大小写不敏感导致主程序与 CLI 相互覆盖。macOS CLI 位于 `SKM.app/Contents/Resources/skm`。

[Wails v2 Windows 运行时说明](https://v2.wails.io/docs/guides/windows/) · [Wails CLI 构建参数](https://v2.wails.io/docs/reference/cli/)
