import 'dart:async';
import 'dart:math' as math;

import 'package:flutter/gestures.dart';
import 'package:flutter/scheduler.dart';
import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';

import 'game_session.dart';
import 'strings.dart';

const _background = Color(0xFF0D111B);
const _muted = Color(0xFF8B96AA);
const _colors = [
  Color(0xFFB5F36D),
  Color(0xFF8CB8FF),
  Color(0xFFFFADCA),
  Color(0xFFFFD478),
  Color(0xFFC3A0FF),
  Color(0xFF69DDCA),
  Color(0xFFFF967F),
  Color(0xFF98E2FF),
  Color(0xFFE7A7EC),
  Color(0xFFE5EA8B),
];

class GameScreen extends StatefulWidget {
  const GameScreen({super.key, this.session, this.clock, this.locale});
  final GameSession? session;
  final Duration Function()? clock;
  final Locale? locale;

  @override
  State<GameScreen> createState() => _GameScreenState();
}

class _GameScreenState extends State<GameScreen>
    with SingleTickerProviderStateMixin, WidgetsBindingObserver {
  late final GameSession _session = widget.session ?? GameSession();
  final Stopwatch _clock = Stopwatch()..start();
  late final Ticker _ticker;
  Duration? _lastDeadline;
  int? _lastSecond;
  bool _active = true;
  Duration get _now => widget.clock?.call() ?? _clock.elapsed;

  GameStrings get _strings => GameStrings(
        widget.locale ?? WidgetsBinding.instance.platformDispatcher.locale,
      );

  @override
  void initState() {
    super.initState();
    WidgetsBinding.instance.addObserver(this);
    _ticker = createTicker((_) => _frame());
  }

  @override
  void didChangeLocales(List<Locale>? locales) => setState(() {});

  @override
  void didChangeAppLifecycleState(AppLifecycleState state) {
    _active = state == AppLifecycleState.resumed;
    if (!_active) {
      _session.suspend();
      _lastDeadline = null;
      _lastSecond = null;
    }
    _refresh();
  }

  void _refresh() {
    if (!mounted) return;
    final animate = _active && _session.round.players.isNotEmpty;
    if (animate && !_ticker.isActive) _ticker.start();
    if (!animate && _ticker.isActive) _ticker.stop();
    setState(() {});
  }

  void _frame() {
    final now = _now;
    final round = _session.round;
    if (round.tick(now) && _session.haptics) {
      unawaited(HapticFeedback.heavyImpact());
    }
    final deadline = round.deadline;
    if (deadline != _lastDeadline) {
      _lastDeadline = deadline;
      _lastSecond = null;
    }
    if (deadline != null) {
      final remaining = (deadline - now).inMicroseconds / 1000000;
      final second = remaining.ceil();
      if (remaining <= 3 && second > 0 && second != _lastSecond) {
        _lastSecond = second;
        if (_session.haptics) unawaited(HapticFeedback.lightImpact());
      }
    }
    setState(() {});
  }

  @override
  void dispose() {
    WidgetsBinding.instance.removeObserver(this);
    _ticker.dispose();
    _clock.stop();
    super.dispose();
  }

  @override
  Widget build(BuildContext context) {
    final strings = _strings;
    final round = _session.round;
    final status = round.winner == null
        ? strings.joined(round.players.length)
        : strings.winner(round.winner!.slot + 1);
    return ColoredBox(
      color: _background,
      child: SafeArea(
        child: LayoutBuilder(
          builder: (context, constraints) {
            final layout = GameLayout(constraints.biggest);
            Control? controlAt(Offset position) {
              if (layout.vibration.contains(position)) return Control.vibration;
              if (round.isFinished && layout.retry.contains(position)) {
                return Control.retry;
              }
              return null;
            }

            return Listener(
              key: const Key('game-surface'),
              behavior: HitTestBehavior.opaque,
              onPointerDown: (event) {
                if (!_active || event.kind != PointerDeviceKind.touch) return;
                final joined = _session.down(
                  event.pointer,
                  event.localPosition.dx,
                  event.localPosition.dy,
                  _now,
                  control: controlAt(event.localPosition),
                );
                if (joined && _session.haptics) {
                  unawaited(HapticFeedback.lightImpact());
                }
                _refresh();
              },
              onPointerMove: (event) {
                if (!_active || event.kind != PointerDeviceKind.touch) return;
                _session.move(
                  event.pointer,
                  event.localPosition.dx,
                  event.localPosition.dy,
                  control: controlAt(event.localPosition),
                );
                _refresh();
              },
              onPointerUp: (event) {
                if (!_active || event.kind != PointerDeviceKind.touch) return;
                _session.up(
                  event.pointer,
                  _now,
                  control: controlAt(event.localPosition),
                );
                _refresh();
              },
              onPointerCancel: (event) {
                _session.cancel(event.pointer);
                _refresh();
              },
              child: Stack(
                children: [
                  Positioned.fill(
                    child: CustomPaint(
                      painter: _GamePainter(
                        session: _session,
                        strings: strings,
                        layout: layout,
                        now: _now,
                        reduceMotion: MediaQuery.disableAnimationsOf(context),
                        fontFamily:
                            DefaultTextStyle.of(context).style.fontFamily,
                      ),
                    ),
                  ),
                  Positioned.fromRect(
                    rect: Rect.fromLTWH(
                      12,
                      layout.scale * 105,
                      layout.size.width - 24,
                      56,
                    ),
                    child: Semantics(
                      label: status,
                      liveRegion: true,
                      child: const IgnorePointer(child: SizedBox.expand()),
                    ),
                  ),
                  Positioned.fromRect(
                    rect: layout.vibration,
                    child: Semantics(
                      label: strings.text(
                        _session.haptics ? 'haptics_on' : 'haptics_off',
                      ),
                      button: true,
                      enabled: _session.allReleased,
                      onTap: _session.allReleased
                          ? () {
                              _session.toggleHaptics();
                              _refresh();
                            }
                          : null,
                      child: const IgnorePointer(child: SizedBox.expand()),
                    ),
                  ),
                  if (round.isFinished)
                    Positioned.fromRect(
                      rect: layout.retry,
                      child: Semantics(
                        label: strings.text(
                          _session.canRetry ? 'again' : 'release',
                        ),
                        button: true,
                        enabled: _session.canRetry,
                        onTap: _session.canRetry
                            ? () {
                                _session.retry();
                                _refresh();
                              }
                            : null,
                        child: const IgnorePointer(child: SizedBox.expand()),
                      ),
                    ),
                ],
              ),
            );
          },
        ),
      ),
    );
  }
}

/// Drawing and hit testing share these bounds, including on small screens.
class GameLayout {
  GameLayout(this.size)
      : scale = math.min(1.0, math.min(size.width / 360, size.height / 600));
  final Size size;
  final double scale;
  Rect get vibration =>
      Rect.fromLTWH(math.max(0, size.width - 112), 0, 112, 44);
  Rect get retry => Rect.fromLTWH(
        24 * scale,
        size.height - 90 * scale,
        size.width - 48 * scale,
        math.max(44, 54 * scale),
      );
  double get middle => math.max(210 * scale, size.height * .46);
}

class _GamePainter extends CustomPainter {
  _GamePainter({
    required this.session,
    required this.strings,
    required this.layout,
    required this.now,
    required this.reduceMotion,
    this.fontFamily,
  });
  final GameSession session;
  final GameStrings strings;
  final GameLayout layout;
  final Duration now;
  final bool reduceMotion;
  final String? fontFamily;

  void _text(
    Canvas canvas,
    String text,
    Offset center,
    double fontSize,
    Color color, {
    bool bold = false,
    double? width,
  }) {
    final painter = TextPainter(
      text: TextSpan(
        text: text,
        style: TextStyle(
          color: color,
          fontFamily: fontFamily,
          fontSize: fontSize,
          fontWeight: bold ? FontWeight.w700 : FontWeight.w400,
        ),
      ),
      textDirection: TextDirection.ltr,
      textAlign: TextAlign.center,
      maxLines: 2,
      ellipsis: '…',
    )..layout(maxWidth: width ?? math.max(0, layout.size.width - 24));
    painter.paint(
      canvas,
      Offset(center.dx - painter.width / 2, center.dy - painter.height / 2),
    );
    painter.dispose();
  }

  void _ring(
    Canvas canvas,
    Offset center,
    double radius,
    Color color,
    double width,
  ) {
    canvas.drawCircle(
      center,
      radius,
      Paint()
        ..color = color
        ..style = PaintingStyle.stroke
        ..strokeWidth = width,
    );
  }

  @override
  void paint(Canvas canvas, Size size) {
    final round = session.round;
    final s = layout.scale, cx = size.width / 2, mid = layout.middle;
    for (double x = 20; x < size.width; x += 28) {
      for (double y = 155 * s; y < size.height - 115 * s; y += 28) {
        canvas.drawCircle(
          Offset(x, y),
          .8,
          Paint()..color = const Color(0xFF18202C),
        );
      }
    }
    _text(canvas, 'CHOUWAZI', Offset(cx, 21 * s), 10 * s, _muted, bold: true);
    _text(
      canvas,
      strings.text('app_name'),
      Offset(cx, 62 * s),
      28 * s,
      const Color(0xFFFFFFFF),
      bold: true,
    );
    _text(canvas, strings.text('tagline'), Offset(cx, 96 * s), 12 * s, _muted);
    final winner = round.winner;
    _text(
      canvas,
      winner == null
          ? strings.joined(round.players.length)
          : strings.winner(winner.slot + 1),
      Offset(cx, 137 * s),
      15 * s,
      winner == null ? _colors[0] : _colors[winner.slot],
      bold: true,
    );
    _text(
      canvas,
      strings.text(session.haptics ? 'haptics_on' : 'haptics_off'),
      layout.vibration.center,
      11,
      _muted,
      width: layout.vibration.width,
    );

    if (!round.isFinished && round.players.length < 2) {
      _ring(canvas, Offset(cx, mid), 48 * s, const Color(0xFF283C38), 2);
      _ring(canvas, Offset(cx, mid), 35 * s, const Color(0xFF536D49), 2);
      _text(canvas, '+', Offset(cx, mid), 36 * s, _colors[0]);
      _text(
        canvas,
        strings.text(round.players.isEmpty ? 'place' : 'one_more'),
        Offset(cx, mid + 76 * s),
        20 * s,
        const Color(0xFFFFFFFF),
        bold: true,
      );
      _text(
        canvas,
        strings.text('range'),
        Offset(cx, mid + 107 * s),
        13 * s,
        _muted,
      );
    }
    if (round.deadline != null) {
      final remaining = (round.deadline! - now).inMicroseconds / 1000000;
      _text(
        canvas,
        remaining > 3
            ? strings.text('ready')
            : '${math.max(1, remaining.ceil())}',
        Offset(cx, mid),
        (remaining > 3 ? 40 : 76) * s,
        const Color(0xFFE7EDE5),
        bold: true,
      );
      _text(
        canvas,
        strings.text('hold'),
        Offset(cx, mid + 56 * s),
        13 * s,
        _muted,
      );
    }
    for (final player in round.players) {
      final center = Offset(player.x, player.y),
          selected = round.winnerId == player.id;
      final dim = round.isFinished && !selected, color = _colors[player.slot];
      final pulse = reduceMotion
          ? 0.0
          : math.sin(now.inMicroseconds / 1000000 * 4.3 + player.slot) * 2;
      final radius = ((selected ? 49 : 36) + pulse) * s;
      canvas.drawCircle(
        center,
        radius + 12 * s,
        Paint()..color = color.withValues(alpha: dim ? .05 : .14),
      );
      _ring(
        canvas,
        center,
        radius,
        dim ? const Color(0xFF384050) : color,
        selected ? 5 : 3,
      );
      canvas.drawCircle(
        center,
        23 * s,
        Paint()..color = dim ? const Color(0xFF202735) : color,
      );
      _text(
        canvas,
        '${player.slot + 1}',
        center,
        17 * s,
        _background,
        bold: true,
      );
      if (selected) {
        _text(
          canvas,
          strings.text('you'),
          Offset(center.dx, math.max(159 * s, center.dy - radius - 26 * s)),
          19 * s,
          color,
          bold: true,
        );
        for (var i = 0; i < 12; i++) {
          final angle = i * math.pi / 6;
          canvas.drawCircle(
            center +
                Offset(math.cos(angle), math.sin(angle)) * (radius + 22 * s),
            2,
            Paint()..color = color,
          );
        }
      }
    }
    if (round.isFinished) {
      canvas.drawRRect(
        RRect.fromRectAndRadius(layout.retry, Radius.circular(18 * s)),
        Paint()
          ..color = session.canRetry ? _colors[0] : const Color(0xFF344336),
      );
      _text(
        canvas,
        strings.text(session.canRetry ? 'again' : 'release'),
        layout.retry.center,
        16 * s,
        session.canRetry ? const Color(0xFF152010) : const Color(0xFFA1B394),
        bold: true,
        width: layout.retry.width - 16,
      );
    } else {
      _text(
        canvas,
        strings.text('restart_hint'),
        Offset(cx, size.height - 66 * s),
        12 * s,
        _muted,
      );
    }
    _text(
      canvas,
      strings.text('footer'),
      Offset(cx, size.height - 18 * s),
      10 * s,
      const Color(0xFF687488),
    );
  }

  @override
  bool shouldRepaint(covariant _GamePainter oldDelegate) => true;
}
