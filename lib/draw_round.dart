import 'dart:collection';
import 'dart:math';

/// A round uses pointer identities and one monotonic deadline on every platform.
class DrawRound {
  DrawRound({int Function(int)? choose})
      : _choose = choose ?? Random.secure().nextInt;

  static const limit = 10;
  static const delay = Duration(milliseconds: 3800);
  final int Function(int) _choose;
  final Map<int, DrawPlayer> _players = {};
  Duration? _deadline;
  int? _winnerId;

  List<DrawPlayer> get players => UnmodifiableListView(_players.values);
  Duration? get deadline => _deadline;
  int? get winnerId => _winnerId;
  DrawPlayer? get winner => _players[_winnerId];
  bool get isFinished => _winnerId != null;

  bool join(int id, double x, double y, Duration now) {
    if (isFinished || _players.containsKey(id) || _players.length >= limit) {
      return false;
    }
    final used = _players.values.map((player) => player.slot).toSet();
    final slot = List.generate(
      limit,
      (i) => i,
    ).firstWhere((i) => !used.contains(i));
    _players[id] = DrawPlayer(id, slot, x, y);
    _restart(now);
    return true;
  }

  void move(int id, double x, double y) {
    final player = _players[id];
    if (!isFinished && player != null) {
      _players[id] = DrawPlayer(id, player.slot, x, y);
    }
  }

  void leave(int id, Duration now) {
    if (!isFinished && _players.remove(id) != null) _restart(now);
  }

  void _restart(Duration now) {
    _deadline = _players.length >= 2 ? now + delay : null;
  }

  bool tick(Duration now) {
    if (isFinished || _deadline == null || now < _deadline!) return false;
    final index = _choose(_players.length);
    RangeError.checkValidIndex(index, _players.values.toList());
    _winnerId = _players.keys.elementAt(index);
    _deadline = null;
    return true;
  }

  void reset() {
    _players.clear();
    _deadline = null;
    _winnerId = null;
  }
}

class DrawPlayer {
  const DrawPlayer(this.id, this.slot, this.x, this.y);
  final int id;
  final int slot;
  final double x;
  final double y;
}
