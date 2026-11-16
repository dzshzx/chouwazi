// Package screenshots captures the unmodified production UI from a debug iOS simulator build.
//
// Only the debug VM service receives synthetic touch events. The ordinary game
// clock, random draw, localization and privacy navigation produce every screen.
// No service URL, auth token or raw launch/attach log is written or uploaded.
// Flutter is the `flutter` on PATH: run the command under `fvm exec` so that is the .fvmrc SDK.
package screenshots

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"time"

	"howett.net/plist"
)

const (
	bundleID = "com.alex.chouwazi"
	app      = "build/ios/iphonesimulator/Runner.app"
	output   = "build/app-store-screenshots"
)

type device struct {
	label, name   string
	width, height int
}

var devices = []device{
	{"iphone-16-pro", "iPhone 16 Pro", 1206, 2622},
	{"ipad-pro-13-m4", "iPad Pro 13-inch (M4)", 2064, 2752},
}

type language struct {
	tag, region, privacyText, privacyTitle string
}

var languages = []language{
	{"en", "en_US", "Privacy", "Privacy Policy"},
	{"zh-Hans", "zh_CN", "隐私", "隐私政策"},
	{"zh-Hant", "zh_TW", "隱私", "隱私政策"},
}

var (
	vmURLPattern = regexp.MustCompile(`The Dart VM service is listening on (http://127\.0\.0\.1:\d+/[A-Za-z0-9=_-]+/)`)
	udidPattern  = regexp.MustCompile(`^[A-Fa-f0-9-]{36}$`)
)

// Screenshot is one captured screen in the public manifest.
type Screenshot struct {
	File        string `json:"file"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Mode        string `json:"mode"`
	SHA256      string `json:"sha256"`
	StateBefore State  `json:"state_before"`
	StateAfter  State  `json:"state_after"`
}

// Locale is the capture of all screens in one language.
type Locale struct {
	Language     string       `json:"language"`
	ActualLocale string       `json:"actual_locale"`
	Screenshots  []Screenshot `json:"screenshots"`
}

// Device is the capture on one simulator model.
type Device struct {
	Name           string   `json:"name"`
	DeviceType     string   `json:"device_type"`
	Runtime        string   `json:"runtime"`
	RuntimeVersion string   `json:"runtime_version"`
	RuntimeBuild   string   `json:"runtime_build"`
	Locales        []Locale `json:"locales"`
}

// Manifest is the public description of the screenshot set.
type Manifest struct {
	SourceSHA   string          `json:"source_sha"`
	Version     any             `json:"version"`
	Build       any             `json:"build"`
	BundleID    string          `json:"bundle_id"`
	BuildMode   string          `json:"build_mode"`
	Capture     string          `json:"capture"`
	Interaction string          `json:"interaction"`
	Screens     []string        `json:"screens"`
	Xcode       string          `json:"xcode"`
	Flutter     json.RawMessage `json:"flutter"`
	Devices     []Device        `json:"devices"`
}

// VMURL extracts the loopback VM service URL from a unified log line.
func VMURL(line string) (string, bool) {
	match := vmURLPattern.FindStringSubmatch(line)
	if match == nil {
		return "", false
	}
	return match[1], true
}

// machineMessages returns the JSON message list on one `flutter --machine` output line.
func machineMessages(line string) ([]any, bool) {
	var messages any
	if err := json.Unmarshal([]byte(line), &messages); err != nil {
		return nil, false
	}
	list, ok := messages.([]any)
	return list, ok
}

// AttachReadyParser parses `flutter attach --machine` output until the app
// started and its debug URI is known.
func AttachReadyParser() func(string) (string, bool) {
	uri, started := "", false
	return func(line string) (string, bool) {
		messages, ok := machineMessages(line)
		if !ok {
			return "", false
		}
		for _, message := range messages {
			switch text(get(message, "event")) {
			case "app.debugPort":
				uri = text(get(get(message, "params"), "wsUri"))
			case "app.started":
				started = true
			}
		}
		return uri, started && uri != ""
	}
}

func pollUntil(condition func() (bool, error), timeout time.Duration, message string) error {
	deadline := time.Now().Add(timeout)
	for {
		done, err := condition()
		if err != nil || done {
			return err
		}
		if err := require(time.Now().Before(deadline), message); err != nil {
			return err
		}
		time.Sleep(200 * time.Millisecond)
	}
}

type session struct {
	vm     *vm
	udid   string
	folder string
	device device
	lang   language
	shots  []Screenshot
	stdout io.Writer
}

func (s *session) capture(name string, expected func(State) bool) error {
	operation := map[string]string{"01-home": "home", "02-result": "result", "03-privacy": "privacy"}[name]
	before, err := s.vm.snapshot(operation + "-before")
	if err != nil {
		return err
	}
	if err := require(expected(before), "Unexpected game state before "+name); err != nil {
		return err
	}
	path := filepath.Join(s.folder, name+".png")
	if _, err := run120("xcrun", "simctl", "io", s.udid, "screenshot", "--type=png", "--mask=ignored", path); err != nil {
		return err
	}
	after, err := s.vm.snapshot(operation + "-after")
	if err != nil {
		return err
	}
	if err := require(expected(after), "Game state changed during "+name+"; refusing to label this screenshot"); err != nil {
		return err
	}
	digest, err := OpaquePNG(path, s.device.width, s.device.height)
	if err != nil {
		return err
	}
	relative, err := filepath.Rel(output, path)
	if err != nil {
		return err
	}
	s.shots = append(s.shots, Screenshot{File: filepath.ToSlash(relative), Width: s.device.width, Height: s.device.height,
		Mode: "RGB", SHA256: digest, StateBefore: before, StateAfter: after})
	fmt.Fprintf(s.stdout, "Captured %s (%d × %d, RGB)\n", filepath.ToSlash(relative), s.device.width, s.device.height)
	return nil
}

// Expected game states for each stable screen.
func homeState(state State) bool { return !truthy(state.Privacy) && intEquals(state.Players, 0) }

func resultState(state State) bool {
	slot := false
	for index := range int64(4) {
		slot = slot || intEquals(state.WinnerSlot, index)
	}
	return !truthy(state.Privacy) && intEquals(state.Players, 4) && slot && truthy(state.AllReleased)
}

func privacyState(state State) bool { return truthy(state.Privacy) && truthy(state.AllReleased) }

// geometry returns the logical width, height and top and bottom padding of the view.
func (s *session) geometry() ([4]float64, error) {
	var values [4]float64
	printed, err := s.vm.string(geometryExpression, "geometry")
	if err != nil {
		return values, err
	}
	var decoded []float64
	if err := json.Unmarshal([]byte(printed), &decoded); err != nil || len(decoded) != 4 {
		return values, errors.New("invalid view geometry")
	}
	copy(values[:], decoded)
	return values, nil
}

func (s *session) playRound() error {
	geometry, err := s.geometry()
	if err != nil {
		return err
	}
	width, height, top, bottom := geometry[0], geometry[1], geometry[2], geometry[3]
	content := height - top - bottom
	var positions []touch
	for index, point := range [][2]float64{{.24, .28}, {.76, .28}, {.24, .73}, {.76, .73}} {
		positions = append(positions, touch{index + 1, width * point[0], top + content*point[1]})
	}
	if err := s.vm.touches("Down", positions, "players-down"); err != nil {
		return err
	}
	// Wait for the ordinary 3.8-second round and its random draw to finish.
	if err := pollUntil(func() (bool, error) {
		state, err := s.vm.snapshot("winner-poll")
		return state.WinnerSlot != nil, err
	}, 10*time.Second, "The normal game clock did not produce a result"); err != nil {
		return err
	}
	if err := s.vm.touches("Up", positions, "players-up"); err != nil {
		return err
	}
	if err := pollUntil(func() (bool, error) {
		state, err := s.vm.snapshot("players-up")
		return truthy(state.AllReleased), err
	}, 10*time.Second, "The normal pointer releases did not finish"); err != nil {
		return err
	}
	if err := s.capture("02-result", resultState); err != nil {
		return err
	}
	return s.openPrivacy(top)
}

func (s *session) openPrivacy(top float64) error {
	// Complete an ordinary touch on the existing privacy button, including its release.
	privacy := []touch{{20, 30, top + 22}}
	if err := s.vm.touches("Down", privacy, "privacy-down"); err != nil {
		return err
	}
	if err := s.vm.touches("Up", privacy, "privacy-up"); err != nil {
		return err
	}
	const titleExpression = `
          (context as Element).toStringDeep().contains('PrivacyScreen') &&
          (context as Element).toStringDeep().contains(_strings.text('privacy_title')) &&
          (context as Element).toStringDeep().contains(_strings.text('privacy_summary'))
        `
	return pollUntil(func() (bool, error) {
		state, err := s.vm.snapshot("privacy-rendered")
		if err != nil || !truthy(state.Privacy) {
			return false, err
		}
		rendered, err := s.vm.string(titleExpression, "privacy-rendered")
		if err != nil || rendered != "true" {
			return false, err
		}
		title, err := s.vm.string("_strings.text('privacy_title')", "privacy-title")
		return title == s.lang.privacyTitle, err
	}, 15*time.Second, "The real localized privacy screen did not render")
}

func (s *session) screens() (Locale, error) {
	language := s.lang
	locale, err := s.vm.string("WidgetsBinding.instance.platformDispatcher.locale.toLanguageTag()", "locale")
	if err != nil {
		return Locale{}, err
	}
	label, err := s.vm.string("_strings.text('privacy')", "privacy-label")
	if err != nil {
		return Locale{}, err
	}
	if err := require(label == language.privacyText, "System locale did not select "+language.tag); err != nil {
		return Locale{}, err
	}
	banner, err := s.vm.string(noDebugBanner, "debug-banner")
	if err != nil {
		return Locale{}, err
	}
	if err := require(banner == "true", "The app is displaying its debug banner"); err != nil {
		return Locale{}, err
	}
	if err := s.capture("01-home", homeState); err != nil {
		return Locale{}, err
	}
	if err := s.playRound(); err != nil {
		return Locale{}, err
	}
	if err := s.capture("03-privacy", privacyState); err != nil {
		return Locale{}, err
	}
	return Locale{Language: language.tag, ActualLocale: locale, Screenshots: s.shots}, nil
}

type resources struct {
	logs, attach *privateProcess
	vm           *vm
}

func (r *resources) steps(udid string) []cleanupStep {
	var steps []cleanupStep
	if r.vm != nil {
		steps = append(steps, cleanupStep{"VM connection", r.vm.close})
	}
	if r.attach != nil {
		steps = append(steps, cleanupStep{"Flutter attach", r.attach.close})
	}
	if r.logs != nil {
		steps = append(steps, cleanupStep{"Simulator unified log", r.logs.close})
	}
	// Flutter may already have terminated the app.
	return append(steps, cleanupStep{"Simulator app termination", func() error {
		_, err := run(30*time.Second, "xcrun", "simctl", "terminate", udid, bundleID)
		var public Error
		if errors.As(err, &public) {
			return nil
		}
		return err
	}})
}

func (r *resources) connect(udid string, language language, stdout io.Writer) error {
	var err error
	// Flutter discovers iOS VM announcements through unified logging. Listen
	// before launching so the startup message is not missed.
	if r.logs, err = startPrivate("xcrun", "simctl", "spawn", udid, "log", "stream", "--style", "compact",
		"--predicate", `eventType = logEvent AND processImagePath ENDSWITH "/Runner"`); err != nil {
		return err
	}
	if _, err := r.logs.waitFor(func(line string) (string, bool) {
		return "", strings.HasPrefix(line, "Filtering the log data using ")
	}, "Simulator unified log startup", 30*time.Second); err != nil {
		return err
	}
	if _, err := run120("xcrun", "simctl", "launch", udid, bundleID, "--enable-checked-mode", "--verify-entry-points",
		"--disable-vm-service-publication", "-AppleLanguages", "("+language.tag+")", "-AppleLocale", language.region); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Simulator launch acknowledged; waiting for private unified log VM announcement")
	uri, err := r.logs.waitFor(VMURL, "Simulator debug service startup", 180*time.Second)
	if err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Simulator loopback VM service discovered; attaching Flutter")
	if r.attach, err = startPrivate("flutter", "attach", "--machine", "--debug", "--no-dds", "--no-devtools",
		"-d", udid, "--debug-url", uri); err != nil {
		return err
	}
	debugURI, err := r.attach.waitFor(AttachReadyParser(), "Flutter debug attach", 180*time.Second)
	if err != nil {
		return err
	}
	time.Sleep(time.Second)
	r.vm, err = connect(debugURI)
	return err
}

func captureLanguage(udid string, model device, language language, stdout, stderr io.Writer) (locale Locale, err error) {
	folder := filepath.Join(output, model.label, language.tag)
	if err := os.MkdirAll(filepath.Dir(folder), 0o755); err != nil {
		return Locale{}, err
	}
	if err := os.Mkdir(folder, 0o755); err != nil {
		return Locale{}, err
	}
	var held resources
	defer func() { err = cleanup(held.steps(udid), err, stderr) }()
	if err := held.connect(udid, language, stdout); err != nil {
		return Locale{}, err
	}
	s := &session{vm: held.vm, udid: udid, folder: folder, device: model, lang: language, stdout: stdout}
	return s.screens()
}

type simDevice struct {
	Name       string `json:"name"`
	Identifier string `json:"identifier"`
}

type simRuntime struct {
	Name           string      `json:"name"`
	Identifier     string      `json:"identifier"`
	Version        string      `json:"version"`
	BuildVersion   string      `json:"buildversion"`
	IsAvailable    bool        `json:"isAvailable"`
	SupportedTypes []simDevice `json:"supportedDeviceTypes"`
}

type inventory struct {
	DeviceTypes []simDevice  `json:"devicetypes"`
	Runtimes    []simRuntime `json:"runtimes"`
}

func versionKey(version string) ([]int, error) {
	var key []int
	for _, part := range strings.Split(version, ".") {
		number, err := strconv.Atoi(part)
		if err != nil {
			return nil, err
		}
		key = append(key, number)
	}
	return key, nil
}

func newerVersion(a, b []int) bool {
	for index := 0; index < len(a) && index < len(b); index++ {
		if a[index] != b[index] {
			return a[index] > b[index]
		}
	}
	return len(a) > len(b)
}

// SelectSimulators picks the required device types and the newest available iOS
// runtime that supports all of them. A missing model is a failure, never a fallback.
func SelectSimulators(listed inventory) ([]simDevice, simRuntime, error) {
	var selected []simDevice
	for _, model := range devices {
		var matches []simDevice
		for _, item := range listed.DeviceTypes {
			if item.Name == model.name {
				matches = append(matches, item)
			}
		}
		if err := require(len(matches) == 1, "Required simulator device type is missing: "+model.name); err != nil {
			return nil, simRuntime{}, err
		}
		selected = append(selected, matches[0])
	}
	var best simRuntime
	var bestKey []int
	for _, candidate := range listed.Runtimes {
		if !candidate.IsAvailable || !strings.HasPrefix(candidate.Identifier, "com.apple.CoreSimulator.SimRuntime.iOS-") ||
			!supportsAll(candidate, selected) {
			continue
		}
		key, err := versionKey(candidate.Version)
		if err != nil {
			return nil, simRuntime{}, errors.New("invalid runtime version")
		}
		if bestKey == nil || newerVersion(key, bestKey) {
			best, bestKey = candidate, key
		}
	}
	if err := require(bestKey != nil, "No installed iOS runtime supports both required simulator models"); err != nil {
		return nil, simRuntime{}, err
	}
	return selected, best, nil
}

func supportsAll(candidate simRuntime, selected []simDevice) bool {
	for _, item := range selected {
		found := false
		for _, supported := range candidate.SupportedTypes {
			found = found || supported.Identifier == item.Identifier
		}
		if !found {
			return false
		}
	}
	return true
}

func startManifest() (Manifest, error) {
	raw, err := os.ReadFile(filepath.Join(app, "Info.plist"))
	if err != nil {
		return Manifest{}, err
	}
	var info map[string]any
	if _, err := plist.Unmarshal(raw, &info); err != nil {
		return Manifest{}, err
	}
	platforms, _ := info["CFBundleSupportedPlatforms"].([]any)
	if err := require(info["CFBundleIdentifier"] == bundleID && len(platforms) == 1 && platforms[0] == "iPhoneSimulator",
		"Requires the real Runner simulator bundle"); err != nil {
		return Manifest{}, err
	}
	head, err := run120("git", "rev-parse", "HEAD")
	if err != nil {
		return Manifest{}, err
	}
	xcode, err := run120("xcodebuild", "-version")
	if err != nil {
		return Manifest{}, err
	}
	flutter, err := run120("flutter", "--version", "--machine")
	if err != nil {
		return Manifest{}, err
	}
	if !json.Valid([]byte(flutter)) {
		return Manifest{}, errors.New("invalid flutter version output")
	}
	version, okVersion := info["CFBundleShortVersionString"]
	build, okBuild := info["CFBundleVersion"]
	if !okVersion || !okBuild {
		return Manifest{}, errors.New("missing app version")
	}
	return Manifest{SourceSHA: strings.TrimSpace(head), Version: version, Build: build, BundleID: bundleID,
		BuildMode: "debug", Capture: "simctl native screen",
		Interaction: "Flutter touch events via loopback debug VM; normal game clock and random draw",
		Screens:     []string{"01-home", "02-result", "03-privacy"}, Xcode: strings.TrimSpace(xcode),
		Flutter: json.RawMessage(flutter), Devices: []Device{}}, nil
}

func captureDevice(model device, kind simDevice, chosen simRuntime, stdout, stderr io.Writer) (record Device, err error) {
	created, err := run120("xcrun", "simctl", "create", "chouwazi-screenshots-"+model.label, kind.Identifier, chosen.Identifier)
	if err != nil {
		return Device{}, err
	}
	udid := strings.TrimSpace(created)
	if err := require(udidPattern.MatchString(udid), "simctl did not return a simulator identifier"); err != nil {
		return Device{}, err
	}
	fmt.Fprintf(stdout, "Starting %s for %d × %d screenshots\n", kind.Name, model.width, model.height)
	defer func() {
		// A failed boot can leave an already-shutdown device to remove.
		err = cleanup([]cleanupStep{
			{"Simulator shutdown", func() error {
				_, err := run(60*time.Second, "xcrun", "simctl", "shutdown", udid)
				var public Error
				if errors.As(err, &public) {
					return nil
				}
				return err
			}},
			{"Simulator deletion", func() error { _, err := run120("xcrun", "simctl", "delete", udid); return err }},
		}, err, stderr)
	}()
	for _, step := range [][]string{
		{"xcrun", "simctl", "boot", udid},
		{"xcrun", "simctl", "status_bar", udid, "override", "--time", "9:41", "--dataNetwork", "wifi", "--wifiMode", "active",
			"--wifiBars", "3", "--batteryState", "charged", "--batteryLevel", "100"},
		{"xcrun", "simctl", "install", udid, app},
	} {
		if _, err := run120(step...); err != nil {
			return Device{}, err
		}
		if step[2] == "boot" {
			if _, err := run(240*time.Second, "xcrun", "simctl", "bootstatus", udid, "-b"); err != nil {
				return Device{}, err
			}
		}
	}
	record = Device{Name: kind.Name, DeviceType: kind.Identifier, Runtime: chosen.Identifier,
		RuntimeVersion: chosen.Version, RuntimeBuild: chosen.BuildVersion}
	for _, language := range languages {
		locale, err := captureLanguage(udid, model, language, stdout, stderr)
		if err != nil {
			return Device{}, err
		}
		record.Locales = append(record.Locales, locale)
	}
	return record, nil
}

func countPNG(root string) (int, error) {
	count := 0
	err := filepath.WalkDir(root, func(path string, entry fs.DirEntry, err error) error {
		if err == nil && strings.HasSuffix(entry.Name(), ".png") {
			count++
		}
		return err
	})
	return count, err
}

func capture(stdout, stderr io.Writer) error {
	if err := require(runtime.GOOS == "darwin", "iOS capture requires macOS with Xcode and a simulator runtime"); err != nil {
		return err
	}
	if _, err := os.Lstat(output); err == nil {
		return Error("Screenshot output already exists; use a fresh build directory")
	}
	manifest, err := startManifest()
	if err != nil {
		return err
	}
	listed, err := run120("xcrun", "simctl", "list", "-j")
	if err != nil {
		return err
	}
	var parsed inventory
	if err := json.Unmarshal([]byte(listed), &parsed); err != nil {
		return err
	}
	kinds, chosen, err := SelectSimulators(parsed)
	if err != nil {
		return err
	}
	fmt.Fprintf(stdout, "Selected installed runtime: %s (%s, %s)\n", chosen.Name, chosen.Version, chosen.BuildVersion)
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}
	for index, model := range devices {
		record, err := captureDevice(model, kinds[index], chosen, stdout, stderr)
		if err != nil {
			return err
		}
		manifest.Devices = append(manifest.Devices, record)
	}
	count, err := countPNG(output)
	if err != nil {
		return err
	}
	if err := require(count == 18, "Expected all three stable screens in three languages on both devices"); err != nil {
		return err
	}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(manifest); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(output, "manifest.json"), encoded.Bytes(), 0o644); err != nil {
		return err
	}
	fmt.Fprintln(stdout, "Verified 18 native RGB PNG screenshots and a public manifest.")
	return nil
}

// Main captures the App Store screenshots from the repository root.
func Main(args []string, stdout, stderr io.Writer) int {
	if len(args) != 0 {
		fmt.Fprintln(stderr, "ios-screenshots takes no arguments")
		return 2
	}
	err := capture(stdout, stderr)
	if err == nil {
		return 0
	}
	var public Error
	if errors.As(err, &public) {
		fmt.Fprintf(stderr, "Screenshot capture failed: %s\n", public)
	} else {
		// External errors may embed an authenticated URL. Keep those messages private.
		fmt.Fprintln(stderr, "Screenshot capture failed; check simulator, Flutter attach and screenshot prerequisites. Debug output was kept private.")
	}
	return 1
}
