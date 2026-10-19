import 'dart:ui' show SemanticsAction;

import 'package:chouwazi/draw_round.dart';
import 'package:chouwazi/game_screen.dart';
import 'package:chouwazi/game_session.dart';
import 'package:chouwazi/privacy_screen.dart';
import 'package:chouwazi/strings.dart';
import 'package:flutter/gestures.dart';
import 'package:flutter/widgets.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  void useView(WidgetTester tester, Size size) {
    tester.view.physicalSize = size;
    tester.view.devicePixelRatio = 1;
    addTearDown(tester.view.resetPhysicalSize);
    addTearDown(tester.view.resetDevicePixelRatio);
  }

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

  testWidgets(
    'privacy gestures cannot join a round or open while fingers hold',
    (tester) async {
      useView(tester, const Size(400, 800));
      final session = GameSession();
      await tester.pumpWidget(
        WidgetsApp(
          color: const Color(0xFF0D111B),
          builder: (context, child) => GameScreen(session: session),
        ),
      );
      final privacy = GameLayout(const Size(400, 800)).privacy.center;
      final finger = await tester.startGesture(
        const Offset(90, 300),
        pointer: 1,
      );
      await tester.tapAt(privacy);
      await tester.pump();
      expect(find.byType(PrivacyScreen), findsNothing);
      expect(session.round.players.single.id, 1);
      await finger.up();

      final cancelled = await tester.startGesture(privacy, pointer: 2);
      await cancelled.cancel();
      await tester.pump();
      expect(find.byType(PrivacyScreen), findsNothing);
      final dragged = await tester.startGesture(privacy, pointer: 3);
      await dragged.moveTo(const Offset(120, 100));
      await dragged.moveTo(privacy);
      await dragged.up();
      await tester.pump();
      expect(find.byType(PrivacyScreen), findsNothing);
      final control = await tester.startGesture(privacy, pointer: 4);
      final extra = await tester.startGesture(
        const Offset(200, 300),
        pointer: 5,
      );
      await extra.up();
      await control.up();
      await tester.pump();
      expect(find.byType(PrivacyScreen), findsNothing);
      expect(session.round.players, isEmpty);
      await tester.tapAt(privacy);
      await tester.pump();
      expect(find.byType(PrivacyScreen), findsOneWidget);
      expect(session.allReleased, isTrue);
      await tester.pumpWidget(const SizedBox());
    },
  );

  testWidgets('privacy stops result animation and returns to the same winner', (
    tester,
  ) async {
    useView(tester, const Size(400, 800));
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
    final first = await tester.startGesture(const Offset(90, 300), pointer: 1);
    final second = await tester.startGesture(
      const Offset(260, 330),
      pointer: 2,
    );
    now = DrawRound.delay;
    await tester.pump(const Duration(milliseconds: 16));
    await first.up();
    await second.up();
    await tester.tapAt(GameLayout(const Size(400, 800)).privacy.center);
    await tester.pump();
    expect(find.byType(PrivacyScreen), findsOneWidget);
    expect(tester.binding.transientCallbackCount, 0);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.inactive);
    tester.binding.handleAppLifecycleStateChanged(AppLifecycleState.resumed);
    await tester.pump();
    expect(tester.binding.transientCallbackCount, 0);
    await tester.tap(find.byKey(const Key('privacy-back')));
    await tester.pump();
    expect(find.byType(PrivacyScreen), findsNothing);
    expect(session.round.winnerId, 2);
    expect(session.canRetry, isTrue);
    expect(tester.binding.transientCallbackCount, greaterThan(0));
    await tester.tapAt(GameLayout(const Size(400, 800)).privacy.center);
    await tester.pump();
    await tester.binding.handlePopRoute();
    await tester.pump();
    expect(find.byType(PrivacyScreen), findsNothing);
    expect(session.round.winnerId, 2);
    expect(tester.takeException(), isNull);
    await tester.pumpWidget(const SizedBox());
  });

  for (final systemBack in [false, true]) {
    testWidgets(
      'privacy ${systemBack ? 'system' : 'button'} back keeps held body contact in retry gate',
      (tester) async {
        useView(tester, const Size(400, 800));
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
          pointer: 1,
        );
        final second = await tester.startGesture(
          const Offset(260, 330),
          pointer: 2,
        );
        now = DrawRound.delay;
        await tester.pump(const Duration(milliseconds: 16));
        await first.up();
        await second.up();
        final layout = GameLayout(const Size(400, 800));
        await tester.tapAt(layout.privacy.center, pointer: 6);
        await tester.pump();
        expect(find.byType(PrivacyScreen), findsOneWidget);
        expect(tester.binding.transientCallbackCount, 0);
        final body = await tester.startGesture(
          const Offset(120, 240),
          pointer: 3,
        );
        await tester.pump();
        if (systemBack) {
          await tester.binding.handlePopRoute();
        } else {
          await tester.tap(find.byKey(const Key('privacy-back')), pointer: 7);
        }
        await tester.pump();
        expect(find.byType(PrivacyScreen), findsNothing);
        expect(session.canRetry, isFalse);
        expect(session.round.winnerId, 2);
        await tester.tapAt(layout.retry.center, pointer: 8);
        await tester.pump();
        expect(session.round.winnerId, 2);
        final extra = await tester.startGesture(
          const Offset(280, 450),
          pointer: 4,
        );
        await body.moveTo(const Offset(130, 250));
        await body.up();
        await tester.pump();
        expect(session.canRetry, isFalse);
        await extra.up();
        await tester.pump();
        expect(session.canRetry, isTrue);
        await tester.tapAt(layout.retry.center, pointer: 9);
        await tester.pump();
        expect(session.round.players, isEmpty);
        final next = await tester.startGesture(
          const Offset(90, 300),
          pointer: 5,
        );
        expect(session.round.players.single.id, 5);
        await next.up();
        expect(tester.takeException(), isNull);
        await tester.pumpWidget(const SizedBox());
      },
    );
  }

  testWidgets('privacy semantics follows physical contact availability', (
    tester,
  ) async {
    useView(tester, const Size(400, 800));
    final semantics = tester.ensureSemantics();
    try {
      final session = GameSession();
      await tester.pumpWidget(
        WidgetsApp(
          color: const Color(0xFF0D111B),
          builder: (context, child) =>
              GameScreen(session: session, locale: const Locale('en')),
        ),
      );
      final finger = await tester.startGesture(
        const Offset(90, 300),
        pointer: 1,
      );
      await tester.pump();
      final held = tester.getSemantics(find.bySemanticsLabel('Privacy'));
      expect(held.getSemanticsData().hasAction(SemanticsAction.tap), isFalse);
      await finger.up();
      await tester.pump();
      final idle = tester.getSemantics(find.bySemanticsLabel('Privacy'));
      expect(idle.getSemanticsData().hasAction(SemanticsAction.tap), isTrue);
      idle.owner!.performAction(idle.id, SemanticsAction.tap);
      await tester.pump();
      expect(find.byType(PrivacyScreen), findsOneWidget);
      expect(session.round.players, isEmpty);
      await tester.pumpWidget(const SizedBox());
    } finally {
      semantics.dispose();
    }
  });

  for (final locale in [
    const Locale('en'),
    const Locale('zh', 'CN'),
    const Locale.fromSubtags(languageCode: 'zh', scriptCode: 'Hant'),
  ]) {
    testWidgets(
      'privacy is readable and scrolls on a small large-text $locale',
      (tester) async {
        useView(tester, const Size(280, 480));
        final strings = GameStrings(locale);
        await tester.pumpWidget(
          WidgetsApp(
            color: const Color(0xFF0D111B),
            builder: (context, child) => MediaQuery(
              data: MediaQuery.of(context)
                  .copyWith(textScaler: const TextScaler.linear(2)),
              child: GameScreen(locale: locale),
            ),
          ),
        );
        await tester.tapAt(GameLayout(const Size(280, 480)).privacy.center);
        await tester.pump();
        expect(find.text(strings.text('privacy_title')), findsOneWidget);
        final scroll = tester.state<ScrollableState>(find.byType(Scrollable));
        expect(scroll.position.maxScrollExtent, greaterThan(0));
        await tester.drag(
          find.byKey(const Key('privacy-scroll')),
          Offset(0, -scroll.position.maxScrollExtent),
        );
        await tester.pumpAndSettle();
        expect(
          find.text(strings.text('privacy_contact')).hitTestable(),
          findsOneWidget,
        );
        expect(tester.takeException(), isNull);
        await tester.tap(find.byKey(const Key('privacy-back')));
        await tester.pump();
        expect(find.byKey(const Key('game-surface')), findsOneWidget);
        await tester.pumpWidget(const SizedBox());
      },
    );
  }
}
