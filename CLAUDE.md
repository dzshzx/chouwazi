# CLAUDE.md

使用与验证状态见 [README.md](README.md)，工具链和产物见 [docs/BUILDING.md](docs/BUILDING.md)。

## 共享实现

- 抽选规则只改 `lib/draw_round.dart`，物理触点、按钮与生命周期规则只改 `lib/game_session.dart`；规则以这两个文件和 `test/` 为准，改规则同步测试。平台壳不复制状态机或界面。
- 测试未覆盖、须保持：随机索引等概率（`Random.secure`）；倒计时源是单调 `Stopwatch`（`lib/game_screen.dart`）。
- 简体、繁体、英文界面统一在 `lib/strings.dart`。桌面应用名属于各端原生资源，改名时同步两端资源。

## 构建与验证

- 只维护 Android 与 iOS 两端，均用 `.fvmrc` 固定的 Flutter 版本（见[构建指南](docs/BUILDING.md#工具链)），升级须复验两端。
- iOS 工程为 `ios/Runner.xcworkspace` / `Runner` scheme。新增原生 iOS 资源需登记 `project.pbxproj`；共享 Dart 文件不需要逐个登记。
- 门禁：`bash scripts/check.sh`（CI 只在 Linux 上跑它：`build.yml` 的 Android 作业与 `release.yml` 的 `gate` 作业，后者只在发版带 `--gate`（工作流输入 `gate=true`）时才跑，默认不跑；macOS 作业经 `.github/actions/ios-setup` 准备，只做 iOS 构建与检查）。
- 发版：`~/workspace/fleet-state/tools/rollout/rollout release chouwazi`（在 `main` 头上派发 `release.yml` 并等运行结束），先试跑加 `--dry-run`，要跑门禁加 `--gate`（默认不跑）；在别的分支上试跑直接派发 `gh workflow run release.yml --ref <分支> -f dry_run=true`。版本号由最高的纯 `vX.Y.Z` tag 算出，构建号按公式由版本号算出，`pubspec.yaml` 只放占位版本，不手改；`minor`、`major` 须维护者先确认。详见[发版指南](docs/TESTFLIGHT.md#发版)。
- 本机 Linux 可构建 APK，iOS 必须在 macOS/Xcode 构建。
- 多指触控、显示和震动仍须真机确认；验证结论变化时更新 README「验证状态」，保留编译、安装与真机体验的区别。

## 平台约束

- Android applicationId 保持 `com.alex.fingerdraw`；iOS 为 `com.alex.chouwazi`。同一包名换了签名也不能覆盖安装。
- 离线承诺适用于最终产物：Android 系统权限仅震动；AndroidX 的应用内部广播权限须保持 signature 级；iOS 保留应用与引擎隐私清单。构建脚本校验真实 APK 权限，新增权限或联网能力时同步隐私说明。
- 签名材料与本机路径不入库。Android Release 是调试签名试用包；`build.yml` 的 iOS 产物未签名。TestFlight 分发使用 `release.yml` 的主分支受保护环境，签名、上传和处理状态按 [发版指南](docs/TESTFLIGHT.md) 分别验证；发布和设备签名按维护者授权处理。
