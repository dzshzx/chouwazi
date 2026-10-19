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
      'privacy': 'Privacy',
      'privacy_title': 'Privacy Policy',
      'back': 'Back',
      'privacy_summary':
          'chouwazi is an offline random selection tool. It has no accounts, '
          'ads, in-app purchases, servers or analytics SDKs.',
      'privacy_data_title': 'Local processing',
      'privacy_data':
          'Touch points and the current selection result are processed only in '
          'your device’s memory for the current draw. The app does not save '
          'selection history or collect or store personal information.',
      'privacy_network_title': 'Network and uploads',
      'privacy_network':
          'The app does not initiate network connections or upload touch data '
          'or selection results.',
      'privacy_system_title': 'System and store diagnostics',
      'privacy_system':
          'Your operating system or app distribution platform may process '
          'diagnostic or usage data under its own settings and privacy '
          'policy. This is separate from the app’s local selection process.',
      'privacy_contact_title': 'Contact',
      'privacy_contact':
          'For privacy questions, use the developer support link on this app’s '
          'store page.',
    },
    'zh-Hans': {
      'app_name': '抽袜子',
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
      'privacy': '隐私',
      'privacy_title': '隐私政策',
      'back': '返回',
      'privacy_summary': '抽袜子是离线随机选择工具，无账号、广告、内购、服务器或分析 SDK。',
      'privacy_data_title': '本地处理',
      'privacy_data':
          '触点与当前抽选结果仅在设备内存中用于本次抽选。'
          '应用不保存抽选历史，也不收集或存储个人信息。',
      'privacy_network_title': '联网与上传',
      'privacy_network': '应用不主动发起网络连接，不上传触点数据或抽选结果。',
      'privacy_system_title': '系统与商店诊断',
      'privacy_system':
          '操作系统或应用分发平台可能按照各自的设置和隐私政策处理诊断或使用数据。'
          '这些处理与应用的本地抽选过程相互独立。',
      'privacy_contact_title': '联系',
      'privacy_contact': '如有隐私问题，请通过本应用商店页面的开发者支持入口联系。',
    },
    'zh-Hant': {
      'app_name': '抽袜子',
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
      'privacy': '隱私',
      'privacy_title': '隱私政策',
      'back': '返回',
      'privacy_summary': '抽袜子是離線隨機選擇工具，無帳號、廣告、App 內購買、伺服器或分析 SDK。',
      'privacy_data_title': '本機處理',
      'privacy_data':
          '觸點與目前抽選結果僅在裝置記憶體中用於本次抽選。'
          '應用程式不儲存抽選歷史，也不收集或儲存個人資訊。',
      'privacy_network_title': '連線與上傳',
      'privacy_network': '應用程式不主動發起網路連線，不上傳觸點資料或抽選結果。',
      'privacy_system_title': '系統與商店診斷',
      'privacy_system':
          '作業系統或應用程式分發平台可能依照各自的設定和隱私政策處理診斷或使用資料。'
          '這些處理與應用程式的本機抽選過程相互獨立。',
      'privacy_contact_title': '聯絡',
      'privacy_contact': '如有隱私問題，請透過本應用程式商店頁面的開發者支援入口聯絡。',
    },
  };
}
