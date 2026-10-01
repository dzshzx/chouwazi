import 'package:chouwazi/draw_round.dart';
import 'package:flutter_test/flutter_test.dart';

Duration ms(int value) => Duration(milliseconds: value);

void main() {
  test('stable identities, a single 3.8s deadline and exact boundary', () {
    final round = DrawRound(choose: (_) => 1);
    round.join(17, 1, 2, ms(0));
    expect(round.deadline, isNull);
    round.join(90, 4, 5, ms(1000));
    expect(round.deadline, ms(4800));
    round.move(17, 9, 10);
    expect(round.deadline, ms(4800));
    expect(round.tick(ms(4799)), isFalse);
    expect(round.tick(ms(4800)), isTrue);
    expect(round.winnerId, 90);
    expect(round.tick(ms(6000)), isFalse);
  });

  test('joins and leaves restart; one contact cancels countdown', () {
    final round = DrawRound();
    round.join(2, 0, 0, ms(0));
    round.join(4, 0, 0, ms(0));
    round.join(6, 0, 0, ms(2000));
    expect(round.deadline, ms(5800));
    round.leave(4, ms(3000));
    expect(round.deadline, ms(6800));
    round.leave(2, ms(4000));
    expect(round.deadline, isNull);
    expect(round.tick(ms(10000)), isFalse);
  });

  test('ten contacts, reusable slots, duplicate and unknown IDs are inert', () {
    final round = DrawRound();
    for (var id = 0; id < 11; id++) {
      round.join(id * 7, 0, 0, ms(id));
    }
    expect(round.players.length, 10);
    final deadline = round.deadline;
    round.join(0, 900, 900, ms(100));
    round.leave(999, ms(200));
    round.move(999, 3, 5);
    expect(round.deadline, deadline);
    expect(round.players.first.x, 0);
    round.leave(28, ms(300));
    round.join(99, 1, 2, ms(400));
    expect(round.players.last.slot, 4);
    expect(round.players.map((p) => p.slot).toSet().length, 10);
  });

  test(
    'every participant has one selection index; result freezes until reset',
    () {
      for (var selected = 0; selected < 10; selected++) {
        final round = DrawRound(
          choose: (count) {
            expect(count, 10);
            return selected;
          },
        );
        for (var i = 0; i < 10; i++) {
          round.join(i * 7, i.toDouble(), 20, ms(0));
        }
        expect(round.tick(ms(3800)), isTrue);
        expect(round.winnerId, selected * 7);
        final winner = round.winner!;
        round.move(winner.id, 999, 999);
        round.leave(winner.id, ms(4000));
        round.join(111, 2, 2, ms(4000));
        expect(round.winner, same(winner));
        expect(round.players.length, 10);
        round.reset();
        expect(round.players, isEmpty);
        expect(round.winnerId, isNull);
        expect(round.deadline, isNull);
      }
    },
  );
}
