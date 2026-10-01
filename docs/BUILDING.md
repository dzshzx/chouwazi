# 三端构建

三端入口均为根目录 `lib/main.dart`。状态机、触控、绘制、本地化与测试共用；平台目录仅承载 Flutter 引擎和系统配置。当前是迁移验证与试用构建，不是应用商店发布。

## 工具链

| 平台 | 固定工具 | 说明 |
| --- | --- | --- |
| Android | Google Flutter 3.27.4 / Dart 3.6.2、Java 17、SDK 35、build-tools 35.0.0、NDK 26.3.11579264 | Gradle 8.9、AGP 8.7.3；最低 API 23 |
| iOS | Google Flutter 3.27.4 / Dart 3.6.2、macOS、Xcode 16.4 | 最低 iOS 15；工程 Runner、产物 Runner.app |
| HarmonyOS | Flutter OH 3.27.5-ohos-1.0.0、Java 17、华为命令行工具 5.1.0.840 / SDK API 18 | 最低 API 12；输出原生 arm64 HAP |

Android/iOS 使用 Flutter 官方工具链；鸿蒙使用 CPF-Flutter 的适配工具链，两者编译同一份 `lib/`。这一基线以可公开获取并真实构建的配套版本为准。升级 Flutter 或鸿蒙 SDK 时同时验证共享依赖、测试和三端编译。

工具链来源：

- [Google Flutter 3.27.4](https://github.com/flutter/flutter/tree/3.27.4)。本机 Flutter 由 mise 管理；使用与项目一致的版本，不修改全局默认版本。CI 使用 `subosito/flutter-action` 安装固定版本。
- [Flutter OH 3.27.5-ohos-1.0.0](https://gitcode.com/CPF-Flutter/flutter_flutter/tree/3.27.5-ohos-1.0.0)，提交 `c8a0f037e39856cdac07bfa31d59d3ea78549b1b`。
- [华为 HarmonyOS 命令行工具 5.1.0.840](https://repo.huaweicloud.com/harmonyos/ohpm/5.1.0/)，下载 zip 的 SHA-256 为 `1dcefa4741dc289277560b2802040e0c45b433898d8db453d711d5376e0478f1`，来自同目录官方 `.sha256` 文件。

鸿蒙适配 SDK 是项目构建依赖，官方 Flutter SDK 不提供 `hap` 目标，故由项目脚本在 `.tooling/` 下载固定上游源码并验证提交。SDK zip 验证官方校验值后解压，不修改用户全局 Flutter、Node 或 Java。需要 Python 3.11+、Git、unzip、Java 17；自动配置当前支持 Linux x86_64。工具包下载约 2 GB，解压后约 6.5 GB，另需 Flutter 引擎和构建缓存空间。

`CHOUWAZI_TOOLCHAINS=/绝对目录` 可以指定独立缓存位置。macOS/Windows 的鸿蒙开发可安装相同配套 DevEco SDK 与 Flutter OH，参考[上游环境说明](https://gitcode.com/CPF-Flutter/flutter_flutter/tree/3.27.5-ohos-1.0.0)；自动 SDK 下载脚本不声称支持这些宿主。

## 共享验证

```sh
flutter pub get --enforce-lockfile
dart format --output=none --set-exit-if-changed lib test
flutter analyze --no-pub
flutter test --no-pub
```

依赖锁文件由 Dart 3.6.2 生成，包源为 pub.dev。应用仅依赖 Flutter SDK，测试与 lint 依赖不进入游戏运行包。工具链下载与依赖安装会联网；应用运行完全离线。

测试覆盖倒计时边界、增减/移动、槽位复用、十指与重复 ID、每个随机索引、结果冻结、超限及结果后物理触点、按钮手势隔离、取消、应用失活、语言选择和真实 Flutter 多指事件路由。

## Android

```sh
flutter build apk --release --no-pub
python3 scripts/check-apk.py "$ANDROID_HOME/build-tools/35.0.0/aapt" build/app/outputs/flutter-apk/app-release.apk
"$ANDROID_HOME/build-tools/35.0.0/apksigner" verify --verbose build/app/outputs/flutter-apk/app-release.apk
```

输出 `build/app/outputs/flutter-apk/app-release.apk`。包名仍为 `com.alex.fingerdraw`，Release 优化但使用调试签名供试用。持续分发或覆盖旧包须由用户提供匹配的稳定签名；不能把 CI 的临时调试签名当作原 APK 的签名。

主清单只申请震动系统权限，debug/profile 也不追加网络权限。AndroidX 合并进应用内部的 `com.alex.fingerdraw.DYNAMIC_RECEIVER_NOT_EXPORTED_PERMISSION`，用于限制广播接收器的调用方，构建检查要求它保持 signature 级；不授予联网或个人数据访问。网络调试服务不作为默认工作流。遇到手机不支持十指或系统三指手势拦截，按设备能力记录。

## iOS

```sh
flutter precache --ios
(cd ios && pod install)
flutter build ios --simulator --debug --no-codesign --no-pub
flutter build ios --release --no-codesign --no-pub
python3 scripts/check-ios.py build/ios/iphonesimulator/Runner.app build/ios/iphoneos/Runner.app
```

必须在 macOS 上执行。使用 Xcode 运行时打开 `ios/Runner.xcworkspace`，选择 Runner scheme。CI 分别打包 `chouwazi-simulator.zip` 与 `chouwazi-device-unsigned.zip`；两者都含 `Runner.app`，只能在匹配的平台使用。真机包未签名，不是可直接安装的 IPA。签名与验收见 [iOS 真机试用指南](../ios/DEVICE-TESTING.md)。

Runner scheme 默认以 Release 运行，避免真机试用启动 Flutter 的本地网络调试服务。CI 的 Debug 模拟器包仅供开发；手工改用 Debug/Profile 调试时，Flutter 可能请求本地网络访问以支持 DevTools，不属于 Release 游戏功能。CI 检查真机 Release 的 Info.plist 不包含这类声明。

## 原生鸿蒙

```sh
bash scripts/build-ohos.sh
```

该命令自动配置并校验固定工具链，下载 Flutter OH 引擎、解析依赖，再通过该版本官方 Hvigor 插件真实编译 Dart、ArkTS 和原生 HAP。旧版 `flutter build hap` 在编译后强制检查签名，因此无签名构建直接调用同一 Hvigor 插件；所有编译错误保持失败退出，不忽略失败、不伪造签名。

输出 `build/ohos/chouwazi-unsigned.hap`。脚本校验包内 `module.json`、`pack.info`、`libapp.so` 和 `libflutter.so`，并拒绝震动以外的权限。版本在 `pubspec.yaml` 与 `ohos/AppScope/app.json5` 中须一致，不一致会停止构建。

安装需要在 DevEco Studio 中打开 `ohos/`，使用自己的开发者账号和设备配置签名。完成签名后再通过该版本 Flutter OH 的 `flutter build hap --release` / `flutter run` 或 DevEco 运行；不要把证书、密码、私钥、描述文件或含个人签名的配置提交进仓库。

鸿蒙验收沿用 [iOS 指南中的游戏验收清单](../ios/DEVICE-TESTING.md#真机验收清单)，另记录 HarmonyOS 版本、设备型号、HAP 签名与安装结果。未签名编译通过不代表商用鸿蒙真机已验证。

## CI 产物

`Build Android, iOS and HarmonyOS` 仅 `workflow_dispatch`，权限为 `contents: read`。Android/iOS 在 Google Flutter 下构建；鸿蒙在固定 Flutter OH 下构建并复跑共享测试，防止源码只兼容其中一个 SDK。

| Artifact | 内容 |
| --- | --- |
| `chouwazi-android-apk` | `app-release.apk`，调试签名试用包 |
| `chouwazi-ios-builds` | 模拟器 zip、未签名真机 zip |
| `chouwazi-harmony-hap` | `chouwazi-unsigned.hap` |

产物受 Actions 保留期限限制。每轮交付以 Actions 的源码 SHA、任务结论和实际下载文件为准；根 README 维护当前验证状态。
