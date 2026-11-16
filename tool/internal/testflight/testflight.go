// Package testflight handles signing assets without printing certificate, profile or key contents.
package testflight

import (
	"bytes"
	"crypto/rand"
	"crypto/sha1"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"regexp"
	"slices"
	"strconv"
	"strings"
	"syscall"
	"time"

	"howett.net/plist"
)

// BundleID is the iOS bundle identifier of the app.
const BundleID = "com.alex.chouwazi"

const project = "ios/Runner.xcodeproj/project.pbxproj"

var (
	teamPattern      = regexp.MustCompile(`^[A-Z0-9]{10}$`)
	uuidPattern      = regexp.MustCompile(`^[A-Fa-f0-9-]{36}$`)
	issuerPattern    = regexp.MustCompile(`^[A-Fa-f0-9]{8}(?:-[A-Fa-f0-9]{4}){3}-[A-Fa-f0-9]{12}$`)
	versionPattern   = regexp.MustCompile(`^\d+\.\d+\.\d+$`)
	buildPattern     = regexp.MustCompile(`^[1-9]\d*$`)
	identityPattern  = regexp.MustCompile(`\b([A-F0-9]{40})\s+"([^"\n]+)"`)
	sdkPattern       = regexp.MustCompile(`^iphoneos(?:2[6-9]|[3-9]\d)\.\d+(?:\.\d+)?$`)
	profileName      = regexp.MustCompile(`^[A-Fa-f0-9-]{36}\.mobileprovision$`)
	subcomponentLine = regexp.MustCompile(`(?m)^In subcomponent:[ \t]*([^\r\n]+)`)
	componentName    = regexp.MustCompile(`(?:^|/)(Runner\.app|Flutter\.framework|App\.framework)/?$`)
)

// ValidationError carries an audited, secret-free public message.
type ValidationError string

func (e ValidationError) Error() string { return string(e) }

func require(condition bool, message string) error {
	if condition {
		return nil
	}
	return ValidationError(message)
}

var codesignOperations = [][2]string{{"--verify", "verify"}, {"--entitlements", "display-entitlements"},
	{"--extract-certificates", "extract-certificates"}}

var codesignReasons = []struct {
	name     string
	patterns []string
}{
	{"unsigned", []string{"code object is not signed at all"}},
	{"sealed-resource-missing-or-invalid", []string{"sealed resource is missing or invalid"}},
	{"main-executable-strict-validation", []string{"main executable failed strict validation"}},
	{"unrecognized-or-ambiguous-bundle", []string{"bundle format unrecognized", "bundle format is ambiguous"}},
	{"invalid-signature", []string{"invalid signature", "signature invalid", "code or signature have been modified",
		"cssmerr_csp_verify_failed"}},
	{"trust-or-certificate-expiry", []string{"cssmerr_tp_cert_expired", "cssmerr_tp_not_trusted",
		"cssmerr_tp_cert_not_valid_yet", "errseccertificateexpired", "certificate has expired",
		"certificate is not trusted", "unable to build chain to self-signed root"}},
	{"unsupported-syntax-or-argument", []string{"invalid argument", "unknown option", "unrecognized option",
		"unsupported option", "invalid option", "illegal option", "requires an argument", "usage: codesign"}},
	{"file-or-path-not-found", []string{"no such file", "file not found", "path not found"}},
}

func codesignOperation(args []string) string {
	for _, operation := range codesignOperations {
		for _, arg := range args[1:] {
			if arg == operation[0] || strings.HasPrefix(arg, operation[0]+"=") {
				return operation[1]
			}
		}
	}
	return "unknown"
}

func codesignReason(stderr string) string {
	lowered := strings.ToLower(stderr)
	for _, reason := range codesignReasons {
		for _, pattern := range reason.patterns {
			if strings.Contains(lowered, pattern) {
				return reason.name
			}
		}
	}
	return "unknown"
}

// CodesignFailure builds the public error for a failed codesign call. stderr is
// untrusted: only fixed operation, reason and component tokens are public.
func CodesignFailure(args []string, exitCode int, stderr []byte) error {
	text := strings.ToValidUTF8(string(stderr), "�")
	component := "none"
	if subcomponents := subcomponentLine.FindAllStringSubmatch(text, -1); subcomponents != nil {
		component = "unknown"
		if match := componentName.FindStringSubmatch(strings.TrimSpace(subcomponents[0][1])); match != nil {
			component = match[1]
		}
	}
	return ValidationError(fmt.Sprintf("codesign %s failed while handling signing assets (exit=%d, reason=%s, subcomponent=%s)",
		codesignOperation(args), exitCode, codesignReason(text), component))
}

func run(args ...string) ([]byte, error) {
	command := exec.Command(args[0], args[1:]...)
	var stdout, stderr bytes.Buffer
	command.Stdout, command.Stderr = &stdout, &stderr
	err := command.Run()
	var exit *exec.ExitError
	if err != nil && !errors.As(err, &exit) {
		return nil, errors.New("command could not start")
	}
	code := command.ProcessState.ExitCode()
	if args[0] == "codesign" && code != 0 {
		return nil, CodesignFailure(args, code, stderr.Bytes())
	}
	if code != 0 {
		return nil, ValidationError(args[0] + " failed while handling signing assets")
	}
	return stdout.Bytes(), nil
}

func isDecimal(value string) bool {
	if value == "" {
		return false
	}
	for _, char := range value {
		if char < '0' || char > '9' {
			return false
		}
	}
	return true
}

func isSymlink(path string) bool {
	info, err := os.Lstat(path)
	return err == nil && info.Mode()&os.ModeSymlink != 0
}

func exists(path string) bool {
	_, err := os.Stat(path)
	return err == nil
}

func lexists(path string) bool {
	_, err := os.Lstat(path)
	return err == nil
}

// StateDirectory is the per-run signing state directory under RUNNER_TEMP.
func StateDirectory() (string, error) {
	if err := require(os.Getenv("GITHUB_ACTIONS") == "true", "Requires a GitHub-hosted workflow"); err != nil {
		return "", err
	}
	runID, okRun := os.LookupEnv("GITHUB_RUN_ID")
	attempt, okAttempt := os.LookupEnv("GITHUB_RUN_ATTEMPT")
	temp, okTemp := os.LookupEnv("RUNNER_TEMP")
	if !okRun || !okAttempt || !okTemp {
		return "", errors.New("missing workflow environment")
	}
	if err := require(isDecimal(runID) && isDecimal(attempt), "Invalid workflow run identifier"); err != nil {
		return "", err
	}
	runnerTemp, err := filepath.Abs(temp)
	if err != nil {
		return "", err
	}
	if resolved, err := filepath.EvalSymlinks(runnerTemp); err == nil {
		runnerTemp = resolved
	}
	state := filepath.Join(runnerTemp, fmt.Sprintf("chouwazi-testflight-%s-%s", runID, attempt))
	if err := require(!isSymlink(state), "Signing state cannot be a symlink"); err != nil {
		return "", err
	}
	return state, nil
}

// Settings is the signing state shared with testflight.sh and its Ruby step.
type Settings struct {
	TeamID           string   `json:"team_id"`
	AppID            string   `json:"app_id"`
	ProfileUUID      string   `json:"profile_uuid"`
	Version          string   `json:"version"`
	Build            string   `json:"build"`
	APIKeyID         string   `json:"api_key_id"`
	APIIssuerID      string   `json:"api_issuer_id"`
	CertificateSHA1s []string `json:"certificate_sha1s"`
	CertificateSHA1  string   `json:"certificate_sha1,omitempty"`
}

func readSettings(state string) (Settings, error) {
	var data Settings
	raw, err := os.ReadFile(filepath.Join(state, "settings.json"))
	if err != nil {
		return data, err
	}
	return data, json.Unmarshal(raw, &data)
}

func writeSettings(state string, data Settings) error {
	encoded, err := json.Marshal(data)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(state, "settings.json"), encoded, 0o600)
}

func stringList(value any) []any {
	list, _ := value.([]any)
	return list
}

func matchesString(pattern *regexp.Regexp, value any) bool {
	text, ok := value.(string)
	return ok && pattern.MatchString(text)
}

func truthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != ""
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	case []byte:
		return len(typed) > 0
	case uint64:
		return typed != 0
	case int64:
		return typed != 0
	case float64:
		return typed != 0
	}
	return true
}

// Profile is a decoded and validated App Store provisioning profile.
type Profile struct {
	TeamID       string
	AppID        string
	UUID         string
	Certificates [][]byte
}

func decodePlist(data []byte) (map[string]any, error) {
	var decoded any
	if _, err := plist.Unmarshal(data, &decoded); err != nil {
		return nil, err
	}
	object, ok := decoded.(map[string]any)
	if !ok {
		return nil, errors.New("plist is not a dictionary")
	}
	return object, nil
}

func dict(value any) map[string]any {
	object, _ := value.(map[string]any)
	return object
}

// ValidateProfile checks a decoded provisioning profile against the App Store distribution rules.
func ValidateProfile(profile map[string]any, now time.Time) (Profile, error) {
	teams, prefixes := stringList(profile["TeamIdentifier"]), stringList(profile["ApplicationIdentifierPrefix"])
	checks := []struct {
		ok      func() bool
		message string
	}{
		{func() bool { return len(teams) == 1 && matchesString(teamPattern, teams[0]) }, "Invalid profile team"},
		{func() bool { return len(prefixes) == 1 && matchesString(teamPattern, prefixes[0]) }, "Invalid App ID prefix"},
		{func() bool { return matchesString(uuidPattern, profile["UUID"]) }, "Invalid profile UUID"},
	}
	for _, check := range checks {
		if err := require(check.ok(), check.message); err != nil {
			return Profile{}, err
		}
	}
	team, prefix := teams[0].(string), prefixes[0].(string)
	entitlements := dict(profile["Entitlements"])
	expires, isTime := profile["ExpirationDate"].(time.Time)
	_, hasDevices := profile["ProvisionedDevices"]
	rules := []struct {
		ok      bool
		message string
	}{
		{entitlements["application-identifier"] == prefix+"."+BundleID, "Profile App ID does not match Runner"},
		{entitlements["com.apple.developer.team-identifier"] == team, "Profile team entitlement mismatch"},
		{entitlements["get-task-allow"] == false, "A development profile cannot sign this upload"},
		{entitlements["beta-reports-active"] == true, "Requires an App Store distribution profile"},
		{!hasDevices && !truthy(profile["ProvisionsAllDevices"]), "Ad hoc/enterprise profiles cannot be used"},
		{isTime, "Missing profile expiry"},
		{!isTime || expires.After(now), "Provisioning profile has expired"},
		{truthy(profile["DeveloperCertificates"]), "Profile contains no distribution certificate"},
	}
	for _, rule := range rules {
		if err := require(rule.ok, rule.message); err != nil {
			return Profile{}, err
		}
	}
	var certificates [][]byte
	for _, item := range stringList(profile["DeveloperCertificates"]) {
		certificate, ok := item.([]byte)
		if !ok {
			return Profile{}, errors.New("certificate is not data")
		}
		certificates = append(certificates, certificate)
	}
	return Profile{TeamID: team, AppID: prefix + "." + BundleID, UUID: profile["UUID"].(string), Certificates: certificates}, nil
}

func profileData(blob string) (Profile, error) {
	decoded, err := run("security", "cms", "-D", "-i", blob)
	if err != nil {
		return Profile{}, err
	}
	profile, err := decodePlist(decoded)
	if err != nil {
		return Profile{}, err
	}
	return ValidateProfile(profile, time.Now())
}

func sha1Hex(data []byte) string {
	sum := sha1.Sum(data)
	return strings.ToUpper(hex.EncodeToString(sum[:]))
}

var secretNames = []string{"IOS_DISTRIBUTION_P12_BASE64", "IOS_DISTRIBUTION_P12_PASSWORD",
	"IOS_APP_STORE_PROFILE_BASE64", "ASC_API_KEY_P8", "ASC_API_KEY_ID", "ASC_API_ISSUER_ID"}

func writeSigningBlobs(state string) error {
	for _, asset := range [][2]string{{secretNames[0], "distribution.p12"}, {secretNames[2], "app.mobileprovision"}} {
		blob, err := base64.StdEncoding.DecodeString(strings.Join(strings.Fields(os.Getenv(asset[0])), ""))
		if err != nil {
			return errors.New("invalid base64 signing asset")
		}
		if err := require(len(blob) > 0, "Empty signing asset"); err != nil {
			return err
		}
		if err := os.WriteFile(filepath.Join(state, asset[1]), blob, 0o600); err != nil {
			return err
		}
	}
	return nil
}

func writePrivateKey(state, keyID string) error {
	privateKeys := filepath.Join(state, "private_keys")
	if err := os.Mkdir(privateKeys, 0o700); err != nil {
		return err
	}
	privateKey := filepath.Join(privateKeys, "AuthKey_"+keyID+".p8")
	if err := os.WriteFile(privateKey, []byte(os.Getenv("ASC_API_KEY_P8")), 0o600); err != nil {
		return err
	}
	_, err := run("openssl", "pkey", "-in", privateKey, "-noout")
	return err
}

func prepare(state string) error {
	all := true
	for _, name := range secretNames {
		all = all && os.Getenv(name) != ""
	}
	if err := require(all, "Required TestFlight environment secrets are missing"); err != nil {
		return err
	}
	if err := require(!exists(state), "Signing state already exists; cleanup the previous attempt first"); err != nil {
		return err
	}
	if err := os.Mkdir(state, 0o700); err != nil {
		return err
	}
	if err := writeSigningBlobs(state); err != nil {
		return err
	}
	profile, err := profileData(filepath.Join(state, "app.mobileprovision"))
	if err != nil {
		return err
	}
	keyID, issuer := os.Getenv("ASC_API_KEY_ID"), os.Getenv("ASC_API_ISSUER_ID")
	if err := require(teamPattern.MatchString(keyID), "Invalid App Store Connect key ID"); err != nil {
		return err
	}
	if err := require(issuerPattern.MatchString(issuer), "Invalid App Store Connect issuer ID"); err != nil {
		return err
	}
	if err := writePrivateKey(state, keyID); err != nil {
		return err
	}
	// release.yml computes these from the release tags; see scripts/release-version.sh.
	version, build := os.Getenv("RELEASE_VERSION"), os.Getenv("RELEASE_BUILD")
	if err := require(versionPattern.MatchString(version) && buildPattern.MatchString(build),
		"Expected RELEASE_VERSION x.y.z and a positive RELEASE_BUILD"); err != nil {
		return err
	}
	data := Settings{TeamID: profile.TeamID, AppID: profile.AppID, ProfileUUID: profile.UUID, Version: version,
		Build: build, APIKeyID: keyID, APIIssuerID: issuer, CertificateSHA1s: []string{}}
	for _, certificate := range profile.Certificates {
		data.CertificateSHA1s = append(data.CertificateSHA1s, sha1Hex(certificate))
	}
	if err := writeSettings(state, data); err != nil {
		return err
	}
	return saveRestorePoints(state)
}

func saveRestorePoints(state string) error {
	password := make([]byte, 32)
	if _, err := rand.Read(password); err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(state, "keychain-password"), []byte(base64.RawURLEncoding.EncodeToString(password)), 0o600); err != nil {
		return err
	}
	listed, err := run("security", "list-keychains", "-d", "user")
	if err != nil {
		return err
	}
	keychains, err := ShellSplit(string(listed))
	if err != nil {
		return err
	}
	encoded, err := json.Marshal(keychains)
	if err != nil {
		return err
	}
	if err := os.WriteFile(filepath.Join(state, "keychains.json"), encoded, 0o600); err != nil {
		return err
	}
	original, err := os.ReadFile(project)
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(state, "project.pbxproj"), original, 0o600)
}

// ShellSplit splits text into words with POSIX shell quoting rules.
func ShellSplit(text string) ([]string, error) {
	words := []string{}
	var word strings.Builder
	inWord := false
	quote := rune(0)
	escaped := false
	for _, char := range text {
		switch {
		case escaped:
			if quote == '"' && char != '"' && char != '\\' {
				word.WriteRune('\\')
			}
			word.WriteRune(char)
			escaped = false
		case quote == '\'':
			if char == '\'' {
				quote = 0
			} else {
				word.WriteRune(char)
			}
		case char == '\\' && quote != '\'':
			escaped, inWord = true, true
		case quote == '"':
			if char == '"' {
				quote = 0
			} else {
				word.WriteRune(char)
			}
		case char == '"' || char == '\'':
			quote, inWord = char, true
		case char == ' ' || char == '\t' || char == '\n' || char == '\r':
			if inWord {
				words = append(words, word.String())
				word.Reset()
				inWord = false
			}
		default:
			word.WriteRune(char)
			inWord = true
		}
	}
	if quote != 0 || escaped {
		return nil, errors.New("unbalanced quoting")
	}
	if inWord {
		words = append(words, word.String())
	}
	return words, nil
}

// MatchIdentities returns the distribution identities whose SHA-1 the profile allows.
func MatchIdentities(identities string, allowed []string) []string {
	var matches []string
	for _, match := range identityPattern.FindAllStringSubmatch(identities, -1) {
		sha, name := match[1], match[2]
		if slices.Contains(allowed, sha) &&
			(strings.HasPrefix(name, "Apple Distribution:") || strings.HasPrefix(name, "iPhone Distribution:")) {
			matches = append(matches, sha)
		}
	}
	return matches
}

func profilesDirectory() (string, error) {
	home, err := os.UserHomeDir()
	if err != nil {
		return "", err
	}
	return filepath.Join(home, "Library", "MobileDevice", "Provisioning Profiles"), nil
}

func install(state string) error {
	data, err := readSettings(state)
	if err != nil {
		return err
	}
	keychain := filepath.Join(state, "signing.keychain-db")
	identities, err := run("security", "find-identity", "-v", "-p", "codesigning", keychain)
	if err != nil {
		return err
	}
	matches := MatchIdentities(string(identities), data.CertificateSHA1s)
	if err := require(len(matches) == 1, "Requires one valid distribution identity matching the profile"); err != nil {
		return err
	}
	data.CertificateSHA1 = matches[0]
	if err := writeSettings(state, data); err != nil {
		return err
	}
	var original []string
	if err := readJSON(filepath.Join(state, "keychains.json"), &original); err != nil {
		return err
	}
	if _, err := run(append([]string{"security", "list-keychains", "-d", "user", "-s", keychain}, original...)...); err != nil {
		return err
	}
	return installProfile(state, data)
}

func installProfile(state string, data Settings) error {
	profiles, err := profilesDirectory()
	if err != nil {
		return err
	}
	if err := os.MkdirAll(profiles, 0o700); err != nil {
		return err
	}
	installed := filepath.Join(profiles, data.ProfileUUID+".mobileprovision")
	if err := require(!exists(installed), "Runner already has a profile with this UUID"); err != nil {
		return err
	}
	// Record ownership before writing, so cleanup also works after interruption.
	if err := os.WriteFile(filepath.Join(state, "installed-profile"), []byte(installed), 0o600); err != nil {
		return err
	}
	blob, err := os.ReadFile(filepath.Join(state, "app.mobileprovision"))
	if err != nil {
		return err
	}
	if err := os.WriteFile(installed, blob, 0o600); err != nil {
		return err
	}
	options := map[string]any{"method": "app-store-connect", "destination": "export", "signingStyle": "manual",
		"teamID": data.TeamID, "signingCertificate": data.CertificateSHA1,
		"provisioningProfiles":           map[string]any{BundleID: data.ProfileUUID},
		"manageAppVersionAndBuildNumber": false, "uploadSymbols": true}
	encoded, err := plist.MarshalIndent(options, plist.XMLFormat, "\t")
	if err != nil {
		return err
	}
	return os.WriteFile(filepath.Join(state, "ExportOptions.plist"), encoded, 0o600)
}

func readJSON(path string, target any) error {
	raw, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(raw, target)
}

func exportedIPA(state string) (string, error) {
	export := filepath.Join(state, "export")
	if err := require(!isSymlink(export), "IPA export directory cannot be a symlink"); err != nil {
		return "", err
	}
	candidates, err := filepath.Glob(filepath.Join(export, "*.ipa"))
	if err != nil {
		return "", err
	}
	var ipas []string
	for _, candidate := range candidates {
		if info, err := os.Lstat(candidate); err == nil && info.Mode().IsRegular() {
			ipas = append(ipas, candidate)
		}
	}
	if err := require(len(ipas) > 0, "Xcode did not export a regular IPA file"); err != nil {
		return "", err
	}
	if err := require(len(ipas) == 1, "Xcode exported multiple IPA files"); err != nil {
		return "", err
	}
	ipa := filepath.Join(export, "Runner.ipa")
	if ipas[0] != ipa {
		if err := require(!lexists(ipa), "Cannot normalize IPA name over an existing export"); err != nil {
			return "", err
		}
		if err := os.Rename(ipas[0], ipa); err != nil {
			return "", err
		}
	}
	return ipa, nil
}

func readPlistFile(path string) (map[string]any, error) {
	raw, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}
	return decodePlist(raw)
}

// CheckExportedInfo checks the Info.plist of the exported device app.
func CheckExportedInfo(info map[string]any, data Settings) error {
	var xcode int64
	switch value := info["DTXcode"].(type) {
	case nil:
	case string:
		parsed, err := strconv.ParseInt(strings.TrimSpace(value), 10, 64)
		if err != nil {
			return errors.New("invalid DTXcode")
		}
		xcode = parsed
	case int64:
		xcode = value
	case uint64:
		xcode = int64(min(value, 1<<62))
	default:
		return errors.New("invalid DTXcode")
	}
	platforms := stringList(info["CFBundleSupportedPlatforms"])
	rules := []struct {
		ok      bool
		message string
	}{
		{info["CFBundleIdentifier"] == BundleID, "Exported bundle ID mismatch"},
		{info["CFBundleShortVersionString"] == data.Version && info["CFBundleVersion"] == data.Build, "Export changed the version/build"},
		{info["ITSAppUsesNonExemptEncryption"] == false, "Missing export compliance declaration"},
		{len(platforms) == 1 && platforms[0] == "iPhoneOS", "Export is not an iPhoneOS device build"},
		{xcode >= 2600, "Export was not built with Xcode 26 or later"},
		{matchesString(sdkPattern, info["DTSDKName"]), "Export requires iOS SDK 26 or later"},
	}
	for _, rule := range rules {
		if err := require(rule.ok, rule.message); err != nil {
			return err
		}
	}
	return nil
}

var allowedEntitlements = []string{"application-identifier", "com.apple.developer.team-identifier", "get-task-allow",
	"beta-reports-active", "keychain-access-groups"}

// CheckEntitlements checks the entitlements the exported app was signed with.
func CheckEntitlements(entitlements map[string]any, data Settings) error {
	onlyAllowed := true
	for key := range entitlements {
		onlyAllowed = onlyAllowed && slices.Contains(allowedEntitlements, key)
	}
	groupsOK := true
	for _, group := range stringList(entitlements["keychain-access-groups"]) {
		groupsOK = groupsOK && group == data.AppID
	}
	rules := []struct {
		ok      bool
		message string
	}{
		{entitlements["application-identifier"] == data.AppID, "Signed App ID mismatch"},
		{entitlements["com.apple.developer.team-identifier"] == data.TeamID, "Signed team mismatch"},
		{entitlements["get-task-allow"] == false, "IPA permits debugging"},
		{entitlements["beta-reports-active"] == true, "IPA is not signed for App Store distribution"},
		{onlyAllowed, "IPA includes an unexpected capability"},
		{groupsOK, "IPA includes an unexpected keychain group"},
	}
	for _, rule := range rules {
		if err := require(rule.ok, rule.message); err != nil {
			return err
		}
	}
	return nil
}

func verify(state string, stdout io.Writer) error {
	data, err := readSettings(state)
	if err != nil {
		return err
	}
	ipa, err := exportedIPA(state)
	if err != nil {
		return err
	}
	app, err := unpackApp(state, ipa)
	if err != nil {
		return err
	}
	info, err := readPlistFile(filepath.Join(app, "Info.plist"))
	if err != nil {
		return err
	}
	if err := CheckExportedInfo(info, data); err != nil {
		return err
	}
	for _, manifest := range []string{filepath.Join(app, "PrivacyInfo.xcprivacy"), filepath.Join(app, "Frameworks/Flutter.framework/PrivacyInfo.xcprivacy")} {
		info, statErr := os.Stat(manifest)
		if err := require(statErr == nil && info.Mode().IsRegular(), "Missing app or Flutter privacy manifest"); err != nil {
			return err
		}
		if _, err := readPlistFile(manifest); err != nil {
			return err
		}
	}
	check := exec.Command("bash", "scripts/check-ios.sh", app)
	check.Stdout, check.Stderr = stdout, os.Stderr
	if err := check.Run(); err != nil {
		return errors.New("check-ios failed")
	}
	if err := verifySignature(state, app, data); err != nil {
		return err
	}
	raw, err := os.ReadFile(ipa)
	if err != nil {
		return err
	}
	sum := sha256.Sum256(raw)
	fmt.Fprintf(stdout, "Verified distribution IPA: %s %s (%s); SHA-256 %s\n", BundleID, data.Version, data.Build, hex.EncodeToString(sum[:]))
	return nil
}

func unpackApp(state, ipa string) (string, error) {
	unpacked := filepath.Join(state, "verified-ipa")
	// Apple's extractor preserves framework symlinks and executable modes.
	if _, err := run("ditto", "-x", "-k", ipa, unpacked); err != nil {
		return "", err
	}
	apps, err := filepath.Glob(filepath.Join(unpacked, "Payload", "*.app"))
	if err != nil {
		return "", err
	}
	if err := require(len(apps) == 1 && filepath.Base(apps[0]) == "Runner.app", "Unexpected IPA app payload"); err != nil {
		return "", err
	}
	return apps[0], nil
}

func verifySignature(state, app string, data Settings) error {
	if _, err := run("codesign", "--verify", "--deep", "--strict", app); err != nil {
		return err
	}
	raw, err := run("codesign", "-d", "--entitlements", ":-", app)
	if err != nil {
		return err
	}
	entitlements, err := decodePlist(raw)
	if err != nil {
		return err
	}
	if err := CheckEntitlements(entitlements, data); err != nil {
		return err
	}
	embedded, err := profileData(filepath.Join(app, "embedded.mobileprovision"))
	if err != nil {
		return err
	}
	if err := require(embedded.UUID == data.ProfileUUID, "IPA uses a different provisioning profile"); err != nil {
		return err
	}
	prefix := filepath.Join(state, "signer-")
	// This optional argument must share an argv item with the option.
	if _, err := run("codesign", "-d", "--extract-certificates="+prefix, app); err != nil {
		return err
	}
	leaf, err := os.ReadFile(prefix + "0")
	if err != nil {
		return err
	}
	return require(sha1Hex(leaf) == data.CertificateSHA1, "IPA signing certificate mismatch")
}

func cleanup(state string) error {
	if !exists(state) {
		return nil
	}
	var failures []string
	if exists(filepath.Join(state, "project.pbxproj")) {
		saved, err := os.ReadFile(filepath.Join(state, "project.pbxproj"))
		if err == nil {
			err = os.WriteFile(project, saved, 0o644)
		}
		if err != nil {
			failures = append(failures, "restore Runner project")
		}
	}
	if exists(filepath.Join(state, "keychains.json")) {
		var original []string
		if err := readJSON(filepath.Join(state, "keychains.json"), &original); err != nil {
			return err
		}
		if _, err := run(append([]string{"security", "list-keychains", "-d", "user", "-s"}, original...)...); err != nil {
			failures = append(failures, "restore keychain search list")
		}
	}
	if keychain := filepath.Join(state, "signing.keychain-db"); exists(keychain) {
		if _, err := run("security", "delete-keychain", keychain); err != nil {
			failures = append(failures, "delete temporary keychain")
		}
	}
	if err := removeInstalledProfile(state, &failures); err != nil {
		return err
	}
	// Keep restoration metadata if security cleanup failed, but remove key files.
	for _, name := range []string{"distribution.p12", "app.mobileprovision", "keychain-password"} {
		if err := os.Remove(filepath.Join(state, name)); err != nil && !errors.Is(err, os.ErrNotExist) {
			return err
		}
	}
	_ = os.RemoveAll(filepath.Join(state, "private_keys"))
	if err := require(len(failures) == 0, "Signing cleanup failed: "+strings.Join(failures, ", ")); err != nil {
		return err
	}
	return os.RemoveAll(state)
}

func removeInstalledProfile(state string, failures *[]string) error {
	record := filepath.Join(state, "installed-profile")
	if !exists(record) {
		return nil
	}
	raw, err := os.ReadFile(record)
	if err != nil {
		return err
	}
	installed := string(raw)
	expected, err := profilesDirectory()
	if err != nil {
		return err
	}
	if err := require(filepath.Dir(installed) == expected && profileName.MatchString(filepath.Base(installed)),
		"Invalid cleanup profile location"); err != nil {
		return err
	}
	if err := os.Remove(installed); err != nil && !errors.Is(err, os.ErrNotExist) {
		*failures = append(*failures, "remove installed profile")
	}
	return nil
}

func operate(args []string, stdout io.Writer) error {
	syscall.Umask(0o077)
	state, err := StateDirectory()
	if err != nil {
		return err
	}
	if len(args) == 0 {
		return errors.New("missing operation")
	}
	switch args[0] {
	case "field":
		if err := require(len(args) == 2 && (args[1] == "api_key_id" || args[1] == "api_issuer_id"), "Unknown upload metadata field"); err != nil {
			return err
		}
		data, err := readSettings(state)
		if err != nil {
			return err
		}
		value := data.APIKeyID
		if args[1] == "api_issuer_id" {
			value = data.APIIssuerID
		}
		fmt.Fprintln(stdout, value)
		return nil
	case "prepare":
		return prepare(state)
	case "install":
		return install(state)
	case "verify":
		return verify(state, stdout)
	case "cleanup":
		return cleanup(state)
	}
	return errors.New("unknown operation")
}

// Main runs one signing-asset operation. ValidationError messages contain only
// fixed text and tokens; other errors can carry key material and are never printed.
func Main(args []string, stdout, stderr io.Writer) int {
	err := operate(args, stdout)
	if err == nil {
		return 0
	}
	var validation ValidationError
	if errors.As(err, &validation) {
		fmt.Fprintf(stderr, "TestFlight asset operation failed: %s\n", validation)
	} else {
		fmt.Fprintln(stderr, "TestFlight asset operation failed; check secrets, signing assets and workflow prerequisites.")
	}
	return 1
}
