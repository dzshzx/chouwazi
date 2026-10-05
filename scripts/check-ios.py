#!/usr/bin/env -S uv run --locked --quiet --script
# /// script
# requires-python = ">=3.11"
# dependencies = []
#
# [tool.uv]
# exclude-newer = "1 day"
# ///
"""Check the actual app bundle has the shared app, privacy and localized names."""
from pathlib import Path
import plistlib
import sys

for path in sys.argv[1:]:
    app = Path(path)
    info = plistlib.loads((app / "Info.plist").read_bytes())
    assert info["CFBundleIdentifier"] == "com.alex.chouwazi"
    assert info["MinimumOSVersion"] == "15.0"
    for name in ["PrivacyInfo.xcprivacy", "Assets.car", "Frameworks/Flutter.framework/Flutter",
                 "Frameworks/App.framework/flutter_assets/AssetManifest.bin"]:
        assert (app / name).is_file(), f"Missing {name} in {app}"
    for language in ["en", "zh-Hans", "zh-Hant"]:
        assert (app / f"{language}.lproj" / "InfoPlist.strings").is_file(), f"Missing {language} app name"
    if info['CFBundleSupportedPlatforms'] == ['iPhoneOS']:
        assert 'NSLocalNetworkUsageDescription' not in info, 'Device release includes debug network service'
        assert 'NSBonjourServices' not in info, 'Device release advertises a network service'
    print(f"Verified {app}: {info['CFBundleSupportedPlatforms']}, "
          f"{info['CFBundleShortVersionString']} ({info['CFBundleVersion']})")
