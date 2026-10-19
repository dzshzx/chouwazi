# iOS 真机试用指南

当前使用 Android 与 iOS 共用的 Flutter 源码。已获内部测试邀请的用户可通过 [TestFlight](../docs/TESTFLIGHT.md#testflight-安装) 安装，无需自备 Mac；本地开发也可通过下述 Xcode 流程签名安装。最新编译与分发证据统一见[根 README](../README.md#验证状态)，已完成的设备验证见下方实测记录，完整验收范围按清单逐项记录。

## 准备与安装

准备一台装有 Xcode 的 Mac、Apple 账号和 iOS / iPadOS 15.0+ 设备。首次验证使用连接线。已验证的构建工具为 Xcode 26.3；设备系统较新时，需使用支持该设备系统的 Xcode。通过 TestFlight 分发使用[签名上传指南](../docs/TESTFLIGHT.md)，安装时不需要连接 Mac。

1. 在 Mac 获取源码并安装 Flutter 3.47.7，在根目录执行 `flutter pub get --enforce-lockfile`、`flutter precache --ios`、`(cd ios && pod install)`、`flutter build ios --config-only --no-codesign`，再打开 `ios/Runner.xcworkspace`，选择 `Runner` scheme。工具链说明见[构建指南](../docs/BUILDING.md)。
2. 在 Xcode 设置中的 Apple Accounts（部分版本称 Accounts）登录自己的 Apple 账号。个人设备开发试用可选择 Personal Team；分发给其他人另行准备签名与分发方案。
3. 选中 `Runner` target → Signing & Capabilities，开启 Automatically manage signing，选择自己的 Team。若 `com.alex.chouwazi` 无法注册，仅在本地改为账号可用的唯一 Bundle Identifier，并记录实际值；不要把个人 Team 或签名材料提交进仓库。
4. 连接并解锁设备，按提示信任 Mac，在 Xcode 的运行目标中选择该设备，等待配对和设备准备完成。
5. 系统要求 Developer Mode 时，在设备「设置 → 隐私与安全 → 开发者模式」开启，按提示重启并确认。没有此入口时，先完成 Xcode 配对，再按 Xcode 的设备提示处理。
6. 真机试用前，在 Product → Scheme → Edit Scheme → Run → Build Configuration 选择 Release，再按 `⌘R` 构建、签名、安装并启动。首次运行若有开发者信任提示，按系统指引处理。Debug/Profile 的 Flutter 开发调试服务可能触发本地网络提示；模拟器开发可保留标准 Debug 配置。
7. 确认桌面图标可以再次打开应用，再断开连接，执行下面的验收。启用飞行模式并关闭 Wi-Fi，完成至少一轮抽签。

步骤依据 Apple 的[设备运行说明](https://developer.apple.com/documentation/xcode/running-your-app-on-simulated-or-physical-devices)、[签名流程](https://help.apple.com/xcode/mac/current/en.lproj/dev60b6fbbc7.html)和[开发者模式说明](https://developer.apple.com/documentation/xcode/enabling-developer-mode-on-a-device)。账号签名与安装准备可能需要联网，应用玩法本身完全离线。

## CI 产物与适用范围

当前工作流的 `chouwazi-ios-builds` artifact 包含以下文件，下载需有私有仓库访问权。Actions 产物有保留期限，过期后可按对应源码重新构建。使用根 README 指向的构建，并核对源码 SHA 和版本。

| 文件 | 用途 |
| --- | --- |
| `chouwazi-simulator.zip` | 模拟器 `Runner.app`，不能安装到 iPhone/iPad |
| `chouwazi-device-unsigned.zip` | 真机 Release 编译产物，未签名、无描述文件，不能直接安装；首次试用使用上面的 Xcode 源码安装流程 |

Bundle Identifier 保持 `com.alex.chouwazi`，最低系统为 iOS 15。CI 检查真实 bundle 中的 Flutter 引擎、共享 Dart 资源、应用隐私清单、图标和三语言桌面名称。源码 SHA、版本与构建结论以对应 Actions 运行及根 README 为准。

## 2026-10-09 实测记录

- iPhone 18 Pro / iOS 27.0.1，应用 1.0.4 (5)，源码 `3f85685`。TestFlight 后台确认已安装；47.74 秒录屏展示首次启动、多指输入、倒计时、随机结果、再次抽选及简体中文隐私页。
- 该机实际最多识别 5 个同时触点。用户确认第六指是在倒计时结束前、抽选区域内加入；在同机 Safari 的[独立触点测试](https://patrickhlauke.github.io/touch/tracker/multi-touch-tracker.html)中也最多出现 5 个圆圈。该测试读取真实 `targetTouches` 列表。
- Android 对照：用户确认 Xiaomi 17 的应用可同时识别 10 点；本次未记录该设备的系统版本和构建号。
- 软件边界验证：在 Flutter 3.27.4 下向实际 `GameScreen` 触点监听器输入 6 点和 10 点，两项测试均通过，末位触点可被选中；2026-10-20 升级到 Flutter 3.47.7 后尚未重做这项验证与真机验证。此项验证覆盖应用事件处理与界面状态，设备实际触点能力由真机结果记录。
- Flutter 3.27.4 对应引擎 `82bd5b7209295a5b7ff8cae0df96e7870171e3a5` 已在 [`FlutterViewController.loadView`](https://github.com/flutter/engine/blob/82bd5b7209295a5b7ff8cae0df96e7870171e3a5/shell/platform/darwin/ios/framework/Source/FlutterViewController.mm#L499) 开启 `multipleTouchEnabled`，并逐个转发 UIKit 提供的触点；应用的软件上限为 10。综合源码、软件边界测试与 Safari 对照，五点限制定位于该设备的 iOS 输入能力。触控芯片、固件与系统内部各自的限制仍缺少公开规格依据。
- 触点测试应记录实际事件数量。WebKit 当前 [`navigator.maxTouchPoints`](https://github.com/WebKit/WebKit/blob/main/Source/WebCore/page/Navigator.cpp#L456) 在 iOS 分支返回固定值 5，不能单独据此判定某款设备的硬件能力。
- 录屏 SHA256：`6e8e91d9fe5ae424c5aa63af57d6d6fbc5533716364acae8bc2df6a66b67cc7f`。视频已用于 App Review 补充材料。iPad、其他语言、震动手感、飞行模式及完整生命周期验收仍待逐项完成。

## 真机验收清单

尽可能覆盖一台 iPhone 和一台 iPad；未测设备和语言保留「待测」。每次记录设备型号、系统版本、源码提交、应用版本、语言与实际最大触点数。

| 项目 | 操作和预期 | 状态 |
| --- | --- | --- |
| 启动与布局 | 冷启动、退出后再打开；标题、底部文字和按钮不被刘海或 Home 指示条遮挡；设备旋转后仍可正常操作 | 待测 |
| 本地化 | 分别使用简体中文、繁体中文、英文；检查桌面名称、首页、结果和按钮是否完整显示 | 待测 |
| 单指等待 | 仅一指按住，显示等待更多手指，不开始抽签 | iPhone 18 Pro 录屏已确认；iPad 待测 |
| 两指抽签 | 第二指加入后先准备约 0.8 秒，再显示 3 秒倒计时，仅选出一个已有编号 | 待测 |
| 移动与增减 | 倒计时中移动手指不重置；加入第三指或从三指减至两指重新准备；减至一指停止倒计时 | 待测 |
| 身份与槽位 | 三指加入后移走中间编号，再加一指；原有手指编号保持，新指复用空缺编号 | 待测 |
| 多指上限 | 逐步增加手指，记录设备真实报告的最大数量；应用最多收录 10 指，实际数量由设备输入能力决定 | iPhone 18 Pro 实测 5 点；iPad 待测 |
| 结果冻结 | 抽中后移动、移开或新增手指，中奖编号与结果位置不变；所有手指松开后才能点「再来一轮」 | 待测 |
| 连续抽选 | 完成至少 10 轮，检查是否有残留编号、无法重开、重复结果事件或崩溃；这不构成随机分布统计检验 | 待测 |
| 系统打断 | 准备中、倒计时中分别打开控制中心、切后台或锁屏，返回后无旧倒计时；已有结果在失活恢复后仍保留并可重开 | 待测 |
| 触控取消 | 触发设备系统多指手势，再回到应用；不残留已取消手指，不出现无法重开的状态 | 待测 |
| 震动 | 支持震动的设备分别测试开/关；关闭后无震动，开启后加入、倒计时和结果反馈正常；不支持的设备记不适用 | 待测 |
| 离线 | 飞行模式且 Wi-Fi 关闭，冷启动并完成抽签和重开，无联网要求或权限弹窗 | 待测 |
| 隐私与返回 | 左上角打开三语隐私说明，滚动并返回；按住抽选区域时切页再返回，须等原手指全部松开才可重新参与或重开 | 待测 |

问题回报应包含：设备/系统、语言、提交与版本、复现步骤、预期与实际、出现频率、截图或录屏。崩溃时附 Xcode 或设备诊断中的对应崩溃日志，去掉账号与设备标识等个人信息。

## 安装问题定位

- 找不到设备或系统不支持：先检查连接、解锁、信任、配对状态及 Xcode 对设备系统的支持。
- 提示缺 Team、Bundle Identifier 不可用或缺 provisioning profile：回到 Signing & Capabilities，核对账号、Team、唯一标识和自动签名；保留 Xcode 的原始错误便于继续排查。
- 提示 Developer Mode 未开启：按系统提示完成开启、重启和再次确认。
- 下载 zip 后无法安装：先区分模拟器产物与未签名真机产物；两者都不是可直接安装的已签名 IPA。
- 曾能安装，后来不能启动：先查看签名或描述文件是否失效，通过自己的 Xcode/Team 重新签名运行；不要把这一现象直接判为应用逻辑崩溃。

完成设备验收后，将实测记录与问题结论写入本文件，并同步根 README 的「验证状态」。
