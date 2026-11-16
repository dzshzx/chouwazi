# 构建

两端构建工作流提供试用产物；iOS 商店签名与上传使用独立的[发版流程](TESTFLIGHT.md)。各端构建与检查命令以 `.github/workflows/build.yml` 为准。

## 工具链

Python 构建辅助脚本经 uv 按各自的 PEP 723 声明运行。
本机先准备 uv；CI 安装固定版本 uv。有第三方依赖的脚本带同名 `*.py.lock`，调用使用
`uv run --locked --script`，依赖声明更新后用 `uv lock --script <脚本>` 更新并提交对应锁文件；
零依赖脚本（`dependencies = []`）不带锁，调用使用 `uv run --script`。

Flutter 版本由仓库根的 `.fvmrc` 固定，各端工具版本见 `android/app/build.gradle`、`android/settings.gradle`、`android/gradle/wrapper/gradle-wrapper.properties` 与 `build.yml`。本机先把 [fvm](https://github.com/btwld/fvm) 装到 PATH 上，命令一律经 `fvm flutter …`、`fvm dart …` 运行，不依赖全局 Flutter。

只维护 Android 与 iOS 两端。升级 Flutter 时同时验证共享依赖、测试和两端编译；如果构建输出出现 CocoaPods 弃用提示，或 Flutter 宣布弃用 CocoaPods，就重开 [dzshzx/chouwazi#4](https://github.com/dzshzx/chouwazi/issues/4)（iOS 迁到 Swift Package Manager）。

- 升级 Flutter 改 `.fvmrc` 与 `scripts/flutter-install.sh` 里的提交号（报错信息给出取提交号的命令）；fvm 本体的版本与各平台 SHA-256 在 `scripts/ci-setup-fvm.sh`。

## Android

Release 优化但使用调试签名供试用。持续分发或覆盖旧包须由维护者提供匹配的稳定签名；不能把 CI 的临时调试签名当作原 APK 的签名。

遇到手机不支持十指或系统三指手势拦截，按设备能力记录。

## iOS

必须在 macOS 上执行。CI 的模拟器 zip 与真机 zip 都含 `Runner.app`，只能在匹配的平台使用；真机包未签名，不是可直接安装的 IPA。签名与验收见 [iOS 真机试用指南](../ios/DEVICE-TESTING.md)。

Apple 自 2026-04-28 起要求上传包使用 Xcode 26 / iOS 26 SDK 或更新版本，见[官方要求](https://developer.apple.com/news/upcoming-requirements/)。签名、IPA 校验与商店上传见[发版指南](TESTFLIGHT.md)；上传成功须继续等待 Apple 处理并核对测试分发状态。

iOS 用 CocoaPods 集成 Flutter 引擎，`pubspec.yaml` 关闭了 Swift Package Manager（原因见该处注释）。`AppDelegate` 有屏幕常亮的自定义代码，Flutter 不会自动迁移到 UIScene 生命周期，构建时会给出手动迁移提示；迁移须在 macOS 上验证屏幕常亮与生命周期。

Runner 保持 Flutter 标准构建 scheme，CLI 通过 `--debug` 或 `--release` 选择模式。Xcode 真机试用时在 Edit Scheme → Run → Build Configuration 选择 Release。Debug/Profile 调试时 Flutter 可能请求本地网络访问以支持 DevTools，不属于 Release 应用功能。

## CI 产物

每轮交付以 Actions 的源码 SHA、任务结论和实际下载文件为准；根 README 维护当前验证状态。
