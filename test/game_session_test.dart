import 'package:chouwazi/draw_round.dart';
import 'package:chouwazi/game_session.dart';
import 'package:flutter_test/flutter_test.dart';

void main() {
  test('overflow and post-result contacts must lift before retry', () {
    final session = GameSession(round: DrawRound(choose: (_) => 0));
    for (var i = 0; i < 11; i++) {
      session.down(i, 0, 0, Duration.zero);
    }
    session.round.tick(DrawRound.delay);
    for (var i = 0; i < 10; i++) {
      session.up(i, DrawRound.delay);
    }
    expect(session.canRetry, isFalse);
    session.retry();
    expect(session.round.isFinished, isTrue);
    session.up(10, DrawRound.delay);
    expect(session.canRetry, isTrue);
    session.down(100, 0, 0, DrawRound.delay);
    expect(session.canRetry, isFalse);
    session.up(100, DrawRound.delay);
    session.retry();
    expect(session.round.players, isEmpty);
  });

  test('control gesture never leaks participants into the round', () {
    final session = GameSession();
    session.down(1, 0, 0, Duration.zero, control: Control.vibration);
    session.down(2, 100, 100, Duration.zero);
    session.up(1, Duration.zero, control: Control.vibration);
    session.down(3, 100, 100, Duration.zero);
    expect(session.round.players, isEmpty);
    session.up(2, Duration.zero);
    session.up(3, Duration.zero);
    expect(session.haptics, isTrue);
    session.down(4, 0, 0, Duration.zero, control: Control.vibration);
    session.up(4, Duration.zero, control: Control.vibration);
    expect(session.haptics, isFalse);
  });

  test('drag out cancels control tap', () {
    final session = GameSession();
    session.down(1, 0, 0, Duration.zero, control: Control.vibration);
    session.move(1, 30, 30);
    session.move(1, 0, 0, control: Control.vibration);
    session.up(1, Duration.zero, control: Control.vibration);
    expect(session.haptics, isTrue);
  });

  test('privacy controls are isolated from an ongoing countdown', () {
    final session = GameSession();
    session.down(1, 50, 200, Duration.zero);
    session.down(2, 150, 200, Duration.zero);
    final deadline = session.round.deadline;
    expect(session.canOpenPrivacy, isFalse);
    expect(session.activateControl(Control.privacy), isNull);
    expect(
      session.down(
        3,
        10,
        10,
        const Duration(seconds: 1),
        control: Control.privacy,
      ),
      isFalse,
    );
    session.move(3, 20, 20, control: Control.privacy);
    expect(session.round.players.map((player) => player.id), [1, 2]);
    expect(session.round.deadline, deadline);
    expect(
      session.up(3, const Duration(seconds: 1), control: Control.privacy),
      isNull,
    );
    expect(session.round.deadline, deadline);
    session.up(1, Duration.zero);
    session.up(2, Duration.zero);
    expect(session.canOpenPrivacy, isTrue);
  });

  test('privacy tap survives no cancellation or extra physical contact', () {
    final session = GameSession();
    session.down(1, 10, 10, Duration.zero, control: Control.privacy);
    session.move(1, 100, 100);
    session.move(1, 10, 10, control: Control.privacy);
    expect(
      session.up(1, Duration.zero, control: Control.privacy),
      isNull,
    );
    session.down(2, 10, 10, Duration.zero, control: Control.privacy);
    session.cancel(2);
    expect(
      session.up(2, Duration.zero, control: Control.privacy),
      isNull,
    );
    session.down(3, 10, 10, Duration.zero, control: Control.privacy);
    session.down(4, 100, 100, Duration.zero);
    session.up(4, Duration.zero);
    expect(
      session.up(3, Duration.zero, control: Control.privacy),
      isNull,
    );
    expect(session.round.players, isEmpty);
    session.down(5, 10, 10, Duration.zero, control: Control.privacy);
    expect(
      session.up(5, Duration.zero, control: Control.privacy),
      Control.privacy,
    );
    expect(session.round.players, isEmpty);
  });

  test('privacy activation preserves a released result', () {
    final session = GameSession(round: DrawRound(choose: (_) => 1));
    session.down(1, 50, 200, Duration.zero);
    session.down(2, 150, 200, Duration.zero);
    session.round.tick(DrawRound.delay);
    session.up(1, DrawRound.delay);
    session.up(2, DrawRound.delay);
    session.down(3, 10, 10, DrawRound.delay, control: Control.privacy);
    expect(
      session.up(3, DrawRound.delay, control: Control.privacy),
      Control.privacy,
    );
    expect(session.round.winnerId, 2);
    expect(session.round.players.map((player) => player.id), [1, 2]);
    expect(session.canRetry, isTrue);
  });

  test('overlay contacts exclude gameplay until every contact releases', () {
    final session = GameSession();
    session.overlayDown(1);
    session.overlayDown(1);
    expect(session.allReleased, isFalse);
    expect(session.canOpenPrivacy, isFalse);
    expect(session.activateControl(Control.vibration), isNull);
    expect(session.down(2, 100, 200, Duration.zero), isFalse);
    session.up(1, Duration.zero);
    expect(session.allReleased, isFalse);
    expect(session.round.players, isEmpty);
    session.up(2, Duration.zero);
    expect(session.allReleased, isTrue);
    expect(session.down(3, 100, 200, Duration.zero), isTrue);
    expect(session.round.players.single.id, 3);
  });

  test('overlay cancellation preserves results and waits for all releases', () {
    final session = GameSession(round: DrawRound(choose: (_) => 1));
    session.down(1, 50, 200, Duration.zero);
    session.down(2, 150, 200, Duration.zero);
    session.round.tick(DrawRound.delay);
    session.up(1, DrawRound.delay);
    session.up(2, DrawRound.delay);
    session.overlayDown(3);
    session.overlayDown(4);
    session.move(3, 60, 300);
    session.retry();
    expect(session.round.winnerId, 2);
    expect(session.round.players.first.x, 50);
    session.cancel(3);
    expect(session.canRetry, isFalse);
    expect(session.round.winnerId, 2);
    session.up(4, DrawRound.delay);
    expect(session.canRetry, isTrue);
    session.retry();
    expect(session.round.players, isEmpty);
  });

  test('cancel clears unfinished round and waits for remaining fingers', () {
    final session = GameSession();
    session.down(1, 0, 0, Duration.zero);
    session.down(2, 0, 0, Duration.zero);
    session.cancel(1);
    expect(session.round.deadline, isNull);
    expect(session.round.players, isEmpty);
    session.down(3, 0, 0, Duration.zero);
    expect(session.round.players, isEmpty);
    session.up(2, Duration.zero);
    session.up(3, Duration.zero);
    session.down(4, 0, 0, Duration.zero);
    expect(session.round.players.single.id, 4);
  });

  test('suspend cancels countdown but preserves finished result', () {
    final session = GameSession(round: DrawRound(choose: (_) => 0));
    session.down(1, 0, 0, Duration.zero);
    session.down(2, 0, 0, Duration.zero);
    session.suspend();
    expect(session.round.players, isEmpty);
    expect(session.allReleased, isTrue);
    session.down(3, 0, 0, Duration.zero);
    session.down(4, 0, 0, Duration.zero);
    session.round.tick(DrawRound.delay);
    session.suspend();
    expect(session.round.winnerId, 3);
    expect(session.canRetry, isTrue);
  });
}
