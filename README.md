# 抽袜子 · chouwazi

Android、iOS 和原生 HarmonyOS 共用一套 Flutter/Dart 源码的多指抽签小游戏。完全离线，不含广告或账号系统。

## 玩法

2～10 根手指按住屏幕，经过 0.8 秒准备和 3 秒倒计时，从参与触点中等概率选中一根。新增或移开参与手指重置倒计时，移动不重置。结果冻结，全部松开后点击「再来一轮」。震动可关闭。

十指是软件上限，实际触点数量取决于设备与系统手势。系统取消触控或应用失活时清理未完成轮次；已产生的结果保留。触控取消后仍按住的手指须先松开，才能重新参与。

## 一套源码

| 路径 | 职责 |
| --- | --- |
| `lib/draw_round.dart` | 三端共用的抽签规则、槽位、单调时钟截止时间及随机选取 |
| `lib/game_session.dart` | 三端共用的物理触点跟踪、按钮手势、取消与结果重开 |
| `lib/game_screen.dart` | 三端共用的多指事件、画面、倒计时反馈与生命周期处理 |
| `lib/strings.dart` | 简体中文、繁体中文、英文界面文字 |
| `test/` | 共享状态模型和 Flutter 多指交互测试 |
| `android/`、`ios/`、`ohos/` | 平台启动壳、包名、图标、权限与签名配置 |

Java/Swift 的旧游戏逻辑已由共享实现替代，历史源码可从 Git 查询。现在引入 Flutter 引擎；应用不依赖额外业务插件或在线服务。平台启动代码仅处理引擎启动和屏幕常亮，不再各写一份游戏。

## 构建

完整安装、工具链来源与校验、各端构建命令见 [构建指南](docs/BUILDING.md)。

- Android / iOS：Flutter **3.27.4**，Dart 3.6.2。
- 鸿蒙：Flutter OH **3.27.5-ohos-1.0.0**（固定提交），HarmonyOS 命令行工具 **5.1.0.840 / API 18**。这是经实际编译验证、能从公开源自动配置的组合，不跟随浮动分支。
- 最低系统：Android 6.0 / API 23，iOS 15，HarmonyOS 5.0 / API 12。鸿蒙使用独立 HAP，不是 Android APK 兼容模式。
- 版本源为 `pubspec.yaml`；鸿蒙 `ohos/AppScope/app.json5` 必须与其一致，构建脚本会校验。

Android / iOS 在根目录运行：

```sh
flutter pub get --enforce-lockfile
flutter analyze
flutter test
flutter build apk --release
# 以下两项需要 macOS 和 Xcode
flutter precache --ios
(cd ios && pod install)
flutter build ios --simulator --debug --no-codesign
flutter build ios --release --no-codesign
```

鸿蒙在 Linux x86_64 上运行：

```sh
bash scripts/build-ohos.sh
```

进入 GitHub Actions → **Build Android, iOS and HarmonyOS** → Run workflow，一次构建三端。工作流仅手动触发，私有仓库会消耗 Actions 额度。产物为试用 APK、iOS 模拟器与未签名真机 app、未签名鸿蒙 HAP，不执行应用商店发布。

Android 包名保持 `com.alex.fingerdraw`，iOS 与鸿蒙为 `com.alex.chouwazi`。Android Release 构建暂用调试签名供试用，不能保证覆盖原本地签名或其他 CI 运行的安装包。iOS 与鸿蒙未签名产物不能直接安装，需自己的开发者签名；iOS 操作见 [真机试用指南](ios/DEVICE-TESTING.md)。仓库不保存签名私钥、证书或描述文件。

## 验证状态

- 共享实现：12 项状态与多指交互测试通过，静态分析通过。
- 鸿蒙：本机已真实编译共享游戏为 arm64 Release HAP，校验产物含编译后的 Dart 代码及 Flutter 引擎，权限仅震动。
- Android / iOS：迁移后完整 CI 构建验收进行中；迁移前的原生构建成功记录不作为共享版通过的证据。
- 三端真机安装、多指上限、显示与震动均待设备验收。编译通过不等于设备验证通过。

## 隐私

完全离线，不收集个人信息、不上传数据、不含广告。Android 与鸿蒙使用的系统权限仅震动，iOS 无需用户授权权限。AndroidX 另自动声明应用自身的 signature 级内部广播保护权限，限定同签名调用；它不授予联网或个人数据访问。构建检查同时验证其保护等级。应用使用单调时钟计时，iOS 同时打包应用隐私清单与 Flutter 引擎自身清单。

下载 SDK、构建依赖与开发者签名可能需要联网；已安装游戏不需要网络。为保持权限承诺，Android 调试配置也未添加 INTERNET 权限，网络调试服务不作为本项目的默认能力。
