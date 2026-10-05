#!/usr/bin/env -S uv run --locked --quiet --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
#
# [tool.uv]
# exclude-newer = "1 day"
# ///
"""Check the compiled APK, not just the source manifest."""
import re
import subprocess
import sys

output = subprocess.check_output([sys.argv[1], "dump", "badging", sys.argv[2]], text=True)
assert "package: name='com.alex.fingerdraw'" in output, "Application ID changed"
assert "sdkVersion:'23'" in output, "Minimum Android version changed"
permissions = set(re.findall(r"uses-permission(?:-sdk-\d+)?: name='([^']+)'", output))
internal = "com.alex.fingerdraw.DYNAMIC_RECEIVER_NOT_EXPORTED_PERMISSION"
assert permissions <= {"android.permission.VIBRATE", internal}, f"Unexpected permissions: {permissions}"
if internal in permissions:
    tree = subprocess.check_output([sys.argv[1], "dump", "xmltree", sys.argv[2], "AndroidManifest.xml"], text=True)
    declarations = re.findall(r"    E: permission .*?(?=\n    E:|\Z)", tree, re.S)
    def is_internal_signature(block):
        name = re.search(r'android:name\([^)]*\)="([^"]+)"', block)
        level = re.search(r'android:protectionLevel\([^)]*\)=\(type 0x[0-9a-f]+\)(0x[0-9a-f]+)\b', block)
        return name is not None and level is not None and name[1] == internal and int(level[1], 16) == 2
    assert any(is_internal_signature(block) for block in declarations), "AndroidX receiver permission must remain signature-only"
print("Verified Android identity, minimum version and offline permissions.")
