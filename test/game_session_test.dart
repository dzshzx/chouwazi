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
