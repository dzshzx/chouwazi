import 'dart:async';

import 'package:flutter/services.dart';
import 'package:flutter/widgets.dart';

import 'game_screen.dart';

Future<void> main() async {
  WidgetsFlutterBinding.ensureInitialized();
  await SystemChrome.setPreferredOrientations([DeviceOrientation.portraitUp]);
  SystemChrome.setSystemUIOverlayStyle(
    const SystemUiOverlayStyle(
      statusBarColor: Color(0x00000000),
      statusBarIconBrightness: Brightness.light,
      statusBarBrightness: Brightness.dark,
      systemNavigationBarColor: Color(0xFF0D111B),
      systemNavigationBarIconBrightness: Brightness.light,
    ),
  );
  runApp(const ChouwaziApp());
}

class ChouwaziApp extends StatelessWidget {
  const ChouwaziApp({super.key});

  @override
  Widget build(BuildContext context) => WidgetsApp(
        color: const Color(0xFF0D111B),
        debugShowCheckedModeBanner: false,
        title: 'chouwazi',
        supportedLocales: const [
          Locale('en'),
          Locale('zh', 'CN'),
          Locale('zh', 'TW'),
        ],
        builder: (context, child) => const GameScreen(),
      );
}
