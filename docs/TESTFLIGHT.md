# TestFlight 上传

`.github/workflows/testflight.yml` 从已合入的 `main` 构建、手动签名并上传 iOS 包。
Google Flutter 版本见[构建指南](BUILDING.md#工具链)，Xcode 固定为 26.3。编译、签名、上传与分发的验证状态统一见 [README 验证状态](../README.md#验证状态)。
Apple 自 2026-04-28 起要求上传的 iOS 包使用 Xcode 26 与 iOS 26 SDK 或更新版本，见[官方要求](https://developer.apple.com/news/upcoming-requirements/)。

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

## 执行与结果

1. 核对 `pubspec.yaml` 的版本和 build number。上传使用该版本，不自动改号；同一版本的后续新包须增加 build number。
2. 先完成 `Build Android and iOS` 的两端构建与共享测试，再在 Actions 中手动运行 `Upload iOS to TestFlight`，分支选择 `main`。Flutter 升级后的首次上传前，先在新版本下手动跑通一次 `build.yml` 与 `screenshots.yml`，结果记入 README「验证状态」；上传完成后把 `testflight.yml` 的结果一并记入。
3. 流程复跑共享检查，在独立临时 keychain 中导入分发身份，仅对 Runner 的 Release 配置设置手动签名，然后执行 `xcodebuild archive` 和 `-exportArchive`，导出方式为 `app-store-connect`。
4. 从本次导出目录中定位唯一的 IPA 并规范化文件名，再检查 Bundle ID、版本、构建号、iOS SDK、应用与 Flutter 隐私清单、分发描述文件、签名证书和 entitlement，最后由 Apple `altool` 验证并上传。
5. 由 Account Holder 到 App Store Connect 网页查看构建处理结果，处理完成后把构建加入测试组。外部测试可能需要 Beta App Review；App Store 上架还需补齐商店资料并单独提交审核。

工作流只有手动入口、读取仓库权限和固定源码 SHA；没有自定义 ref 或外部 artifact 输入，并发上传串行执行。
secrets 只交给签名上传步骤。正常结束、失败或取消时恢复 Runner 工程与 keychain 搜索列表，清除临时 keychain、描述文件和 key；`always()` 步骤再次清理。
签名材料、原始上传日志与 IPA 均不发布为 Actions artifacts。

“上传被 Apple 接受”与“处理完成、TestFlight 可安装”是不同状态。[Apple 上传说明](https://developer.apple.com/help/app-store-connect/manage-builds/upload-builds)明确要求构建先完成处理。
上传出错或取消后先查看 App Store Connect 的实际状态，再决定是否重跑，避免重复上传同一构建。
仍须按 [iOS 真机试用指南](../ios/DEVICE-TESTING.md) 验收多指、画面与震动。

## TestFlight 安装

在 iPhone 或 iPad 上安装 Apple 的 [TestFlight](https://testflight.apple.com/)，打开测试邀请邮件并接受邀请，即可安装对应构建。用户不需要自备 Mac，也不需要为 TestFlight 安装开启开发者模式。内部测试组使用手动添加构建，后续成功上传并处理的新构建仍需在网页加入该组。

外部测试的[公开邀请链接](https://testflight.apple.com/join/CRjBmUDp)对应「公开测试」组，设为 Open to Anyone。2026-10-09 已关联 1.0.4 (5) 并提交 Beta App Review，启用获批后自动通知测试者。2026-10-14 匿名打开公开链接，页面已提供加入入口，可用另一个 Apple 账号加入并安装。首次外部构建需要审核，见 [Apple 外部测试说明](https://developer.apple.com/help/app-store-connect/test-a-beta-version/invite-external-testers/)。

本应用离线运行且没有自有加密，因此 Info.plist 声明 `ITSAppUsesNonExemptEncryption=false`，并在 IPA 检查该字段。
声明含义见 [Apple 字段文档](https://developer.apple.com/documentation/bundleresources/information-property-list/itsappusesnonexemptencryption)；若将来引入新的加密或联网能力，须重新核对声明和隐私说明。
