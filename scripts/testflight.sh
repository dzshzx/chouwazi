#!/usr/bin/env bash
# Only run on a trusted main checkout in the protected testflight environment.
set +x
set -euo pipefail
umask 077

if [[ "${GITHUB_ACTIONS:-}" != true || "${GITHUB_EVENT_NAME:-}" != workflow_dispatch || "${GITHUB_REF:-}" != refs/heads/main ]]; then
  echo 'TestFlight signing is restricted to a manual main workflow.' >&2
  exit 1
fi

trap 'bash scripts/testflight-cleanup.sh' EXIT
trap 'exit 130' INT
trap 'exit 143' TERM HUP

uv run --locked --script scripts/testflight-assets.py prepare
state_dir="${RUNNER_TEMP}/chouwazi-testflight-${GITHUB_RUN_ID}-${GITHUB_RUN_ATTEMPT}"
export TESTFLIGHT_STATE_DIR="$state_dir"
keychain_path="$state_dir/signing.keychain-db"
keychain_password="$(< "$state_dir/keychain-password")"

security create-keychain -p "$keychain_password" "$keychain_path" >/dev/null 2>&1
security set-keychain-settings -lut 21600 "$keychain_path" >/dev/null 2>&1
security unlock-keychain -p "$keychain_password" "$keychain_path" >/dev/null 2>&1
sw_vers
openssl version
if openssl pkcs12 -in "$state_dir/distribution.p12" \
  -passin env:IOS_DISTRIBUTION_P12_PASSWORD -noout >/dev/null 2>&1; then
  echo 'OpenSSL PKCS#12 validation succeeded.'
else
  echo 'OpenSSL PKCS#12 validation failed.' >&2
fi
# The protected state directory is removed by cleanup. Never print raw import
# errors: they may contain signing metadata or paths to private material.
if security import "$state_dir/distribution.p12" -k "$keychain_path" -P "$IOS_DISTRIBUTION_P12_PASSWORD" -T /usr/bin/codesign -T /usr/bin/security >/dev/null 2>"$state_dir/import.log"; then
  :
else
  import_status=$?
  if LC_ALL=C grep -Eiq 'MAC verification failed|mac verify failure' "$state_dir/import.log"; then
    import_error='MAC verification failed'
  elif LC_ALL=C grep -Eiq 'unknown format|unsupported (format|algorithm)|format (is )?not supported' "$state_dir/import.log"; then
    import_error='unsupported or unknown format'
  elif LC_ALL=C grep -Eiq 'user interaction is not allowed|interaction not allowed|permission denied|authorization denied|not unlocked|keychain is locked|errSecInteractionNotAllowed|errSecAuthFailed' "$state_dir/import.log"; then
    import_error='permission, interaction, or keychain unlock failure'
  else
    import_error='unclassified import error'
  fi
  printf 'Distribution certificate import failed (exit code %s): %s.\n' "$import_status" "$import_error" >&2
  exit 1
fi
security set-key-partition-list -S apple-tool:,apple:,codesign: -k "$keychain_password" "$keychain_path" >/dev/null 2>&1
unset keychain_password IOS_DISTRIBUTION_P12_PASSWORD IOS_DISTRIBUTION_P12_BASE64 IOS_APP_STORE_PROFILE_BASE64 ASC_API_KEY_P8 ASC_API_KEY_ID ASC_API_ISSUER_ID
uv run --locked --script scripts/testflight-assets.py install

# xcodebuild command-line signing settings would also reach Pods. Configure only
# Runner's Release target and restore the original project during cleanup.
ruby <<'RUBY'
require 'json'
require 'xcodeproj'
settings = JSON.parse(File.read(File.join(ENV.fetch('TESTFLIGHT_STATE_DIR'), 'settings.json')))
project = Xcodeproj::Project.open('ios/Runner.xcodeproj')
target = project.targets.find { |item| item.name == 'Runner' }
raise 'Runner target is missing' unless target
config = target.build_configurations.find { |item| item.name == 'Release' }
raise 'Runner Release configuration is missing' unless config
config.build_settings['CODE_SIGN_STYLE'] = 'Manual'
config.build_settings['DEVELOPMENT_TEAM'] = settings.fetch('team_id')
config.build_settings['CODE_SIGN_IDENTITY[sdk=iphoneos*]'] = settings.fetch('certificate_sha1')
config.build_settings['PROVISIONING_PROFILE_SPECIFIER'] = settings.fetch('profile_uuid')
config.build_settings['CODE_SIGNING_ALLOWED'] = 'YES'
config.build_settings['CODE_SIGNING_REQUIRED'] = 'YES'
project.save
RUBY

xcodebuild -workspace ios/Runner.xcworkspace -scheme Runner -configuration Release \
  -destination 'generic/platform=iOS' -archivePath "$state_dir/Runner.xcarchive" \
  -derivedDataPath "$state_dir/DerivedData" archive
xcodebuild -exportArchive -archivePath "$state_dir/Runner.xcarchive" \
  -exportPath "$state_dir/export" -exportOptionsPlist "$state_dir/ExportOptions.plist"
uv run --locked --script scripts/testflight-assets.py verify

api_key_id="$(uv run --locked --quiet --script scripts/testflight-assets.py field api_key_id)"
api_issuer_id="$(uv run --locked --quiet --script scripts/testflight-assets.py field api_issuer_id)"
# altool resolves AuthKey_<key ID>.p8 from this directory. Neither the key nor
# the raw delivery logs are printed or published as an Actions artifact.
export API_PRIVATE_KEYS_DIR="$state_dir/private_keys"
if ! xcrun altool --validate-app -f "$state_dir/export/Runner.ipa" -t ios \
  --apiKey "$api_key_id" --apiIssuer "$api_issuer_id" --output-format json \
  >"$state_dir/validation.json" 2>"$state_dir/validation.log"; then
  echo 'Apple binary validation failed. Check App Store Connect delivery status.' >&2
  exit 1
fi
if ! xcrun altool --upload-app -f "$state_dir/export/Runner.ipa" -t ios \
  --apiKey "$api_key_id" --apiIssuer "$api_issuer_id" --output-format json \
  >"$state_dir/upload.json" 2>"$state_dir/upload.log"; then
  echo 'Apple upload failed. Check App Store Connect before retrying this build.' >&2
  exit 1
fi
echo 'Upload accepted by Apple. Processing and TestFlight availability are still pending.'
if [[ -n "${GITHUB_STEP_SUMMARY:-}" ]]; then
  printf '%s\n' 'The signed IPA passed local checks and Apple validation; Apple accepted the upload. Wait for processing in App Store Connect, then assign the build to testers. This run does not establish TestFlight availability or App Store approval.' >> "$GITHUB_STEP_SUMMARY"
fi
