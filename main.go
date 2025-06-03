package main

import (
	"MathReX/model_controller"
	"embed"
	"encoding/json"
	"fmt"
	"image/png"
	"io"
	"io/ioutil"
	"log"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"runtime/debug"
	"strings"
	"sync"
	"time"

	"github.com/atotto/clipboard"
	"github.com/getlantern/systray"
	"github.com/kbinani/screenshot"
	"github.com/sqweek/dialog"
	onnxruntime "github.com/yalue/onnxruntime_go"
)

//go:embed all:model katex.min.js mathml2omml.js icon.png icon.ico
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
var shortcutMutex sync.Mutex
var shortcutCaptureChan <-chan string

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

	// Use runtime-downloaded ONNX library instead of embedded one
	extractedLibPath := getDefaultSharedLibPath()
	if extractedLibPath == "" {
		return "", "", "", "", fmt.Errorf("could not determine library path for %s/%s", runtime.GOOS, runtime.GOARCH)
	}

	// Check if the library file exists
	if _, err := os.Stat(extractedLibPath); os.IsNotExist(err) {
		return "", "", "", "", fmt.Errorf("ONNX runtime library not found at %s. Please ensure download_onnxruntime.py has been run", extractedLibPath)
	}
	log.Printf("Using ONNX runtime library at: %s\n", extractedLibPath)

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

func getDefaultSharedLibPath() string {
	const onnxVersion = "1.21.0" // Align with download script and user feedback

	// For macOS, check if we're running from an app bundle first
	if runtime.GOOS == "darwin" {
		execPath, err := os.Executable()
		if err == nil {
			// Check if we're in an app bundle
			if strings.Contains(execPath, ".app/Contents/MacOS/") {
				// We're in an app bundle, look for libraries in Resources
				appBundlePath := filepath.Dir(filepath.Dir(execPath)) // Go up from MacOS to Contents
				resourcesPath := filepath.Join(appBundlePath, "Resources", "onnxruntime")
				var libName string
				if runtime.GOARCH == "arm64" {
					libName = fmt.Sprintf("libonnxruntime.%s.dylib", onnxVersion)
				} else {
					libName = fmt.Sprintf("libonnxruntime.%s.dylib", onnxVersion)
				}
				bundleLibPath := filepath.Join(resourcesPath, libName)
				if _, err := os.Stat(bundleLibPath); err == nil {
					log.Printf("Found ONNX runtime in app bundle: %s", bundleLibPath)
					return bundleLibPath
				}
			}
		}
		// Fall back to relative path
		if runtime.GOARCH == "arm64" {
			return filepath.Join("onnxruntime", fmt.Sprintf("libonnxruntime.%s.dylib", onnxVersion))
		}
		if runtime.GOARCH == "amd64" {
			return filepath.Join("onnxruntime", fmt.Sprintf("libonnxruntime.%s.dylib", onnxVersion))
		}
	}

	if runtime.GOOS == "windows" {
		if runtime.GOARCH == "amd64" {
			// DLL name usually doesn't include version
			return filepath.Join("onnxruntime", "onnxruntime.dll")
		}
		if runtime.GOARCH == "arm64" {
			// DLL name usually doesn't include version
			return filepath.Join("onnxruntime", "onnxruntime.dll")
		}
	}

	if runtime.GOOS == "linux" {
		// Assuming .so for Linux, and versioned name similar to dylib
		if runtime.GOARCH == "arm64" {
			return filepath.Join("onnxruntime", fmt.Sprintf("libonnxruntime.%s.so", onnxVersion))
		}
		return filepath.Join("onnxruntime", fmt.Sprintf("libonnxruntime.%s.so", onnxVersion))
	}
	log.Printf("Error: Unable to determine onnxruntime path for OS=%s Arch=%s\n", runtime.GOOS, runtime.GOARCH)
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
	// Set up logging to both console and file for debugging
	logFile, err := os.OpenFile("mathrex_debug.log", os.O_CREATE|os.O_WRONLY|os.O_APPEND, 0666)
	if err != nil {
		log.Printf("Warning: Could not create log file: %v", err)
	} else {
		defer logFile.Close()
		log.SetOutput(io.MultiWriter(os.Stdout, logFile))
	}

	log.Println("=== MathReX Starting ===")
	log.Printf("OS: %s, Arch: %s", runtime.GOOS, runtime.GOARCH)
	log.Printf("Go version: %s", runtime.Version())

	// Add panic recovery
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC RECOVERED: %v", r)
			log.Printf("Stack trace: %s", debug.Stack())
			if runtime.GOOS == "windows" {
				log.Println("Press Enter to exit...")
				fmt.Scanln()
			}
		}
	}()

	onExit := func() {
		log.Println("MathReX onExit: Shutting down hotkey manager...")
		ShutdownHotkeyManager()
		log.Println("MathReX onExit: Systray cleanup.")
		log.Println("MathReX application finished.")
	}

	log.Println("Starting systray...")
	systray.Run(onReady, onExit)
}

func onReady() {
	log.Println("=== onReady() called ===")

	// Add error recovery for onReady
	defer func() {
		if r := recover(); r != nil {
			log.Printf("PANIC in onReady: %v", r)
			log.Printf("Stack trace: %s", debug.Stack())
			if runtime.GOOS == "windows" {
				log.Println("onReady panic - keeping app alive for debugging")
				// Don't exit, just log the error
			}
		}
	}()

	log.Println("Loading settings...")
	loadSettings()
	log.Println("Settings loaded successfully")

	// Initialize hotkey manager
	log.Println("Initializing hotkey manager...")
	err := InitializeHotkeyManager()
	if err != nil {
		log.Printf("Warning: Failed to initialize hotkey manager: %v", err)
		// Don't exit on hotkey manager failure
	} else {
		log.Println("Hotkey manager initialized successfully")
	}

	log.Println("Extracting embedded files...")
	extractedLibPath, extractedTokenizerPath, extractedEncoderPath, extractedDecoderPath, err := extractAndGetPaths()

	var tempModelDir string
	var modelsInitialized bool = false

	if err != nil {
		log.Printf("ERROR: Failed to extract embedded files: %v", err)
		if runtime.GOOS == "windows" {
			log.Println("Continuing without ONNX runtime for debugging...")
		} else {
			log.Fatalf("Failed to extract embedded files: %v", err)
		}
	} else {
		log.Printf("Files extracted successfully. Lib: %s", extractedLibPath)

		tempModelDir = filepath.Dir(extractedTokenizerPath)
		defer os.RemoveAll(filepath.Dir(extractedLibPath))
		defer os.RemoveAll(tempModelDir)

		log.Printf("Setting ONNX runtime library path: %s", extractedLibPath)
		onnxruntime.SetSharedLibraryPath(extractedLibPath)

		log.Println("Initializing ONNX runtime environment...")
		if err := onnxruntime.InitializeEnvironment(); err != nil {
			log.Printf("ERROR: ONNX Init fail: %v", err)
			if runtime.GOOS == "windows" {
				log.Println("Continuing without ONNX runtime for debugging...")
			} else {
				log.Fatalf("ONNX Init fail: %v", err)
			}
		} else {
			log.Println("ONNX runtime initialized successfully")

			log.Println("Initializing tokenizer...")
			if err := model_controller.InitTokenizer(extractedTokenizerPath); err != nil {
				log.Printf("ERROR: Tokenizer Init fail: %v", err)
				if runtime.GOOS == "windows" {
					log.Println("Continuing without tokenizer for debugging...")
				} else {
					log.Fatalf("Tokenizer Init fail: %v", err)
				}
			} else {
				log.Println("Tokenizer initialized successfully")

				log.Println("Initializing models...")
				if err := model_controller.InitModels(extractedEncoderPath, extractedDecoderPath); err != nil {
					log.Printf("ERROR: Models Init fail: %v", err)
					if runtime.GOOS == "windows" {
						log.Println("Continuing without models for debugging...")
					} else {
						log.Fatalf("Models Init fail: %v", err)
					}
				} else {
					log.Println("Models initialized successfully")
					modelsInitialized = true
				}
			}
		}
	}

	log.Println("Initializing KaTeX...")
	katexJSData, err := GetEmbeddedKaTeXJS()
	if err != nil {
		log.Printf("ERROR: Failed to get KaTeX JS: %v", err)
		if runtime.GOOS != "windows" {
			log.Fatalf("Failed to get embedded KaTeX JS: %v", err)
		}
	} else {
		model_controller.InitKaTeX(katexJSData)
		log.Println("KaTeX initialized successfully")
	}

	log.Println("Initializing MathML2OMML...")
	mathml2ommlJSData, err := GetEmbeddedMathML2OMMLJS()
	if err != nil || len(mathml2ommlJSData) == 0 {
		log.Printf("ERROR: Failed to get embedded mathml2omml.js: %v or data is empty", err)
		if runtime.GOOS != "windows" {
			log.Fatalf("Failed to get embedded mathml2omml.js: %v or data is empty", err)
		}
	} else {
		model_controller.InitMathML2OMMLJS(mathml2ommlJSData)
		log.Println("MathML2OMML initialized successfully")
	}

	if modelsInitialized {
		log.Println("All core components initialized successfully.")
	} else {
		log.Println("Core components partially initialized (debugging mode).")
	}

	// Set systray icon
	log.Println("Setting up systray...")
	iconData, err := embeddedFS.ReadFile("icon.png")
	if err != nil {
		log.Printf("Warning: Could not read embedded icon: %v", err)
	} else {
		log.Printf("Icon data loaded, size: %d bytes", len(iconData))
		systray.SetIcon(iconData)
		log.Println("Systray icon set successfully.")
	}

	log.Println("Setting systray title and tooltip...")
	systray.SetTitle("MathReX")
	systray.SetTooltip("MathReX - Screenshot to Math")
	log.Println("Systray title and tooltip set")

	log.Println("Adding menu items...")
	mCapture := systray.AddMenuItem("Capture & Recognize", "Capture a screen region via menu")
	log.Println("Added Capture menu item")
	mFromFile := systray.AddMenuItem("Recognize from File...", "Select an image file")
	log.Println("Added From File menu item")
	systray.AddSeparator()
	log.Println("Added separator")

	mOutputFormat := systray.AddMenuItem("Output Format", "Select output format")
	log.Println("Added Output Format menu item")
	mFormatLatex := mOutputFormat.AddSubMenuItemCheckbox("LaTeX", "LaTeX", currentSettings.OutputFormat == "latex")
	mFormatMathML := mOutputFormat.AddSubMenuItemCheckbox("MathML", "MathML", currentSettings.OutputFormat == "mathml")
	log.Println("Added format submenu items")

	systray.AddSeparator()
	mCaptureShortcut = systray.AddMenuItem(fmt.Sprintf("Capture Shortcut: %s", currentSettings.CaptureShortcut), "Current capture shortcut")
	mCaptureShortcut.Disable()
	log.Println("Added capture shortcut display item")

	mSetShortcut := systray.AddMenuItem("Set Capture Shortcut...", "Set a new shortcut for capture")
	log.Println("Added set shortcut menu item")

	systray.AddSeparator()
	mQuit := systray.AddMenuItem("Quit MathReX", "Exit the application")
	log.Println("Added quit menu item")

	// Register the capture hotkey
	log.Println("Registering capture hotkey...")
	registerCaptureHotkey()
	log.Println("Hotkey registration completed")

	log.Println("Starting event loop...")
	go func() {
		defer func() {
			if r := recover(); r != nil {
				log.Printf("PANIC in event loop: %v", r)
				log.Printf("Stack trace: %s", debug.Stack())
			}
		}()

		log.Println("Event loop started successfully")
		for {
			select {
			case <-mCapture.ClickedCh:
				log.Println("Capture menu clicked")
				go handleCaptureAndRecognize()
			case <-mFromFile.ClickedCh:
				log.Println("From File menu clicked")
				go handleRecognizeFromFile()
			case <-mFormatLatex.ClickedCh:
				log.Println("LaTeX format selected")
				currentSettings.OutputFormat = "latex"
				updateFormatCheckmarks(mFormatLatex, mFormatMathML)
				saveSettings()
			case <-mFormatMathML.ClickedCh:
				log.Println("MathML format selected")
				currentSettings.OutputFormat = "mathml"
				updateFormatCheckmarks(mFormatLatex, mFormatMathML)
				saveSettings()
			case <-mSetShortcut.ClickedCh:
				log.Println("Set Shortcut menu clicked")
				go handleChangeShortcutGUI()
			case <-mQuit.ClickedCh:
				log.Println("Quit menu item clicked. Shutting down...")
				systray.Quit()
				return
			}
		}
	}()

	log.Println("=== onReady() completed successfully ===")
	if runtime.GOOS == "windows" {
		log.Println("Windows: Application should now be visible in system tray")
		log.Println("Windows: If you see this message, the app started successfully!")
		log.Println("Windows: Check your system tray for the MathReX icon")

		// Keep a debug goroutine running to prevent exit
		go func() {
			for {
				time.Sleep(30 * time.Second)
				log.Println("Windows debug: App is still running...")
			}
		}()
	}
}

func registerCaptureHotkey() {
	if currentSettings.CaptureShortcut == "" {
		log.Println("No capture shortcut configured.")
		return
	}

	err := RegisterGlobalHotkey(currentSettings.CaptureShortcut, func() {
		shortcutMutex.Lock()
		if !isSettingShortcut {
			shortcutMutex.Unlock()
			log.Printf("Global hotkey pressed: %s", currentSettings.CaptureShortcut)
			go handleCaptureAndRecognize()
		} else {
			shortcutMutex.Unlock()
			log.Printf("Global hotkey ignored (setting mode): %s", currentSettings.CaptureShortcut)
		}
	})

	if err != nil {
		log.Printf("Failed to register hotkey '%s': %v", currentSettings.CaptureShortcut, err)
	} else {
		log.Printf("Registered hotkey: %s", currentSettings.CaptureShortcut)
	}
}

func updateFormatCheckmarks(latexItem, mathmlItem *systray.MenuItem) {
	latexItem.Uncheck()
	mathmlItem.Uncheck()
	switch currentSettings.OutputFormat {
	case "latex":
		latexItem.Check()
	case "mathml":
		mathmlItem.Check()
	}
}

func handleCaptureAndRecognize() {
	log.Println("Capture & Recognize triggered.")
	tempImagePath := filepath.Join(os.TempDir(), "mathrex_capture.png")
	defer os.Remove(tempImagePath)

	var cmd *exec.Cmd
	var err error

	if runtime.GOOS == "darwin" {
		cmd = exec.Command("screencapture", "-s", tempImagePath)
		err = cmd.Run()
	} else if runtime.GOOS == "linux" {
		cmd = exec.Command("scrot", "-s", tempImagePath)
		err = cmd.Run()
	} else if runtime.GOOS == "windows" {
		log.Println("Attempting full-screen capture for Windows using kbinani/screenshot.")
		n := screenshot.NumActiveDisplays()
		if n <= 0 {
			err = fmt.Errorf("no active displays found for screenshot")
		} else {
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
						go dialog.Message("Captured the full primary screen. For region selection on Windows, please use Win+Shift+S, save the image, and then use 'Recognize from File'.").Title("Windows Capture Note").Info()
					}
				}
			}
		}
	} else {
		log.Println("Unsupported OS for screenshot capture.")
		dialog.Message("Screenshot capture is not supported on this OS.").Title("Error").Error()
		return
	}

	if err != nil {
		if runtime.GOOS != "windows" {
			if _, statErr := os.Stat(tempImagePath); os.IsNotExist(statErr) {
				log.Println("Screenshot selection cancelled or no file created.")
				return
			}
		}
		log.Printf("Failed to execute screenshot command: %v", err)
		dialog.Message(fmt.Sprintf("Failed to capture screenshot: %v", err)).Title("Error").Error()
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
		dialog.Message(fmt.Sprintf("Error selecting file: %v", err)).Title("Error").Error()
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
		dialog.Message(fmt.Sprintf("Failed to read image file: %v", err)).Title("Error").Error()
		return
	}
	if len(imageBytes) == 0 {
		log.Printf("Image file %s is empty.", imagePath)
		dialog.Message(fmt.Sprintf("Image file is empty: %s", imagePath)).Title("Error").Error()
		return
	}

	outputFmt := currentSettings.OutputFormat
	if outputFmt == "omml" {
		outputFmt = "mathml"
		log.Println("Warning: OMML output format encountered unexpectedly, falling back to MathML.")
	}
	resultText, _, err := model_controller.ProcessImagePrediction(imageBytes, outputFmt)
	if err != nil {
		log.Printf("Failed to process image prediction: %v", err)
		dialog.Message(fmt.Sprintf("Failed to process image: %v", err)).Title("Error").Error()
		return
	}

	err = clipboard.WriteAll(resultText)
	if err != nil {
		log.Printf("Failed to copy result to clipboard: %v", err)
		dialog.Message(fmt.Sprintf("Failed to copy to clipboard: %v\n\nResult was:\n%s", err, resultText)).Title("Clipboard Error").Error()
	} else {
		log.Printf("Result (%s) copied to clipboard.", currentSettings.OutputFormat)
		successMessage := fmt.Sprintf("Recognition successful!\nFormat: %s\n\nResult copied to clipboard.", currentSettings.OutputFormat)
		go dialog.Message(successMessage).Title("Success").Info()
	}
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

	// Inform the user
	go func() {
		dialog.Message("Press the new key combination for capture, or press ESC to cancel.").Title("Set New Shortcut").Info()
	}()

	// Start shortcut capture
	captureChan, err := StartGlobalShortcutCapture()
	if err != nil {
		log.Printf("Failed to start shortcut capture: %v", err)
		shortcutMutex.Lock()
		isSettingShortcut = false
		shortcutMutex.Unlock()
		dialog.Message(fmt.Sprintf("Failed to start shortcut capture: %v", err)).Title("Error").Error()
		return
	}

	// Goroutine to wait for the key press or timeout
	go func() {
		defer func() {
			shortcutMutex.Lock()
			isSettingShortcut = false
			shortcutMutex.Unlock()
			StopGlobalShortcutCapture()
			log.Println("Exited shortcut setting mode.")
		}()

		select {
		case newShortcutStr := <-captureChan:
			log.Printf("Received shortcut: %s", newShortcutStr)

			// Check for ESC key
			if strings.Contains(newShortcutStr, "esc") {
				log.Println("Shortcut setting cancelled by user (ESC pressed).")
				dialog.Message("Shortcut setting cancelled.").Title("Cancelled").Info()
				return
			}

			if newShortcutStr == "" {
				log.Println("Failed to convert event to shortcut string.")
				dialog.Message("Could not recognize the pressed key combination. Please try again.").Title("Error").Error()
				return
			}

			log.Printf("New shortcut candidate: %s", newShortcutStr)

			confirmed := dialog.Message(fmt.Sprintf("Set shortcut to: %s?", newShortcutStr)).Title("Confirm Shortcut").YesNo()
			if confirmed {
				// Unregister old hotkey
				UnregisterGlobalHotkey(currentSettings.CaptureShortcut)

				currentSettings.CaptureShortcut = newShortcutStr
				saveSettings()
				log.Printf("New shortcut set and saved: %s", newShortcutStr)

				// Register new hotkey
				registerCaptureHotkey()

				dialog.Message(fmt.Sprintf("Shortcut changed to: %s", newShortcutStr)).Title("Success").Info()
				if mCaptureShortcut != nil {
					mCaptureShortcut.SetTitle(fmt.Sprintf("Capture Shortcut: %s", currentSettings.CaptureShortcut))
				}
			} else {
				log.Println("User did not confirm new shortcut.")
				dialog.Message("Shortcut change cancelled by user.").Title("Cancelled").Info()
			}

		case <-time.After(30 * time.Second):
			log.Println("Shortcut setting timed out.")
			dialog.Message("Shortcut setting timed out.").Title("Timeout").Info()
		}
	}()
}
