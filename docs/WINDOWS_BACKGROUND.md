# Windows 后台控制与截图边界

这份文档记录 Windows runtime 对“后台”的定义、可配置输入模式，以及游戏 / raw-input 应用的兼容边界。

## 后台定义

- 支持目标窗口不在前台、被其他窗口遮挡但未最小化的场景。
- 不承诺最小化、锁屏、未登录桌面或脱离交互式 session 的 GUI 操作。
- `get_app_state` 截图优先使用 `PrintWindow(PW_RENDERFULLCONTENT)` 捕获目标窗口；如果失败或返回明显空图，再回退到 `CopyFromScreen`。
- `get_app_state` 文本会标注 `Screenshot source: print_window|screen_copy|none`，用于区分真正窗口截图、屏幕拷贝 fallback 和无可用截图。

## 输入模式

默认模式：

```powershell
$env:OPEN_COMPUTER_USE_WINDOWS_INPUT_MODE = "background"
```

`background` 会保持非抢占策略：优先使用 UI Automation pattern、可写文本控件 HWND 的 `EM_REPLACESEL`、以及 Win32 `PostMessage` / `SendMessage`。这个模式不主动把目标窗口带到前台，但不同 GUI toolkit 对后台消息支持不一致。

action 返回的文本会标注 `Input mode` 和 `Foreground changed`。如果 `background` 模式下 `Foreground changed: true`，说明本轮调用或外部环境导致前台窗口变化，需要按风险处理，而不是把 `isError=false` 直接理解成完全非抢占。

隔离会话 / VM 模式：

```powershell
$env:OPEN_COMPUTER_USE_WINDOWS_INPUT_MODE = "session-foreground"
```

`session-foreground` 会在当前 Windows 桌面 session 内对目标窗口执行 `SetForegroundWindow`，并使用 `SendInput` 执行 click、drag、scroll、type_text 和 press_key。它只适合目标 app 运行在隔离桌面 session 或 VM 里的场景：对用户主桌面来说仍是“后台”，但对目标 session 来说目标窗口是前台。

游戏移动通常需要“按住”而不是短按。`press_key` 支持可选的 `duration_ms` 参数，例如 `{"app":"45176","key":"right","duration_ms":800}` 会按住右方向键约 800ms。

## 游戏与 raw-input 应用

- 同一桌面内对游戏、DirectX 或 raw-input 应用做真正后台输入不可保证。
- 可靠做法是把游戏放到隔离 VM / 远程桌面 / 独立交互式 session 中运行，再在该 session 内使用 `OPEN_COMPUTER_USE_WINDOWS_INPUT_MODE=session-foreground`。
- 项目不实现注入、hook、内核驱动、反作弊绕过或其它高风险输入路径。

## 验证建议

- 普通应用：打开 Notepad，用其他窗口遮挡它，运行 `get_app_state`，确认截图仍来自 Notepad 窗口。
- 默认后台输入：保持 Codex 或终端在前台，运行 `get_app_state -> type_text -> press_key -> get_app_state`，确认目标应用响应且前台不被抢。
- 游戏路径：在隔离 VM / session 内启动目标游戏，设置 `OPEN_COMPUTER_USE_WINDOWS_INPUT_MODE=session-foreground`，确认动作只影响该隔离 session。
