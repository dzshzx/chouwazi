#!/usr/bin/env -S uv run --locked --quiet --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
#
# [tool.uv]
# exclude-newer = "1 day"
# ///
"""Render the fixed store/site sources; never deploy or fetch remote resources.

Default mode requires confirmed public details in store/site-config.json.
Use --preview for an explicitly marked, non-indexable local draft.
"""

import argparse
from html import escape
import json
from pathlib import Path
import re
from string import Template
import sys
from urllib.parse import quote, urlsplit


ROOT = Path(__file__).resolve().parent.parent
SOURCE = ROOT / "store" / "site"
CONFIG = ROOT / "store" / "site-config.json"
LANGUAGES = ("zh-Hans", "zh-Hant", "en")
PAGE_KINDS = ("support", "privacy")
REQUIRED = ("public_base_url", "support_email", "copyright_holder")
REGISTRATION_FIELDS = ("website_icp_registration", "website_public_security_registration", "website_public_security_code")
PUBLIC_SECURITY_ICON = "assets/public-security-beian.png"


def require(condition, message):
    if not condition:
        raise ValueError(message)


def is_public_host(host):
    if not host or "." not in host:
        return False
    try:
        host = host.lower().rstrip(".").encode("idna").decode("ascii")
    except UnicodeError:
        return False
    labels = host.split(".")
    if len(host) > 253 or labels[-1].isdecimal() or not all(
            re.fullmatch(r"[a-z0-9](?:[a-z0-9-]{0,61}[a-z0-9])?", label) for label in labels):
        return False
    reserved = ("localhost", "local", "internal", "invalid", "test", "example", "example.com", "example.net", "example.org")
    return not any(host == name or host.endswith("." + name) for name in reserved)


def validate_config(config, preview):
    require(isinstance(config, dict), "Site configuration must be a JSON object")
    require(config.get("schema_version") == 1, "Unsupported site configuration schema_version")
    missing = []
    for field in REQUIRED:
        value = config.get(field)
        if value is None or value == "":
            missing.append(field)
            continue
        require(isinstance(value, str) and value.strip() == value and bool(value),
                f"{field} must be a nonempty string without surrounding whitespace, or null")
        require(not any(ord(char) < 32 for char in value), f"{field} contains a control character")
    for field in REGISTRATION_FIELDS:
        value = config.get(field)
        require(value is None or isinstance(value, str) and value.strip() == value
                and bool(value) and not any(ord(char) < 32 for char in value),
                f"{field} must be a nonempty string without surrounding whitespace or control characters, or null")
    if config.get("website_public_security_registration"):
        require(re.fullmatch(r"[0-9]{14}", config.get("website_public_security_code") or ""),
                "website_public_security_code must contain the 14 digits of the website registration")
    if config.get("public_base_url"):
        url = urlsplit(config["public_base_url"])
        require(url.scheme == "https" and is_public_host(url.hostname)
                and not url.username and not url.password and not url.query and not url.fragment
                and not re.search(r"\s", config["public_base_url"]),
                "public_base_url must be a real public HTTPS URL without credentials, query or fragment")
        require(url.port in (None, 443), "public_base_url must use the default HTTPS port")
        require(".." not in url.path.split("/"), "public_base_url cannot contain parent path segments")
    if config.get("support_email"):
        email = config["support_email"]
        require(re.fullmatch(r"[A-Za-z0-9.!#$%&'*+/=?^_`{|}~-]+@[A-Za-z0-9.-]+", email)
                and is_public_host(email.rsplit("@", 1)[-1]),
                "support_email must be a real public email address, without a mailto: prefix")
    require(preview or not missing,
            "Public details are not confirmed: " + ", ".join(missing)
            + ". Set store/site-config.json, or use --preview for a local draft.")
    return missing


def link(href, label, attributes=""):
    return f'<a href="{escape(href, quote=True)}"{attributes}>{escape(label)}</a>'


def paragraph(text, css_class=""):
    attribute = f' class="{css_class}"' if css_class else ""
    return f"<p{attribute}>{escape(text)}</p>"


def contact(config, translation):
    email = config.get("support_email")
    if email:
        href = "mailto:" + quote(email, safe="@.+-_")
        return '<p class="contact">' + link(href, email) + "</p>"
    return paragraph(translation["contact_pending"], "pending")


def website_registrations(config, translation, prefix):
    links = []
    if config.get("website_icp_registration"):
        links.append(link("https://beian.miit.gov.cn/", config["website_icp_registration"]))
    if config.get("website_public_security_registration"):
        href = "https://beian.mps.gov.cn/#/query/webSearch?code=" + config["website_public_security_code"]
        icon = f'<img src="{escape(prefix + PUBLIC_SECURITY_ICON, quote=True)}" alt="" width="16" height="16">'
        links.append(f'<a href="{escape(href, quote=True)}">{icon}{escape(config["website_public_security_registration"])}</a>')
    if not links:
        return ""
    return '<div class="website-registrations">' + paragraph(translation["website_registration_label"]) + "\n".join(links) + "</div>"


def page_content(translation, kind, config):
    page = translation[kind]
    sections = page["sections"]
    contents = "".join("<li>" + link("#" + section["id"], section["title"]) + "</li>"
                       for section in sections)
    output = [f'<h1>{escape(page["title"])}</h1>', paragraph(page["lead"], "lead"),
              f'<nav class="contents" aria-label="{escape(translation["contents_label"])}"><ul>{contents}</ul></nav>']
    for section in sections:
        output.append(f'<section id="{escape(section["id"], quote=True)}"><h2>{escape(section["title"])}</h2>')
        output.extend(paragraph(text) for text in section.get("paragraphs", []))
        if "steps" in section:
            output.append("<ol>" + "".join(f"<li>{escape(step)}</li>" for step in section["steps"]) + "</ol>")
        for question in section.get("questions", []):
            output.append(f'<h3>{escape(question["title"])}</h3>' + paragraph(question["text"]))
        if section.get("contact"):
            output.append(contact(config, translation))
        output.append("</section>")
    return "\n".join(output)


def render_page(template, translations, config, missing, preview, language, kind=None):
    translation = translations[language]
    is_index = kind is None
    prefix = "" if is_index else "../"
    relative_path = "index.html" if is_index else f"{language}/{kind}.html"
    title = translation["index_title"] if is_index else translation[kind]["title"]
    title += " | " + translation["app_name"]
    head_extra = ['<meta http-equiv="Content-Security-Policy" content="default-src \'none\'; style-src \'self\'; img-src \'self\'; base-uri \'none\'; form-action \'none\'">']
    if preview:
        title = translation["draft_title"] + " | " + title
        head_extra.append('<meta name="robots" content="noindex, nofollow">')
    base_url = config.get("public_base_url")
    if base_url and not preview:
        base_url = base_url.rstrip("/") + "/"
        head_extra.append(f'<link rel="canonical" href="{escape(base_url + relative_path, quote=True)}">')
        if not is_index:
            for alternate in LANGUAGES:
                head_extra.append(f'<link rel="alternate" hreflang="{alternate}" href="{escape(base_url + alternate + "/" + kind + ".html", quote=True)}">')
    language_links = []
    for alternate in LANGUAGES:
        target = f"{prefix}{alternate}/{kind or 'support'}.html"
        attributes = f' lang="{alternate}" hreflang="{alternate}"'
        if alternate == language and not is_index:
            attributes += ' aria-current="page"'
        language_links.append(link(target, translations[alternate]["name"], attributes))
    if is_index:
        content = [f'<h1>{escape(translation["index_title"])}</h1>', paragraph(translation["index_lead"], "lead"), '<div class="index-languages">']
        for alternate in LANGUAGES:
            localized = translations[alternate]
            content.append(f'<section lang="{alternate}"><h2>{escape(localized["name"])}</h2><p>'
                           + " ".join(link(f"{alternate}/{page}.html", localized[page]["title"])
                                      for page in PAGE_KINDS) + "</p></section>")
        content.append("</div>")
        content = "\n".join(content)
    else:
        content = page_content(translation, kind, config)
    notice = ""
    if preview:
        notice_text = translation["draft"]
        if missing:
            notice_text += " " + translation["pending_label"] + ", ".join(translation["missing"][field] for field in missing)
        notice = '<aside class="draft-notice">' + paragraph(notice_text) + "</aside>"
    copyright_text = "© " + config["copyright_holder"] if config.get("copyright_holder") else translation["copyright_pending"]
    return template.substitute(
        language=language, title=escape(title), app_name=escape(translation["app_name"]),
        description=escape(translation["index_lead"] if is_index else translation[kind]["description"], quote=True),
        head_extra="\n  ".join(head_extra), stylesheet=prefix + "styles.css", home=prefix + "index.html",
        skip=escape(translation["skip"]), draft_notice=notice,
        language_label=escape(translation["language_label"]), language_links="\n".join(language_links),
        content=content, footer_label=escape(translation["footer_label"]),
        footer_links="\n".join(link(prefix + language + "/" + page + ".html", translation[page]["title"])
                                for page in PAGE_KINDS), copyright=escape(copyright_text),
        website_registrations=website_registrations(config, translation, prefix))


def main(argv=None):
    parser = argparse.ArgumentParser(description=__doc__)
    parser.add_argument("--preview", action="store_true", help="Generate an explicitly marked local draft, even with missing public details")
    parser.add_argument("--output", type=Path, default=ROOT / "build" / "store-site", help="Output directory (default: build/store-site)")
    args = parser.parse_args(argv)
    try:
        config = json.loads(CONFIG.read_text(encoding="utf-8"))
        missing = validate_config(config, args.preview)
        output = args.output.resolve()
        require(output != ROOT and output != SOURCE and not SOURCE.is_relative_to(output)
                and not output.is_relative_to(SOURCE), "Output must not overwrite the project root or store/site sources")
        translations = json.loads((SOURCE / "content.json").read_text(encoding="utf-8"))
        require(set(translations) == set(LANGUAGES), "Content must contain exactly zh-Hans, zh-Hant and en")
        template = Template((SOURCE / "page.html").read_text(encoding="utf-8"))
        pages = {"index.html": render_page(template, translations, config, missing, args.preview, "zh-Hans")}
        for language in LANGUAGES:
            for kind in PAGE_KINDS:
                pages[f"{language}/{kind}.html"] = render_page(template, translations, config, missing, args.preview, language, kind)
        # Render and validate before creating any output, so missing details never
        # leave behind something that could be mistaken for a publishable build.
        styles = (SOURCE / "styles.css").read_text(encoding="utf-8")
        assets = {PUBLIC_SECURITY_ICON: (SOURCE / PUBLIC_SECURITY_ICON).read_bytes()} if config.get("website_public_security_registration") else {}
        manifest = {
            "schema_version": 1,
            "status": "preview-draft" if args.preview else "ready-for-deployment",
            "preview": args.preview,
            "missing_public_details": missing,
            "public_base_url": config.get("public_base_url"),
            "files": sorted([*pages, "styles.css", *assets]),
            "pages": {language: {kind: f"{language}/{kind}.html" for kind in PAGE_KINDS} for language in LANGUAGES},
        }
        output.mkdir(parents=True, exist_ok=True)
        for relative, html in pages.items():
            destination = output / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_text(html, encoding="utf-8")
        (output / "styles.css").write_text(styles, encoding="utf-8")
        for relative, data in assets.items():
            destination = output / relative
            destination.parent.mkdir(parents=True, exist_ok=True)
            destination.write_bytes(data)
        (output / "manifest.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n", encoding="utf-8")
    except (ValueError, OSError, KeyError, TypeError) as error:
        print(f"Store site not generated: {error}", file=sys.stderr)
        return 1
    print(f"{'PREVIEW DRAFT' if args.preview else 'Ready for deployment'}: {output}")
    return 0


if __name__ == "__main__":
    sys.exit(main())
