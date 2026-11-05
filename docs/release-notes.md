欢迎从哔哩哔哩来到 SKM！感谢大家使用和分享这个本地 Skills 管理工具。

## 下载与使用

- **macOS**：下载 `macos-universal.zip`，解压后把 `SKM.app` 拖到“应用程序”。支持 Apple Silicon 和 Intel。
- **Windows 10/11 x64**：下载 `windows-x64.zip`，完整解压到固定目录，再打开 `SKM.exe`。请保留旁边的 `cli` 目录。
- 首次打开后，在 Providers 中添加你的技能目录，扫描后即可浏览和管理 Skills。
- 数据默认保存在用户目录的 `.skm` 文件夹中，升级时保留此目录即可。

## 首次打开

目前的发布包未使用开发者证书签名或 Apple 公证。macOS 若阻止打开，可在“系统设置 → 隐私与安全性”中确认来源后选择“仍要打开”；Windows 可能显示 SmartScreen 提示。

Windows 需要 Microsoft WebView2 Runtime；缺少时程序会提示运行官方安装引导器，安装过程需要网络。浏览、扫描和管理技能不需要 Git Bash；执行 package.json 中的技能命令时，请安装 Git for Windows，并把包含 `sh.exe` 的 `Git\bin` 目录加入 PATH。技能脚本自身使用的 Node.js、Python 等工具也需要安装。

## 反馈

欢迎在哔哩哔哩评论区分享使用体验，也欢迎到 [GitHub Issues](https://github.com/sichang824/skm/issues) 提交问题。反馈时请注明 SKM 版本、系统版本和复现步骤。

`SHA256SUMS.txt` 提供下载包的 SHA-256 校验值。
