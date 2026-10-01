import 'dart:ui';

class GameStrings {
  GameStrings(Locale locale)
      : _language = locale.languageCode != 'zh'
            ? 'en'
            : locale.scriptCode == 'Hant' ||
                    ['TW', 'HK', 'MO'].contains(locale.countryCode)
                ? 'zh-Hant'
                : 'zh-Hans';

  final String _language;
  String text(String key) => _translations[_language]![key]!;
  String winner(int number) => text('winner').replaceFirst('%d', '$number');
  String joined(int count) => text('joined').replaceFirst('%d', '$count');

  static const _translations = {
    'en': {
      'app_name': 'chouwazi',
      'tagline': 'Let this moment decide',
      'again': 'Another round',
      'release': 'Lift all fingers to continue',
      'haptics_on': 'Haptics on',
      'haptics_off': 'Haptics off',
      'winner': 'No. %d · You are chosen!',
      'joined': '%d / 10 fingers joined',
      'place': 'Place your fingers together',
      'one_more': 'One more finger',
      'range': 'At least 2 · Up to 10',
      'ready': 'Ready',
      'hold': 'Keep holding. Choosing one of you…',
      'you': 'It’s you!',
      'restart_hint': 'Joining or leaving restarts the countdown',
      'footer': 'Offline draw · Touch limit depends on your device',
    },
    'zh-Hans': {
      'app_name': '抽袜子',
      'tagline': '把选择，交给这一刻',
      'again': '再来一轮',
      'release': '松开所有手指后再来一轮',
      'haptics_on': '震动 开',
      'haptics_off': '震动 关',
      'winner': '第 %d 号 · 就是你！',
      'joined': '%d / 10 根手指已加入',
      'place': '大家一起按住屏幕',
      'one_more': '再来一根手指',
      'range': '至少 2 根 · 最多 10 根',
      'ready': '准备',
      'hold': '保持按住，即将随机选出一位',
      'you': '就是你！',
      'restart_hint': '新增或移开手指，会重新倒计时',
      'footer': '离线随机抽选 · 实际触点数取决于设备',
    },
    'zh-Hant': {
      'app_name': '抽袜子',
      'tagline': '把選擇，交給這一刻',
      'again': '再來一輪',
      'release': '鬆開所有手指後再來一輪',
      'haptics_on': '震動 開',
      'haptics_off': '震動 關',
      'winner': '第 %d 號 · 就是你！',
      'joined': '%d / 10 根手指已加入',
      'place': '大家一起按住螢幕',
      'one_more': '再來一根手指',
      'range': '至少 2 根 · 最多 10 根',
      'ready': '準備',
      'hold': '保持按住，即將隨機選出一位',
      'you': '就是你！',
      'restart_hint': '新增或移開手指，會重新倒數',
      'footer': '離線隨機抽選 · 實際觸點數取決於裝置',
    },
  };
}
