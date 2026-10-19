# 构建

Android 与 iOS 的入口均为根目录 `lib/main.dart`。状态机、触控、绘制、本地化与测试共用；平台目录仅承载 Flutter 引擎和系统配置。两端构建工作流提供试用产物；iOS 商店签名与上传使用独立的 [TestFlight 流程](TESTFLIGHT.md)。

## 工具链

Python 构建辅助脚本经 uv 按各自的 PEP 723 声明和 `*.py.lock` 运行。
本机先准备 uv；CI 安装固定版本 uv。调用使用 `uv run --locked --script`，
脚本依赖声明更新后用 `uv lock --script <脚本>` 更新并提交对应锁文件。

| 平台 | 固定工具 | 说明 |
| --- | --- | --- |
| Android | Google Flutter 3.27.4 / Dart 3.6.2、Java 17、SDK 35、build-tools 35.0.0、NDK 26.3.11579264 | Gradle 8.9、AGP 8.7.3；最低 API 23 |
| iOS | Google Flutter 3.27.4 / Dart 3.6.2、macOS、Xcode 26.3 | 最低 iOS 15；工程 Runner、产物 Runner.app；已通过真实 CI 编译 |

Android 与 iOS 都使用 Flutter 官方工具链，编译同一份 `lib/`。升级 Flutter 时同时验证共享依赖、测试和两端编译。鸿蒙端已于 2026-10-20 放弃，取舍见 [ADR 0001](adr/0001-drop-harmonyos.md)。

工具链来源：

- [Google Flutter 3.27.4](https://github.com/flutter/flutter/tree/3.27.4)。本机 Flutter 由 mise 管理：仓库根的 `mise.toml` 与 `mise.lock` 把它钉到同一版本，在仓库内运行 `flutter` 即用该版本，不修改全局默认版本；升级时两处与本表一起改。CI 使用 `subosito/flutter-action` 安装固定版本。

## 共享验证

```sh
flutter pub get --enforce-lockfile
dart format --output=none --set-exit-if-changed lib test
flutter analyze --no-pub
flutter test --no-pub
```

依赖锁文件由 Dart 3.6.2 生成，包源为 pub.dev。应用仅依赖 Flutter SDK，测试与 lint 依赖不进入应用运行包。工具链下载与依赖安装会联网；应用运行完全离线。

测试覆盖倒计时边界、增减/移动、槽位复用、十指与重复 ID、每个随机索引、结果冻结、超限及结果后物理触点、按钮手势隔离、取消、应用失活、语言选择和真实 Flutter 多指事件路由。

## Android

```sh
flutter build apk --release --no-pub
uv run --locked --script scripts/check-apk.py "$ANDROID_HOME/build-tools/35.0.0/aapt" build/app/outputs/flutter-apk/app-release.apk
"$ANDROID_HOME/build-tools/35.0.0/apksigner" verify --verbose build/app/outputs/flutter-apk/app-release.apk
```

输出 `build/app/outputs/flutter-apk/app-release.apk`。包名为 `com.alex.fingerdraw`，Release 优化但使用调试签名供试用。持续分发或覆盖旧包须由用户提供匹配的稳定签名；不能把 CI 的临时调试签名当作原 APK 的签名。

主清单只申请震动系统权限，debug/profile 也不追加网络权限。AndroidX 合并进应用内部的 `com.alex.fingerdraw.DYNAMIC_RECEIVER_NOT_EXPORTED_PERMISSION`，用于限制广播接收器的调用方，构建检查要求它保持 signature 级；不授予联网或个人数据访问。网络调试服务不作为默认工作流。遇到手机不支持十指或系统三指手势拦截，按设备能力记录。

## iOS

```sh
export DEVELOPER_DIR=/Applications/Xcode_26.3.app/Contents/Developer
xcodebuild -version
xcrun --sdk iphoneos --show-sdk-version
flutter precache --ios
(cd ios && pod install)
flutter build ios --simulator --debug --no-codesign --no-pub
flutter build ios --release --no-codesign --no-pub
uv run --locked --script scripts/check-ios.py build/ios/iphonesimulator/Runner.app build/ios/iphoneos/Runner.app
```

必须在 macOS 上执行。使用 Xcode 运行时打开 `ios/Runner.xcworkspace`，选择 Runner scheme。CI 分别打包 `chouwazi-simulator.zip` 与 `chouwazi-device-unsigned.zip`；两者都含 `Runner.app`，只能在匹配的平台使用。真机包未签名，不是可直接安装的 IPA。签名与验收见 [iOS 真机试用指南](../ios/DEVICE-TESTING.md)。

CI 明确选择 Xcode 26.3 并打印实际 Xcode 和 iPhoneOS SDK 版本，Flutter 固定为 3.27.4。2026-10-08 的真实 CI 已编译模拟器与未签名真机包；下载读回真机包为 1.0.2 (3)，`DTXcode=2630`、`DTSDKName=iphoneos26.2`、最低 iOS 15，应用与引擎隐私清单均存在。签名上传须再由 TestFlight 流程验证。

Apple 自 2026-04-28 起要求上传包使用 Xcode 26 / iOS 26 SDK 或更新版本，见[官方要求](https://developer.apple.com/news/upcoming-requirements/)。签名、IPA 校验与商店上传见 [TestFlight 上传指南](TESTFLIGHT.md)；上传成功须继续等待 Apple 处理并核对测试分发状态。

Runner 保持 Flutter 标准构建 scheme，CLI 通过 `--debug` 或 `--release` 选择模式。Xcode 真机试用时在 Edit Scheme → Run → Build Configuration 选择 Release。CI 的 Debug 模拟器包仅供开发；Debug/Profile 调试时 Flutter 可能请求本地网络访问以支持 DevTools，不属于 Release 应用功能。CI 检查真机 Release 的 Info.plist 不包含这类声明。

## CI 产物

`Build Android and iOS` 仅 `workflow_dispatch`，权限为 `contents: read`。Android 与 iOS 都在 Google Flutter 下构建，Android 任务同时执行共享测试。

| Artifact | 内容 |
| --- | --- |
| `chouwazi-android-apk` | `app-release.apk`，调试签名试用包 |
| `chouwazi-ios-builds` | 模拟器 zip、未签名真机 zip |

产物受 Actions 保留期限限制。每轮交付以 Actions 的源码 SHA、任务结论和实际下载文件为准；根 README 维护当前验证状态。
