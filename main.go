package main

import (
	"MathReX/model_controller"
	"embed"
	"encoding/json"
	"fmt"
	"image/png" // Added for saving screenshot on Windows
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort" // Keep for sorting modifiers
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/getlantern/systray"
	"github.com/kbinani/screenshot" // Added for Windows screenshot
	gohook "github.com/robotn/gohook"
	"github.com/sqweek/dialog"
	onnxruntime "github.com/yalue/onnxruntime_go"
)

// Modifier key constants are no longer needed here as hook.Register uses strings.

//go:embed all:onnxruntime all:model katex.min.js mathml2omml.js
var embeddedFS embed.FS

type AppSettings struct {
	OutputFormat    string `json:"outputFormat"`
	CaptureShortcut string `json:"captureShortcut"`
}

var currentSettings AppSettings
var settingsFilePath string
var mCaptureShortcut *systray.MenuItem

// Global state for shortcut setting
var isSettingShortcut bool = false
var shortcutMutex sync.Mutex // To protect isSettingShortcut and newShortcutEventChannel
var newShortcutEventChannel = make(chan gohook.Event, 1)

// Global hook system management
var hookEventChannel chan gohook.Event
var hookProcessDone chan struct{} // Closed when gohook.Process goroutine finishes
var hookSystemMutex sync.Mutex    // To protect hookEventChannel and hookProcessDone

// Hardcoded raw code to string map for common keys.
// This is a fallback since gohook.Key... constants are not available as expected.
var rawCodeToNamedKey map[uint16]string = map[uint16]string{
	27:  "esc",   // Escape
	32:  "space", // Space
	13:  "enter", // Enter (might vary, 10 or 13) - check ev.Keychar as well
	9:   "tab",   // Tab
	8:   "backspace",
	46:  "delete",   // Delete (might vary)
	36:  "home",     // Home (might vary)
	35:  "end",      // End (might vary)
	33:  "pageup",   // Page Up (might vary)
	34:  "pagedown", // Page Down (might vary)
	37:  "left",     // Left arrow
	38:  "up",       // Up arrow
	39:  "right",    // Right arrow
	40:  "down",     // Down arrow
	20:  "capslock",
	145: "scrolllock",
	144: "numlock",
	44:  "printscreen",
	19:  "pause",
	45:  "insert", // Insert (might vary)
	112: "f1", 113: "f2", 114: "f3", 115: "f4", 116: "f5", 117: "f6",
	118: "f7", 119: "f8", 120: "f9", 121: "f10", 122: "f11", 123: "f12",
	// Keypad numbers (Rawcodes might differ significantly from ASCII)
	// These are just placeholders, actual rawcodes for keypad keys are specific.
	96: "kp0", 97: "kp1", 98: "kp2", 99: "kp3", 100: "kp4",
	101: "kp5", 102: "kp6", 103: "kp7", 104: "kp8", 105: "kp9",
	110: "kp.", 107: "kp+", 109: "kp-", 106: "kp*", 111: "kp/",
	// Punctuation - for these, ev.Keychar is usually more reliable if it's a printable char.
	188: ",", 190: ".", 191: "/", 220: "\\", 186: ";", 222: "'",
	219: "[", 221: "]", 189: "-", 187: "=", 192: "`",
}

// These are common bitmasks for modifiers.
// They might match gohook's internal values but are used here as a fallback.
const (
	ModShift uint16 = 1 // Standard Shift mask bit
	ModCtrl  uint16 = 2 // Standard Ctrl mask bit
	ModAlt   uint16 = 4 // Standard Alt mask bit (Option on macOS)
	ModCmd   uint16 = 8 // Standard Cmd/Meta/Win mask bit
)

// eventToShortcutString converts a gohook.Event to a "key+mod1+mod2" string.
func eventToShortcutString(ev gohook.Event) string {
	var parts []string
	keyStr := ""

	// Try to get a friendly name for the key from raw code first for special keys
	if name, ok := rawCodeToNamedKey[ev.Rawcode]; ok {
		keyStr = name
	} else if ev.Keychar != 0 && ev.Keychar != 65535 { // 65535 is NoChar from gohook
		// If it's a printable character, use it.
		// This handles A-Z, a-z, 0-9, and common symbols directly.
		keyStr = strings.ToLower(string(ev.Keychar))
	} else {
		// Fallback for unknown raw codes that don't produce a Keychar
		keyStr = fmt.Sprintf("raw%d", ev.Rawcode)
	}

	if keyStr == "" {
		// This case should ideally not be reached if the above logic is sound.
		log.Printf("Warning: Could not determine key string for event: %+v", ev)
		return ""
	}

	var modifiers []string
	// Ctrl
	if ev.Mask&ModCtrl > 0 { // ModCtrl = 2
		modifiers = append(modifiers, "ctrl")
	}
	// Shift
	if ev.Mask&ModShift > 0 { // ModShift = 1
		modifiers = append(modifiers, "shift")
	}

	// Alt and Cmd/Meta handling
	if runtime.GOOS == "darwin" {
		// On macOS, based on user feedback (Cmd identified as Alt):
		// - If ModAlt bit (4) is set, interpret as Command key.
		// - If ModCmd bit (8) is set, interpret as Option (Alt) key.
		if ev.Mask&ModAlt > 0 { // Original ModAlt = 4
			modifiers = append(modifiers, "cmd")
		}
		if ev.Mask&ModCmd > 0 { // Original ModCmd = 8
			// Check if "cmd" was already added from ModAlt. If so, this could be Cmd+Option.
			// Or, if they are mutually exclusive from gohook's Mask, this is just Option.
			// To prevent adding "alt" if "cmd" was already added due to ModAlt also being set
			// (which would be unusual unless gohook sets both for a single modifier),
			// we can add a check. However, typical modifier masks are independent bits.
			// For now, assume they are independent or that gohook reports them cleanly.
			// If Option key alone sets ModCmd bit, this will add "alt".
			// If Cmd+Option is pressed, and Cmd sets ModAlt bit & Option sets ModCmd bit, both will be added.
			// The variable isCmdAlreadyAdded was declared but not used in the simplified logic.
			// It's removed to fix the "declared and not used" error.
			// The logic now simply checks if ModCmd is set and adds "alt" if not already present,
			// assuming ModAlt (for "cmd") and ModCmd (for "alt" on Darwin) are distinct flags from gohook.

			// A simpler interpretation is that ModAlt means Cmd, ModCmd means Alt on Darwin.
			// Let's stick to the simpler interpretation:
			// if ev.Mask&ModAlt > 0 means "cmd"
			// if ev.Mask&ModCmd > 0 means "alt"
			// These are not mutually exclusive if user presses Cmd+Option.
			// The previous logic for ModAlt (value 4) on Darwin was adding "alt".
			// The previous logic for ModCmd (value 8) on Darwin was adding "cmd".
			// User says Cmd (which should be ModCmd=8) is seen as Alt (which was ModAlt=4).
			// So, when Cmd is pressed, ev.Mask has bit 4 set.
			// We need: when bit 4 is set on Darwin -> "cmd"
			// For actual Alt/Option key on Darwin, let's assume it uses bit 8.
			// So: when bit 8 is set on Darwin -> "alt"
			// This is a swap of interpretation.
			// The code `if ev.Mask&ModAlt > 0 { modifiers = append(modifiers, "cmd") }` is already above.
			// The following `if ev.Mask&ModCmd > 0` should now add "alt".
			// The previous code for ModCmd on darwin was: `if ev.Mask&ModCmd > 0 && runtime.GOOS == "darwin" { modifiers = append(modifiers, "cmd") }`
			// This needs to change to "alt".
			// The code for ModAlt on darwin was: `if ev.Mask&ModAlt > 0 { modifiers = append(modifiers, "alt") }`
			// This needs to change to "cmd".

			// The current block for Darwin already handles `if ev.Mask&ModAlt > 0 { modifiers = append(modifiers, "cmd") }`
			// So we just need to handle the `ModCmd` part for "alt" on Darwin.
			// The `else if ev.Mask&ModCmd > 0` for non-Darwin should remain "meta".
		}
		// This structure was getting confusing. Let's simplify the logic block.
	} else { // Not Darwin
		if ev.Mask&ModAlt > 0 { // ModAlt = 4
			modifiers = append(modifiers, "alt")
		}
		if ev.Mask&ModCmd > 0 { // ModCmd = 8
			modifiers = append(modifiers, "meta")
		}
	}
	// Corrected logic structure for Alt/Cmd:
	if runtime.GOOS == "darwin" {
		if ev.Mask&ModAlt > 0 { // If mask bit 4 is set on macOS
			// Check if "cmd" is already added to avoid duplicates if gohook is weird
			// However, this check is probably not needed if ModAlt and ModCmd are distinct bits
			// and gohook sets them correctly for distinct keys.
			// Based on user feedback, mask bit 4 IS Cmd.
			if !contains(modifiers, "cmd") {
				modifiers = append(modifiers, "cmd")
			}
		}
		if ev.Mask&ModCmd > 0 { // If mask bit 8 is set on macOS, assume it's Option/Alt
			if !contains(modifiers, "alt") {
				modifiers = append(modifiers, "alt")
			}
		}
	} else { // Not Darwin
		if ev.Mask&ModAlt > 0 { // ModAlt = 4
			if !contains(modifiers, "alt") {
				modifiers = append(modifiers, "alt")
			}
		}
		if ev.Mask&ModCmd > 0 { // ModCmd = 8
			if !contains(modifiers, "meta") {
				modifiers = append(modifiers, "meta")
			}
		}
	}

	modOrder := map[string]int{"ctrl": 1, "alt": 2, "shift": 3, "cmd": 4, "meta": 5}
	sort.SliceStable(modifiers, func(i, j int) bool {
		return modOrder[modifiers[i]] < modOrder[modifiers[j]]
	})

	parts = append(parts, keyStr)
	parts = append(parts, modifiers...)
	return strings.Join(parts, "+")
}

func GetEmbeddedKaTeXJS() ([]byte, error) {
	return embeddedFS.ReadFile("katex.min.js")
}
func GetEmbeddedMathML2OMMLJS() ([]byte, error) {
	return embeddedFS.ReadFile("mathml2omml.js")
}

func extractAndGetPaths() (string, string, string, string, error) {
	tempDir := os.TempDir()
	libDir := filepath.Join(tempDir, "MathReX_lib")
	modelDir := filepath.Join(tempDir, "MathReX_model")

	if err := os.MkdirAll(libDir, 0755); err != nil {
		return "", "", "", "", fmt.Errorf("failed to create lib temp dir: %w", err)
	}
	if err := os.MkdirAll(modelDir, 0755); err != nil {
		return "", "", "", "", fmt.Errorf("failed to create model temp dir: %w", err)
	}

	libEmbedPath := getDefaultSharedLibEmbedPath()
	if libEmbedPath == "" {
		return "", "", "", "", fmt.Errorf("could not determine embedded library path for %s/%s", runtime.GOOS, runtime.GOARCH)
	}
	libData, err := embeddedFS.ReadFile(libEmbedPath)
	if err != nil {
		return "", "", "", "", fmt.Errorf("failed to read embedded library %s: %w", libEmbedPath, err)
	}
	libFileName := filepath.Base(libEmbedPath)
	extractedLibPath := filepath.Join(libDir, libFileName)
	if err := os.WriteFile(extractedLibPath, libData, 0755); err != nil {
		return "", "", "", "", fmt.Errorf("failed to write library %s: %w", extractedLibPath, err)
	}
	log.Printf("Extracted library to: %s\n", extractedLibPath)

	modelFiles := []string{"encoder_model.onnx", "decoder_model.onnx", "tokenizer.json"}
	var extractedTokenizerPath, extractedEncoderPath, extractedDecoderPath string
	for _, fName := range modelFiles {
		embedPath := filepath.Join("model", fName)
		data, err := embeddedFS.ReadFile(embedPath)
		if err != nil {
			return "", "", "", "", fmt.Errorf("failed to read embedded model file %s: %w", fName, err)
		}
		extractedPath := filepath.Join(modelDir, fName)
		if err := os.WriteFile(extractedPath, data, 0644); err != nil {
			return "", "", "", "", fmt.Errorf("failed to write model file %s: %w", fName, err)
		}
		log.Printf("Extracted model file to: %s\n", extractedPath)
		switch fName {
		case "tokenizer.json":
			extractedTokenizerPath = extractedPath
		case "encoder_model.onnx":
			extractedEncoderPath = extractedPath
		case "decoder_model.onnx":
			extractedDecoderPath = extractedPath
		}
	}
	if extractedTokenizerPath == "" || extractedEncoderPath == "" || extractedDecoderPath == "" {
		return "", "", "", "", fmt.Errorf("one or more critical model files not found")
	}
	return extractedLibPath, extractedTokenizerPath, extractedEncoderPath, extractedDecoderPath, nil
}

func getDefaultSharedLibEmbedPath() string {
	const onnxVersion = "1.21.0" // Align with download script and user feedback
	if runtime.GOOS == "windows" {
		if runtime.GOARCH == "amd64" {
			// DLL name usually doesn't include version
			return fmt.Sprintf("onnxruntime/amd64_windows/onnxruntime.dll")
		}

		if runtime.GOARCH == "arm64" {
			// DLL name usually doesn't include version
			return fmt.Sprintf("onnxruntime/arm64_windows/onnxruntime.dll")
		}
	}
	if runtime.GOOS == "darwin" {
		if runtime.GOARCH == "arm64" {
			return fmt.Sprintf("onnxruntime/arm64_darwin/libonnxruntime.%s.dylib", onnxVersion)
		}
		if runtime.GOARCH == "amd64" {
			return fmt.Sprintf("onnxruntime/amd64_darwin/libonnxruntime.%s.dylib", onnxVersion)
		}
	}
	if runtime.GOOS == "linux" {
		// Assuming .so for Linux, and versioned name similar to dylib
		if runtime.GOARCH == "arm64" {
			return fmt.Sprintf("onnxruntime/arm64_linux/libonnxruntime.%s.so", onnxVersion)
		}
		return fmt.Sprintf("onnxruntime/amd64_linux/libonnxruntime.%s.so", onnxVersion)
	}
	log.Printf("Error: Unable to determine embedded onnxruntime path for OS=%s Arch=%s\n", runtime.GOOS, runtime.GOARCH)
	return ""
}

func getDefaultShortcut() string {
	if runtime.GOOS == "darwin" {
		return "cmd+shift+c"
	}
	return "ctrl+shift+s"
}

func loadSettings() {
	defaultShortcut := getDefaultShortcut()
	currentSettings = AppSettings{OutputFormat: "mathml", CaptureShortcut: defaultShortcut}

	homeDir, err := os.UserHomeDir()
	if err != nil {
		log.Printf("Warning: Could not get user home directory: %v. Using default settings.", err)
		return
	}
	settingsFilePath = filepath.Join(homeDir, ".config", "MathReX", "settings.json")

	if err := os.MkdirAll(filepath.Dir(settingsFilePath), 0750); err != nil {
		log.Printf("Warning: Could not create config directory %s: %v. Using default settings.", filepath.Dir(settingsFilePath), err)
		return
	}

	data, err := ioutil.ReadFile(settingsFilePath)
	if err != nil {
		if os.IsNotExist(err) {
			log.Printf("Settings file not found at %s. Creating with default settings.", settingsFilePath)
			saveSettings()
		} else {
			log.Printf("Warning: Could not read settings file %s: %v. Using default settings.", settingsFilePath, err)
		}
		return
	}

	err = json.Unmarshal(data, &currentSettings)
	if err != nil {
		log.Printf("Warning: Could not parse settings file %s: %v. Using default settings.", settingsFilePath, err)
		currentSettings = AppSettings{OutputFormat: "mathml", CaptureShortcut: defaultShortcut}
	}
	if currentSettings.OutputFormat != "latex" && currentSettings.OutputFormat != "mathml" {
		// If OMML was somehow set (e.g. old config file), default to mathml
		if currentSettings.OutputFormat == "omml" {
			log.Printf("Warning: OMML output format is no longer supported. Defaulting to mathml.")
		} else {
			log.Printf("Warning: Invalid outputFormat '%s' loaded. Defaulting to mathml.", currentSettings.OutputFormat)
		}
		currentSettings.OutputFormat = "mathml"
	}
	if currentSettings.CaptureShortcut == "" {
		currentSettings.CaptureShortcut = defaultShortcut
	}
	log.Printf("Settings loaded: %+v", currentSettings)
}

func saveSettings() {
	if settingsFilePath == "" {
		homeDir, _ := os.UserHomeDir()
		settingsFilePath = filepath.Join(homeDir, ".config", "MathReX", "settings.json")
		os.MkdirAll(filepath.Dir(settingsFilePath), 0750)
	}

	data, err := json.MarshalIndent(currentSettings, "", "  ")
	if err != nil {
		log.Printf("Error: Could not marshal settings to JSON: %v", err)
		return
	}
	err = ioutil.WriteFile(settingsFilePath, data, 0644)
	if err != nil {
		log.Printf("Error: Could not write settings file %s: %v", settingsFilePath, err)
	}
	log.Printf("Settings saved: %+v", currentSettings)
	if mCaptureShortcut != nil {
		mCaptureShortcut.SetTitle(fmt.Sprintf("Capture Shortcut: %s", currentSettings.CaptureShortcut))
	}
}

func main() {
	onExit := func() {
		// gohook.End() is now called before systray.Quit()
		log.Println("MathReX onExit: Systray cleanup.")
		log.Println("MathReX application finished.")
	}
	systray.Run(onReady, onExit)
}

// Helper function for eventToShortcutString
func contains(slice []string, item string) bool {
	for _, s := range slice {
		if s == item {
			return true
		}
	}
	return false
}

func onReady() {
	log.Println("MathReX is ready.")
	loadSettings()
	// initializeRawCodeMap() // Removed as we are hardcoding or using Keychar

	extractedLibPath, extractedTokenizerPath, extractedEncoderPath, extractedDecoderPath, err := extractAndGetPaths()
	if err != nil {
		log.Fatalf("Failed to extract embedded files: %v", err)
	}
	tempModelDir := filepath.Dir(extractedTokenizerPath)
	defer os.RemoveAll(filepath.Dir(extractedLibPath))
	defer os.RemoveAll(tempModelDir)
	onnxruntime.SetSharedLibraryPath(extractedLibPath)
	if err := onnxruntime.InitializeEnvironment(); err != nil {
		log.Fatalf("ONNX Init fail: %v", err)
	}
	if err := model_controller.InitTokenizer(extractedTokenizerPath); err != nil {
		log.Fatalf("Tokenizer Init fail: %v", err)
	}
	if err := model_controller.InitModels(extractedEncoderPath, extractedDecoderPath); err != nil {
		log.Fatalf("Models Init fail: %v", err)
	}
	katexJSData, _ := GetEmbeddedKaTeXJS()
	model_controller.InitKaTeX(katexJSData)

	mathml2ommlJSData, err := GetEmbeddedMathML2OMMLJS()
	if err != nil || len(mathml2ommlJSData) == 0 {
		log.Fatalf("Failed to get embedded mathml2omml.js: %v or data is empty", err)
	}
	model_controller.InitMathML2OMMLJS(mathml2ommlJSData)

	log.Println("All core components initialized successfully.")

	log.Println("Systray icon handling skipped as icon.png is not embedded.")
	systray.SetTitle("MathReX")
	systray.SetTooltip("MathReX - Screenshot to Math")

	mCapture := systray.AddMenuItem("Capture & Recognize", "Capture a screen region via menu")
	mFromFile := systray.AddMenuItem("Recognize from File...", "Select an image file")
	systray.AddSeparator()

	mOutputFormat := systray.AddMenuItem("Output Format", "Select output format")
	mFormatLatex := mOutputFormat.AddSubMenuItemCheckbox("LaTeX", "LaTeX", currentSettings.OutputFormat == "latex")
	mFormatMathML := mOutputFormat.AddSubMenuItemCheckbox("MathML", "MathML", currentSettings.OutputFormat == "mathml")
	// OMML option removed

	systray.AddSeparator()
	mCaptureShortcut = systray.AddMenuItem(fmt.Sprintf("Capture Shortcut: %s", currentSettings.CaptureShortcut), "Edit settings.json to change (restart needed for hotkey)")
	mCaptureShortcut.Disable()

	mSetShortcut := systray.AddMenuItem("Set Capture Shortcut...", "Set a new shortcut for capture")

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit MathReX", "Exit the application")

	// Start the global hook system
	startGlobalHookSystem() // This now registers the hotkey as well

	go func() {
		for {
			select {
			case <-mCapture.ClickedCh:
				go handleCaptureAndRecognize()
			case <-mFromFile.ClickedCh:
				go handleRecognizeFromFile()
			case <-mFormatLatex.ClickedCh:
				currentSettings.OutputFormat = "latex"
				updateFormatCheckmarks(mFormatLatex, mFormatMathML, nil) // Pass nil for ommlItem
				saveSettings()
			case <-mFormatMathML.ClickedCh:
				currentSettings.OutputFormat = "mathml"
				updateFormatCheckmarks(mFormatLatex, mFormatMathML, nil) // Pass nil for ommlItem
				saveSettings()
			// OMML case (and mFormatOMML.ClickedCh) removed
			case <-mSetShortcut.ClickedCh:
				go handleChangeShortcutGUI() // Changed to new GUI handler
			case <-mQuit.ClickedCh:
				log.Println("Quit menu item clicked. Stopping hook system...")
				stopGlobalHookSystem() // Stop our managed hook system
				log.Println("Hook system stopped. Quitting systray...")
				systray.Quit()
				return
			}
		}
	}()
}

func startGlobalHookSystem() {
	hookSystemMutex.Lock()
	defer hookSystemMutex.Unlock()

	if hookEventChannel != nil {
		log.Println("Hook system already running. Stopping first.")
		// Internal stop without lock
		internalStopGlobalHookSystem()
	}

	log.Println("Starting global hook system...")
	hookEventChannel = gohook.Start()
	hookProcessDone = make(chan struct{}) // New channel for this instance

	// Goroutine to process hook events
	go func(currentHookEventChannel chan gohook.Event, currentHookProcessDone chan struct{}) {
		log.Println("gohook.Process starting...")
		// gohook.Process blocks until gohook.End() is called on the *same* event channel.
		// Or, more accurately, until the channel passed to it is used by End()
		<-gohook.Process(currentHookEventChannel) // This was the issue: Process needs the specific channel
		close(currentHookProcessDone)             // Signal that this Process goroutine has finished
		log.Println("gohook.Process finished.")
	}(hookEventChannel, hookProcessDone)

	// Register the primary capture hotkey
	registerGlobalHotkey() // This will use the current hookEventChannel implicitly if gohook allows

	// Add a generic event listener for when we are setting a new shortcut
	// This listener captures *any* key down event if isSettingShortcut is true.
	gohook.Register(gohook.KeyDown, []string{}, func(e gohook.Event) { // Empty filter means all KeyDown events
		shortcutMutex.Lock()
		if isSettingShortcut {
			log.Printf("Shortcut setting mode: Key event captured: %v", e)
			// Try to send to channel, but don't block if not ready (e.g., race condition)
			select {
			case newShortcutEventChannel <- e:
				log.Println("Sent event to newShortcutEventChannel")
			default:
				log.Println("newShortcutEventChannel not ready to receive")
			}
		}
		shortcutMutex.Unlock()
	})
	log.Println("Global hook system started and generic listener registered.")
}

func internalStopGlobalHookSystem() {
	// This function assumes hookSystemMutex is already locked or not needed (e.g. during onExit)
	if hookEventChannel != nil {
		log.Println("internalStopGlobalHookSystem: Stopping gohook...")
		gohook.End() // This should use the stored hookEventChannel

		// Wait for the gohook.Process goroutine to finish
		if hookProcessDone != nil {
			log.Println("internalStopGlobalHookSystem: Waiting for gohook.Process to finish...")
			select {
			case <-hookProcessDone:
				log.Println("internalStopGlobalHookSystem: gohook.Process finished.")
			case <-time.After(2 * time.Second): // Timeout to prevent deadlock
				log.Println("internalStopGlobalHookSystem: Timeout waiting for gohook.Process to finish.")
			}
		}
		hookEventChannel = nil
		hookProcessDone = nil
		log.Println("internalStopGlobalHookSystem: Hook system resources released.")
	} else {
		log.Println("internalStopGlobalHookSystem: Hook system not running.")
	}
}
func stopGlobalHookSystem() {
	hookSystemMutex.Lock()
	defer hookSystemMutex.Unlock()
	internalStopGlobalHookSystem()
}

func handleChangeShortcutGUI() {
	shortcutMutex.Lock()
	if isSettingShortcut {
		shortcutMutex.Unlock()
		log.Println("Already in shortcut setting mode.")
		dialog.Message("Already setting a shortcut. Press keys or ESC.").Title("In Progress").Info()
		return
	}
	isSettingShortcut = true
	shortcutMutex.Unlock()

	log.Println("Entered shortcut setting mode.")
	// Ensure the global hook system is running to capture keys
	// startGlobalHookSystem() // It should already be running. If not, this might be a place to ensure it.

	// Inform the user
	go func() { // Run dialog in a goroutine so it doesn't block main UI loop
		dialog.Message("Press the new key combination for capture, or press ESC to cancel.").Title("Set New Shortcut").Info()
	}()

	// Goroutine to wait for the key press or timeout
	go func() {
		defer func() {
			shortcutMutex.Lock()
			isSettingShortcut = false
			shortcutMutex.Unlock()
			log.Println("Exited shortcut setting mode.")
		}()

		select {
		case event := <-newShortcutEventChannel:
			log.Printf("Received shortcut event: Rawcode %d, Keychar %c (%s), Modifiers %x", event.Rawcode, event.Keychar, string(event.Keychar), event.Mask)
			// Check for ESC key using its common raw code 27
			if event.Rawcode == 27 { // ESC key raw code
				log.Println("Shortcut setting cancelled by user (ESC pressed).")
				dialog.Message("Shortcut setting cancelled.").Title("Cancelled").Info()
				return
			}

			newShortcutStr := eventToShortcutString(event)
			if newShortcutStr == "" {
				log.Println("Failed to convert event to shortcut string.")
				dialog.Message("Could not recognize the pressed key combination. Please try again.").Title("Error").Error()
				return
			}

			log.Printf("New shortcut candidate: %s", newShortcutStr)

			// Corrected dialog usage
			confirmed := dialog.Message(fmt.Sprintf("Set shortcut to: %s?", newShortcutStr)).Title("Confirm Shortcut").YesNo()
			if confirmed {
				currentSettings.CaptureShortcut = newShortcutStr
				saveSettings()
				log.Printf("New shortcut set and saved: %s", newShortcutStr)

				// Restart hook system to apply new hotkey
				// This needs to be done carefully.
				// gohook.Unregister might be needed if we could identify the old hotkey registration.
				// Simpler to stop and restart the whole hook listener.
				stopGlobalHookSystem()
				startGlobalHookSystem() // This will re-register with the new shortcut

				dialog.Message(fmt.Sprintf("Shortcut changed to: %s", newShortcutStr)).Title("Success").Info()
				if mCaptureShortcut != nil {
					mCaptureShortcut.SetTitle(fmt.Sprintf("Capture Shortcut: %s", currentSettings.CaptureShortcut))
				}
			} else {
				log.Println("User did not confirm new shortcut.")
				dialog.Message("Shortcut change cancelled by user.").Title("Cancelled").Info()
			}

		case <-time.After(30 * time.Second): // Timeout for setting shortcut
			log.Println("Shortcut setting timed out.")
			shortcutMutex.Lock()
			if isSettingShortcut { // Check if still in setting mode (could have been resolved by ESC)
				isSettingShortcut = false                                             // Ensure we exit setting mode
				dialog.Message("Shortcut setting timed out.").Title("Timeout").Info() // Changed Warning to Info
			}
			shortcutMutex.Unlock()
		}
	}()
}

func updateFormatCheckmarks(latexItem, mathmlItem, ommlItem *systray.MenuItem) { // ommlItem will be nil
	latexItem.Uncheck()
	mathmlItem.Uncheck()
	if ommlItem != nil { // Should always be nil now, but good practice
		ommlItem.Uncheck()
	}
	switch currentSettings.OutputFormat {
	case "latex":
		latexItem.Check()
	case "mathml":
		mathmlItem.Check()
		// OMML case removed
	}
}

// parseShortcutString converts a shortcut string like "cmd+shift+c"
// into a slice of strings suitable for gohook.Register.
func parseShortcutString(shortcut string) []string {
	parts := strings.Split(strings.ToLower(shortcut), "+")
	if len(parts) == 0 {
		return nil
	}

	var key string
	var modifiers []string

	for _, part := range parts {
		p := strings.ToLower(strings.TrimSpace(part))
		switch p {
		case "ctrl", "control":
			modifiers = append(modifiers, "ctrl")
		case "alt", "option":
			modifiers = append(modifiers, "alt")
		case "shift":
			modifiers = append(modifiers, "shift")
		case "cmd", "command", "super", "win": // "cmd" is standard for gohook on darwin, "meta" might be for others.
			// gohook's Register function seems to handle "cmd" correctly on macOS.
			// For other OS, "meta" or specific OS key might be needed if "cmd" doesn't work.
			// The example uses "ctrl", "shift".
			if runtime.GOOS == "darwin" {
				modifiers = append(modifiers, "cmd")
			} else {
				// For non-Darwin, "meta" is a common term, or "ctrl" if "cmd" implies "ctrl" on other platforms.
				// Given gohook's string matching, we should use what it expects.
				// If "cmd" is intended as "ctrl" on non-darwin, the shortcut string should reflect that.
				// For now, let's assume "cmd" is primarily for macOS.
				// If a generic "super" or "win" key is desired, the string in settings should use "meta" or similar if supported by gohook.
				// The gohook example doesn't show "meta" or "win", so we stick to "ctrl", "alt", "shift", "cmd".
				// If "cmd" is in the shortcut string on non-darwin, it might not work as expected unless gohook maps it.
				// We will pass "cmd" as is, and let gohook handle it.
				modifiers = append(modifiers, p) // Pass "cmd", "super", "win" as is.
			}
		default:
			if key == "" {
				key = p // Assume the first non-modifier is the main key
			} else {
				log.Printf("Warning: Multiple non-modifier keys found in shortcut string: '%s'. Using '%s'.", shortcut, key)
			}
		}
	}

	if key == "" {
		log.Printf("Warning: No main key found in shortcut string: '%s'", shortcut)
		return nil
	}

	// gohook.Register expects []string{"key", "modifier1", "modifier2", ...}
	return append([]string{key}, modifiers...)
}

func registerGlobalHotkey() {
	if currentSettings.CaptureShortcut == "" {
		log.Println("No capture shortcut configured.")
		return
	}

	shortcutParts := parseShortcutString(currentSettings.CaptureShortcut)
	if shortcutParts == nil || len(shortcutParts) < 1 { // Must have at least the key
		log.Printf("Failed to parse shortcut string or no valid parts: '%s'", currentSettings.CaptureShortcut)
		return
	}

	log.Printf("Attempting to register global hotkey: %s -> %v", currentSettings.CaptureShortcut, shortcutParts)

	// This function is now called by startGlobalHookSystem.
	// No need to check evId from gohook.Register as its specific return for this usage might not be an error indicator.
	gohook.Register(gohook.KeyDown, shortcutParts, func(e gohook.Event) {
		shortcutMutex.Lock()
		if !isSettingShortcut {
			shortcutMutex.Unlock()
			log.Printf("Global hotkey pressed: %s (isSettingShortcut=false)", currentSettings.CaptureShortcut)
			go handleCaptureAndRecognize()
		} else {
			shortcutMutex.Unlock()
			log.Printf("Global hotkey pressed but ignored (isSettingShortcut=true): %s", currentSettings.CaptureShortcut)
		}
	})
	log.Printf("Registered hotkey: %s", currentSettings.CaptureShortcut)
}

// startHotkeyListener is no longer used.

func handleCaptureAndRecognize() {
	log.Println("Capture & Recognize triggered.")
	tempImagePath := filepath.Join(os.TempDir(), "mathrex_capture.png")
	defer os.Remove(tempImagePath)

	var cmd *exec.Cmd
	var err error
	// captureSuccess removed as it was unused

	if runtime.GOOS == "darwin" {
		cmd = exec.Command("screencapture", "-s", tempImagePath)
		err = cmd.Run()
	} else if runtime.GOOS == "linux" {
		cmd = exec.Command("scrot", "-s", tempImagePath)
		err = cmd.Run()
	} else if runtime.GOOS == "windows" {
		log.Println("Attempting full-screen capture for Windows using kbinani/screenshot.")
		// This captures the primary display. Region selection is more complex.
		// We'll inform the user that it's a full-screen capture.
		// For multiple displays, screenshot.NumActiveDisplays() and screenshot.GetDisplayBounds(i) can be used.
		// Capturing primary display (index 0)
		n := screenshot.NumActiveDisplays()
		if n <= 0 {
			err = fmt.Errorf("no active displays found for screenshot")
		} else {
			// Capture the primary display (index 0)
			// To capture all displays into one image: bounds := screenshot.GetDisplayBounds(0) ... then loop and combine
			// For simplicity, let's capture the primary display.
			bounds := screenshot.GetDisplayBounds(0)
			img, captureErr := screenshot.CaptureRect(bounds)
			if captureErr != nil {
				err = fmt.Errorf("failed to capture screen: %w", captureErr)
			} else {
				file, createErr := os.Create(tempImagePath)
				if createErr != nil {
					err = fmt.Errorf("failed to create temp image file: %w", createErr)
				} else {
					defer file.Close()
					encodeErr := png.Encode(file, img)
					if encodeErr != nil {
						err = fmt.Errorf("failed to encode screenshot to png: %w", encodeErr)
					} else {
						log.Println("Full screen captured to", tempImagePath)
						// Inform user it was a full screen capture
						go dialog.Message("Captured the full primary screen. For region selection on Windows, please use Win+Shift+S, save the image, and then use 'Recognize from File'.").Title("Windows Capture Note").Info()
					}
				}
			}
		}
	} else {
		log.Println("Unsupported OS for screenshot capture.")
		dialog.Message("%s", "Screenshot capture is not supported on this OS.").Title("Error").Error()
		return
	}

	if err != nil { // This 'err' is from cmd.Run() or a placeholder for Windows
		if runtime.GOOS == "windows" { // Should not be reached if we return early for windows
			// Error handling for any windows specific attempt would go here
		} else if _, statErr := os.Stat(tempImagePath); os.IsNotExist(statErr) {
			log.Println("Screenshot selection cancelled or no file created.")
			return
		}
		log.Printf("Failed to execute screenshot command: %v", err)
		dialog.Message("Failed to capture screenshot: %v", err).Title("Error").Error()
		return
	}
	if _, err := os.Stat(tempImagePath); os.IsNotExist(err) {
		log.Println("Screenshot file not created (selection likely cancelled).")
		return
	}
	processImageFile(tempImagePath)
}

func handleRecognizeFromFile() {
	log.Println("Recognize from File triggered.")
	filePath, err := dialog.File().Filter("Image Files", "png", "jpg", "jpeg", "bmp", "webp").Load()
	if err != nil {
		if err == dialog.ErrCancelled {
			log.Println("File selection cancelled.")
			return
		}
		log.Printf("Error selecting file: %v", err)
		dialog.Message("Error selecting file: %v", err).Title("Error").Error()
		return
	}
	log.Printf("File selected: %s", filePath)
	processImageFile(filePath)
}

func processImageFile(imagePath string) {
	log.Printf("Processing image file: %s", imagePath)
	imageBytes, err := ioutil.ReadFile(imagePath)
	if err != nil {
		log.Printf("Failed to read image file %s: %v", imagePath, err)
		dialog.Message("Failed to read image file: %v", err).Title("Error").Error()
		return
	}
	if len(imageBytes) == 0 {
		log.Printf("Image file %s is empty.", imagePath)
		dialog.Message("Image file is empty: %s", imagePath).Title("Error").Error()
		return
	}

	outputFmt := currentSettings.OutputFormat
	if outputFmt == "omml" { // Fallback if omml is somehow still set
		outputFmt = "mathml"
		log.Println("Warning: OMML output format encountered unexpectedly, falling back to MathML.")
	}
	resultText, _, err := model_controller.ProcessImagePrediction(imageBytes, outputFmt)
	if err != nil {
		log.Printf("Failed to process image prediction: %v", err)
		dialog.Message("Failed to process image: %v", err).Title("Error").Error()
		return
	}

	err = clipboard.WriteAll(resultText)
	if err != nil {
		log.Printf("Failed to copy result to clipboard: %v", err)
		dialog.Message("Failed to copy to clipboard: %v\n\nResult was:\n%s", err, resultText).Title("Clipboard Error").Error()
	} else {
		log.Printf("Result (%s) copied to clipboard.", currentSettings.OutputFormat)
		// Show success notification
		successMessage := fmt.Sprintf("Recognition successful!\nFormat: %s\n\nResult copied to clipboard.", currentSettings.OutputFormat)
		go dialog.Message(successMessage).Title("Success").Info()
	}
}
