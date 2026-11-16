// Package storesite renders the fixed store/site sources; it never deploys or fetches remote resources.
//
// Default mode requires confirmed public details in store/site-config.json.
// Preview mode generates an explicitly marked, non-indexable local draft.
package storesite

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"net/url"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"

	"golang.org/x/net/idna"
)

var (
	languages          = []string{"zh-Hans", "zh-Hant", "en"}
	pageKinds          = []string{"support", "privacy"}
	required           = []string{"public_base_url", "support_email", "copyright_holder"}
	registrationFields = []string{"website_icp_registration", "website_public_security_registration", "website_public_security_code"}
	reservedHosts      = []string{"localhost", "local", "internal", "invalid", "test", "example", "example.com", "example.net", "example.org"}

	labelPattern    = regexp.MustCompile(`^[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?$`)
	digitsPattern   = regexp.MustCompile(`^[0-9]+$`)
	securityCode    = regexp.MustCompile(`^[0-9]{14}$`)
	emailPattern    = regexp.MustCompile("^[A-Za-z0-9.!#$%&'*+/=?^_`{|}~-]+@[A-Za-z0-9.-]+$")
	whitespace      = regexp.MustCompile(`\s`)
	templatePattern = regexp.MustCompile(`(?i)\$(?:(\$)|([_a-z][_a-z0-9]*)|\{([_a-z][_a-z0-9]*)\}|())`)
)

// PublicSecurityIcon is the asset copied when a public security registration is shown.
const PublicSecurityIcon = "assets/public-security-beian.png"

type validationError string

func (e validationError) Error() string { return string(e) }

func require(condition bool, message string) error {
	if condition {
		return nil
	}
	return validationError(message)
}

// Config is the decoded store/site-config.json object.
type Config map[string]any

// str returns the string value of a field, or "" when it is absent, null or not a string.
func (c Config) str(field string) string {
	value, _ := c[field].(string)
	return value
}

// truthy follows the source semantics of a JSON value used as a condition.
func (c Config) truthy(field string) bool {
	switch value := c[field].(type) {
	case nil:
		return false
	case string:
		return value != ""
	case bool:
		return value
	case float64:
		return value != 0
	case []any:
		return len(value) > 0
	case map[string]any:
		return len(value) > 0
	}
	return true
}

func isPublicHost(host string) bool {
	if host == "" || !strings.Contains(host, ".") {
		return false
	}
	host = strings.TrimRight(strings.ToLower(host), ".")
	ascii, err := idna.Lookup.ToASCII(host)
	if err != nil {
		return false
	}
	host = ascii
	labels := strings.Split(host, ".")
	if len(host) > 253 || digitsPattern.MatchString(labels[len(labels)-1]) {
		return false
	}
	for _, label := range labels {
		if !labelPattern.MatchString(label) {
			return false
		}
	}
	for _, name := range reservedHosts {
		if host == name || strings.HasSuffix(host, "."+name) {
			return false
		}
	}
	return true
}

func hasControl(value string) bool {
	for _, char := range value {
		if char < 32 {
			return true
		}
	}
	return false
}

func validString(value any) bool {
	text, ok := value.(string)
	return ok && text != "" && strings.TrimSpace(text) == text && !hasControl(text)
}

func validateRequired(config Config) ([]string, error) {
	missing := []string{}
	for _, field := range required {
		value := config[field]
		if value == nil || value == "" {
			missing = append(missing, field)
			continue
		}
		text, ok := value.(string)
		if err := require(ok && strings.TrimSpace(text) == text,
			field+" must be a nonempty string without surrounding whitespace, or null"); err != nil {
			return nil, err
		}
		if err := require(!hasControl(text), field+" contains a control character"); err != nil {
			return nil, err
		}
	}
	for _, field := range registrationFields {
		value := config[field]
		if err := require(value == nil || validString(value),
			field+" must be a nonempty string without surrounding whitespace or control characters, or null"); err != nil {
			return nil, err
		}
	}
	return missing, nil
}

func validateBaseURL(raw string) error {
	const message = "public_base_url must be a real public HTTPS URL without credentials, query or fragment"
	parsed, err := url.Parse(raw)
	if err != nil {
		return validationError(message)
	}
	if err := require(parsed.Scheme == "https" && isPublicHost(strings.ToLower(parsed.Hostname())) &&
		parsed.User == nil && parsed.RawQuery == "" && !parsed.ForceQuery && parsed.Fragment == "" &&
		!strings.Contains(raw, "#") && !whitespace.MatchString(raw), message); err != nil {
		return err
	}
	if err := require(parsed.Port() == "" || parsed.Port() == "443", "public_base_url must use the default HTTPS port"); err != nil {
		return err
	}
	for _, segment := range strings.Split(parsed.Path, "/") {
		if segment == ".." {
			return validationError("public_base_url cannot contain parent path segments")
		}
	}
	return nil
}

// ValidateConfig checks the site configuration and returns the missing public details.
func ValidateConfig(config Config, preview bool) ([]string, error) {
	if err := require(config["schema_version"] == float64(1), "Unsupported site configuration schema_version"); err != nil {
		return nil, err
	}
	missing, err := validateRequired(config)
	if err != nil {
		return nil, err
	}
	if config.truthy("website_public_security_registration") {
		if err := require(securityCode.MatchString(config.str("website_public_security_code")),
			"website_public_security_code must contain the 14 digits of the website registration"); err != nil {
			return nil, err
		}
	}
	if config.truthy("public_base_url") {
		if err := validateBaseURL(config.str("public_base_url")); err != nil {
			return nil, err
		}
	}
	if config.truthy("support_email") {
		email := config.str("support_email")
		domain := email[strings.LastIndex(email, "@")+1:]
		if err := require(emailPattern.MatchString(email) && isPublicHost(domain),
			"support_email must be a real public email address, without a mailto: prefix"); err != nil {
			return nil, err
		}
	}
	if err := require(preview || len(missing) == 0,
		"Public details are not confirmed: "+strings.Join(missing, ", ")+
			". Set store/site-config.json, or use --preview for a local draft."); err != nil {
		return nil, err
	}
	return missing, nil
}

// escape matches HTML escaping with quotes: &, <, >, " and '.
var escaper = strings.NewReplacer("&", "&amp;", "<", "&lt;", ">", "&gt;", `"`, "&quot;", "'", "&#x27;")

func escape(text string) string { return escaper.Replace(text) }

// quoteEmail percent-encodes everything except letters, digits and _.-~@+.
func quoteEmail(text string) string {
	var out strings.Builder
	for _, b := range []byte(text) {
		if b >= 'a' && b <= 'z' || b >= 'A' && b <= 'Z' || b >= '0' && b <= '9' || strings.IndexByte("_.-~@+", b) >= 0 {
			out.WriteByte(b)
		} else {
			fmt.Fprintf(&out, "%%%02X", b)
		}
	}
	return out.String()
}

func link(href, label, attributes string) string {
	return `<a href="` + escape(href) + `"` + attributes + ">" + escape(label) + "</a>"
}

func paragraph(text, class string) string {
	attribute := ""
	if class != "" {
		attribute = ` class="` + class + `"`
	}
	return "<p" + attribute + ">" + escape(text) + "</p>"
}

// object is a decoded JSON object from content.json.
type object map[string]any

func (o object) text(key string) string {
	value, ok := o[key]
	if !ok {
		panic(contentError("missing key " + key))
	}
	text, ok := value.(string)
	if !ok {
		panic(contentError(key + " must be a string"))
	}
	return text
}

func (o object) obj(key string) object {
	value, ok := o[key].(map[string]any)
	if !ok {
		panic(contentError(key + " must be an object"))
	}
	return value
}

func (o object) list(key string) []any {
	value, present := o[key]
	if !present {
		return nil
	}
	items, ok := value.([]any)
	if !ok {
		panic(contentError(key + " must be a list"))
	}
	return items
}

func asObject(value any) object {
	item, ok := value.(map[string]any)
	if !ok {
		panic(contentError("list item must be an object"))
	}
	return item
}

func asText(value any) string {
	text, ok := value.(string)
	if !ok {
		panic(contentError("list item must be a string"))
	}
	return text
}

type contentError string

func contact(config Config, translation object) string {
	if email := config.str("support_email"); config.truthy("support_email") {
		return `<p class="contact">` + link("mailto:"+quoteEmail(email), email, "") + "</p>"
	}
	return paragraph(translation.text("contact_pending"), "pending")
}

func websiteRegistrations(config Config, translation object, prefix string) string {
	var links []string
	if config.truthy("website_icp_registration") {
		links = append(links, link("https://beian.miit.gov.cn/", config.str("website_icp_registration"), ""))
	}
	if config.truthy("website_public_security_registration") {
		href := "https://beian.mps.gov.cn/#/query/webSearch?code=" + config.str("website_public_security_code")
		icon := `<img src="` + escape(prefix+PublicSecurityIcon) + `" alt="" width="16" height="16">`
		links = append(links, `<a href="`+escape(href)+`">`+icon+escape(config.str("website_public_security_registration"))+"</a>")
	}
	if len(links) == 0 {
		return ""
	}
	return `<div class="website-registrations">` + paragraph(translation.text("website_registration_label"), "") +
		strings.Join(links, "\n") + "</div>"
}

func sectionContent(section object, config Config, translation object) []string {
	output := []string{`<section id="` + escape(section.text("id")) + `"><h2>` + escape(section.text("title")) + "</h2>"}
	for _, text := range section.list("paragraphs") {
		output = append(output, paragraph(asText(text), ""))
	}
	if _, ok := section["steps"]; ok {
		var steps strings.Builder
		for _, step := range section.list("steps") {
			steps.WriteString("<li>" + escape(asText(step)) + "</li>")
		}
		output = append(output, "<ol>"+steps.String()+"</ol>")
	}
	for _, item := range section.list("questions") {
		question := asObject(item)
		output = append(output, "<h3>"+escape(question.text("title"))+"</h3>"+paragraph(question.text("text"), ""))
	}
	if truthyValue(section["contact"]) {
		output = append(output, contact(config, translation))
	}
	return append(output, "</section>")
}

func truthyValue(value any) bool { return Config{"v": value}.truthy("v") }

func pageContent(translation object, kind string, config Config) string {
	page := translation.obj(kind)
	sections := page.list("sections")
	if _, ok := page["sections"]; !ok {
		panic(contentError("missing key sections"))
	}
	var contents strings.Builder
	for _, item := range sections {
		section := asObject(item)
		contents.WriteString("<li>" + link("#"+section.text("id"), section.text("title"), "") + "</li>")
	}
	output := []string{"<h1>" + escape(page.text("title")) + "</h1>", paragraph(page.text("lead"), "lead"),
		`<nav class="contents" aria-label="` + escape(translation.text("contents_label")) + `"><ul>` + contents.String() + "</ul></nav>"}
	for _, item := range sections {
		output = append(output, sectionContent(asObject(item), config, translation)...)
	}
	return strings.Join(output, "\n")
}

func headExtras(config Config, preview bool, kind, relativePath string) []string {
	head := []string{`<meta http-equiv="Content-Security-Policy" content="default-src 'none'; style-src 'self'; img-src 'self'; base-uri 'none'; form-action 'none'">`}
	if preview {
		head = append(head, `<meta name="robots" content="noindex, nofollow">`)
	}
	if config.truthy("public_base_url") && !preview {
		base := strings.TrimRight(config.str("public_base_url"), "/") + "/"
		head = append(head, `<link rel="canonical" href="`+escape(base+relativePath)+`">`)
		if kind != "" {
			for _, alternate := range languages {
				head = append(head, `<link rel="alternate" hreflang="`+alternate+`" href="`+escape(base+alternate+"/"+kind+".html")+`">`)
			}
		}
	}
	return head
}

func languageSwitchLinks(translations map[string]object, language, kind, prefix string) []string {
	var links []string
	for _, alternate := range languages {
		target := prefix + alternate + "/" + orDefault(kind, "support") + ".html"
		attributes := ` lang="` + alternate + `" hreflang="` + alternate + `"`
		if alternate == language && kind != "" {
			attributes += ` aria-current="page"`
		}
		links = append(links, link(target, translations[alternate].text("name"), attributes))
	}
	return links
}

func orDefault(value, fallback string) string {
	if value == "" {
		return fallback
	}
	return value
}

func indexContent(translations map[string]object, translation object) string {
	content := []string{"<h1>" + escape(translation.text("index_title")) + "</h1>", paragraph(translation.text("index_lead"), "lead"), `<div class="index-languages">`}
	for _, alternate := range languages {
		localized := translations[alternate]
		var pages []string
		for _, page := range pageKinds {
			pages = append(pages, link(alternate+"/"+page+".html", localized.obj(page).text("title"), ""))
		}
		content = append(content, `<section lang="`+alternate+`"><h2>`+escape(localized.text("name"))+"</h2><p>"+strings.Join(pages, " ")+"</p></section>")
	}
	content = append(content, "</div>")
	return strings.Join(content, "\n")
}

func draftNotice(translation object, missing []string, preview bool) string {
	if !preview {
		return ""
	}
	notice := translation.text("draft")
	if len(missing) > 0 {
		names := translation.obj("missing")
		var labels []string
		for _, field := range missing {
			labels = append(labels, names.text(field))
		}
		notice += " " + translation.text("pending_label") + strings.Join(labels, ", ")
	}
	return `<aside class="draft-notice">` + paragraph(notice, "") + "</aside>"
}

// substitute fills $name and ${name} placeholders; $$ is a literal dollar sign.
func substitute(template string, values map[string]string) string {
	return templatePattern.ReplaceAllStringFunc(template, func(match string) string {
		groups := templatePattern.FindStringSubmatch(match)
		if groups[1] != "" {
			return "$"
		}
		name := groups[2] + groups[3]
		if name == "" {
			panic(contentError("invalid placeholder in page template"))
		}
		value, ok := values[name]
		if !ok {
			panic(contentError("unknown placeholder " + name))
		}
		return value
	})
}

type renderer struct {
	template     string
	translations map[string]object
	config       Config
	missing      []string
	preview      bool
}

func (r renderer) page(language, kind string) string {
	translation := r.translations[language]
	isIndex := kind == ""
	prefix, relative := "../", language+"/"+kind+".html"
	title, description := "", ""
	if isIndex {
		prefix, relative = "", "index.html"
		title, description = translation.text("index_title"), translation.text("index_lead")
	} else {
		title, description = translation.obj(kind).text("title"), translation.obj(kind).text("description")
	}
	title += " | " + translation.text("app_name")
	if r.preview {
		title = translation.text("draft_title") + " | " + title
	}
	content := ""
	if isIndex {
		content = indexContent(r.translations, translation)
	} else {
		content = pageContent(translation, kind, r.config)
	}
	copyright := translation.text("copyright_pending")
	if r.config.truthy("copyright_holder") {
		copyright = "© " + r.config.str("copyright_holder")
	}
	var footer []string
	for _, page := range pageKinds {
		footer = append(footer, link(prefix+language+"/"+page+".html", translation.obj(page).text("title"), ""))
	}
	return substitute(r.template, map[string]string{
		"language": language, "title": escape(title), "app_name": escape(translation.text("app_name")),
		"description": escape(description), "head_extra": strings.Join(headExtras(r.config, r.preview, kind, relative), "\n  "),
		"stylesheet": prefix + "styles.css", "home": prefix + "index.html",
		"skip": escape(translation.text("skip")), "draft_notice": draftNotice(translation, r.missing, r.preview),
		"language_label": escape(translation.text("language_label")),
		"language_links": strings.Join(languageSwitchLinks(r.translations, language, kind, prefix), "\n"),
		"content":        content, "footer_label": escape(translation.text("footer_label")),
		"footer_links": strings.Join(footer, "\n"), "copyright": escape(copyright),
		"website_registrations": websiteRegistrations(r.config, translation, prefix),
	})
}

type pageSet struct {
	Support string `json:"support"`
	Privacy string `json:"privacy"`
}

type pageIndex struct {
	ZhHans pageSet `json:"zh-Hans"`
	ZhHant pageSet `json:"zh-Hant"`
	En     pageSet `json:"en"`
}

type manifest struct {
	SchemaVersion        int       `json:"schema_version"`
	Status               string    `json:"status"`
	Preview              bool      `json:"preview"`
	MissingPublicDetails []string  `json:"missing_public_details"`
	PublicBaseURL        any       `json:"public_base_url"`
	Files                []string  `json:"files"`
	Pages                pageIndex `json:"pages"`
}

// resolve makes a path absolute and resolves symlinks in its longest existing prefix.
func resolve(path string) (string, error) {
	path, err := filepath.Abs(path)
	if err != nil {
		return "", err
	}
	rest := ""
	for current := path; ; current = filepath.Dir(current) {
		if real, err := filepath.EvalSymlinks(current); err == nil {
			return filepath.Join(real, rest), nil
		}
		if current == filepath.Dir(current) {
			return path, nil
		}
		rest = filepath.Join(filepath.Base(current), rest)
	}
}

func within(path, parent string) bool {
	relative, err := filepath.Rel(parent, path)
	return err == nil && relative != ".." && !strings.HasPrefix(relative, ".."+string(filepath.Separator))
}

func readJSON(path string, target any) error {
	data, err := os.ReadFile(path)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, target)
}

// Result describes what Render wrote.
type Result struct {
	Output  string
	Preview bool
}

// Render validates the configuration and content, then writes the site into output.
func Render(root, output string, preview bool) (result Result, err error) {
	defer func() {
		if recovered := recover(); recovered != nil {
			message, ok := recovered.(contentError)
			if !ok {
				panic(recovered)
			}
			err = errors.New(string(message))
		}
	}()
	config, err := loadConfig(root)
	if err != nil {
		return result, err
	}
	missing, err := ValidateConfig(config, preview)
	if err != nil {
		return result, err
	}
	source, output, err := checkOutput(root, output)
	if err != nil {
		return result, err
	}
	r, err := loadContent(source, config, missing, preview)
	if err != nil {
		return result, err
	}
	pages := map[string]string{"index.html": r.page("zh-Hans", "")}
	for _, language := range languages {
		for _, kind := range pageKinds {
			pages[language+"/"+kind+".html"] = r.page(language, kind)
		}
	}
	// Render and validate before creating any output, so missing details never
	// leave behind something that could be mistaken for a publishable build.
	writes, err := siteFiles(source, config, missing, preview, pages)
	if err != nil {
		return result, err
	}
	if err := writeAll(output, writes); err != nil {
		return result, err
	}
	return Result{Output: output, Preview: preview}, nil
}

func loadConfig(root string) (Config, error) {
	var raw any
	if err := readJSON(filepath.Join(root, "store", "site-config.json"), &raw); err != nil {
		return nil, err
	}
	decoded, ok := raw.(map[string]any)
	return decoded, require(ok, "Site configuration must be a JSON object")
}

// checkOutput resolves the source and output directories and refuses an output
// that would overwrite the project root or the store/site sources.
func checkOutput(root, output string) (string, string, error) {
	source, err := resolve(filepath.Join(root, "store", "site"))
	if err != nil {
		return "", "", err
	}
	if output, err = resolve(output); err != nil {
		return "", "", err
	}
	if root, err = resolve(root); err != nil {
		return "", "", err
	}
	return source, output, require(output != root && output != source && !within(source, output) && !within(output, source),
		"Output must not overwrite the project root or store/site sources")
}

func loadContent(source string, config Config, missing []string, preview bool) (renderer, error) {
	var raw map[string]map[string]any
	if err := readJSON(filepath.Join(source, "content.json"), &raw); err != nil {
		return renderer{}, err
	}
	translations := map[string]object{}
	for language, translation := range raw {
		translations[language] = translation
	}
	if err := require(sameKeys(translations, languages), "Content must contain exactly zh-Hans, zh-Hant and en"); err != nil {
		return renderer{}, err
	}
	template, err := os.ReadFile(filepath.Join(source, "page.html"))
	if err != nil {
		return renderer{}, err
	}
	return renderer{template: string(template), translations: translations, config: config, missing: missing, preview: preview}, nil
}

// siteFiles returns every output file by its path relative to the output directory.
func siteFiles(source string, config Config, missing []string, preview bool, pages map[string]string) (map[string][]byte, error) {
	styles, err := os.ReadFile(filepath.Join(source, "styles.css"))
	if err != nil {
		return nil, err
	}
	writes := map[string][]byte{"styles.css": styles}
	if config.truthy("website_public_security_registration") {
		if writes[PublicSecurityIcon], err = os.ReadFile(filepath.Join(source, PublicSecurityIcon)); err != nil {
			return nil, err
		}
	}
	for name, html := range pages {
		writes[name] = []byte(html)
	}
	files := make([]string, 0, len(writes))
	for name := range writes {
		files = append(files, name)
	}
	sort.Strings(files)
	status := "ready-for-deployment"
	if preview {
		status = "preview-draft"
	}
	set := func(language string) pageSet {
		return pageSet{Support: language + "/support.html", Privacy: language + "/privacy.html"}
	}
	record := manifest{SchemaVersion: 1, Status: status, Preview: preview, MissingPublicDetails: missing,
		PublicBaseURL: config["public_base_url"], Files: files, Pages: pageIndex{set("zh-Hans"), set("zh-Hant"), set("en")}}
	var encoded bytes.Buffer
	encoder := json.NewEncoder(&encoded)
	encoder.SetEscapeHTML(false)
	encoder.SetIndent("", "  ")
	if err := encoder.Encode(record); err != nil {
		return nil, err
	}
	writes["manifest.json"] = encoded.Bytes()
	return writes, nil
}

func writeAll(output string, writes map[string][]byte) error {
	if err := os.MkdirAll(output, 0o755); err != nil {
		return err
	}
	for name, data := range writes {
		destination := filepath.Join(output, filepath.FromSlash(name))
		if err := os.MkdirAll(filepath.Dir(destination), 0o755); err != nil {
			return err
		}
		if err := os.WriteFile(destination, data, 0o644); err != nil {
			return err
		}
	}
	return nil
}

func sameKeys(translations map[string]object, want []string) bool {
	if len(translations) != len(want) {
		return false
	}
	for _, key := range want {
		if _, ok := translations[key]; !ok {
			return false
		}
	}
	return true
}

// Main runs the store-site command from the repository root.
func Main(args []string, stdout, stderr io.Writer) int {
	flags := flag.NewFlagSet("store-site", flag.ContinueOnError)
	flags.SetOutput(stderr)
	preview := flags.Bool("preview", false, "Generate an explicitly marked local draft, even with missing public details")
	output := flags.String("output", filepath.Join("build", "store-site"), "Output directory")
	if err := flags.Parse(args); err != nil {
		return 2
	}
	if flags.NArg() != 0 {
		fmt.Fprintln(stderr, "store-site takes no positional arguments")
		return 2
	}
	result, err := Render(".", *output, *preview)
	if err != nil {
		fmt.Fprintf(stderr, "Store site not generated: %v\n", err)
		return 1
	}
	label := "Ready for deployment"
	if result.Preview {
		label = "PREVIEW DRAFT"
	}
	fmt.Fprintf(stdout, "%s: %s\n", label, result.Output)
	return 0
}
