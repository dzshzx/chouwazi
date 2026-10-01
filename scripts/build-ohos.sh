#!/usr/bin/env bash
set -euo pipefail
source "$(dirname "$0")/ohos-env.sh"
cd "$CHOUWAZI_ROOT"
python3 scripts/setup-ohos.py
flutter --suppress-analytics precache --ohos
flutter --suppress-analytics pub get --enforce-lockfile

# The upstream Hvigor plugin catches Flutter subprocess errors. Compile first
# with strict exit-code propagation, including when old build outputs exist.
flutter --suppress-analytics assemble --no-version-check \
  --depfile build/ohos/intermediates/flutter/defaultRelease/flutter_build.d \
  --output build/ohos/intermediates/flutter/defaultRelease \
  -dTargetFile=lib/main.dart -dTargetPlatform=ohos -dBuildMode=release \
  -dOhosArchs=ohos-arm64 -dTreeShakeIcons=true ohos_aot_bundle_release_ohos-arm64

# The pinned Flutter CLI checks signing after compiling. Invoke its upstream
# Hvigor plugin directly to explicitly build an unsigned HAP, without swallowing
# a Flutter error or creating a fake signing configuration.
python3 - <<'PY'
import json, os, re
from pathlib import Path
root = Path.cwd()
version = re.search(r'^version:\s*(\d+\.\d+\.\d+)\+(\d+)\s*$', (root / 'pubspec.yaml').read_text(), re.M)
if not version:
    raise SystemExit('pubspec.yaml must contain version: x.y.z+build')
name, number = version.groups()
config = root / 'ohos/AppScope/app.json5'
app = json.loads(config.read_text())
if app['app']['versionName'] != name or app['app']['versionCode'] != int(number):
    raise SystemExit('Harmony app version must match pubspec.yaml; update ohos/AppScope/app.json5.')
plugin = str(Path(os.environ['FLUTTER_OH_ROOT']) / 'packages/flutter_tools/hvigor')
(root / 'ohos/package.json').write_text(json.dumps({'private': True, 'dependencies': {'flutter-hvigor-plugin': 'file:' + plugin}}))
properties = {
    'hwsdk.dir': os.environ['DEVECO_SDK_HOME'],
    'nodejs.dir': os.environ['NODE_HOME'],
    'flutter.sdk': os.environ['FLUTTER_OH_ROOT'],
    'flutter.versionName': name,
    'flutter.versionCode': number,
}
(root / 'ohos/local.properties').write_text(''.join(f'{k}={v}\n' for k,v in properties.items()))
PY
(
  cd ohos
  npm install --ignore-scripts --no-audit --no-fund
  hvigorw assembleHap -p product=default -p buildMode=release --no-daemon \
    -p FLUTTER_TARGET=lib/main.dart -p TARGET_PLATFORM=ohos-arm64 \
    -p TREE_SHAKE_ICONS=true 2>&1 | tee ../build/ohos/hvigor.log
)
if grep -q 'flutter assemble: Command failed:' build/ohos/hvigor.log; then
  echo 'The Hvigor plugin swallowed a Flutter compiler failure.' >&2
  exit 1
fi
python3 scripts/check-hap.py \
  ohos/entry/build/default/outputs/default/entry-default-unsigned.hap \
  build/ohos/intermediates/flutter/defaultRelease/arm64-v8a/app.so
python3 - <<'PY'
import shutil
from pathlib import Path
source = Path('ohos/entry/build/default/outputs/default/entry-default-unsigned.hap')
destination = Path('build/ohos/chouwazi-unsigned.hap')
destination.parent.mkdir(parents=True, exist_ok=True)
shutil.copy2(source, destination)
print(f'Built unsigned HarmonyOS arm64 app: {destination}')
PY
