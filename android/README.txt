抽袜子 / chouwazi — Android 平台壳

玩法、界面、触控和本地化由根目录 lib/ 的 Dart 源码提供，Android 目录仅保留 FlutterActivity、资源、权限与 Gradle 配置。

在仓库根目录使用 Flutter 3.47.7：
  flutter pub get --enforce-lockfile
  flutter test
  flutter build apk --release

输出：build/app/outputs/flutter-apk/app-release.apk。
包名：com.alex.fingerdraw；最低 Android 6.0 / API 23。
试用构建采用调试签名，非应用商店发布签名。与旧本地签名包不同，不能保证覆盖安装；版本来自根 pubspec.yaml。
完全离线，系统权限仅震动；AndroidX 另声明 signature 级应用内部广播保护权限，构建会校验保护等级。实际触点数取决于设备与系统手势。

自动构建、版本与验证证据见根 README.md 和 docs/BUILDING.md。旧 build-local.sh 与 Java 游戏逻辑已由共享构建替代。
