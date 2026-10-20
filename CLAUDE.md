# CLAUDE.md

「抽袜子 / chouwazi」是离线随机选择工具，通过一套 Flutter/Dart 源码编译 Android 与 iOS。使用与验证状态见 [README.md](README.md)，工具链和产物见 [docs/BUILDING.md](docs/BUILDING.md)。

## 共享实现

- 抽选规则只改 `lib/draw_round.dart`，物理触点、按钮与生命周期规则只改 `lib/game_session.dart`，并更新 `test/`。平台壳不复制状态机或界面。
- 参与触点有上限；触点用稳定 pointer ID；增减参与者重置唯一的截止时间（准备期加倒计时），移动不重置；结果冻结；空缺槽位复用；随机索引等概率。
- 倒计时源是单调 `Stopwatch`。所有物理触点（含超限和结果后的新增触点）松开才允许重开；控制按钮手势不参与抽签。系统取消触控或失活清理未完成轮次，保留已产生结果。
- 简体、繁体、英文界面统一在 `lib/strings.dart`。桌面应用名属于各端原生资源，改名时同步两端资源。

## 构建与验证

- 只维护 Android 与 iOS 两端（取舍见 [ADR 0001](docs/adr/0001-drop-harmonyos.md)），均用[构建指南](docs/BUILDING.md#工具链)固定的 Flutter 版本，升级须复验两端。
- iOS 工程为 `ios/Runner.xcworkspace` / `Runner` scheme。新增原生 iOS 资源需登记 `project.pbxproj`；共享 Dart 文件不需要逐个登记。
- GitHub Actions `build.yml` 仅手动触发，执行共享测试并真实构建两端。本机 Linux 可构建 APK，iOS 必须在 macOS/Xcode 构建。
- 多指触控、显示和震动仍须真机确认；验证结论变化时更新 README「验证状态」，保留编译、安装与真机体验的区别。

## 平台约束

- Android applicationId 保持 `com.alex.fingerdraw`；iOS 为 `com.alex.chouwazi`。同一包名换了签名也不能覆盖安装。
- 版本统一来自 `pubspec.yaml`。
- 离线承诺适用于最终产物：Android 系统权限仅震动；AndroidX 的应用内部广播权限须保持 signature 级；iOS 保留应用与引擎隐私清单。构建脚本校验真实 APK 权限，新增权限或联网能力时同步隐私说明。
- 签名材料与本机路径不入库。Android Release 是调试签名试用包；`build.yml` 的 iOS 产物未签名。TestFlight 分发使用 `testflight.yml` 的主分支受保护环境，签名、上传和处理状态按 [发布指南](docs/TESTFLIGHT.md) 分别验证；发布和设备签名沿用户授权处理。
