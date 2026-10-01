import 'draw_round.dart';

enum Control { vibration, retry }

/// Owns physical contacts, including overflow fingers and touches after a result.
/// A control gesture excludes gameplay until every contact is released.
class GameSession {
  GameSession({DrawRound? round}) : round = round ?? DrawRound();
  final DrawRound round;
  final Set<int> _contacts = {};
  int? _controlPointer;
  Control? _armedControl;
  bool _controlGesture = false;
  bool haptics = true;

  bool get allReleased => _contacts.isEmpty;
  bool get canRetry => round.isFinished && allReleased;

  bool down(int id, double x, double y, Duration now, {Control? control}) {
    if (_contacts.contains(id)) return false;
    final wasEmpty = allReleased;
    _contacts.add(id);
    if (wasEmpty &&
        control != null &&
        (control != Control.retry || round.isFinished)) {
      _controlGesture = true;
      _controlPointer = id;
      _armedControl = control;
      return false;
    }
    _armedControl = null;
    if (_controlGesture) return false;
    return round.join(id, x, y, now);
  }

  void move(int id, double x, double y, {Control? control}) {
    if (_controlPointer == id && control != _armedControl) _armedControl = null;
    if (!_controlGesture) round.move(id, x, y);
  }

  void up(int id, Duration now, {Control? control}) {
    if (!_contacts.remove(id)) return;
    if (!_controlGesture) round.leave(id, now);
    if (id == _controlPointer && allReleased && control == _armedControl) {
      if (_armedControl == Control.retry) round.reset();
      if (_armedControl == Control.vibration) haptics = !haptics;
    }
    if (allReleased) _clearControl();
  }

  // Platform cancellations invalidate the unfinished round as a whole. The
  // result remains frozen, and any still-held contacts must lift before retry.
  void cancel(int id) {
    if (!_contacts.remove(id)) return;
    if (!round.isFinished) round.reset();
    _armedControl = null;
    _controlGesture = !allReleased;
    if (allReleased) _clearControl();
  }

  void suspend() {
    _contacts.clear();
    _clearControl();
    if (!round.isFinished) round.reset();
  }

  void retry() {
    if (canRetry) round.reset();
  }

  void toggleHaptics() {
    if (allReleased) haptics = !haptics;
  }

  void _clearControl() {
    _controlPointer = null;
    _armedControl = null;
    _controlGesture = false;
  }
}
