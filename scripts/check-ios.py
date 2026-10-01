#!/usr/bin/env python3
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
        assert (app / language / "InfoPlist.strings").is_file(), f"Missing {language} app name"
    print(f"Verified {app}: {info['CFBundleSupportedPlatforms']}, "
          f"{info['CFBundleShortVersionString']} ({info['CFBundleVersion']})")
