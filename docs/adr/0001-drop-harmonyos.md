---
status: accepted
---

# 放弃鸿蒙端，只保留 Android 与 iOS

本仓原来用一套 Flutter/Dart 源码编译 Android、iOS 和原生鸿蒙 HAP，三端共用一份 `pubspec.lock`。Flutter SDK 会把 characters、collection、meta、test_api 等自带包钉成固定版本，不同版本线钉的值不同，所以共用锁文件要求 Google Flutter 与 Flutter OH 走同一条版本线。2026-10-20 调研时，Flutter OH 最新正式线 3.41.9+ohos 需要 HarmonyOS 命令行工具 26.0.0，这个版本只能登录华为账号下载，公开镜像没有，现有「公开地址加 SHA-256」的自动配置做不到；鸿蒙最低系统也要从 API 12 提到 API 17。维护者 2026-10-20 在问答中决定放弃鸿蒙端，原话「不再要求鸿蒙版本，去掉鸿蒙端的开发」「不需要鸿蒙版了」；同一天还要求提升依赖版本，原话「两件事，1修，2提升依赖版本，都做」。去掉鸿蒙后 Google Flutter 不再受鸿蒙版本线约束，直接升到最新稳定版。

## Considered Options

- 本机登录华为账号下载命令行工具 26.0.0，停掉 CI 的鸿蒙构建，三端同走 3.41.9 线（调研时的推荐项）。没有选：鸿蒙构建只能在本机手动做，并且要提高最低系统。
- CI 用私有附件提供命令行工具 26.0.0。没有选。
- 鸿蒙不升级，只升 Android/iOS，维护两份锁文件。没有选。
- 三端都不动，继续停在 3.27 线。没有选。

## Consequences

- 不再构建和发布鸿蒙包；`build.yml` 只构建 Android 与 iOS，Flutter 版本只跟 Google 稳定版。
- 要恢复鸿蒙端，可从删除前的提交 `3abd21049bc3eef3a4a0040ad517dbefe0fc9786` 找回 `ohos/` 工程、`scripts/setup-ohos.py`、`build-ohos.sh`、`ohos-env.sh`、`check-hap.py` 及其锁文件和 CI 任务；恢复时须重新对齐当时的 Flutter OH 版本线与共用锁文件。
