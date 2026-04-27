package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestToolDefinitionCount(t *testing.T) {
	if got := len(toolDefinitions()); got != 9 {
		t.Fatalf("toolDefinitions() count = %d, want 9", got)
	}
}

func TestCallSequenceStopsAfterFirstToolError(t *testing.T) {
	output, hasError, err := runCallCommand([]string{
		"--calls",
		`[{"tool":"not_a_tool"},{"tool":"list_apps"}]`,
	}, newService())
	if err != nil {
		t.Fatal(err)
	}
	if !hasError {
		t.Fatal("expected hasError")
	}
	items, ok := output.([]map[string]any)
	if !ok {
		t.Fatalf("output type = %T", output)
	}
	if len(items) != 1 {
		t.Fatalf("sequence output count = %d, want 1", len(items))
	}
}

func TestReadArgumentsAcceptsJSONObject(t *testing.T) {
	args, err := readArguments(`{"app":"Notepad","pages":2}`, "")
	if err != nil {
		t.Fatal(err)
	}
	if args["app"] != "Notepad" {
		t.Fatalf("app = %v", args["app"])
	}
	if args["pages"].(json.Number).String() != "2" {
		t.Fatalf("pages = %v", args["pages"])
	}
}

func TestMCPInitializeResponseContainsToolsCapability(t *testing.T) {
	request := map[string]any{
		"jsonrpc": "2.0",
		"id":      float64(1),
		"method":  "initialize",
		"params":  map[string]any{},
	}
	response := handleMCPRequest(request, newService())
	result, ok := response["result"].(map[string]any)
	if !ok {
		t.Fatalf("missing result: %#v", response)
	}
	capabilities := result["capabilities"].(map[string]any)
	if _, ok := capabilities["tools"]; !ok {
		t.Fatalf("missing tools capability: %#v", capabilities)
	}
}

func TestCLIHelpMentionsWindowsRuntime(t *testing.T) {
	var out bytes.Buffer
	if err := runCLI([]string{"--help"}, &out); err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(out.String(), "Open Computer Use for Windows") {
		t.Fatalf("help text did not mention Windows runtime:\n%s", out.String())
	}
}

func TestWindowsRuntimeForegroundActionsRequireOptIn(t *testing.T) {
	if !strings.Contains(windowsRuntimeScript, "OPEN_COMPUTER_USE_WINDOWS_ALLOW_APP_LAUNCH") {
		t.Fatal("Windows app launch fallback must remain opt-in")
	}
	if !strings.Contains(windowsRuntimeScript, "OPEN_COMPUTER_USE_WINDOWS_ALLOW_FOCUS_ACTIONS") {
		t.Fatal("Windows SetFocus action must remain opt-in")
	}
	if !strings.Contains(windowsRuntimeScript, "OPEN_COMPUTER_USE_WINDOWS_ALLOW_UIA_TEXT_FALLBACK") {
		t.Fatal("Windows UIA text fallback must remain opt-in")
	}
	if !strings.Contains(serverInstructions, "does not auto-launch apps, perform SetFocus, or use UIA text fallback by default") {
		t.Fatal("MCP instructions must document the Windows background-focus policy")
	}
}

func TestWindowsRuntimeBackgroundScreenshotUsesPrintWindowFallback(t *testing.T) {
	if !strings.Contains(windowsRuntimeScript, "PrintWindow") {
		t.Fatal("Windows screenshot capture should try PrintWindow before screen copy")
	}
	if !strings.Contains(windowsRuntimeScript, "PW_RENDERFULLCONTENT") {
		t.Fatal("Windows PrintWindow capture should request full window content")
	}
	if !strings.Contains(windowsRuntimeScript, "Test-BitmapMostlyBlank") {
		t.Fatal("Windows screenshot capture should detect blank PrintWindow results before falling back")
	}
	if !strings.Contains(windowsRuntimeScript, "screenshotSource") {
		t.Fatal("Windows snapshots should report screenshotSource diagnostics")
	}
}

func TestWindowsRuntimeSessionForegroundModeIsExplicit(t *testing.T) {
	if !strings.Contains(windowsRuntimeScript, "OPEN_COMPUTER_USE_WINDOWS_INPUT_MODE") {
		t.Fatal("Windows input mode must be controlled by an explicit environment variable")
	}
	if !strings.Contains(windowsRuntimeScript, "session-foreground") {
		t.Fatal("Windows runtime must expose a session-foreground input mode")
	}
	if !strings.Contains(windowsRuntimeScript, "SetForegroundWindow") {
		t.Fatal("session-foreground mode should be able to foreground the target inside an isolated session")
	}
	if !strings.Contains(windowsRuntimeScript, "SendInput") {
		t.Fatal("session-foreground mode should use SendInput for raw-input compatible actions")
	}
	if !strings.Contains(serverInstructions, "isolated desktop session or VM") {
		t.Fatal("MCP instructions must document the VM/session boundary for game compatibility")
	}
}

func TestWindowsPressKeySupportsDurationForGames(t *testing.T) {
	definitions := toolDefinitions()
	var pressKey *toolDefinition
	for index := range definitions {
		if definitions[index].Name == "press_key" {
			pressKey = &definitions[index]
			break
		}
	}
	if pressKey == nil {
		t.Fatal("press_key tool definition is missing")
	}
	properties, _ := pressKey.InputSchema["properties"].(map[string]any)
	if _, ok := properties["duration_ms"]; !ok {
		t.Fatal("press_key schema must expose duration_ms for hold-style game input")
	}
	if !strings.Contains(windowsRuntimeScript, "Send-ForegroundKey $hwnd $operation.key ([int]$operation.duration_ms)") {
		t.Fatal("session-foreground press_key should pass duration_ms to Send-ForegroundKey")
	}
	if !strings.Contains(windowsRuntimeScript, "Send-Key $hwnd $operation.key ([int]$operation.duration_ms)") {
		t.Fatal("background press_key should pass duration_ms to Send-Key")
	}
	if !strings.Contains(windowsRuntimeScript, "Start-Sleep -Milliseconds $durationMs") {
		t.Fatal("PowerShell foreground key path should hold the key for duration_ms")
	}
}

func TestWindowsSnapshotRendersDiagnostics(t *testing.T) {
	changed := false
	snapshot := appSnapshot{
		App:               appDescriptor{Name: "Game", PID: 42},
		WindowTitle:       "Neon",
		ScreenshotSource:  "print_window",
		InputMode:         "background",
		ForegroundChanged: &changed,
	}
	text := snapshot.renderedText()
	for _, expected := range []string{
		"Screenshot source: print_window",
		"Input mode: background",
		"Foreground changed: false",
	} {
		if !strings.Contains(text, expected) {
			t.Fatalf("snapshot text missing %q:\n%s", expected, text)
		}
	}
}

func TestWindowsRuntimeTracksForegroundChangesForActions(t *testing.T) {
	if !strings.Contains(windowsRuntimeScript, "GetForegroundWindow") {
		t.Fatal("Windows action path should sample foreground window")
	}
	if !strings.Contains(windowsRuntimeScript, "foregroundChanged") {
		t.Fatal("Windows action snapshots should include foregroundChanged diagnostics")
	}
	if !strings.Contains(windowsRuntimeScript, "inputMode = Get-WindowsInputMode") {
		t.Fatal("Windows action snapshots should include input mode diagnostics")
	}
}

func TestWindowsRuntimePacksSignedWheelDeltaForSendInput(t *testing.T) {
	if !strings.Contains(windowsRuntimeScript, "ConvertTo-UnsignedInt32") {
		t.Fatal("session-foreground wheel input should pack signed wheel deltas without UInt32 cast failures")
	}
	if strings.Contains(windowsRuntimeScript, "[UInt32]($delta -band 0xffffffff)") {
		t.Fatal("PowerShell cannot cast a negative wheel delta directly to UInt32")
	}
}

func TestDoctorMentionsWindowsInputModeBoundary(t *testing.T) {
	var out bytes.Buffer
	if err := runCLI([]string{"doctor"}, &out); err != nil {
		t.Fatal(err)
	}
	text := out.String()
	if !strings.Contains(text, "OPEN_COMPUTER_USE_WINDOWS_INPUT_MODE=background") {
		t.Fatalf("doctor should document default input mode:\n%s", text)
	}
	if !strings.Contains(text, "OPEN_COMPUTER_USE_WINDOWS_INPUT_MODE=session-foreground") {
		t.Fatalf("doctor should document session-foreground mode:\n%s", text)
	}
	if !strings.Contains(text, "isolated desktop session or VM") {
		t.Fatalf("doctor should document the game compatibility boundary:\n%s", text)
	}
}

func TestCodexPluginMCPConfigUsesCrossPlatformLauncher(t *testing.T) {
	configPath := filepath.FromSlash("../../plugins/open-computer-use/.mcp.json")
	raw, err := os.ReadFile(configPath)
	if err != nil {
		t.Fatal(err)
	}
	text := string(raw)
	if strings.Contains(text, "launch-open-computer-use.sh") {
		t.Fatal("Codex plugin MCP config should not require a POSIX shell on Windows")
	}
	if !strings.Contains(text, "launch-open-computer-use.mjs") {
		t.Fatal("Codex plugin MCP config should use the cross-platform Node launcher")
	}
}
