# Windows 后台控制与截图兼容

## 用户诉求

先卸载本地 Codex 的 `mcp_servers.open-computer-use` 直连配置，然后处理 Windows 兼容问题：Windows 被控制的应用或游戏不应必须占用用户主桌面前台，需要支持后台操作和截图。

## 本次改动

- Windows runtime 截图优先尝试 `PrintWindow(PW_RENDERFULLCONTENT)`，用于目标窗口被遮挡但未最小化的后台观察；失败或明显空图时回退 `CopyFromScreen`。
- 新增 `OPEN_COMPUTER_USE_WINDOWS_INPUT_MODE=background|session-foreground`：
  - `background` 是默认模式，继续使用 UIA pattern / Win32 window message，不主动抢用户前台。
  - `session-foreground` 面向隔离桌面 session / VM，允许在该 session 内前台化目标窗口并使用 `SendInput`，用于游戏或 raw-input 应用的兼容路径。
- 更新 Windows MCP instructions、`doctor` 输出、架构、安全、质量评分和 Windows active execution plan。
- 新增 `docs/WINDOWS_BACKGROUND.md`，明确后台定义、截图 fallback、输入模式和游戏兼容边界。
- 将 Codex plugin MCP 启动入口从 POSIX shell 脚本切到跨平台 Node launcher，避免 Windows 本地 Codex 部署后依赖 `.sh`。
- 修正 Codex plugin config helper，让它能移除 `[mcp_servers.open-computer-use]` 这种无引号历史 TOML section。
- 已把本地构建的 Windows runtime 部署到 Codex plugin cache，并启用 `open-computer-use@open-computer-use-local`。
- 同步更新直接调试文档，把旧 `.sh` launcher 示例替换为 Windows 可用的 exe 或 Node launcher。
- 将 release 版本 bump 到 `0.1.37`，并重新部署到本机 Codex 的 `open-computer-use-local/open-computer-use/0.1.37` plugin cache。
- 根据 Neon Assault 实测反馈补 `press_key.duration_ms`，让 Windows `session-foreground` 输入可以按住按键一段时间，而不是只能发送极短 tap；版本继续 bump 到 `0.1.38` 并部署到本机 Codex plugin cache。
- 给 Windows background 模式补截图来源和前台变化诊断：`get_app_state` 会标注 `Screenshot source`，action 会标注 `Input mode` 与 `Foreground changed`；版本继续 bump 到 `0.1.39` 并部署到本机 Codex plugin cache。

## 设计动机

Windows 没有一套对所有 GUI toolkit 和游戏都等价于 macOS AX 的后台输入模型。普通桌面应用可以优先走 UIA / Win32 message，减少抢前台；DirectX / raw-input 游戏通常只处理所在桌面 session 的前台输入。因此本次把“游戏后台”定义为隔离 session / VM 场景：目标对它自己的 session 是前台，对用户主桌面是后台，同时明确不实现注入、hook、驱动或反作弊绕过。

## 受影响文件

- `apps/OpenComputerUseWindows/runtime.ps1`
- `apps/OpenComputerUseWindows/main.go`
- `apps/OpenComputerUseWindows/main_test.go`
- `apps/OpenComputerUseLinux/main.go`
- `packages/OpenComputerUseKit/Sources/OpenComputerUseKit/OpenComputerUseVersion.swift`
- `apps/OpenComputerUseSmokeSuite/Sources/OpenComputerUseSmokeSuite/main.swift`
- `scripts/computer-use-cli/main.go`
- `docs/releases/feature-release-notes.md`
- `plugins/open-computer-use/.mcp.json`
- `plugins/open-computer-use/scripts/launch-open-computer-use.mjs`
- `scripts/install-config-helper.mjs`
- `scripts/computer-use-cli/README.md`
- `docs/WINDOWS_BACKGROUND.md`
- `docs/references/codex-computer-use-cli.md`
- `docs/references/codex-network-capture.md`
- `docs/ARCHITECTURE.md`
- `docs/SECURITY.md`
- `docs/QUALITY_SCORE.md`
- `docs/exec-plans/active/20260422-windows-computer-use-runtime.md`
