import 'package:chouwazi/draw_round.dart';
import 'package:chouwazi/game_screen.dart';
import 'package:chouwazi/game_session.dart';
import 'package:chouwazi/strings.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('language resolution covers scripts, regions and English fallback', () {
    expect(GameStrings(const Locale('zh', 'CN')).text('ready'), '准备');
    expect(GameStrings(const Locale('zh', 'TW')).text('ready'), '準備');
    expect(GameStrings(const Locale('zh', 'HK')).text('ready'), '準備');
    expect(
      GameStrings(
        const Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hant'),
      ).text('ready'),
      '準備',
    );
    expect(GameStrings(const Locale('fr')).text('ready'), 'Ready');
  });

  testWidgets('raw multi-touch reaches shared rules and release-gated retry', (
    tester,
  ) async {
    tester.view.physicalSize = const Size(400, 800);
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
    final session = GameSession(round: DrawRound(choose: (_) => 1));
    var now = Duration.zero;
    await tester.pumpWidget(
      WidgetsApp(
        color: const Color(0xFF0D111B),
        builder: (context, child) => GameScreen(
          session: session,
          clock: () => now,
          locale: const Locale('en'),
        ),
      ),
    );
    final first = await tester.startGesture(
      const Offset(90, 300),
      pointer: 17,
      kind: PointerDeviceKind.touch,
    );
    final second = await tester.startGesture(
      const Offset(260, 330),
      pointer: 90,
      kind: PointerDeviceKind.touch,
    );
    await tester.pump();
    final deadline = session.round.deadline;
    await first.moveTo(const Offset(100, 310));
    expect(session.round.deadline, deadline);
    now = const Duration(milliseconds: 3799);
    await tester.pump(const Duration(milliseconds: 16));
    expect(session.round.isFinished, isFalse);
    now = DrawRound.delay;
    await tester.pump(const Duration(milliseconds: 16));
    expect(session.round.winner?.slot, 1);
    expect(session.canRetry, isFalse);
    await first.up();
    await second.up();
    await tester.pump();
    expect(session.canRetry, isTrue);
    await tester.tapAt(GameLayout(const Size(400, 800)).retry.center);
    await tester.pump();
    expect(session.round.players, isEmpty);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
  });

  testWidgets('backgrounding clears an unfinished round and resumes cleanly', (
    tester,
  ) async {
    final session = GameSession();
    await tester.pumpWidget(
      WidgetsApp(
        color: const Color(0xFF0D111B),
        builder: (context, child) => GameScreen(session: session),
      ),
    );
    final first = await tester.startGesture(const Offset(90, 300), pointer: 1);
    final second = await tester.startGesture(
      const Offset(260, 330),
      pointer: 2,
    );
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
    await tester.pump();
    expect(session.round.deadline, isNull);
    expect(session.round.players, isEmpty);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await first.up();
    await second.up();
    await tester.pump();
    expect(session.round.players, isEmpty);
    await tester.pumpWidget(const SizedBox());
  });
}
