#!/usr/bin/env python3
"""Install the pinned project-local Harmony toolchain; never alter global SDKs."""
import hashlib
import os
from pathlib import Path
import platform
import subprocess
import sys
import urllib.request

FLUTTER_REPO = "https://gitcode.com/CPF-Flutter/flutter_flutter.git"
FLUTTER_TAG = "3.27.5-ohos-1.0.0"
FLUTTER_COMMIT = "c8a0f037e39856cdac07bfa31d59d3ea78549b1b"
SDK_ARCHIVE = "commandline-tools-linux-x64-5.1.0.840.zip"
SDK_URL = "https://repo.huaweicloud.com/harmonyos/ohpm/5.1.0/" + SDK_ARCHIVE
SDK_SHA256 = "1dcefa4741dc289277560b2802040e0c45b433898d8db453d711d5376e0478f1"


def main():
    if platform.system() != "Linux" or platform.machine() != "x86_64":
        sys.exit("Automatic Harmony SDK setup requires Linux x86_64; see docs/BUILDING.md for DevEco setup.")
    root = Path(os.environ.get("CHOUWAZI_TOOLCHAINS", Path(__file__).resolve().parents[1] / ".tooling"))
    root.mkdir(parents=True, exist_ok=True)
    flutter = root / "flutter-oh"
    if not flutter.exists():
        subprocess.run(["git", "clone", "--depth", "1", "--branch", FLUTTER_TAG,
                        FLUTTER_REPO, str(flutter)], check=True)
    actual = subprocess.check_output(["git", "-C", str(flutter), "rev-parse", "HEAD"], text=True).strip()
    if actual != FLUTTER_COMMIT:
        sys.exit(f"Unexpected Flutter OH revision {actual}; expected {FLUTTER_COMMIT}. Existing files retained.")
    if subprocess.check_output(["git", "-C", str(flutter), "status", "--porcelain", "--untracked-files=no"], text=True):
        sys.exit("Flutter OH has tracked edits; existing SDK retained. Use a clean pinned SDK.")
    sdk = root / "harmony" / "command-line-tools"
    receipt = root / "harmony" / ".sdk-sha256"
    if not receipt.exists() or receipt.read_text().strip() != SDK_SHA256:
        if sdk.exists():
            sys.exit("An unverified Harmony SDK already exists here; use a new CHOUWAZI_TOOLCHAINS directory.")
        archive = root / SDK_ARCHIVE
        if not archive.exists():
            print(f"Downloading {SDK_URL}", flush=True)
            partial = archive.with_suffix(".zip.part")
            urllib.request.urlretrieve(SDK_URL, partial)
            partial.rename(archive)
        with archive.open("rb") as stream:
            digest = hashlib.file_digest(stream, "sha256").hexdigest()
        if digest != SDK_SHA256:
            sys.exit(f"Harmony SDK checksum mismatch: {digest}. Archive retained for inspection.")
        (root / "harmony").mkdir(exist_ok=True)
        subprocess.run(["unzip", "-q", str(archive), "-d", str(root / "harmony")], check=True)
        receipt.write_text(SDK_SHA256 + "\n")
    for required in ["sdk/default/openharmony/toolchains/hdc", "hvigor/bin/hvigorw", "ohpm/bin/ohpm", "tool/node/bin/node"]:
        if not (sdk / required).is_file():
            sys.exit(f"Incomplete Harmony SDK: missing {required}")
    print(f"Pinned Harmony toolchain ready: {root}")


if __name__ == "__main__":
    main()
