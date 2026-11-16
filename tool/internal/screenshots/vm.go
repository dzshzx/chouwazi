package screenshots

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/url"
	"slices"
	"strconv"
	"strings"
	"time"

	"github.com/coder/websocket"
)

const snapshotExpression = `[
  _showPrivacy,
  _session.round.players.length,
  _session.round.winner?.slot,
  _session.allReleased,
  _session.round.deadline == null ? null :
    (_session.round.deadline! - _now).inMilliseconds
].toString()`

const geometryExpression = `(() {
  final view = WidgetsBinding.instance.platformDispatcher.views.single;
  final ratio = view.devicePixelRatio;
  return [view.physicalSize.width / ratio, view.physicalSize.height / ratio,
    view.padding.top / ratio, view.padding.bottom / ratio].toString();
})()`

const noDebugBanner = "!context.findAncestorWidgetOfExactType<WidgetsApp>()!.debugShowCheckedModeBanner"

var (
	vmOperations = []string{"evaluate", "string", "snapshot", "state-mounted", "locale", "privacy-label",
		"debug-banner", "home-before", "home-after", "geometry", "players-down",
		"winner-poll", "players-up", "result-before", "result-after", "privacy-down",
		"privacy-up", "privacy-rendered", "privacy-title", "privacy-before", "privacy-after"}
	vmResultTypes  = []string{"@Instance", "@Error", "Error", "Sentinel"}
	vmMethods      = []string{"getVM", "getIsolate", "getObject", "getInstances", "evaluate"}
	vmFailureKinds = []string{"UnhandledException", "LanguageError", "InternalError", "TerminationError",
		"Collected", "Expired", "NotInitialized", "BeingInitialized", "OptimizedOut", "Free"}
)

func vmErrorCode(code int64) bool {
	if code >= 100 && code < 115 {
		return true
	}
	return slices.Contains([]int64{-32700, -32600, -32601, -32602, -32603, -32010, 130, 131}, code)
}

// vmTimeout bounds every send and receive on the VM service connection.
const vmTimeout = 60 * time.Second

// vm is a JSON-RPC client for the loopback Dart VM service of the real app.
type vm struct {
	connection *websocket.Conn
	sequence   int
	isolate    string
	state      string
}

type object = map[string]any

func list(value any) []any {
	items, _ := value.([]any)
	return items
}

func text(value any) string {
	result, _ := value.(string)
	return result
}

func get(value any, key string) any {
	item, _ := value.(map[string]any)
	return item[key]
}

func connect(uri string) (*vm, error) {
	parsed, err := url.Parse(uri)
	if err != nil || parsed.Scheme != "ws" || parsed.Hostname() != "127.0.0.1" {
		return nil, Error("VM service must use a loopback WebSocket")
	}
	ctx, cancel := context.WithTimeout(context.Background(), vmTimeout)
	defer cancel()
	connection, _, err := websocket.Dial(ctx, uri, nil)
	if err != nil {
		return nil, errors.New("VM service connection failed")
	}
	// VM service responses (isolate libraries, class lists) exceed the default 32 KiB limit.
	connection.SetReadLimit(256 << 20)
	machine := &vm{connection: connection}
	if err := machine.locateState(); err != nil {
		_ = machine.close()
		return nil, err
	}
	return machine, nil
}

func (m *vm) locateState() error {
	result, err := m.call("getVM", "", nil)
	if err != nil {
		return err
	}
	isolates := list(result["isolates"])
	if err := require(len(isolates) == 1, "Expected exactly one application isolate"); err != nil {
		return err
	}
	m.isolate = text(get(isolates[0], "id"))
	if result, err = m.call("getIsolate", "", object{"isolateId": m.isolate}); err != nil {
		return err
	}
	var game []any
	for _, library := range list(result["libraries"]) {
		uri := text(get(library, "uri"))
		if uri == "package:chouwazi/game_screen.dart" || strings.HasSuffix(uri, "/lib/game_screen.dart") {
			game = append(game, library)
		}
	}
	if err := require(len(game) == 1, "The real game_screen library was not loaded"); err != nil {
		return err
	}
	if result, err = m.call("getObject", "", object{"isolateId": m.isolate, "objectId": get(game[0], "id")}); err != nil {
		return err
	}
	var states []any
	for _, class := range list(result["classes"]) {
		if text(get(class, "name")) == "_GameScreenState" {
			states = append(states, class)
		}
	}
	if err := require(len(states) == 1, "The real GameScreen state class was not loaded"); err != nil {
		return err
	}
	return m.locateMounted(get(states[0], "id"))
}

func (m *vm) locateMounted(class any) error {
	set, err := m.call("getInstances", "", object{"isolateId": m.isolate, "objectId": class, "limit": 10})
	if err != nil {
		return err
	}
	// Dart 3.6 uses 'samples'; newer protocol implementations use 'instances'.
	instances, ok := set["instances"]
	if !ok {
		instances = set["samples"]
	}
	total, ok := set["totalCount"].(json.Number)
	count, convErr := total.Int64()
	if !ok || convErr != nil {
		return errors.New("missing instance count")
	}
	if err := require(count <= 10, "Unexpected number of GameScreen states"); err != nil {
		return err
	}
	var mounted []string
	for _, instance := range list(instances) {
		id := text(get(instance, "id"))
		result, err := m.evaluate("mounted", id, "state-mounted")
		if err != nil {
			return err
		}
		if result["valueAsString"] == "true" {
			mounted = append(mounted, id)
		}
	}
	if err := require(len(mounted) == 1, "Expected exactly one mounted real GameScreen state"); err != nil {
		return err
	}
	m.state = mounted[0]
	return nil
}

type rpcResponse struct {
	ID     json.RawMessage     `json:"id"`
	Error  *struct{ Code any } `json:"error"`
	Result object              `json:"result"`
}

func (m *vm) call(method, operation string, params object) (object, error) {
	if err := require(slices.Contains(vmMethods, method), "Unsupported VM service method"); err != nil {
		return nil, err
	}
	if err := require(operation == "" || slices.Contains(vmOperations, operation), "Unsupported VM operation label"); err != nil {
		return nil, err
	}
	if params == nil {
		params = object{}
	}
	m.sequence++
	id := strconv.Itoa(m.sequence)
	request, err := json.Marshal(object{"jsonrpc": "2.0", "id": m.sequence, "method": method, "params": params})
	if err != nil {
		return nil, err
	}
	ctx, cancel := context.WithTimeout(context.Background(), vmTimeout)
	defer cancel()
	if err := m.connection.Write(ctx, websocket.MessageText, request); err != nil {
		return nil, errors.New("VM service send failed")
	}
	for {
		response, err := m.receive()
		if err != nil {
			return nil, err
		}
		if string(response.ID) != id && string(response.ID) != strconv.Quote(id) {
			continue
		}
		if response.Error != nil {
			code := "unknown"
			if number, ok := response.Error.Code.(json.Number); ok {
				if value, err := number.Int64(); err == nil && vmErrorCode(value) {
					code = number.String()
				}
			}
			stage := ""
			if operation != "" {
				stage = ", operation=" + operation
			}
			return nil, Error(fmt.Sprintf("VM %s failed (code=%s%s); debug details kept private", method, code, stage))
		}
		if response.Result == nil {
			return nil, errors.New("VM response has no result")
		}
		return response.Result, nil
	}
}

func (m *vm) receive() (rpcResponse, error) {
	ctx, cancel := context.WithTimeout(context.Background(), vmTimeout)
	defer cancel()
	_, data, err := m.connection.Read(ctx)
	if err != nil {
		return rpcResponse{}, errors.New("VM service receive failed")
	}
	var response rpcResponse
	decoder := json.NewDecoder(bytes.NewReader(data))
	decoder.UseNumber()
	if err := decoder.Decode(&response); err != nil {
		return rpcResponse{}, errors.New("VM service sent invalid JSON")
	}
	return response, nil
}

func (m *vm) evaluate(expression, target, operation string) (object, error) {
	// Flutter's compile-expression stdin protocol accepts one line.
	expression = strings.TrimSpace(strings.Join(splitLines(expression), " "))
	if target == "" {
		target = m.state
	}
	if operation == "" {
		operation = "evaluate"
	}
	result, err := m.call("evaluate", operation, object{"isolateId": m.isolate, "targetId": target,
		"expression": expression, "disableBreakpoints": true})
	if err != nil {
		return nil, err
	}
	resultType, kind := text(result["type"]), text(result["kind"])
	if !slices.Contains(vmResultTypes, resultType) {
		resultType = "unknown"
	}
	if !slices.Contains(vmFailureKinds, kind) {
		kind = "unknown"
	}
	if err := require(result["type"] == "@Instance", fmt.Sprintf(
		"Debug expression failed to evaluate (operation=%s, type=%s, kind=%s); debug details kept private",
		operation, resultType, kind)); err != nil {
		return nil, err
	}
	return result, nil
}

// splitLines splits like str.splitlines for the line breaks that occur in expressions.
func splitLines(text string) []string {
	text = strings.ReplaceAll(text, "\r\n", "\n")
	text = strings.ReplaceAll(text, "\r", "\n")
	return strings.Split(strings.TrimSuffix(text, "\n"), "\n")
}

func (m *vm) string(expression, operation string) (string, error) {
	result, err := m.evaluate(expression, "", operation)
	if err != nil {
		return "", err
	}
	if err := require(!truthy(result["valueAsStringIsTruncated"]), "Debug expression result was truncated (operation="+operation+")"); err != nil {
		return "", err
	}
	value, ok := result["valueAsString"].(string)
	if !ok {
		return "", errors.New("missing valueAsString")
	}
	return value, nil
}

// State is the game state read through the debug VM around each screenshot.
type State struct {
	Privacy     any `json:"privacy"`
	Players     any `json:"players"`
	WinnerSlot  any `json:"winner_slot"`
	AllReleased any `json:"all_released"`
	RemainingMS any `json:"remaining_ms"`
}

// ParseState decodes the Dart list printed by the snapshot expression.
func ParseState(printed string) (State, error) {
	decoder := json.NewDecoder(strings.NewReader(printed))
	decoder.UseNumber()
	var values []any
	if err := decoder.Decode(&values); err != nil || len(values) < 5 {
		return State{}, errors.New("invalid game state snapshot")
	}
	return State{values[0], values[1], values[2], values[3], values[4]}, nil
}

func (m *vm) snapshot(operation string) (State, error) {
	printed, err := m.string(snapshotExpression, operation)
	if err != nil {
		return State{}, err
	}
	return ParseState(printed)
}

type touch struct {
	pointer int
	x, y    float64
}

// floatLiteral writes the shortest round-trip decimal, always with a decimal point, as a Dart double literal.
func floatLiteral(value float64) string {
	formatted := strconv.FormatFloat(value, 'f', -1, 64)
	if !strings.ContainsAny(formatted, ".eEn") {
		formatted += ".0"
	}
	return formatted
}

// TouchExpression builds the Dart expression that dispatches pointer events on the next event-loop turn.
func TouchExpression(action string, positions []touch) string {
	var events []string
	for _, position := range positions {
		events = append(events, fmt.Sprintf("GestureBinding.instance.handlePointerEvent("+
			"Pointer%sEvent(pointer: %d, kind: PointerDeviceKind.touch, position: Offset(%s, %s)));",
			action, position.pointer, floatLiteral(position.x), floatLiteral(position.y)))
	}
	return "(() { Future<void>(() { " + strings.Join(events, "\n") + " }); return true; })()"
}

func (m *vm) touches(action string, positions []touch, operation string) error {
	if err := require(action == "Down" || action == "Up", "Unsupported pointer event"); err != nil {
		return err
	}
	// VM evaluation can interrupt build/layout/paint; dispatch on the next
	// ordinary event-loop turn so pointer handlers can safely schedule frames.
	_, err := m.evaluate(TouchExpression(action, positions), "", operation)
	return err
}

func (m *vm) close() error {
	return m.connection.Close(websocket.StatusNormalClosure, "")
}

func truthy(value any) bool {
	switch typed := value.(type) {
	case nil:
		return false
	case bool:
		return typed
	case string:
		return typed != ""
	case json.Number:
		number, err := typed.Float64()
		return err != nil || number != 0
	case []any:
		return len(typed) > 0
	case map[string]any:
		return len(typed) > 0
	}
	return true
}

func intEquals(value any, want int64) bool {
	switch typed := value.(type) {
	case bool:
		return (typed && want == 1) || (!typed && want == 0)
	case json.Number:
		number, err := typed.Float64()
		return err == nil && number == float64(want)
	}
	return false
}
