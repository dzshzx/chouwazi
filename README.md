# 抽袜子 · chouwazi

Android 和 iOS 原生多人手指抽签小游戏。中文系统应用名称为「抽袜子」，英文为「chouwazi」。本项目拟发布到私有 GitHub 仓库。

## 玩法

2～10 根手指同时按住屏幕，经过 0.8 秒准备和 3 秒倒计时，从参与触点中等概率选中一根。加入或移开手指会重置倒计时，移动手指不会重置。结果保留，全部松开后点击「再来一轮」。震动可关闭。

上限是应用逻辑上限；手机、平板实际可报告的同时触点数量由硬件和系统决定，不能保证所有设备支持十指。系统多指手势可能中断游戏。

## 源码

| 目录 | 内容 |
| --- | --- |
| `android/` | Java 原生 Activity、Canvas、状态模型及独立测试 |
| `ios/Chouwazi.xcodeproj` | 可直接打开的 Xcode 工程和共享 Chouwazi scheme |
| `ios/Chouwazi/` | UIKit、多点触控、中文和英文资源、图标 |
| `ios/Sources/DrawCore/` | 不依赖 UIKit 的抽签状态模型 |
| `ios/Tests/DrawCoreTests/` | Swift Package 状态模型测试 |
| `.github/workflows/build.yml` | 手动触发的 Android 和 iOS 构建 |

## Android

Android 6.0+，Java 17、Android SDK 35、Gradle 8.9、Android Gradle Plugin 8.7.3。

Android Studio 打开 `android/` 并同步。工程未包含 Gradle wrapper，命令行需安装 Gradle 8.9：

```sh
cd android
gradle :app:assembleDebug
```

生成文件：`app/build/outputs/apk/debug/app-debug.apk`。
另有 `build-local.sh` 支持直接使用官方 SDK build-tools 构建，见 `android/README.txt`。

## iOS

iOS / iPadOS 15.0+，建议 Xcode 16 或更新版本。

1. 打开 `ios/Chouwazi.xcodeproj`。
2. 选择 `Chouwazi` target，在 Signing & Capabilities 中选择自己的 Apple 开发团队。
3. 根据账号情况将 Bundle Identifier 修改为可用的唯一标识。
4. 选择 iPhone/iPad 模拟器或已配置签名的真机并运行。

```sh
cd ios
swift test
xcodebuild -project Chouwazi.xcodeproj -scheme Chouwazi \
  -sdk iphonesimulator -destination 'generic/platform=iOS Simulator' \
  CODE_SIGNING_ALLOWED=NO build
```

默认 Bundle Identifier 为 `com.alex.chouwazi`。UI 使用 UIKit，系统正常处理安全区域；没有网络依赖和第三方运行库。多点触控基于稳定的 UITouch 身份跟踪，触控取消或应用失活时清理未完成状态。

## GitHub Actions

进入仓库 Actions → Build Android and iOS → Run workflow，构建过程会执行两端的核心状态测试，输出 Android debug APK、iOS 模拟器 app 和未签名真机 app 压缩包。

工作流仅手动触发，私有仓库执行可能占用账号 Actions 额度。iOS 未签名产物不能直接安装到 iPhone；真机分发需要 Apple 签名、描述文件或 TestFlight。项目不包含证书、令牌或签名私钥。

原对话中生成的 Android APK 使用独立的本地签名；CI 调试包的签名与它不同，通常不能直接覆盖旧安装。持续分发需自行配置稳定签名。

## 验证状态

- Android：前一版原生源码已编译，APK 签名校验通过；状态模型测试已执行通过。
- iOS：已提供完整源码、Xcode 工程、状态模型测试及构建工作流；创建环境为 Linux，没有 Xcode 或 Swift 工具链，尚未完成 Swift 编译、模拟器或真机验证。
- 双端十指触控与实际显示效果仍需设备测试。

## 隐私

完全离线，不收集个人信息，不上传数据，不含广告。Android 只申请震动权限；iOS 无需网络、相册、摄像头、位置等权限。系统运行时间只用于应用内倒计时，隐私清单已声明该用途。
