#!/usr/bin/env bash
# Check the actual iOS app bundles have the shared app, privacy manifest and localized names,
# and that a device build carries no debug network service. Runs on macOS (plutil).
# Usage: bash scripts/check-ios.sh <Runner.app>...
# With RELEASE_VERSION and RELEASE_BUILD set (release.yml), each app must carry that version and build.
set -euo pipefail
fail() {
  echo "$1" >&2
  exit 1
}
# value APP KEYPATH prints a string or number from Info.plist; has APP KEYPATH tests presence of any type.
value() { plutil -extract "$2" raw -o - "$1/Info.plist"; }
has() { plutil -extract "$2" xml1 -o - "$1/Info.plist" >/dev/null 2>&1; }

for app in "$@"; do
  [[ "$(value "$app" CFBundleIdentifier)" == com.alex.chouwazi ]] || fail "Unexpected bundle identifier in $app"
  [[ "$(value "$app" MinimumOSVersion)" == 15.0 ]] || fail "Unexpected minimum iOS version in $app"
  for name in PrivacyInfo.xcprivacy Assets.car Frameworks/Flutter.framework/Flutter \
    Frameworks/App.framework/flutter_assets/AssetManifest.bin; do
    [[ -f "$app/$name" ]] || fail "Missing $name in $app"
  done
  for language in en zh-Hans zh-Hant; do
    [[ -f "$app/$language.lproj/InfoPlist.strings" ]] || fail "Missing $language app name"
  done
  platform=$(value "$app" CFBundleSupportedPlatforms.0)
  if [[ "$platform" == iPhoneOS ]] && ! has "$app" CFBundleSupportedPlatforms.1; then
    ! has "$app" NSLocalNetworkUsageDescription || fail 'Device release includes debug network service'
    ! has "$app" NSBonjourServices || fail 'Device release advertises a network service'
  fi
  version=$(value "$app" CFBundleShortVersionString)
  build=$(value "$app" CFBundleVersion)
  [[ -z "${RELEASE_VERSION:-}" || "$version" == "$RELEASE_VERSION" ]] || fail "$app has version $version, expected $RELEASE_VERSION"
  [[ -z "${RELEASE_BUILD:-}" || "$build" == "$RELEASE_BUILD" ]] || fail "$app has build $build, expected $RELEASE_BUILD"
  echo "Verified $app: $platform, $version ($build)"
done
