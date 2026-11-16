package screenshots

import (
	"bytes"
	"compress/zlib"
	"context"
	"encoding/json"
	"errors"
	"image"
	"image/color"
	"image/png"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/coder/websocket"
)

// The fixture pins the exact expression text sent to the VM service for a known geometry.
func TestTouchExpressionMatchesFixture(t *testing.T) {
	width, height, top, bottom := 402.0, 874.0, 62.0, 34.0
	content := height - top - bottom
	var positions []touch
	for index, point := range [][2]float64{{.24, .28}, {.76, .28}, {.24, .73}, {.76, .73}} {
		positions = append(positions, touch{index + 1, width * point[0], top + content*point[1]})
	}
	want := "(() { Future<void>(() { GestureBinding.instance.handlePointerEvent(PointerDownEvent(pointer: 1, kind: PointerDeviceKind.touch, position: Offset(96.47999999999999, 279.84000000000003)));\n" +
		"GestureBinding.instance.handlePointerEvent(PointerDownEvent(pointer: 2, kind: PointerDeviceKind.touch, position: Offset(305.52, 279.84000000000003)));\n" +
		"GestureBinding.instance.handlePointerEvent(PointerDownEvent(pointer: 3, kind: PointerDeviceKind.touch, position: Offset(96.47999999999999, 629.9399999999999)));\n" +
		"GestureBinding.instance.handlePointerEvent(PointerDownEvent(pointer: 4, kind: PointerDeviceKind.touch, position: Offset(305.52, 629.9399999999999))); }); return true; })()"
	if got := TouchExpression("Down", positions); got != want {
		t.Fatalf("got %s", got)
	}
	up := "(() { Future<void>(() { GestureBinding.instance.handlePointerEvent(PointerUpEvent(pointer: 20, kind: PointerDeviceKind.touch, position: Offset(30.0, 84.0))); }); return true; })()"
	if got := TouchExpression("Up", []touch{{20, 30, top + 22}}); got != up {
		t.Fatalf("got %s", got)
	}
}

func TestVMURLAndAttachParser(t *testing.T) {
	if uri, ok := VMURL("x The Dart VM service is listening on http://127.0.0.1:50123/AbC_d-e=/\n"); !ok || uri != "http://127.0.0.1:50123/AbC_d-e=/" {
		t.Fatalf("got %q %v", uri, ok)
	}
	if _, ok := VMURL("The Dart VM service is listening on http://10.0.0.1:5/x/\n"); ok {
		t.Fatal("non-loopback service accepted")
	}
	ready := AttachReadyParser()
	for _, line := range []string{"not json\n", `{"event":"app.started"}` + "\n", `[{"event":"app.debugPort","params":{"wsUri":"ws://127.0.0.1:5/t=/ws"}}]` + "\n"} {
		if _, ok := ready(line); ok {
			t.Fatalf("ready after %q", line)
		}
	}
	if uri, ok := ready(`[{"event":"app.started"}]` + "\n"); !ok || uri != "ws://127.0.0.1:5/t=/ws" {
		t.Fatalf("got %q %v", uri, ok)
	}
}

func TestExpectedStates(t *testing.T) {
	state := func(printed string) State {
		parsed, err := ParseState(printed)
		if err != nil {
			t.Fatal(err)
		}
		return parsed
	}
	if !homeState(state("[false, 0, null, false, null]")) || homeState(state("[false, 1, null, false, 3000]")) {
		t.Error("home state")
	}
	if !resultState(state("[false, 4, 3, true, null]")) || resultState(state("[false, 4, 4, true, null]")) ||
		resultState(state("[false, 4, null, true, null]")) || resultState(state("[false, 4, 0, false, null]")) {
		t.Error("result state")
	}
	if !privacyState(state("[true, 4, 2, true, null]")) || privacyState(state("[true, 4, 2, false, null]")) {
		t.Error("privacy state")
	}
	if _, err := ParseState("[false, 0]"); err == nil {
		t.Error("short snapshot accepted")
	}
}

func TestSelectSimulators(t *testing.T) {
	types := []simDevice{{"iPhone 16 Pro", "phone"}, {"iPad Pro 13-inch (M4)", "pad"}, {"iPhone SE", "se"}}
	both := []simDevice{{Identifier: "phone"}, {Identifier: "pad"}}
	listed := inventory{DeviceTypes: types, Runtimes: []simRuntime{
		{Identifier: "com.apple.CoreSimulator.SimRuntime.iOS-18-6", Version: "18.6", IsAvailable: true, SupportedTypes: both},
		{Identifier: "com.apple.CoreSimulator.SimRuntime.iOS-26-2", Version: "26.2", IsAvailable: true, SupportedTypes: both},
		{Identifier: "com.apple.CoreSimulator.SimRuntime.iOS-26-10", Version: "26.10", IsAvailable: false, SupportedTypes: both},
		{Identifier: "com.apple.CoreSimulator.SimRuntime.iOS-27-0", Version: "27.0", IsAvailable: true, SupportedTypes: both[:1]},
		{Identifier: "com.apple.CoreSimulator.SimRuntime.watchOS-30-0", Version: "30.0", IsAvailable: true, SupportedTypes: both},
	}}
	kinds, chosen, err := SelectSimulators(listed)
	if err != nil || chosen.Version != "26.2" || kinds[0].Identifier != "phone" || kinds[1].Identifier != "pad" {
		t.Fatalf("got %v %v %v", kinds, chosen, err)
	}
	listed.DeviceTypes = types[:1]
	if _, _, err := SelectSimulators(listed); err == nil || err.Error() != "Required simulator device type is missing: iPad Pro 13-inch (M4)" {
		t.Fatalf("missing model: %v", err)
	}
}

func TestCleanupRunsEveryStep(t *testing.T) {
	var ran []string
	step := func(label string, fail bool) cleanupStep {
		return cleanupStep{label, func() error {
			ran = append(ran, label)
			if fail {
				return errors.New("secret detail")
			}
			return nil
		}}
	}
	var stderr bytes.Buffer
	err := cleanup([]cleanupStep{step("A", true), step("B", false), step("C", true)}, nil, &stderr)
	if err == nil || err.Error() != "Cleanup failed: A, C; details kept private" || strings.Join(ran, "") != "ABC" {
		t.Fatalf("err %v, ran %v", err, ran)
	}
	primary := Error("primary")
	if err := cleanup([]cleanupStep{step("D", true)}, primary, &stderr); err != primary ||
		stderr.String() != "Cleanup failed: D; details kept private\n" {
		t.Fatalf("err %v, stderr %q", err, stderr.String())
	}
}

func TestPrivateProcessKeepsOutputPrivate(t *testing.T) {
	process, err := startPrivate("sh", "-c", "echo secret-token; echo 'The Dart VM service is listening on x'; echo READY; exec sleep 30")
	if err != nil {
		t.Fatal(err)
	}
	value, err := process.waitFor(func(line string) (string, bool) { return "ok", line == "READY\n" }, "test", 10*time.Second)
	if err != nil || value != "ok" {
		t.Fatalf("got %q %v", value, err)
	}
	if err := process.close(); err != nil {
		t.Fatal(err)
	}
	quiet, err := startPrivate("sh", "-c", "echo [ERROR: secret-token")
	if err != nil {
		t.Fatal(err)
	}
	_, err = quiet.waitFor(func(string) (string, bool) { return "", false }, "Startup", 10*time.Second)
	want := "Startup: process exited before it was ready (lines=1, vm_marker=False, error_marker=True, process_running="
	if err == nil || !strings.HasPrefix(err.Error(), want) || strings.Contains(err.Error(), "secret") {
		t.Fatalf("got %v", err)
	}
	_ = quiet.close()
}

func writePNG(t *testing.T, path string, img image.Image, extra ...chunk) {
	t.Helper()
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, img); err != nil {
		t.Fatal(err)
	}
	chunks, err := readChunks(encoded.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	var out bytes.Buffer
	out.Write(pngSignature)
	writeChunk(&out, chunks[0])
	for _, item := range extra {
		writeChunk(&out, item)
	}
	for _, item := range chunks[1:] {
		writeChunk(&out, item)
	}
	if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func iccChunk(t *testing.T, profile []byte) chunk {
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	if _, err := writer.Write(profile); err != nil {
		t.Fatal(err)
	}
	writer.Close()
	return chunk{"iCCP", append([]byte("Display P3\x00\x00"), compressed.Bytes()...)}
}

// screenshot is a small synthetic screen whose pixel (2, 1) has the given alpha.
func screenshot(alpha uint8) *image.NRGBA {
	img := image.NewNRGBA(image.Rect(0, 0, 6, 4))
	for y := range 4 {
		for x := range 6 {
			img.Set(x, y, color.NRGBA{uint8(40 * x), uint8(60 * y), 200, 255})
		}
	}
	img.Set(2, 1, color.NRGBA{9, 8, 7, alpha})
	return img
}

func TestOpaquePNG(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "shot.png")
	profile := []byte("synthetic icc profile")
	gamma := chunk{"gAMA", []byte{0, 0, 0xb1, 0x8f}}
	text := chunk{"tEXt", []byte("Comment\x00private")}
	// An opaque image would be encoded as RGB, so write a real RGBA file by hand.
	rgba := screenshot(255)
	writeRGBA(t, path, rgba, iccChunk(t, profile), gamma, text)
	if _, err := OpaquePNG(path, 6, 4); err != nil {
		t.Fatal(err)
	}
	data, _ := os.ReadFile(path)
	chunks, err := readChunks(data)
	if err != nil {
		t.Fatal(err)
	}
	if head := readHeader(chunks); head.color != 2 || head.depth != 8 {
		t.Fatalf("header %+v", head)
	}
	kinds := ""
	for _, item := range chunks {
		kinds += item.kind + " "
	}
	if !strings.HasPrefix(kinds, "IHDR iCCP gAMA IDAT") || strings.Contains(kinds, "tEXt") {
		t.Fatalf("chunks %s", kinds)
	}
	if saved, _ := iccProfile(chunks); !bytes.Equal(saved, profile) {
		t.Fatal("color profile changed")
	}
	decoded, err := png.Decode(bytes.NewReader(data))
	if err != nil {
		t.Fatal(err)
	}
	for y := range 4 {
		for x := range 6 {
			r, g, b, _ := decoded.At(x, y).RGBA()
			wr, wg, wb, _ := rgba.At(x, y).RGBA()
			if r != wr || g != wg || b != wb {
				t.Fatalf("pixel %d,%d changed", x, y)
			}
		}
	}
}

// writeRGBA writes img as an 8-bit RGBA PNG (color type 6) even when every pixel is opaque.
func writeRGBA(t *testing.T, path string, img *image.NRGBA, extra ...chunk) {
	t.Helper()
	// A pixel with alpha 254 makes the encoder choose RGBA.
	marked := screenshot(254)
	var encoded bytes.Buffer
	if err := png.Encode(&encoded, marked); err != nil {
		t.Fatal(err)
	}
	chunks, err := readChunks(encoded.Bytes())
	if err != nil {
		t.Fatal(err)
	}
	// Keep that RGBA header (color type 6) and write img's pixels as unfiltered scanlines.
	var scanlines bytes.Buffer
	for y := range 4 {
		scanlines.WriteByte(0)
		scanlines.Write(img.Pix[y*img.Stride : y*img.Stride+6*4])
	}
	var compressed bytes.Buffer
	writer := zlib.NewWriter(&compressed)
	writer.Write(scanlines.Bytes())
	writer.Close()
	var out bytes.Buffer
	out.Write(pngSignature)
	writeChunk(&out, chunks[0])
	for _, item := range extra {
		writeChunk(&out, item)
	}
	writeChunk(&out, chunk{"IDAT", compressed.Bytes()})
	writeChunk(&out, chunk{"IEND", nil})
	if err := os.WriteFile(path, out.Bytes(), 0o644); err != nil {
		t.Fatal(err)
	}
}

func TestOpaquePNGRefusals(t *testing.T) {
	dir := t.TempDir()
	cases := map[string]struct {
		write   func(path string)
		message string
	}{
		"transparent": {func(path string) { writeRGBA(t, path, screenshot(254)) }, "Native screenshot has transparent pixels; refusing to change its appearance"},
		"size":        {func(path string) { writeRGBA(t, path, screenshot(255)) }, "Screenshot has an unexpected format or pixel size"},
		"gray":        {func(path string) { writePNG(t, path, image.NewGray(image.Rect(0, 0, 6, 4))) }, "Screenshot has an unexpected pixel format"},
		"not png":     {func(path string) { os.WriteFile(path, []byte("GIF89a"), 0o644) }, "Screenshot has an unexpected format or pixel size"},
	}
	for name, test := range cases {
		path := filepath.Join(dir, name+".png")
		test.write(path)
		width := 6
		if name == "size" {
			width = 7
		}
		before, _ := os.ReadFile(path)
		if _, err := OpaquePNG(path, width, 4); err == nil || err.Error() != test.message {
			t.Errorf("%s: got %v", name, err)
		}
		if after, _ := os.ReadFile(path); !bytes.Equal(before, after) {
			t.Errorf("%s: refused screenshot was rewritten", name)
		}
	}
}

// fakeVM serves the subset of the Dart VM service protocol the client uses.
func fakeVM(t *testing.T, mounted int, evaluateError bool) string {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		connection, err := websocket.Accept(w, r, nil)
		if err != nil {
			return
		}
		defer connection.CloseNow()
		ctx := context.Background()
		for {
			_, data, err := connection.Read(ctx)
			if err != nil {
				return
			}
			var request struct {
				ID     int            `json:"id"`
				Method string         `json:"method"`
				Params map[string]any `json:"params"`
			}
			json.Unmarshal(data, &request)
			var result any
			switch request.Method {
			case "getVM":
				result = map[string]any{"isolates": []any{map[string]any{"id": "isolates/1"}}}
			case "getIsolate":
				result = map[string]any{"libraries": []any{map[string]any{"id": "libraries/9", "uri": "package:chouwazi/game_screen.dart"},
					map[string]any{"id": "libraries/2", "uri": "dart:core"}}}
			case "getObject":
				result = map[string]any{"classes": []any{map[string]any{"id": "classes/5", "name": "_GameScreenState"}}}
			case "getInstances":
				result = map[string]any{"totalCount": 2, "samples": []any{map[string]any{"id": "objects/1"}, map[string]any{"id": "objects/2"}}}
			case "evaluate":
				if evaluateError && request.Params["expression"] != "mounted" {
					reply, _ := json.Marshal(map[string]any{"id": request.ID, "error": map[string]any{"code": 113, "message": "secret"}})
					connection.Write(ctx, websocket.MessageText, reply)
					continue
				}
				value := "false"
				if request.Params["targetId"] == "objects/"+string(rune('0'+mounted)) || request.Params["expression"] != "mounted" {
					value = "true"
				}
				result = map[string]any{"type": "@Instance", "kind": "Bool", "valueAsString": value}
			}
			// An unrelated event first: the client must wait for its own response id.
			noise, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "method": "streamNotify", "params": map[string]any{}})
			connection.Write(ctx, websocket.MessageText, noise)
			reply, _ := json.Marshal(map[string]any{"jsonrpc": "2.0", "id": request.ID, "result": result})
			connection.Write(ctx, websocket.MessageText, reply)
		}
	}))
	t.Cleanup(server.Close)
	return "ws" + strings.TrimPrefix(server.URL, "http") + "/token=/ws"
}

func TestVMLocatesMountedState(t *testing.T) {
	machine, err := connect(fakeVM(t, 2, false))
	if err != nil {
		t.Fatal(err)
	}
	defer machine.close()
	if machine.isolate != "isolates/1" || machine.state != "objects/2" {
		t.Fatalf("isolate %q, state %q", machine.isolate, machine.state)
	}
	if value, err := machine.string("_strings.text('privacy')", "privacy-label"); err != nil || value != "true" {
		t.Fatalf("got %q %v", value, err)
	}
	if _, err := machine.call("streamListen", "", nil); err == nil || err.Error() != "Unsupported VM service method" {
		t.Fatalf("method allowlist: %v", err)
	}
	if _, err := machine.evaluate("1", "", "made-up"); err == nil || err.Error() != "Unsupported VM operation label" {
		t.Fatalf("operation allowlist: %v", err)
	}
}

func TestVMErrorsKeepOnlyPublicCodes(t *testing.T) {
	machine, err := connect(fakeVM(t, 1, true))
	if err != nil {
		t.Fatal(err)
	}
	defer machine.close()
	_, err = machine.string("_strings.text('privacy')", "privacy-label")
	if err == nil || err.Error() != "VM evaluate failed (code=113, operation=privacy-label); debug details kept private" {
		t.Fatalf("got %v", err)
	}
	if _, err := connect("ws://192.0.2.1:1/x"); err == nil || err.Error() != "VM service must use a loopback WebSocket" {
		t.Fatalf("non-loopback: %v", err)
	}
}
