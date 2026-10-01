#!/usr/bin/env bash
set -euo pipefail
cd "$(dirname "$0")"
: "${ANDROID_JAR:?Set ANDROID_JAR to SDK platforms/android-35/android.jar}"
: "${BUILD_TOOLS:?Set BUILD_TOOLS to SDK build-tools/35.0.0}"
mkdir -p build/classes build/dex
"$BUILD_TOOLS/aapt2" compile --dir app/src/main/res -o build/res.zip
"$BUILD_TOOLS/aapt2" link -o build/base.apk -I "$ANDROID_JAR" --manifest app/src/main/AndroidManifest.xml build/res.zip --java build/generated
java com.sun.tools.javac.Main -source 8 -target 8 -bootclasspath "$ANDROID_JAR" -d build/classes app/src/main/java/com/alex/fingerdraw/*.java build/generated/com/alex/fingerdraw/R.java
"$BUILD_TOOLS/d8" --lib "$ANDROID_JAR" --min-api 23 --output build/dex build/classes/com/alex/fingerdraw/*.class
python3 - <<'PY'
import zipfile
with zipfile.ZipFile('build/base.apk','a') as z:z.write('build/dex/classes.dex','classes.dex')
PY
"$BUILD_TOOLS/zipalign" -f 4 build/base.apk build/aligned.apk
if [ ! -f build/local-signing.p12 ]; then
 keytool -genkeypair -keystore build/local-signing.p12 -storepass android -keypass android -alias fingerdraw -keyalg RSA -keysize 2048 -validity 10000 -dname 'CN=Finger Draw Local Build' -noprompt
fi
"$BUILD_TOOLS/apksigner" sign --ks build/local-signing.p12 --ks-pass pass:android --out build/chouwazi-1.0.1.apk build/aligned.apk
"$BUILD_TOOLS/apksigner" verify build/chouwazi-1.0.1.apk
