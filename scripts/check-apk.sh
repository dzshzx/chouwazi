#!/usr/bin/env bash
# Check the compiled APK, not just the source manifest: application ID, minimum Android
# version, and that the only permissions are vibration plus AndroidX's signature-level
# internal receiver permission.
# Usage: bash scripts/check-apk.sh <aapt> <apk>
set -euo pipefail
aapt=$1
apk=$2
internal=com.alex.fingerdraw.DYNAMIC_RECEIVER_NOT_EXPORTED_PERMISSION
fail() {
  echo "$1" >&2
  exit 1
}

badging=$("$aapt" dump badging "$apk")
grep -qF "package: name='com.alex.fingerdraw'" <<<"$badging" || fail 'Application ID changed'
grep -qF "sdkVersion:'23'" <<<"$badging" || fail 'Minimum Android version changed'
permissions=$(grep -oE "uses-permission(-sdk-[0-9]+)?: name='[^']+'" <<<"$badging" | sed -E "s/.*name='([^']+)'$/\1/" | sort -u || true)
unexpected=$(grep -vxF -e android.permission.VIBRATE -e "$internal" <<<"$permissions" || true)
[[ -z "$unexpected" ]] || fail "Unexpected permissions: $(tr '\n' ' ' <<<"$unexpected")"
if grep -qxF "$internal" <<<"$permissions"; then
  # A top-level <permission> element runs until the next element at the same depth.
  "$aapt" dump xmltree "$apk" AndroidManifest.xml | awk -v want="$internal" '
    function close_block() { if (in_permission && name == want && level == "2") ok = 1 }
    /^    E:/ { close_block(); in_permission = 0 }
    index($0, "    E: permission ") { in_permission = 1; name = ""; level = "" }
    in_permission && name == "" && match($0, /android:name\([^)]*\)="[^"]+"/) {
      name = substr($0, RSTART, RLENGTH); sub(/^[^"]*"/, "", name); sub(/"$/, "", name)
    }
    in_permission && level == "" && match($0, /android:protectionLevel\([^)]*\)=\(type 0x[0-9a-f]+\)0x[0-9a-f]+/) {
      level = substr($0, RSTART, RLENGTH); sub(/.*\)0x/, "", level); sub(/^0+/, "", level)
    }
    END { close_block(); exit !ok }
  ' || fail 'AndroidX receiver permission must remain signature-only'
fi
echo 'Verified Android identity, minimum version and offline permissions.'
