#!/usr/bin/env -S uv run --locked --quiet --script
# /// script
# requires-python = ">=3.11"
# dependencies = ["pillow==12.3.0", "websocket-client==1.9.0"]
# ///
"""Capture the unmodified production UI from a debug iOS simulator build.

Only the debug VM service receives synthetic touch events. The ordinary game
clock, random draw, localization and privacy navigation produce every screen.
No service URL, auth token or raw launch/attach log is written or uploaded.
"""
import hashlib
import json
from pathlib import Path
import plistlib
import queue
import re
import struct
import subprocess
import sys
import threading
import time
from urllib.parse import urlparse

from PIL import Image, PngImagePlugin
import websocket


BUNDLE_ID = "com.alex.chouwazi"
APP = Path("build/ios/iphonesimulator/Runner.app")
OUTPUT = Path("build/app-store-screenshots")
DEVICES = (
    ("iphone-16-pro", "iPhone 16 Pro", (1206, 2622)),
    ("ipad-pro-13-m4", "iPad Pro 13-inch (M4)", (2064, 2752)),
)
LANGUAGES = (
    ("en", "en_US", "Privacy", "Privacy Policy"),
    ("zh-Hans", "zh_CN", "隐私", "隐私政策"),
    ("zh-Hant", "zh_TW", "隱私", "隱私政策"),
)
SNAPSHOT = """[
  _showPrivacy,
  _session.round.players.length,
  _session.round.winner?.slot,
  _session.allReleased,
  _session.round.deadline == null ? null :
    (_session.round.deadline! - _now).inMilliseconds
].toString()"""
GEOMETRY = """(() {
  final view = WidgetsBinding.instance.platformDispatcher.views.single;
  final ratio = view.devicePixelRatio;
  return [view.physicalSize.width / ratio, view.physicalSize.height / ratio,
    view.padding.top / ratio, view.padding.bottom / ratio].toString();
})()"""
NO_DEBUG_BANNER = "!context.findAncestorWidgetOfExactType<WidgetsApp>()!.debugShowCheckedModeBanner"
VM_OPERATIONS = frozenset({
    "evaluate", "string", "snapshot", "state-mounted", "locale", "privacy-label",
    "debug-banner", "home-before", "home-after", "geometry", "players-down",
    "winner-poll", "players-up", "result-before", "result-after", "privacy-down",
    "privacy-up", "privacy-rendered", "privacy-title", "privacy-before", "privacy-after",
})
VM_RESULT_TYPES = frozenset({"@Instance", "@Error", "Error", "Sentinel"})
VM_METHODS = frozenset({"getVM", "getIsolate", "getObject", "getInstances", "evaluate"})
VM_FAILURE_KINDS = frozenset({
    "UnhandledException", "LanguageError", "InternalError", "TerminationError",
    "Collected", "Expired", "NotInitialized", "BeingInitialized", "OptimizedOut", "Free",
})
VM_ERROR_CODES = frozenset({-32700, -32600, -32601, -32602, -32603, -32010,
                            *range(100, 115), 130, 131})


class ScreenshotError(ValueError):
    """Messages contain only public, locally controlled metadata."""


def require(condition, message):
    if not condition:
        raise ScreenshotError(message)


def run(*args, timeout=120):
    result = subprocess.run(args, capture_output=True, timeout=timeout, check=False)
    require(result.returncode == 0, f"{Path(args[0]).name} operation failed (exit={result.returncode})")
    return result.stdout.decode("utf-8")


def cleanup(steps):
    """Attempt every bounded cleanup, reporting only fixed local step labels."""
    primary_failure = sys.exception() is not None
    failures = []
    for label, action in steps:
        try:
            action()
        except Exception:
            failures.append(label)
    if failures:
        message = "Cleanup failed: " + ", ".join(failures) + "; details kept private"
        if primary_failure:
            print(message, file=sys.stderr)
        else:
            raise ScreenshotError(message) from None


class PrivateProcess:
    """Consume potentially authenticated debug output without public logging."""

    def __init__(self, *args):
        self.process = subprocess.Popen(
            args, stdout=subprocess.PIPE, stderr=subprocess.STDOUT,
            stdin=subprocess.PIPE, text=True, bufsize=1,
        )
        self.lines = queue.Queue()
        self.reader = threading.Thread(target=self._read, daemon=True)
        self.reader.start()

    def _read(self):
        for line in self.process.stdout:
            self.lines.put(line)
        self.lines.put(None)

    def wait_for(self, parser, description, timeout=180):
        deadline = time.monotonic() + timeout
        lines = 0
        vm_marker = error_marker = False

        def failure(reason):
            return ScreenshotError(
                f"{description}: {reason} (lines={lines}, vm_marker={vm_marker}, "
                f"error_marker={error_marker}, process_running={self.process.poll() is None}); "
                "debug output was kept private")

        while time.monotonic() < deadline:
            try:
                line = self.lines.get(timeout=min(1, max(0.01, deadline - time.monotonic())))
            except queue.Empty:
                continue
            if line is None:
                raise failure("process exited before it was ready")
            lines += 1
            vm_marker |= "The Dart VM service is listening on " in line
            error_marker |= any(marker in line for marker in ("[ERROR:", "Unhandled Exception", "FATAL"))
            parsed = parser(line)
            if parsed is not None:
                return parsed
        raise failure("timed out")

    def close(self):
        if self.process.poll() is None:
            self.process.terminate()
            try:
                self.process.wait(timeout=10)
            except subprocess.TimeoutExpired:
                self.process.kill()
                self.process.wait(timeout=10)
        self.reader.join(timeout=2)
        self.process.stdout.close()
        self.process.stdin.close()


class VM:
    def __init__(self, uri):
        parsed = urlparse(uri)
        require(parsed.scheme == "ws" and parsed.hostname == "127.0.0.1", "VM service must use a loopback WebSocket")
        self.connection = websocket.create_connection(
            uri, timeout=60, http_no_proxy=["127.0.0.1", "localhost"],
        )
        self.sequence = 0
        isolates = self.call("getVM")["isolates"]
        require(len(isolates) == 1, "Expected exactly one application isolate")
        self.isolate = isolates[0]["id"]
        libraries = self.call("getIsolate", isolateId=self.isolate)["libraries"]
        game = [library for library in libraries if library["uri"] == "package:chouwazi/game_screen.dart"
                or library["uri"].endswith("/lib/game_screen.dart")]
        require(len(game) == 1, "The real game_screen library was not loaded")
        self.library = game[0]["id"]
        classes = self.call("getObject", isolateId=self.isolate, objectId=self.library)["classes"]
        states = [klass for klass in classes if klass["name"] == "_GameScreenState"]
        require(len(states) == 1, "The real GameScreen state class was not loaded")
        instance_set = self.call("getInstances", isolateId=self.isolate, objectId=states[0]["id"], limit=10)
        # Dart 3.6 uses 'samples'; newer protocol implementations use 'instances'.
        instances = instance_set.get("instances", instance_set.get("samples", []))
        require(instance_set["totalCount"] <= 10, "Unexpected number of GameScreen states")
        mounted = [instance for instance in instances
                   if self.evaluate("mounted", target=instance["id"], operation="state-mounted").get("valueAsString") == "true"]
        require(len(mounted) == 1, "Expected exactly one mounted real GameScreen state")
        self.state = mounted[0]["id"]

    def call(self, method, *, operation=None, **params):
        require(isinstance(method, str) and method in VM_METHODS, "Unsupported VM service method")
        require(operation is None or isinstance(operation, str) and operation in VM_OPERATIONS,
                "Unsupported VM operation label")
        self.sequence += 1
        request_id = self.sequence
        self.connection.send(json.dumps({"jsonrpc": "2.0", "id": request_id, "method": method, "params": params}))
        while True:
            response = json.loads(self.connection.recv())
            if response.get("id") != request_id:
                continue
            if "error" in response:
                code = response["error"].get("code")
                code = code if type(code) is int and code in VM_ERROR_CODES else "unknown"
                stage = f", operation={operation}" if operation else ""
                require(False, f"VM {method} failed (code={code}{stage}); debug details kept private")
            return response["result"]

    def evaluate(self, expression, target=None, *, operation="evaluate"):
        # Flutter 3.27's compile-expression stdin protocol accepts one line.
        expression = " ".join(expression.splitlines()).strip()
        result = self.call("evaluate", isolateId=self.isolate,
                           targetId=target or self.state, expression=expression,
                           disableBreakpoints=True, operation=operation)
        result_type = result.get("type")
        kind = result.get("kind")
        result_type = result_type if isinstance(result_type, str) and result_type in VM_RESULT_TYPES else "unknown"
        kind = kind if isinstance(kind, str) and kind in VM_FAILURE_KINDS else "unknown"
        require(result.get("type") == "@Instance",
                f"Debug expression failed to evaluate (operation={operation}, type={result_type}, kind={kind}); "
                "debug details kept private")
        return result

    def string(self, expression, *, operation="string"):
        result = self.evaluate(expression, operation=operation)
        require(not result.get("valueAsStringIsTruncated"), f"Debug expression result was truncated (operation={operation})")
        return result["valueAsString"]

    def snapshot(self, *, operation="snapshot"):
        values = json.loads(self.string(SNAPSHOT, operation=operation))
        return dict(zip(("privacy", "players", "winner_slot", "all_released", "remaining_ms"), values))

    def touches(self, action, positions, *, operation="evaluate"):
        require(action in {"Down", "Up"}, "Unsupported pointer event")
        events = "\n".join(
            "GestureBinding.instance.handlePointerEvent("
            f"Pointer{action}Event(pointer: {pointer}, kind: PointerDeviceKind.touch, "
            f"position: Offset({float(x)}, {float(y)})));"
            for pointer, x, y in positions
        )
        # VM evaluation can interrupt build/layout/paint; dispatch on the next
        # ordinary event-loop turn so pointer handlers can safely schedule frames.
        self.evaluate("(() { Future<void>(() { " + events + " }); return true; })()", operation=operation)

    def close(self):
        self.connection.close(timeout=3)


def opaque_png(path, expected_size):
    """Remove only a redundant, fully opaque alpha channel, without compositing."""
    with Image.open(path) as source:
        require(source.format == "PNG" and source.size == expected_size, "Screenshot has an unexpected format or pixel size")
        require(source.mode in {"RGB", "RGBA"}, "Screenshot has an unexpected pixel format")
        if source.mode == "RGBA":
            require(source.getchannel("A").getextrema() == (255, 255), "Native screenshot has transparent pixels; refusing to change its appearance")
        profile = source.info.get("icc_profile")
        colors = PngImagePlugin.PngInfo()
        if "srgb" in source.info:
            colors.add(b"sRGB", bytes([source.info["srgb"]]))
        if "gamma" in source.info:
            colors.add(b"gAMA", struct.pack(">I", round(source.info["gamma"] * 100000)))
        if "chromaticity" in source.info:
            colors.add(b"cHRM", struct.pack(">8I", *(round(value * 100000) for value in source.info["chromaticity"])))
        image = source.convert("RGB")
    # PNG encoding is lossless: RGB values, resolution and the entire screen stay intact.
    image.save(path, format="PNG", icc_profile=profile, pnginfo=colors)
    with Image.open(path) as saved:
        require(saved.mode == "RGB" and saved.size == expected_size, "Saved PNG must be RGB without alpha")
        require(saved.info.get("icc_profile") == profile, "Saved PNG must retain its native color profile")
    return hashlib.sha256(path.read_bytes()).hexdigest()


def capture(vm, udid, folder, name, size, expected, screenshots):
    operation = {"01-home": "home", "02-result": "result", "03-privacy": "privacy"}[name]
    before = vm.snapshot(operation=f"{operation}-before")
    require(expected(before), f"Unexpected game state before {name}")
    path = folder / f"{name}.png"
    run("xcrun", "simctl", "io", udid, "screenshot", "--type=png", "--mask=ignored", str(path))
    after = vm.snapshot(operation=f"{operation}-after")
    require(expected(after), f"Game state changed during {name}; refusing to label this screenshot")
    digest = opaque_png(path, size)
    screenshots.append({"file": str(path.relative_to(OUTPUT)), "width": size[0], "height": size[1],
                        "mode": "RGB", "sha256": digest, "state_before": before, "state_after": after})
    print(f"Captured {path.relative_to(OUTPUT)} ({size[0]} × {size[1]}, RGB)", flush=True)


def vm_url(line):
    match = re.search(r"The Dart VM service is listening on (http://127\.0\.0\.1:\d+/[A-Za-z0-9=_-]+/)", line)
    return match[1] if match else None


def capture_language(udid, label, size, language, region, privacy_text, privacy_title):
    folder = OUTPUT / label / language
    folder.mkdir(parents=True)
    logs = attach = vm = None
    try:
        # Flutter 3.27.4 discovers iOS >= 11 VM announcements through unified
        # logging. Listen before launching so the startup message is not missed.
        logs = PrivateProcess("xcrun", "simctl", "spawn", udid, "log", "stream",
                              "--style", "compact", "--predicate",
                              'eventType = logEvent AND processImagePath ENDSWITH "/Runner"')
        logs.wait_for(lambda line: True if line.startswith("Filtering the log data using ") else None,
                      "Simulator unified log startup", timeout=30)
        run("xcrun", "simctl", "launch", udid, BUNDLE_ID,
            "--enable-checked-mode", "--verify-entry-points",
            "--disable-vm-service-publication", "-AppleLanguages", f"({language})",
            "-AppleLocale", region)
        print("Simulator launch acknowledged; waiting for private unified log VM announcement", flush=True)
        uri = logs.wait_for(vm_url, "Simulator debug service startup")
        print("Simulator loopback VM service discovered; attaching Flutter", flush=True)
        attach = PrivateProcess("flutter", "attach", "--machine", "--debug",
                                "--no-dds", "--no-devtools", "-d", udid, "--debug-url", uri)
        connection = {}

        def ready(line):
            try:
                messages = json.loads(line)
            except json.JSONDecodeError:
                return None
            if not isinstance(messages, list):
                return None
            for message in messages:
                if message.get("event") == "app.debugPort":
                    connection["uri"] = message["params"]["wsUri"]
                if message.get("event") == "app.started":
                    connection["started"] = True
            return connection["uri"] if connection.get("started") and connection.get("uri") else None

        debug_uri = attach.wait_for(ready, "Flutter debug attach")
        time.sleep(1)
        vm = VM(debug_uri)
        actual_locale = vm.string("WidgetsBinding.instance.platformDispatcher.locale.toLanguageTag()", operation="locale")
        require(vm.string("_strings.text('privacy')", operation="privacy-label") == privacy_text, f"System locale did not select {language}")
        require(vm.string(NO_DEBUG_BANNER, operation="debug-banner") == "true", "The app is displaying its debug banner")
        screenshots = []
        capture(vm, udid, folder, "01-home", size,
                lambda state: not state["privacy"] and state["players"] == 0, screenshots)
        width, height, top, bottom = json.loads(vm.string(GEOMETRY, operation="geometry"))
        content_height = height - top - bottom
        positions = [(index + 1, width * x, top + content_height * y)
                     for index, (x, y) in enumerate(((.24, .28), (.76, .28), (.24, .73), (.76, .73)))]
        vm.touches("Down", positions, operation="players-down")
        # Wait for the ordinary 3.8-second round and its random draw to finish.
        deadline = time.monotonic() + 10
        while vm.snapshot(operation="winner-poll")["winner_slot"] is None:
            require(time.monotonic() < deadline, "The normal game clock did not produce a result")
            time.sleep(.2)
        vm.touches("Up", positions, operation="players-up")
        deadline = time.monotonic() + 10
        while not vm.snapshot(operation="players-up")["all_released"]:
            require(time.monotonic() < deadline, "The normal pointer releases did not finish")
            time.sleep(.2)
        capture(vm, udid, folder, "02-result", size,
                lambda state: not state["privacy"] and state["players"] == 4 and
                state["winner_slot"] in range(4) and state["all_released"], screenshots)
        # Complete an ordinary touch on the existing privacy button, including its release.
        privacy_position = [(20, 30, top + 22)]
        vm.touches("Down", privacy_position, operation="privacy-down")
        vm.touches("Up", privacy_position, operation="privacy-up")
        title_expression = """
          (context as Element).toStringDeep().contains('PrivacyScreen') &&
          (context as Element).toStringDeep().contains(_strings.text('privacy_title')) &&
          (context as Element).toStringDeep().contains(_strings.text('privacy_summary'))
        """
        deadline = time.monotonic() + 15
        while not (vm.snapshot(operation="privacy-rendered")["privacy"] and
                   vm.string(title_expression, operation="privacy-rendered") == "true" and
                   vm.string("_strings.text('privacy_title')", operation="privacy-title") == privacy_title):
            require(time.monotonic() < deadline, "The real localized privacy screen did not render")
            time.sleep(.2)
        capture(vm, udid, folder, "03-privacy", size,
                lambda state: state["privacy"] and state["all_released"], screenshots)
        return {"language": language, "actual_locale": actual_locale, "screenshots": screenshots}
    finally:
        steps = [(label, resource.close) for label, resource in (
            ("VM connection", vm), ("Flutter attach", attach), ("Simulator unified log", logs),
        ) if resource is not None]
        # Flutter may already have terminated the app.
        steps.append(("Simulator app termination", lambda: subprocess.run(
            ("xcrun", "simctl", "terminate", udid, BUNDLE_ID),
            capture_output=True, timeout=30, check=False)))
        cleanup(steps)


def main():
    require(sys.platform == "darwin", "iOS capture requires macOS with Xcode and a simulator runtime")
    require(not OUTPUT.exists(), "Screenshot output already exists; use a fresh build directory")
    info = plistlib.loads((APP / "Info.plist").read_bytes())
    require(info.get("CFBundleIdentifier") == BUNDLE_ID and
            info.get("CFBundleSupportedPlatforms") == ["iPhoneSimulator"], "Requires the real Runner simulator bundle")
    manifest = {"source_sha": run("git", "rev-parse", "HEAD").strip(),
                "version": info["CFBundleShortVersionString"], "build": info["CFBundleVersion"],
                "bundle_id": BUNDLE_ID, "build_mode": "debug", "capture": "simctl native screen",
                "interaction": "Flutter touch events via loopback debug VM; normal game clock and random draw",
                "screens": ["01-home", "02-result", "03-privacy"],
                "xcode": run("xcodebuild", "-version").strip(),
                "flutter": json.loads(run("flutter", "--version", "--machine")), "devices": []}
    # The installed inventory decides the identifiers. A missing model is a failure, never a fallback.
    inventory = json.loads(run("xcrun", "simctl", "list", "-j"))
    types = inventory["devicetypes"]
    selected_types = []
    for label, name, size in DEVICES:
        matches = [device for device in types if device["name"] == name]
        require(len(matches) == 1, f"Required simulator device type is missing: {name}")
        selected_types.append((label, matches[0], size))
    identifiers = {device["identifier"] for _, device, _ in selected_types}
    runtimes = [runtime for runtime in inventory["runtimes"]
                if runtime.get("isAvailable") and runtime["identifier"].startswith("com.apple.CoreSimulator.SimRuntime.iOS-")
                and identifiers <= {device["identifier"] for device in runtime.get("supportedDeviceTypes", [])}]
    require(bool(runtimes), "No installed iOS runtime supports both required simulator models")
    runtime = max(runtimes, key=lambda item: tuple(int(part) for part in item["version"].split(".")))
    print(f"Selected installed runtime: {runtime['name']} ({runtime['version']}, {runtime['buildversion']})", flush=True)
    OUTPUT.mkdir(parents=True)
    for label, device, size in selected_types:
        udid = run("xcrun", "simctl", "create", f"chouwazi-screenshots-{label}",
                   device["identifier"], runtime["identifier"]).strip()
        require(re.fullmatch(r"[A-Fa-f0-9-]{36}", udid), "simctl did not return a simulator identifier")
        print(f"Starting {device['name']} for {size[0]} × {size[1]} screenshots", flush=True)
        try:
            run("xcrun", "simctl", "boot", udid)
            run("xcrun", "simctl", "bootstatus", udid, "-b", timeout=240)
            run("xcrun", "simctl", "status_bar", udid, "override", "--time", "9:41",
                "--dataNetwork", "wifi", "--wifiMode", "active", "--wifiBars", "3",
                "--batteryState", "charged", "--batteryLevel", "100")
            run("xcrun", "simctl", "install", udid, str(APP))
            locales = [capture_language(udid, label, size, *language) for language in LANGUAGES]
            manifest["devices"].append({"name": device["name"], "device_type": device["identifier"],
                                        "runtime": runtime["identifier"], "runtime_version": runtime["version"],
                                        "runtime_build": runtime["buildversion"], "locales": locales})
        finally:
            # A failed boot can leave an already-shutdown device to remove.
            cleanup((
                ("Simulator shutdown", lambda: subprocess.run(
                    ("xcrun", "simctl", "shutdown", udid),
                    capture_output=True, timeout=60, check=False)),
                ("Simulator deletion", lambda: run("xcrun", "simctl", "delete", udid)),
            ))
    require(len(list(OUTPUT.rglob("*.png"))) == 18, "Expected all three stable screens in three languages on both devices")
    (OUTPUT / "manifest.json").write_text(json.dumps(manifest, ensure_ascii=False, indent=2) + "\n")
    print("Verified 18 native RGB PNG screenshots and a public manifest.", flush=True)


if __name__ == "__main__":
    try:
        main()
    except ScreenshotError as error:
        print(f"Screenshot capture failed: {error}", file=sys.stderr)
        sys.exit(1)
    except Exception:
        # External exceptions may embed an authenticated URL. Keep those messages private.
        print("Screenshot capture failed; check simulator, Flutter attach and screenshot prerequisites. Debug output was kept private.", file=sys.stderr)
        sys.exit(1)
