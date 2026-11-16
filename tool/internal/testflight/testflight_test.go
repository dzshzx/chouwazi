package testflight

import (
	"bytes"
	"errors"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
	"time"

	"howett.net/plist"
)

func profile() map[string]any {
	return map[string]any{
		"TeamIdentifier":              []any{"ABCDE12345"},
		"ApplicationIdentifierPrefix": []any{"ABCDE12345"},
		"UUID":                        "12345678-1234-1234-1234-123456789ABC",
		"Entitlements": map[string]any{
			"application-identifier":              "ABCDE12345." + BundleID,
			"com.apple.developer.team-identifier": "ABCDE12345",
			"get-task-allow":                      false,
			"beta-reports-active":                 true,
		},
		"ExpirationDate":        time.Date(2099, 1, 1, 0, 0, 0, 0, time.UTC),
		"DeveloperCertificates": []any{[]byte("leaf")},
	}
}

var now = time.Date(2026, 11, 16, 0, 0, 0, 0, time.UTC)

func TestValidProfile(t *testing.T) {
	got, err := ValidateProfile(profile(), now)
	if err != nil {
		t.Fatal(err)
	}
	if got.TeamID != "ABCDE12345" || got.AppID != "ABCDE12345."+BundleID || len(got.Certificates) != 1 {
		t.Fatalf("profile = %+v", got)
	}
}

func TestProfileRejections(t *testing.T) {
	entitlement := func(key string, value any) func(map[string]any) {
		return func(p map[string]any) { p["Entitlements"].(map[string]any)[key] = value }
	}
	cases := map[string]func(map[string]any){
		"Invalid profile team":                          func(p map[string]any) { p["TeamIdentifier"] = []any{"abc"} },
		"Invalid App ID prefix":                         func(p map[string]any) { p["ApplicationIdentifierPrefix"] = []any{"A", "B"} },
		"Invalid profile UUID":                          func(p map[string]any) { delete(p, "UUID") },
		"Profile App ID does not match Runner":          entitlement("application-identifier", "ABCDE12345.other"),
		"Profile team entitlement mismatch":             entitlement("com.apple.developer.team-identifier", "ZZZZZ12345"),
		"A development profile cannot sign this upload": entitlement("get-task-allow", true),
		"Requires an App Store distribution profile":    entitlement("beta-reports-active", false),
		"Ad hoc/enterprise profiles cannot be used":     func(p map[string]any) { p["ProvisionsAllDevices"] = true },
		"Missing profile expiry":                        func(p map[string]any) { p["ExpirationDate"] = "2099-01-01" },
		"Provisioning profile has expired":              func(p map[string]any) { p["ExpirationDate"] = now.Add(-time.Second) },
		"Profile contains no distribution certificate":  func(p map[string]any) { p["DeveloperCertificates"] = []any{} },
	}
	for message, change := range cases {
		p := profile()
		change(p)
		if _, err := ValidateProfile(p, now); err == nil || err.Error() != message {
			t.Errorf("%s: got %v", message, err)
		}
	}
	devices := profile()
	devices["ProvisionedDevices"] = []any{}
	if _, err := ValidateProfile(devices, now); err == nil {
		t.Error("a profile listing devices was accepted")
	}
}

func TestCodesignFailureKeepsOnlyFixedTokens(t *testing.T) {
	cases := []struct {
		args   []string
		stderr string
		want   string
	}{
		{[]string{"codesign", "--verify", "--deep", "x"},
			"x: a sealed resource is missing or invalid\nIn subcomponent: /secret/Payload/Runner.app/Frameworks/Flutter.framework\n",
			"codesign verify failed while handling signing assets (exit=3, reason=sealed-resource-missing-or-invalid, subcomponent=Flutter.framework)"},
		{[]string{"codesign", "-d", "--entitlements", ":-", "x"}, "codesign: Unknown option secret-token\n",
			"codesign display-entitlements failed while handling signing assets (exit=3, reason=unsupported-syntax-or-argument, subcomponent=none)"},
		{[]string{"codesign", "-d", "--extract-certificates=/tmp/signer-", "x"}, "CSSMERR_TP_CERT_EXPIRED\nIn subcomponent: /p/Other.bundle\n",
			"codesign extract-certificates failed while handling signing assets (exit=3, reason=trust-or-certificate-expiry, subcomponent=unknown)"},
		{[]string{"codesign", "--sign", "x"}, "\xff\xfe binary noise",
			"codesign unknown failed while handling signing assets (exit=3, reason=unknown, subcomponent=none)"},
	}
	for _, test := range cases {
		if got := CodesignFailure(test.args, 3, []byte(test.stderr)).Error(); got != test.want {
			t.Errorf("got %q\nwant %q", got, test.want)
		}
	}
}

func TestShellSplit(t *testing.T) {
	got, err := ShellSplit(`    "/Users/r/Library/Keychains/login.keychain-db"
    "/Users/r/My Keys.keychain-db" plain 'single quoted' esc\ aped "a\"b\c"`)
	if err != nil {
		t.Fatal(err)
	}
	want := []string{"/Users/r/Library/Keychains/login.keychain-db", "/Users/r/My Keys.keychain-db", "plain",
		"single quoted", "esc aped", `a"b\c`}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("got %q", got)
	}
	if _, err := ShellSplit(`"open`); err == nil {
		t.Fatal("unbalanced quote accepted")
	}
}

func TestMatchIdentities(t *testing.T) {
	sha, other := strings.Repeat("A", 40), strings.Repeat("B", 40)
	listing := "  1) " + sha + ` "Apple Distribution: Team (X)"` + "\n  2) " + other + ` "Apple Development: Team"` +
		"\n  3) " + other + ` "iPhone Distribution: Team"` + "\n"
	if got := MatchIdentities(listing, []string{sha}); !reflect.DeepEqual(got, []string{sha}) {
		t.Fatalf("got %v", got)
	}
	if got := MatchIdentities(listing, []string{sha, other}); len(got) != 2 {
		t.Fatalf("distribution identities = %v", got)
	}
}

func settings() Settings {
	return Settings{TeamID: "ABCDE12345", AppID: "ABCDE12345." + BundleID, Version: "1.2.3", Build: "10203"}
}

func exportedInfo() map[string]any {
	return map[string]any{"CFBundleIdentifier": BundleID, "CFBundleShortVersionString": "1.2.3", "CFBundleVersion": "10203",
		"ITSAppUsesNonExemptEncryption": false, "CFBundleSupportedPlatforms": []any{"iPhoneOS"},
		"DTXcode": "2630", "DTSDKName": "iphoneos26.2"}
}

func TestCheckExportedInfo(t *testing.T) {
	if err := CheckExportedInfo(exportedInfo(), settings()); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(map[string]any){
		"Exported bundle ID mismatch":                 func(i map[string]any) { i["CFBundleIdentifier"] = "x" },
		"Export changed the version/build":            func(i map[string]any) { i["CFBundleVersion"] = "1" },
		"Missing export compliance declaration":       func(i map[string]any) { delete(i, "ITSAppUsesNonExemptEncryption") },
		"Export is not an iPhoneOS device build":      func(i map[string]any) { i["CFBundleSupportedPlatforms"] = []any{"iPhoneSimulator"} },
		"Export was not built with Xcode 26 or later": func(i map[string]any) { delete(i, "DTXcode") },
		"Export requires iOS SDK 26 or later":         func(i map[string]any) { i["DTSDKName"] = "iphoneos18.4" },
	}
	for message, change := range cases {
		info := exportedInfo()
		change(info)
		if err := CheckExportedInfo(info, settings()); err == nil || err.Error() != message {
			t.Errorf("%s: got %v", message, err)
		}
	}
}

func TestCheckEntitlements(t *testing.T) {
	good := func() map[string]any {
		return map[string]any{"application-identifier": "ABCDE12345." + BundleID, "com.apple.developer.team-identifier": "ABCDE12345",
			"get-task-allow": false, "beta-reports-active": true, "keychain-access-groups": []any{"ABCDE12345." + BundleID}}
	}
	if err := CheckEntitlements(good(), settings()); err != nil {
		t.Fatal(err)
	}
	cases := map[string]func(map[string]any){
		"Signed App ID mismatch":                       func(e map[string]any) { e["application-identifier"] = "x" },
		"Signed team mismatch":                         func(e map[string]any) { e["com.apple.developer.team-identifier"] = "x" },
		"IPA permits debugging":                        func(e map[string]any) { e["get-task-allow"] = true },
		"IPA is not signed for App Store distribution": func(e map[string]any) { delete(e, "beta-reports-active") },
		"IPA includes an unexpected capability":        func(e map[string]any) { e["aps-environment"] = "production" },
		"IPA includes an unexpected keychain group":    func(e map[string]any) { e["keychain-access-groups"] = []any{"ABCDE12345.*"} },
	}
	for message, change := range cases {
		entitlements := good()
		change(entitlements)
		if err := CheckEntitlements(entitlements, settings()); err == nil || err.Error() != message {
			t.Errorf("%s: got %v", message, err)
		}
	}
}

func TestStateDirectory(t *testing.T) {
	temp := t.TempDir()
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_RUN_ID", "77")
	t.Setenv("GITHUB_RUN_ATTEMPT", "2")
	t.Setenv("RUNNER_TEMP", temp)
	state, err := StateDirectory()
	if err != nil || filepath.Base(state) != "chouwazi-testflight-77-2" {
		t.Fatalf("state %q, %v", state, err)
	}
	if err := os.Symlink(temp, state); err != nil {
		t.Fatal(err)
	}
	if _, err := StateDirectory(); err == nil || err.Error() != "Signing state cannot be a symlink" {
		t.Fatalf("symlink state: %v", err)
	}
	t.Setenv("GITHUB_RUN_ID", "7x")
	if _, err := StateDirectory(); err == nil || err.Error() != "Invalid workflow run identifier" {
		t.Fatalf("run id: %v", err)
	}
	t.Setenv("GITHUB_ACTIONS", "")
	if _, err := StateDirectory(); err == nil || err.Error() != "Requires a GitHub-hosted workflow" {
		t.Fatalf("outside Actions: %v", err)
	}
}

func TestMainHidesUnexpectedErrors(t *testing.T) {
	t.Setenv("GITHUB_ACTIONS", "true")
	t.Setenv("GITHUB_RUN_ID", "1")
	t.Setenv("GITHUB_RUN_ATTEMPT", "1")
	t.Setenv("RUNNER_TEMP", t.TempDir())
	var stdout, stderr bytes.Buffer
	if code := Main([]string{"install"}, &stdout, &stderr); code != 1 ||
		stderr.String() != "TestFlight asset operation failed; check secrets, signing assets and workflow prerequisites.\n" {
		t.Fatalf("code %d, stderr %q", code, stderr.String())
	}
	stderr.Reset()
	if code := Main([]string{"field", "secret"}, &stdout, &stderr); code != 1 ||
		stderr.String() != "TestFlight asset operation failed: Unknown upload metadata field\n" {
		t.Fatalf("code %d, stderr %q", code, stderr.String())
	}
	if !errors.As(error(ValidationError("x")), new(ValidationError)) {
		t.Fatal("ValidationError is not detectable")
	}
}

func TestExportOptionsPlist(t *testing.T) {
	state := t.TempDir()
	t.Setenv("HOME", t.TempDir())
	if err := os.WriteFile(filepath.Join(state, "app.mobileprovision"), []byte("profile"), 0o600); err != nil {
		t.Fatal(err)
	}
	data := settings()
	data.ProfileUUID, data.CertificateSHA1 = "12345678-1234-1234-1234-123456789ABC", strings.Repeat("A", 40)
	if err := installProfile(state, data); err != nil {
		t.Fatal(err)
	}
	raw, err := os.ReadFile(filepath.Join(state, "ExportOptions.plist"))
	if err != nil {
		t.Fatal(err)
	}
	var options map[string]any
	if _, err := plist.Unmarshal(raw, &options); err != nil {
		t.Fatal(err)
	}
	if options["method"] != "app-store-connect" || options["manageAppVersionAndBuildNumber"] != false ||
		options["uploadSymbols"] != true || options["teamID"] != "ABCDE12345" ||
		!reflect.DeepEqual(options["provisioningProfiles"], map[string]any{BundleID: data.ProfileUUID}) {
		t.Fatalf("options = %v", options)
	}
	if err := installProfile(state, data); err == nil || err.Error() != "Runner already has a profile with this UUID" {
		t.Fatalf("second install: %v", err)
	}
}
