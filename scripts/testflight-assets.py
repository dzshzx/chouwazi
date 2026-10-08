#!/usr/bin/env -S uv run --locked --quiet --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
#
# [tool.uv]
# exclude-newer = "1 day"
# ///
"""Handle signing assets without printing certificate/profile/key contents."""
import base64
from datetime import datetime, timezone
import hashlib
import json
import os
from pathlib import Path
import plistlib
import re
import runpy
import secrets
import shlex
import shutil
import subprocess
import sys

BUNDLE_ID = "com.alex.chouwazi"
PROJECT = Path("ios/Runner.xcodeproj/project.pbxproj")


class SigningAssetValidationError(ValueError):
    """A validation failure with an audited, secret-free public message."""


def require(condition, message):
    if not condition:
        raise SigningAssetValidationError(message)


def run(*args):
    result = subprocess.run(args, capture_output=True, check=False)
    if args[0] == "codesign" and result.returncode != 0:
        operations = (("--verify", "verify"), ("--entitlements", "display-entitlements"),
                      ("--extract-certificates", "extract-certificates"))
        operation = next((name for flag, name in operations
                          if any(arg == flag or arg.startswith(flag + "=") for arg in args[1:])), "unknown")
        stderr = result.stderr.decode("utf-8", errors="replace")
        lowered_stderr = stderr.lower()
        reasons = (
            ("unsigned", ("code object is not signed at all",)),
            ("sealed-resource-missing-or-invalid", ("sealed resource is missing or invalid",)),
            ("main-executable-strict-validation", ("main executable failed strict validation",)),
            ("unrecognized-or-ambiguous-bundle", ("bundle format unrecognized", "bundle format is ambiguous")),
            ("invalid-signature", ("invalid signature", "signature invalid", "code or signature have been modified",
                                   "cssmerr_csp_verify_failed")),
            ("trust-or-certificate-expiry", ("cssmerr_tp_cert_expired", "cssmerr_tp_not_trusted",
                                             "cssmerr_tp_cert_not_valid_yet", "errseccertificateexpired",
                                             "certificate has expired", "certificate is not trusted",
                                             "unable to build chain to self-signed root")),
            ("unsupported-syntax-or-argument", ("invalid argument", "unknown option", "unrecognized option",
                                                "unsupported option", "invalid option", "illegal option",
                                                "requires an argument", "usage: codesign")),
            ("file-or-path-not-found", ("no such file", "file not found", "path not found")),
        )
        reason = next((name for name, patterns in reasons
                       if any(pattern in lowered_stderr for pattern in patterns)), "unknown")
        subcomponents = re.findall(r"^In subcomponent:[ \t]*([^\r\n]+)", stderr, re.MULTILINE)
        component = "none" if not subcomponents else "unknown"
        if subcomponents:
            match = re.search(r"(?:^|/)(Runner\.app|Flutter\.framework|App\.framework)/?$",
                              subcomponents[0].strip())
            if match:
                component = match[1]
        # stderr is untrusted; only fixed operation/reason/component tokens are public.
        require(False, f"codesign {operation} failed while handling signing assets "
                       f"(exit={result.returncode}, reason={reason}, subcomponent={component})")
    require(result.returncode == 0, f"{args[0]} failed while handling signing assets")
    return result.stdout


def state_directory():
    require(os.environ.get("GITHUB_ACTIONS") == "true", "Requires a GitHub-hosted workflow")
    run_id = os.environ["GITHUB_RUN_ID"]
    attempt = os.environ["GITHUB_RUN_ATTEMPT"]
    require(run_id.isdecimal() and attempt.isdecimal(), "Invalid workflow run identifier")
    runner_temp = Path(os.environ["RUNNER_TEMP"]).resolve()
    state = runner_temp / f"chouwazi-testflight-{run_id}-{attempt}"
    require(not state.is_symlink(), "Signing state cannot be a symlink")
    return state


def settings(state):
    return json.loads((state / "settings.json").read_text())


def profile_data(blob):
    profile = plistlib.loads(run("security", "cms", "-D", "-i", str(blob)))
    teams = profile.get("TeamIdentifier", [])
    prefixes = profile.get("ApplicationIdentifierPrefix", [])
    require(len(teams) == 1 and re.fullmatch(r"[A-Z0-9]{10}", teams[0]), "Invalid profile team")
    require(len(prefixes) == 1 and re.fullmatch(r"[A-Z0-9]{10}", prefixes[0]), "Invalid App ID prefix")
    require(re.fullmatch(r"[A-Fa-f0-9-]{36}", profile.get("UUID", "")), "Invalid profile UUID")
    entitlements = profile.get("Entitlements", {})
    require(entitlements.get("application-identifier") == f"{prefixes[0]}.{BUNDLE_ID}", "Profile App ID does not match Runner")
    require(entitlements.get("com.apple.developer.team-identifier") == teams[0], "Profile team entitlement mismatch")
    require(entitlements.get("get-task-allow") is False, "A development profile cannot sign this upload")
    require(entitlements.get("beta-reports-active") is True, "Requires an App Store distribution profile")
    require("ProvisionedDevices" not in profile and not profile.get("ProvisionsAllDevices"), "Ad hoc/enterprise profiles cannot be used")
    expires = profile.get("ExpirationDate")
    require(isinstance(expires, datetime), "Missing profile expiry")
    require(expires.replace(tzinfo=timezone.utc) > datetime.now(timezone.utc), "Provisioning profile has expired")
    require(profile.get("DeveloperCertificates"), "Profile contains no distribution certificate")
    return profile


def prepare(state):
    names = ["IOS_DISTRIBUTION_P12_BASE64", "IOS_DISTRIBUTION_P12_PASSWORD",
             "IOS_APP_STORE_PROFILE_BASE64", "ASC_API_KEY_P8", "ASC_API_KEY_ID", "ASC_API_ISSUER_ID"]
    require(all(os.environ.get(name) for name in names), "Required TestFlight environment secrets are missing")
    require(not state.exists(), "Signing state already exists; cleanup the previous attempt first")
    state.mkdir(mode=0o700)
    for name, filename in [(names[0], "distribution.p12"), (names[2], "app.mobileprovision")]:
        try:
            blob = base64.b64decode("".join(os.environ[name].split()), validate=True)
        except ValueError:
            raise ValueError("Invalid base64 signing asset") from None
        require(bool(blob), "Empty signing asset")
        (state / filename).write_bytes(blob)
    profile = profile_data(state / "app.mobileprovision")
    key_id = os.environ["ASC_API_KEY_ID"]
    issuer = os.environ["ASC_API_ISSUER_ID"]
    require(re.fullmatch(r"[A-Z0-9]{10}", key_id), "Invalid App Store Connect key ID")
    require(re.fullmatch(r"[A-Fa-f0-9]{8}(?:-[A-Fa-f0-9]{4}){3}-[A-Fa-f0-9]{12}", issuer), "Invalid App Store Connect issuer ID")
    private_keys = state / "private_keys"
    private_keys.mkdir(mode=0o700)
    private_key = private_keys / f"AuthKey_{key_id}.p8"
    private_key.write_text(os.environ["ASC_API_KEY_P8"])
    run("openssl", "pkey", "-in", str(private_key), "-noout")
    version = re.search(r"^version:\s*(\d+\.\d+\.\d+)\+(\d+)\s*$", Path("pubspec.yaml").read_text(), re.MULTILINE)
    require(version is not None, "Expected version name+build in pubspec.yaml")
    data = {"team_id": profile["TeamIdentifier"][0], "app_id": profile["Entitlements"]["application-identifier"],
            "profile_uuid": profile["UUID"], "version": version[1], "build": version[2],
            "api_key_id": key_id, "api_issuer_id": issuer,
            "certificate_sha1s": [hashlib.sha1(cert).hexdigest().upper() for cert in profile["DeveloperCertificates"]]}
    (state / "settings.json").write_text(json.dumps(data))
    (state / "keychain-password").write_text(secrets.token_urlsafe(32))
    original_keychains = shlex.split(run("security", "list-keychains", "-d", "user").decode())
    (state / "keychains.json").write_text(json.dumps(original_keychains))
    (state / "project.pbxproj").write_bytes(PROJECT.read_bytes())


def install(state):
    data = settings(state)
    keychain = str(state / "signing.keychain-db")
    identities = run("security", "find-identity", "-v", "-p", "codesigning", keychain).decode()
    matches = [sha for sha, name in re.findall(r'\b([A-F0-9]{40})\s+"([^"\n]+)"', identities)
               if sha in data["certificate_sha1s"] and name.startswith(("Apple Distribution:", "iPhone Distribution:"))]
    require(len(matches) == 1, "Requires one valid distribution identity matching the profile")
    data["certificate_sha1"] = matches[0]
    (state / "settings.json").write_text(json.dumps(data))
    original_keychains = json.loads((state / "keychains.json").read_text())
    run("security", "list-keychains", "-d", "user", "-s", keychain, *original_keychains)
    profiles = Path.home() / "Library/MobileDevice/Provisioning Profiles"
    profiles.mkdir(parents=True, exist_ok=True)
    installed = profiles / f"{data['profile_uuid']}.mobileprovision"
    require(not installed.exists(), "Runner already has a profile with this UUID")
    # Record ownership before writing, so cleanup also works after interruption.
    (state / "installed-profile").write_text(str(installed))
    shutil.copyfile(state / "app.mobileprovision", installed)
    options = {"method": "app-store-connect", "destination": "export", "signingStyle": "manual",
               "teamID": data["team_id"], "signingCertificate": data["certificate_sha1"],
               "provisioningProfiles": {BUNDLE_ID: data["profile_uuid"]},
               "manageAppVersionAndBuildNumber": False, "uploadSymbols": True}
    (state / "ExportOptions.plist").write_bytes(plistlib.dumps(options))


def exported_ipa(state):
    export = state / "export"
    require(not export.is_symlink(), "IPA export directory cannot be a symlink")
    ipas = [ipa for ipa in export.glob("*.ipa") if ipa.is_file() and not ipa.is_symlink()]
    require(bool(ipas), "Xcode did not export a regular IPA file")
    require(len(ipas) == 1, "Xcode exported multiple IPA files")
    ipa = export / "Runner.ipa"
    if ipas[0] != ipa:
        require(not ipa.exists() and not ipa.is_symlink(), "Cannot normalize IPA name over an existing export")
        ipas[0].rename(ipa)
    return ipa


def verify(state):
    data = settings(state)
    ipa = exported_ipa(state)
    unpacked = state / "verified-ipa"
    # Apple's extractor preserves framework symlinks and executable modes.
    run("ditto", "-x", "-k", str(ipa), str(unpacked))
    apps = list((unpacked / "Payload").glob("*.app"))
    require(len(apps) == 1 and apps[0].name == "Runner.app", "Unexpected IPA app payload")
    app = apps[0]
    info = plistlib.loads((app / "Info.plist").read_bytes())
    require(info.get("CFBundleIdentifier") == BUNDLE_ID, "Exported bundle ID mismatch")
    require(info.get("CFBundleShortVersionString") == data["version"] and info.get("CFBundleVersion") == data["build"], "Export changed the version/build")
    require(info.get("ITSAppUsesNonExemptEncryption") is False, "Missing export compliance declaration")
    require(info.get("CFBundleSupportedPlatforms") == ["iPhoneOS"], "Export is not an iPhoneOS device build")
    require(int(info.get("DTXcode", "0")) >= 2600, "Export was not built with Xcode 26 or later")
    require(re.fullmatch(r"iphoneos(?:2[6-9]|[3-9]\d)\.\d+(?:\.\d+)?", info.get("DTSDKName", "")), "Export requires iOS SDK 26 or later")
    for manifest in [app / "PrivacyInfo.xcprivacy", app / "Frameworks/Flutter.framework/PrivacyInfo.xcprivacy"]:
        require(manifest.is_file(), "Missing app or Flutter privacy manifest")
        plistlib.loads(manifest.read_bytes())
    sys.argv = ["scripts/check-ios.py", str(app)]
    runpy.run_path("scripts/check-ios.py", run_name="__main__")
    run("codesign", "--verify", "--deep", "--strict", str(app))
    entitlements = plistlib.loads(run("codesign", "-d", "--entitlements", ":-", str(app)))
    require(entitlements.get("application-identifier") == data["app_id"], "Signed App ID mismatch")
    require(entitlements.get("com.apple.developer.team-identifier") == data["team_id"], "Signed team mismatch")
    require(entitlements.get("get-task-allow") is False, "IPA permits debugging")
    require(entitlements.get("beta-reports-active") is True, "IPA is not signed for App Store distribution")
    allowed = {"application-identifier", "com.apple.developer.team-identifier", "get-task-allow", "beta-reports-active", "keychain-access-groups"}
    require(set(entitlements) <= allowed, "IPA includes an unexpected capability")
    require(all(group == data["app_id"] for group in entitlements.get("keychain-access-groups", [])), "IPA includes an unexpected keychain group")
    embedded = profile_data(app / "embedded.mobileprovision")
    require(embedded["UUID"] == data["profile_uuid"], "IPA uses a different provisioning profile")
    certificate_prefix = str(state / "signer-")
    # This optional argument must share an argv item with the option.
    run("codesign", "-d", f"--extract-certificates={certificate_prefix}", str(app))
    leaf_sha1 = hashlib.sha1(Path(certificate_prefix + "0").read_bytes()).hexdigest().upper()
    require(leaf_sha1 == data["certificate_sha1"], "IPA signing certificate mismatch")
    print(f"Verified distribution IPA: {BUNDLE_ID} {data['version']} ({data['build']}); SHA-256 {hashlib.sha256(ipa.read_bytes()).hexdigest()}")


def cleanup(state):
    if not state.exists():
        return
    errors = []
    if (state / "project.pbxproj").exists():
        try:
            PROJECT.write_bytes((state / "project.pbxproj").read_bytes())
        except OSError:
            errors.append("restore Runner project")
    if (state / "keychains.json").exists():
        original = json.loads((state / "keychains.json").read_text())
        try:
            run("security", "list-keychains", "-d", "user", "-s", *original)
        except ValueError:
            errors.append("restore keychain search list")
    keychain = state / "signing.keychain-db"
    if keychain.exists():
        try:
            run("security", "delete-keychain", str(keychain))
        except ValueError:
            errors.append("delete temporary keychain")
    if (state / "installed-profile").exists():
        installed = Path((state / "installed-profile").read_text())
        expected = Path.home() / "Library/MobileDevice/Provisioning Profiles"
        require(installed.parent == expected and re.fullmatch(r"[A-Fa-f0-9-]{36}\.mobileprovision", installed.name), "Invalid cleanup profile location")
        try:
            installed.unlink(missing_ok=True)
        except OSError:
            errors.append("remove installed profile")
    # Keep restoration metadata if security cleanup failed, but remove key files.
    for filename in ["distribution.p12", "app.mobileprovision", "keychain-password"]:
        (state / filename).unlink(missing_ok=True)
    shutil.rmtree(state / "private_keys", ignore_errors=True)
    require(not errors, "Signing cleanup failed: " + ", ".join(errors))
    shutil.rmtree(state)


def main():
    os.umask(0o077)
    state = state_directory()
    operation = sys.argv[1]
    if operation == "field":
        require(sys.argv[2] in {"api_key_id", "api_issuer_id"}, "Unknown upload metadata field")
        print(settings(state)[sys.argv[2]])
    else:
        {"prepare": prepare, "install": install, "verify": verify, "cleanup": cleanup}[operation](state)


if __name__ == "__main__":
    try:
        main()
    except SigningAssetValidationError as error:
        # require() messages contain only fixed text/tokens, command names and exit codes.
        print(f"TestFlight asset operation failed: {error}", file=sys.stderr)
        sys.exit(1)
    except Exception:
        # External exceptions can carry binary or key material. Never print them.
        print("TestFlight asset operation failed; check secrets, signing assets and workflow prerequisites.", file=sys.stderr)
        sys.exit(1)
