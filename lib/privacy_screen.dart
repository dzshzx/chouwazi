import 'package:flutter/widgets.dart';

import 'strings.dart';

class PrivacyScreen extends StatelessWidget {
  const PrivacyScreen({super.key, required this.strings, required this.onBack});

  final GameStrings strings;
  final VoidCallback onBack;

  @override
  Widget build(BuildContext context) => ColoredBox(
        color: const Color(0xFF0D111B),
        child: SafeArea(
          child: Center(
            child: ConstrainedBox(
              constraints: const BoxConstraints(maxWidth: 560),
              child: Column(
                crossAxisAlignment: CrossAxisAlignment.stretch,
                children: [
                  Align(
                    alignment: Alignment.centerLeft,
                    child: Semantics(
                      button: true,
                      child: GestureDetector(
                        key: const Key('privacy-back'),
                        behavior: HitTestBehavior.opaque,
                        onTap: onBack,
                        child: Padding(
                          padding: const EdgeInsets.symmetric(
                            horizontal: 24,
                            vertical: 16,
                          ),
                          child: Text(
                            strings.text('back'),
                            style: const TextStyle(
                              fontSize: 15,
                              color: Color(0xFF8B96AA),
                            ),
                          ),
                        ),
                      ),
                    ),
                  ),
                  Expanded(
                    child: SingleChildScrollView(
                      key: const Key('privacy-scroll'),
                      padding: const EdgeInsets.fromLTRB(24, 8, 24, 32),
                      child: Column(
                        crossAxisAlignment: CrossAxisAlignment.start,
                        children: [
                          Semantics(
                            header: true,
                            namesRoute: true,
                            child: Text(
                              strings.text('privacy_title'),
                              style: const TextStyle(
                                fontSize: 28,
                                height: 1.3,
                                fontWeight: FontWeight.w700,
                                color: Color(0xFFFFFFFF),
                              ),
                            ),
                          ),
                          const SizedBox(height: 20),
                          _paragraph('privacy_summary'),
                          for (final section in [
                            'data',
                            'network',
                            'system',
                            'contact',
                          ]) ...[
                            const SizedBox(height: 28),
                            Semantics(
                              header: true,
                              child: Text(
                                strings.text('privacy_${section}_title'),
                                style: const TextStyle(
                                  fontSize: 18,
                                  height: 1.4,
                                  fontWeight: FontWeight.w700,
                                  color: Color(0xFFFFFFFF),
                                ),
                              ),
                            ),
                            const SizedBox(height: 8),
                            _paragraph('privacy_$section'),
                          ],
                        ],
                      ),
                    ),
                  ),
                ],
              ),
            ),
          ),
        ),
      );

  Widget _paragraph(String key) => Text(
        strings.text(key),
        style: const TextStyle(
          fontSize: 16,
          height: 1.6,
          color: Color(0xFFBAC3D3),
        ),
      );
}
