package storesite

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func confirmed() Config {
	return Config{
		"schema_version":                       float64(1),
		"public_base_url":                      "https://chouwazi.example-real.cn/",
		"support_email":                        "support@mail.example-real.cn",
		"copyright_holder":                     "Holder",
		"website_icp_registration":             "ICP-1",
		"website_public_security_registration": "PS-1",
		"website_public_security_code":         "12345678901234",
	}
}

func TestPublicHosts(t *testing.T) {
	for host, want := range map[string]bool{
		"chouwazi.example-real.cn": true, "ChouWaZi.Real.CN.": true, "抽袜子.中国": true,
		"localhost": false, "a.example.com": false, "site.test": false, "1.2.3.4": false,
		"bad_label.cn": false, "-dash.cn": false, "nodot": false, "": false, "a..b.cn": false,
	} {
		if got := isPublicHost(host); got != want {
			t.Errorf("isPublicHost(%q) = %v, want %v", host, got, want)
		}
	}
}

func TestValidateConfig(t *testing.T) {
	cases := map[string]struct {
		change  Config
		preview bool
		message string
	}{
		"confirmed":         {nil, false, ""},
		"missing details":   {Config{"support_email": nil, "copyright_holder": ""}, false, "Public details are not confirmed: support_email, copyright_holder."},
		"preview missing":   {Config{"support_email": nil}, true, ""},
		"schema":            {Config{"schema_version": float64(2)}, false, "Unsupported site configuration schema_version"},
		"http":              {Config{"public_base_url": "http://chouwazi.example-real.cn/"}, false, "real public HTTPS URL"},
		"port":              {Config{"public_base_url": "https://chouwazi.example-real.cn:8443/"}, false, "default HTTPS port"},
		"default port":      {Config{"public_base_url": "https://chouwazi.example-real.cn:443/x/"}, false, ""},
		"query":             {Config{"public_base_url": "https://chouwazi.example-real.cn/?a=1"}, false, "real public HTTPS URL"},
		"credentials":       {Config{"public_base_url": "https://u@chouwazi.example-real.cn/"}, false, "real public HTTPS URL"},
		"parent segment":    {Config{"public_base_url": "https://chouwazi.example-real.cn/a/../b"}, false, "parent path segments"},
		"mailto":            {Config{"support_email": "mailto:a@mail.example-real.cn"}, false, "real public email address"},
		"reserved email":    {Config{"support_email": "a@localhost"}, false, "real public email address"},
		"surrounding space": {Config{"copyright_holder": " Holder"}, false, "copyright_holder must be a nonempty string"},
		"control":           {Config{"copyright_holder": "Hol\x07der"}, false, "copyright_holder contains a control character"},
		"not a string":      {Config{"copyright_holder": float64(5)}, false, "copyright_holder must be a nonempty string"},
		"empty registration": {Config{"website_icp_registration": ""}, false,
			"website_icp_registration must be a nonempty string without surrounding whitespace or control characters, or null"},
		"security code": {Config{"website_public_security_code": "123"}, false, "website_public_security_code must contain the 14 digits"},
	}
	for name, test := range cases {
		config := confirmed()
		for key, value := range test.change {
			config[key] = value
		}
		_, err := ValidateConfig(config, test.preview)
		if test.message == "" && err != nil {
			t.Errorf("%s: unexpected error %v", name, err)
		}
		if test.message != "" && (err == nil || !strings.Contains(err.Error(), test.message)) {
			t.Errorf("%s: error %v, want %q", name, err, test.message)
		}
	}
}

func TestEscapingMatchesHTMLQuoteEscape(t *testing.T) {
	if got := escape(`A&B <c> "d" 'e'`); got != "A&amp;B &lt;c&gt; &quot;d&quot; &#x27;e&#x27;" {
		t.Fatal(got)
	}
	if got := quoteEmail("a+b&c d@mail.cn"); got != "a+b%26c%20d@mail.cn" {
		t.Fatal(got)
	}
}

func TestTemplateSubstitution(t *testing.T) {
	got := substitute("$a ${b} $$ $a_c", map[string]string{"a": "1", "b": "2", "a_c": "3"})
	if got != "1 2 $ 3" {
		t.Fatal(got)
	}
	defer func() {
		if recover() == nil {
			t.Fatal("unknown placeholder accepted")
		}
	}()
	substitute("$missing", map[string]string{})
}

// site copies the real store/site sources into a temporary root with the given configuration.
func site(t *testing.T, config Config) string {
	t.Helper()
	root := t.TempDir()
	source := filepath.Join("..", "..", "..", "store", "site")
	for _, name := range []string{"content.json", "page.html", "styles.css", PublicSecurityIcon} {
		data, err := os.ReadFile(filepath.Join(source, name))
		if err != nil {
			t.Fatal(err)
		}
		destination := filepath.Join(root, "store", "site", name)
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(destination, data, 0o644); err != nil {
			t.Fatal(err)
		}
	}
	encoded, err := json.Marshal(config)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "store", "site-config.json"), encoded, 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func read(t *testing.T, path string) string {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return string(data)
}

func TestRenderReadySite(t *testing.T) {
	root := site(t, confirmed())
	output := filepath.Join(root, "build", "store-site")
	if _, err := Render(root, output, false); err != nil {
		t.Fatal(err)
	}
	page := read(t, filepath.Join(output, "en", "privacy.html"))
	for _, want := range []string{
		`<link rel="canonical" href="https://chouwazi.example-real.cn/en/privacy.html">`,
		`<link rel="alternate" hreflang="zh-Hant" href="https://chouwazi.example-real.cn/zh-Hant/privacy.html">`,
		`<a href="https://beian.mps.gov.cn/#/query/webSearch?code=12345678901234"><img src="../assets/public-security-beian.png"`,
		`© Holder`, `aria-current="page"`,
	} {
		if !strings.Contains(page, want) {
			t.Errorf("privacy page lacks %q", want)
		}
	}
	if strings.Contains(page, "noindex") || strings.Contains(page, "$") {
		t.Error("ready page is marked as draft or has an unfilled placeholder")
	}
	support := read(t, filepath.Join(output, "zh-Hans", "support.html"))
	if !strings.Contains(support, `<a href="mailto:support@mail.example-real.cn">support@mail.example-real.cn</a>`) {
		t.Error("support page lacks the contact link")
	}
	var manifest map[string]any
	if err := json.Unmarshal([]byte(read(t, filepath.Join(output, "manifest.json"))), &manifest); err != nil {
		t.Fatal(err)
	}
	if manifest["status"] != "ready-for-deployment" || len(manifest["files"].([]any)) != 9 {
		t.Errorf("manifest = %v", manifest)
	}
	if !strings.Contains(read(t, filepath.Join(output, "manifest.json")), "\"pages\": {\n    \"zh-Hans\"") {
		t.Error("manifest pages are not in zh-Hans, zh-Hant, en order")
	}
}

func TestRenderPreviewDraft(t *testing.T) {
	config := confirmed()
	config["support_email"], config["website_public_security_registration"] = nil, nil
	root := site(t, config)
	output := filepath.Join(root, "out")
	if _, err := Render(root, output, false); err == nil {
		t.Fatal("rendered a publishable site without confirmed details")
	}
	if _, err := os.Stat(output); !os.IsNotExist(err) {
		t.Fatal("a refused render left output behind")
	}
	if _, err := Render(root, output, true); err != nil {
		t.Fatal(err)
	}
	page := read(t, filepath.Join(output, "index.html"))
	if !strings.Contains(page, `<meta name="robots" content="noindex, nofollow">`) || strings.Contains(page, "canonical") {
		t.Error("preview index is indexable")
	}
	if !strings.Contains(page, `<aside class="draft-notice">`) {
		t.Error("preview lacks the draft notice")
	}
	if _, err := os.Stat(filepath.Join(output, PublicSecurityIcon)); !os.IsNotExist(err) {
		t.Error("icon copied without a public security registration")
	}
}

func TestRenderRefusesSourceOutput(t *testing.T) {
	root := site(t, confirmed())
	for _, output := range []string{root, filepath.Join(root, "store"), filepath.Join(root, "store", "site", "x")} {
		if _, err := Render(root, output, false); err == nil || !strings.Contains(err.Error(), "must not overwrite") {
			t.Errorf("Render to %s: %v", output, err)
		}
	}
}
