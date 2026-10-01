# AGENTS.md

「抽袜子 / chouwazi」通过一套 Flutter/Dart 游戏源码编译 Android、iOS 和原生鸿蒙 HAP。使用与验证状态见 [README.md](README.md)，工具链和产物见 [docs/BUILDING.md](docs/BUILDING.md)。

## 共享实现

- 游戏规则只改 `lib/draw_round.dart`，物理触点、按钮与生命周期规则只改 `lib/game_session.dart`，并更新 `test/`。平台壳不复制游戏状态机或界面。
- 上限 10 指；触点用稳定 pointer ID；增减参与者重置唯一的 3.8 秒截止时间（0.8 秒准备 + 3 秒倒计时），移动不重置；结果冻结；空缺槽位复用；随机索引等概率。
- 倒计时源是单调 `Stopwatch`。所有物理触点（含超限和结果后的新增触点）松开才允许重开；控制按钮手势不参与抽签。系统取消触控或失活清理未完成轮次，保留已产生结果。
- 简体、繁体、英文界面统一在 `lib/strings.dart`。桌面应用名属于各端原生资源，改名时同步三端资源。

## 构建与验证

- Android / iOS 使用 Flutter 3.27.4；鸿蒙使用固定提交的 Flutter OH 3.27.5-ohos-1.0.0 与 SDK 5.1.0.840。升级须复验三端，不把 Google Flutter 的 APK 支持当作鸿蒙 HAP 支持。
- 在根目录执行 `flutter analyze`、`flutter test`。鸿蒙自动构建入口为 `bash scripts/build-ohos.sh`，工具链落在 gitignored `.tooling/`，不改变全局 SDK。
- iOS 工程为 `ios/Runner.xcworkspace` / `Runner` scheme。新增原生 iOS 资源需登记 `project.pbxproj`；共享 Dart 文件不需要逐个登记。
- GitHub Actions `build.yml` 仅手动触发，执行共享测试并真实构建三端。本机 Linux 可构建 APK/HAP，iOS 必须在 macOS/Xcode 构建。
- 多指触控、显示和震动仍须真机确认；验证结论变化时更新 README「验证状态」，保留编译、安装与真机体验的区别。

## 平台约束

- Android applicationId 保持 `com.alex.fingerdraw`；iOS 与鸿蒙为 `com.alex.chouwazi`。包名相同也不代表不同签名可以覆盖安装。
- 版本统一来自 `pubspec.yaml`，鸿蒙 `ohos/AppScope/app.json5` 同步，脚本会拒绝版本不一致。
- 离线承诺适用于最终产物：Android 和鸿蒙系统权限仅震动；AndroidX 的应用内部广播权限须保持 signature 级；iOS 保留应用与引擎隐私清单。构建脚本校验真实 APK/HAP 权限，新增权限或联网能力时同步隐私说明。
- 签名材料与本机路径不入库。Android 当前 Release 是调试签名试用包；iOS/HAP 的 CI 产物未签名。发布和设备签名沿用户授权处理。
