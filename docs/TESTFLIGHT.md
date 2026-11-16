# 发版与 TestFlight 上传

发版由 `.github/workflows/release.yml` 从已合入的 `main` 构建、签名并上传 iOS 包到 TestFlight。
Flutter 版本由 `.fvmrc` 固定（见[构建指南](BUILDING.md#工具链)），Xcode 版本见 `release.yml`。编译、签名、上传与分发的验证状态统一见 [README 验证状态](../README.md#验证状态)。

## 发版

```sh
gh workflow run release.yml --ref main -f bump=patch
```

- `bump` 取 `patch`（默认）、`minor` 或 `major`。`minor` 用于用户能感知的新能力，`major` 用于破坏性变化（含 0.x→1.0），这两档须维护者先确认；内部重构、目录搬迁、删兼容层不算破坏性，用 `patch`。
- 加 `-f dry_run=true` 是试跑，可在任意分支运行：跑门禁、带新版本号构建未签名的 iOS 真机包并核对产物版本，在运行摘要里列出将要发布的版本、源提交和产物 sha256；不进受保护环境、不签名、不打 tag、不上传。
- 正式发版只能在 `main` 上运行；`main` 的头提交已带纯 `vX.Y.Z` tag 时直接失败，说明这个提交已发过版。

### 版本号与构建号

- 版本号：仓里形如 `vX.Y.Z`（纯数字三段，无前后缀）的 tag 中最高的一个，按 `bump` 升一档。
- 构建号（iOS `CFBundleVersion`）= 主版本 × 1000000 + 次版本 × 1000 + 修订号，次版本与修订号须小于 1000，例如 1.0.5 的构建号是 1000005。构建号随版本单调递增。
- 两者由 `scripts/release-version.sh` 算出，构建时经 `--build-name`、`--build-number` 传给 Flutter，不提交回仓库。`pubspec.yaml` 的 `version` 是开发占位值 `0.0.0+1`，本地构建与 `build.yml` 试用产物显示的都是这个占位版本。

### 流程与出错处理

1. 托管 macOS 上跑门禁 `scripts/check.sh`，带新版本号构建未签名真机包，核对产物里的版本号与构建号。dry_run 到此结束。
2. 进入受保护环境 `testflight`，签名、导出并检查 IPA，再经 Apple 校验（`altool --validate-app`）。
3. 校验通过后在源提交上推送附注 tag `vX.Y.Z`，再上传 TestFlight；运行摘要写明版本、`source_commit` 与 IPA 的 sha256。签名与上传细节见 `scripts/testflight.sh`。

推送 tag 之前任何一步失败都不留 tag：修好、合入后用同一 `bump` 重跑即可。tag 推送后上传失败，先到 App Store Connect 查看实际状态，再修好、合入并发下一个 patch。已发布的 tag 不移动、不删除、不复用；版本只升不降，出问题向前修。

## 账号与签名前提

- Apple Developer Program 会员有效，相关协议已接受。
- 开发者账号中有显式 App ID `com.alex.chouwazi`，App Store Connect 已创建使用同一 Bundle ID 的 iOS App 记录。
- Apple Distribution 证书及其私钥已导出为带密码的 `.p12`；App Store 分发描述文件选择此证书和此 App ID，且均未过期。
- 有 Access 设为 **Developer** 的 App Store Connect **团队 API key**：`.p8`、Key ID、Issuer ID。CI 用该 key 认证上传，签名使用证书私钥。个人 API key 不适用于本流程的 issuer 认证。[Apple 上传说明](https://developer.apple.com/help/app-store-connect/manage-builds/upload-builds/)列出的可上传角色包含 Developer。

本流程由 Account Holder 在 Apple Developer 和 App Store Connect 网页处理 App ID、分发证书、描述文件、App 记录与测试组，并创建上述 Developer 团队 key。CI 使用现有材料完成签名和上传，不通过该 key 申请证书或描述文件。
团队 key 的角色权限作用于团队的所有 App，不能仅限定为「抽袜子」；创建团队 key 的账号角色与 key 的 Access 角色不同，见 [Apple API key 说明](https://developer.apple.com/help/app-store-connect/get-started/app-store-connect-api/)。

证书和描述文件须属于同一开发者团队。脚本从描述文件读取 Team ID，不把账号标识写死在工程里。
App ID 若已被别的账号占用，先解决所属关系，不通过改变项目包名绕开。

从 Linux 导出供当前 macOS CI 使用的 `.p12` 时，已验证以下封装参数可被钥匙串导入。`SIGNING_DIR` 应指向权限为 0700 的受保护目录，密码通过环境变量安全注入；证书、私钥和输出文件不放进仓库。

```sh
openssl pkcs12 -export \
  -inkey "$SIGNING_DIR/distribution-key.pem" \
  -in "$SIGNING_DIR/distribution.pem" \
  -out "$SIGNING_DIR/distribution.p12" \
  -passout env:IOS_DISTRIBUTION_P12_PASSWORD \
  -keypbe PBE-SHA1-3DES -certpbe PBE-SHA1-3DES -macalg sha1
```

这只改变 `.p12` 的封装，保留同一证书和私钥。本次 OpenSSL 4.0.3 默认导出文件可被云端 OpenSSL 解密，但 `security import` 报 MAC 校验失败；使用以上封装后，macOS 15.7.9 已成功完成签名归档和导出。遇到同类问题先对同一文件和同一密码核对 OpenSSL 校验与钥匙串导入结果，避免重建或撤销证书。

恢复资料保存在私有凭据库的独立条目（条目与附件清单见内部 secret 清单）；当前 CI 使用兼容封装的 p12 与 provisioning profile，原始封装另行保留，恢复时使用已验证的兼容附件。签名材料变更后，须同步受保护环境并验证备份可读。

## 配置 GitHub secrets

先在仓库 Settings → Environments 创建 `testflight`，将 Deployment branches and tags 设为 **Selected branches and tags → `main`**，只允许该分支。
把以下值加入此 **Environment 的 secrets**，不另放同名 Repository secrets，以便分支规则保护凭据。
还应保护 `main` 的写入权限，并在 CI 或签名流程合入前完成非作者检查。

| Secret | 内容 |
| --- | --- |
| `IOS_DISTRIBUTION_P12_BASE64` | 含 Apple Distribution 证书和私钥的 `.p12` 的 base64 |
| `IOS_DISTRIBUTION_P12_PASSWORD` | 导出 `.p12` 时设置的密码 |
| `IOS_APP_STORE_PROFILE_BASE64` | App Store `.mobileprovision` 的 base64 |
| `ASC_API_KEY_P8` | `.p8` 文件原文，保留实际换行 |
| `ASC_API_KEY_ID` | 该团队 API key 的 Key ID |
| `ASC_API_ISSUER_ID` | App Store Connect 的 Issuer ID |

用可信本机工具编码后直接填入 GitHub secret 输入框，不把 base64、密码或私钥粘进聊天、终端日志、工单或仓库。
base64 只是编码，具有与原始签名文件相同的保密要求。GitHub secrets 的设置与使用见[官方指南](https://docs.github.com/en/actions/how-tos/write-workflows/choose-what-workflows-do/use-secrets)。

## 发版前后

1. Flutter 升级后的首次发版前，先在新版本下手动跑通一次 `screenshots.yml`，结果记入 README「验证状态」；发版完成后把 `release.yml` 的结果一并记入。
2. 由 Account Holder 到 App Store Connect 网页查看构建处理结果，处理完成后把构建加入测试组。外部测试可能需要 Beta App Review；App Store 上架还需补齐商店资料并单独提交审核。

“上传被 Apple 接受”与“处理完成、TestFlight 可安装”是不同状态。[Apple 上传说明](https://developer.apple.com/help/app-store-connect/manage-builds/upload-builds)明确要求构建先完成处理。
签名材料、原始上传日志与 IPA 均不发布为 Actions artifacts。
仍须按 [iOS 真机试用指南](../ios/DEVICE-TESTING.md) 验收多指、画面与震动。

## TestFlight 安装

在 iPhone 或 iPad 上安装 Apple 的 [TestFlight](https://testflight.apple.com/)，打开测试邀请邮件并接受邀请，即可安装对应构建。用户不需要自备 Mac，也不需要为 TestFlight 安装开启开发者模式。内部测试组使用手动添加构建，后续成功上传并处理的新构建仍需在网页加入该组。

外部测试的[公开邀请链接](https://testflight.apple.com/join/CRjBmUDp)对应「公开测试」组，设为 Open to Anyone，启用获批后自动通知测试者。首次外部构建需要审核，见 [Apple 外部测试说明](https://developer.apple.com/help/app-store-connect/test-a-beta-version/invite-external-testers/)。

Info.plist 的 `ITSAppUsesNonExemptEncryption=false` 声明含义见 [Apple 字段文档](https://developer.apple.com/documentation/bundleresources/information-property-list/itsappusesnonexemptencryption)；若将来引入新的加密或联网能力，须重新核对声明和隐私说明。
