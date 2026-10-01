# CLAUDE.md

This file provides guidance to Claude Code (claude.ai/code) when working with code in this repository.

「抽袜子 / chouwazi」是 Android 与 iOS 两端各自独立实现的原生多指抽签小游戏，无第三方运行库、无网络。玩法、环境要求与构建步骤见 [README.md](README.md)，Android 侧补充说明见 [android/README.txt](android/README.txt)。

## 架构

- 两端不共享代码，抽签规则在各端的平台无关状态模型里各写一份：`android/app/src/main/java/com/alex/fingerdraw/DrawState.java` 与 `ios/Sources/DrawCore/DrawRound.swift`。改规则（上限 10 指、加入/离开重置倒计时、移动不重置、结果冻结、槽位复用、等概率选取）须两端模型和两端测试（`android/tests/DrawStateTest.java`、`ios/Tests/DrawCoreTests/`）同步修改。
- 倒计时只有一个截止时间：`3.8s = 0.8s 准备 + 3s 倒计时`；UI 根据剩余时间大于 3 秒显示「准备」阶段。时间源为单调时钟（Android `SystemClock.uptimeMillis()`、iOS `ProcessInfo.systemUptime`），iOS 的 `PrivacyInfo.xcprivacy` 因此声明了 SystemBootTime 用途。
- 玩家以触点身份标识（Android pointer ID、iOS `UITouch`），不用数组下标；UI 层（`MainActivity` 的 Canvas 视图、`GameView` 的 UIKit 视图）只负责把触控事件转成模型调用并定时 `tick`。
- iOS 的 Xcode 工程把 `Sources/DrawCore/DrawRound.swift` 直接编进 App target，`GameView` 不 `import DrawCore`；`ios/Package.swift` 只服务 `swift test`。`project.pbxproj` 为手写、顺序 ID，新增源文件或资源需手动登记。
- 本地化：iOS 有 `en`、`zh-Hans`、`zh-Hant` 三套 `Localizable.strings` 与 `InfoPlist.strings`（应用名）；Android 只有应用名走 `values` / `values-zh`，界面文字直接写在 `MainActivity.java`（中文）。
- 标识历史沿用：Android `applicationId` / 包名为 `com.alex.fingerdraw`（Gradle 工程名 FingerDraw），iOS Bundle Identifier 为 `com.alex.chouwazi`。改 Android `applicationId` 会让已安装版本无法覆盖升级。
- Android 版本号在 `android/app/build.gradle`（`versionCode` / `versionName`）；`build-local.sh` 输出文件名与 `android/README.txt` 也写死了 `1.0.1`，升版时一起改。

## 验证

- Android 状态测试（本机 Java 17 可跑，与 CI 相同）：

  ```sh
  cd android && mkdir -p build/test-classes \
    && javac -d build/test-classes app/src/main/java/com/alex/fingerdraw/DrawState.java tests/DrawStateTest.java \
    && java -cp build/test-classes DrawStateTest
  ```

- 本机没有 Gradle（工程不带 wrapper）和 Swift/Xcode。APK 构建、`swift test` 与 `xcodebuild` 依赖 GitHub Actions 的 `Build Android and iOS`（`.github/workflows/build.yml`，仅 `workflow_dispatch`，私有仓库消耗 Actions 额度），可用 `gh workflow run build.yml` 触发。
- 多指触控、显示与震动效果只能在真机确认；README 的「验证状态」一节记录各端已验证到哪一步，结论变化时同步更新。

## 约束

- 隐私承诺是完全离线：Android 只申请震动权限，iOS 不申请任何权限。新增网络、权限或受限 API 时，同步更新 README「隐私」一节与 `PrivacyInfo.xcprivacy`。
- 仓库不含签名材料，`.gitignore` 屏蔽证书、描述文件与 keystore；`build-local.sh` 在 `android/build/` 生成本地自签名 keystore，不入库。
