import 'draw_round.dart';

enum Control { vibration, retry, privacy }

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
  bool get canRetry => canActivate(Control.retry);
  bool get canOpenPrivacy => canActivate(Control.privacy);

  bool canActivate(Control control) =>
      allReleased && (control != Control.retry || round.isFinished);

  /// Overlay contacts cannot join or activate a game control, even if the
  /// overlay closes while they are held. Gameplay resumes after every release.
  void overlayDown(int id) {
    if (!_contacts.add(id)) return;
    _armedControl = null;
    _controlGesture = true;
  }

  bool down(int id, double x, double y, Duration now, {Control? control}) {
    if (_contacts.contains(id)) return false;
    final canArm = control != null && canActivate(control);
    _contacts.add(id);
    if (control != null) {
      if (canArm) {
        _controlGesture = true;
        _controlPointer = id;
        _armedControl = control;
      } else {
        _armedControl = null;
      }
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

  Control? up(int id, Duration now, {Control? control}) {
    if (!_contacts.remove(id)) return null;
    if (!_controlGesture) round.leave(id, now);
    Control? activated;
    if (id == _controlPointer && allReleased && control == _armedControl) {
      if (_armedControl != null) activated = activateControl(_armedControl!);
    }
    if (allReleased) _clearControl();
    return activated;
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
    activateControl(Control.retry);
  }

  void toggleHaptics() {
    activateControl(Control.vibration);
  }

  /// Semantics and completed physical taps use the same availability rule.
  /// Navigation stays with the UI; privacy activation leaves the round intact.
  Control? activateControl(Control control) {
    if (!canActivate(control)) return null;
    if (control == Control.retry) round.reset();
    if (control == Control.vibration) haptics = !haptics;
    return control;
  }

  void _clearControl() {
    _controlPointer = null;
    _armedControl = null;
    _controlGesture = false;
  }
}
