# 抽袜子 · chouwazi

Android 与 iOS 共用一套 Flutter/Dart 源码的离线随机选择工具。多人同时触屏，随机选出一位；不含广告或账号系统。

## 使用方式

2～10 根手指按住屏幕，经过 0.8 秒准备和 3 秒倒计时，从参与触点中等概率选中一根。新增或移开参与手指重置倒计时，移动不重置。结果冻结，全部松开后点击「再来一轮」。震动可关闭。

十指是软件上限，实际触点数量取决于设备与系统手势。系统取消触控或应用失活时清理未完成轮次；已产生的结果保留。触控取消后仍按住的手指须先松开，才能重新参与。

## 一套源码

| 路径 | 职责 |
| --- | --- |
| `lib/draw_round.dart` | 两端共用的抽签规则、槽位、单调时钟截止时间及随机选取 |
| `lib/game_session.dart` | 两端共用的物理触点跟踪、按钮手势、取消与结果重开 |
| `lib/game_screen.dart` | 两端共用的多指事件、画面、倒计时反馈与生命周期处理 |
| `lib/privacy_screen.dart` | App 内可离线访问的隐私说明 |
| `lib/strings.dart` | 简体中文、繁体中文、英文界面文字 |
| `test/` | 共享状态模型和 Flutter 多指交互测试 |
| `android/`、`ios/` | 平台启动壳、包名、图标、权限与签名配置 |

应用基于 Flutter 引擎，不依赖额外业务插件或在线服务。平台启动代码仅处理引擎启动和屏幕常亮，抽选行为由共享 Dart 实现统一处理。

## 构建

完整安装、工具链固定版本与来源、校验和各端构建命令见 [构建指南](docs/BUILDING.md)。

- Android / iOS：Flutter 官方工具链。
- 最低系统：Android 6.0 / API 23，iOS 15。
- 版本源为 `pubspec.yaml`。

Android / iOS 在根目录运行：

```sh
flutter pub get --enforce-lockfile
flutter analyze
flutter test
flutter build apk --release
# 以下 iOS 命令需要 macOS 和 Xcode
flutter precache --ios
(cd ios && pod install)
flutter build ios --simulator --debug --no-codesign
flutter build ios --release --no-codesign
```

进入 GitHub Actions → **Build Android and iOS** → Run workflow，一次构建两端。工作流仅手动触发，私有仓库会消耗 Actions 额度。产物为试用 APK、iOS 模拟器与未签名真机 app，不执行应用商店发布。

TestFlight 使用独立的 **Upload iOS to TestFlight** 手动工作流，从 `main` 签名并上传。账号、签名材料和受保护环境的配置见 [TestFlight 上传指南](docs/TESTFLIGHT.md)。上传之后还须确认 Apple 处理完成并分配测试组。

正式 App Store 的审核状态、三语商店材料、[支持与隐私网站](https://chouwazi.646598.xyz/)及地区要求见 [发布与审核](docs/APP-STORE.md)。

Android 包名保持 `com.alex.fingerdraw`，iOS 为 `com.alex.chouwazi`。Android Release 构建暂用调试签名供试用，不能保证覆盖原本地签名或其他 CI 运行的安装包。iOS 未签名产物不能直接安装到普通真机，需自己的开发者签名；模拟器要求与真机不同，详见[构建指南](docs/BUILDING.md)。iOS 操作见 [真机试用指南](ios/DEVICE-TESTING.md)。仓库不保存签名私钥、证书或描述文件。

## 验证状态

- 2026-10-20 升级到 Flutter 3.47.7 后，本机 Linux 的共享测试、静态分析与 Release APK 构建及检查通过；[`build.yml` 两端构建](https://github.com/dzshzx/chouwazi/actions/runs/39290633740)已在升级提交 `dbe5d0e` 上通过；`testflight.yml` 与 `screenshots.yml` 尚未在新版本下运行，真机验证尚未在新版本下重做。以下 2026-10-08 起的构建、TestFlight、App Store 与真机记录都基于 Flutter 3.27.4。
- 2026-10-08，[同一提交的三端 Actions 构建](https://github.com/dzshzx/chouwazi/actions/runs/37801507597)全部成功，源码为 `3f85685`，应用版本为 `1.0.4 (5)`。
- 共享实现：25 项状态与多指交互测试及静态分析通过；已检查共享渲染器的界面与隐私页。iOS 商店截图的实际生成状态见发布与审核文档。
- Android：真实编译 Release APK，CI 通过签名及权限检查，保持原包名、最低 API 23 与目标 API 35；系统权限仅震动，内部广播权限保持 signature 级。
- iOS：macOS / Xcode 26.3、iOS 26.2 SDK 真实编译模拟器 Debug 与真机 Release；CI 回读版本、包名、最低 iOS 15、三语言桌面名、应用与 Flutter 引擎隐私清单及非豁免加密声明。真机包无本地网络调试声明。
- 鸿蒙：2026-10-20 起不再开发与发布，取舍见 [ADR 0001](docs/adr/0001-drop-harmonyos.md)；上面 2026-10-08 的三端构建是放弃前的记录。
- 真机：2026-10-09 已确认 iPhone 18 Pro / iOS 27.0.1 安装 1.0.4 (5)，录屏展示多指抽选、结果、再次抽选与隐私页；该机在应用和独立 Safari 触点测试中均报告最多 5 点。用户确认 Xiaomi 17 的 Android 版支持 10 点。完整设备验收、震动手感及其他机型覆盖见 [iOS 真机记录](ios/DEVICE-TESTING.md#2026-10-09-实测记录)。
- TestFlight：2026-10-08，[签名上传运行](https://github.com/dzshzx/chouwazi/actions/runs/37802819518)通过，源码 `3f85685`，版本 `1.0.4 (5)`，构建 ID `a22f5a2e-3f9d-43c7-8f42-1c9c39de3e38`。Apple 已处理为 `VALID`，内部测试状态为 `IN_BETA_TESTING`，构建未过期；「本人测试」组的账号本人已安装该构建。2026-10-09 已创建「公开测试」外部组及[公开邀请链接](https://testflight.apple.com/join/CRjBmUDp)，并提交同一构建的 Beta App Review；2026-10-14 匿名打开公开链接，页面已提供加入入口。
- App Store：1.0.4 (5) 已于 2026-10-11 22:46（Asia/Shanghai）获批并自动发布，免费。此前 2026-10-09 针对 Guideline 2.1 的信息要求补交了真机录屏和完整六项说明后重新提交。2026-10-14 用公开查询接口抽查 30 个地区（含中国大陆、港澳台、美国、日本和欧洲主要地区），均可查到 1.0.4；供应范围共 175 个国家和地区，其余未逐一核对。上架状态与备案见[发布与审核](docs/APP-STORE.md)。

## 隐私

完全离线，不收集个人信息、不上传数据、不含广告。Android 使用的系统权限仅震动，iOS 无需用户授权权限。AndroidX 另自动声明应用自身的 signature 级内部广播保护权限，限定同签名调用；它不授予联网或个人数据访问。构建检查同时验证其保护等级。应用使用单调时钟计时，iOS 同时打包应用隐私清单与 Flutter 引擎自身清单。

下载 SDK、构建依赖与开发者签名可能需要联网；已安装应用不需要网络。为保持权限承诺，Android 调试配置也未添加 INTERNET 权限，网络调试服务不作为本项目的默认能力。

iOS 真机试用使用 Release，Xcode 操作见试用指南；Debug 模拟器包仅供开发，Flutter 的 Debug/Profile 网络调试服务不属于发布版应用。
