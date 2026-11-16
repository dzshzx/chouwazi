# 抽袜子：App Store 发布与审核

核对日期：2026-10-09；上架与供应状态 2026-10-14 复核。目标为全球供应，包含中国大陆。本文把官方规则、开发者自述和本项目的待办分别记录；社区成功经历不构成审核或备案豁免保证。

## 当前状态与材料入口

- 个人 Apple 开发者会员。云端 macOS 的签名、上传和恢复流程见 [TESTFLIGHT.md](TESTFLIGHT.md)。
- 应用在设备内完成多指随机选择；没有账号、广告、内购、分析 SDK 或业务服务器。触点和结果只在运行时内存中处理。
- 支持与隐私页已发布到独立站点 [chouwazi.646598.xyz](https://chouwazi.646598.xyz/)，公开支持邮箱为 `dzshzxdev@163.com`，已实测收信；本 App 尚未办理 App 备案。
- [三语商店文案](../store/metadata.json)与[审核备注](../store/review-notes.txt)已提交；[中国大陆适用性咨询草稿](../store/mainland-inquiry.txt)保留，尚未发送。
- 已上架版本为 1.0.4 (5)，三语说明按离线随机选择工具的实际功能撰写。版权署名为 `dzshzx`，商店版权字段为 `2026 dzshzx`。
- App Store Connect 已保存 1.0.4 的三语名称、副标题、描述和关键词，以及版权、Utilities 主类目和无需审核登录的设置；三语支持与隐私的六个链接已填入。专用审核联系方式与公开邮箱分开。年龄问卷按实际功能完成，常规分级为 4+（地区分级以后台为准）；内容权利声明已保存，Data Not Collected 隐私标签已发布。
- 1.0.4 (5) 已于 2026-10-11 22:46（Asia/Shanghai）获批，并按 `AFTER_APPROVAL` 自动发布。
- 首次送审收到 `Guideline 2.1 - Information Needed`，补交完整六项说明与真机录屏后获批，定稿见[仓库审核备注](../store/review-notes.txt)。软件上限 10 指与真机实际触点能力的区别已在备注中说明，设备证据见 [iOS 实测记录](../ios/DEVICE-TESTING.md#2026-10-09-实测记录)。
- 免费发布，175 个国家和地区供应，包含中国大陆；大陆上架不等于备案适用性已获确认。DSA 按个人非商家声明。iOS 分发设备为 iPhone、iPad；需要实体多指触屏，不启用 Mac 或 Apple Vision Pro 分发。

Apple 的[必填属性表](https://developer.apple.com/help/app-store-connect/reference/app-information/required-localizable-and-editable-properties)与[版本信息说明](https://developer.apple.com/help/app-store-connect/reference/app-information/platform-version-information/)是字段依据。隐私入口等代码变更后须重新构建并增加构建号。

## 独立支持站点与备案

三语页面位于 `https://chouwazi.646598.xyz/` 的 `zh-Hans/`、`zh-Hant/`、`en/` 子目录下，文件名为 `privacy.html` 和 `support.html`。站点部署在独立静态目录，路由配置与部署契约由内部运维仓维护。网站页脚显示现有网站备案信息，其用途与 App 备案不同。

工信部通知要求在境内从事 App 互联网信息服务的主办者备案，已有网站备案主体可补充 App 信息；网站备案号不能直接当成本 App 已备案的证明。官方通知未直接给出本项目这种纯本地随机选择应用的个案判定。[工信部通知](https://www.gov.cn/zhengce/zhengceku/202308/content_6897341.htm)、[工信部解读](https://wap.miit.gov.cn/zwgk/zcjd/art/2023/art_39b4f1acc36745b98478e0ec3e07128d.html)

Apple 当前说明是“部分 App”需要 ICP 信息；腾讯备案渠道 FAQ 则明确把上应用商店的单机离线 App 纳入办理口径。应向 Apple 确认本 App 的大陆供应要求，再向原接入商或所在地管局核对真实填报方式。备案表的后台域名应反映真实接入；本 App 没有业务服务器，隐私政策网站仅提供网页。[Apple App 信息](https://developer.apple.com/cn/help/app-store-connect/reference/app-information/app-information/)、[腾讯 FAQ](https://cloud.tencent.cn/document/product/243/97691)、[阿里 FAQ](https://help.aliyun.com/zh/icp-filing/basic-icp-service/support/basics-about-icp-filling-for-apps)

本 App 的核心用途是多人同时触屏后随机选出一位。产品定位为离线随机选择工具，商店主类目采用 Utilities（工具）；按这一真实用途准备描述、截图及审核说明。[Apple 类目说明](https://developer.apple.com/app-store/categories/)

若需 App 备案，可先整理下列资料，不把不适用字段编造完整：

- 现有备案主体、证件、省份、接入商、域名及备案号，由维护者提供或确认；证件只交给实际办理平台。
- App 名称“抽袜子”、iOS Bundle ID `com.alex.chouwazi`、实际用途、分发平台及运营安排。
- iOS 特征信息按办理平台要求从当前分发证书取得，包括公钥与证书 SHA-1；这不是域名的 HTTPS 证书。证书更新时核对是否需变更备案。[阿里特征信息说明](https://help.aliyun.com/zh/icp-filing/basic-icp-service/user-guide/fill-in-app-feature-information)
- Android 当前只是调试签名试用包，不能把每次 CI 的临时签名当成持续分发的备案签名。

## 商店材料：必需与可选

| 材料 | 本项目准备方式 |
| --- | --- |
| 名称、描述、关键词 | 简体、繁体、英文；只描述真实随机选择、倒计时和离线功能 |
| 类别、年龄分级、内容权利 | Utilities（工具）；按随机选择功能完成问卷 |
| 隐私政策、隐私标签 | 公共网页和 App 内入口；核对应用与 SDK 不向设备外发送用户数据 |
| 支持 URL、版权、审核联系人 | 版权署名 `dzshzx`；审核联系人与公开邮箱用途分开 |
| iPhone/iPad 截图 | 当前应用真实运行画面，三语首页、抽选结果与隐私页 |
| 审核备注与演示 | 完整六项说明，以及从启动到抽选、再次抽选和隐私页的真机录屏 |
| 价格、全球供应与发布方式 | 175 个国家和地区免费供应，包含中国大陆 |
| App Preview、推广文字、Marketing URL、自定义 EULA | 可选，不是首次送审的必填项；暂不制作 |
| Product page Header / Search Results 创意素材 | 与截图独立，Apple 明确列为可选；暂不制作 |

Apple 要求所有 App 在商店资料和 App 内提供易于访问的隐私政策。仅在设备内处理、不发送到设备外的数据不属于其隐私标签定义的“收集”；是否申报 Data Not Collected 还须结合实际嵌入 SDK 和产物核对。[审核指南 5.1.1](https://developer.apple.com/app-store/review/guidelines/#privacy)、[隐私标签定义](https://developer.apple.com/app-store/app-privacy-details/)

### 支持与隐私网页

源文件在 `store/site/`，公开地址、邮箱与版权署名在 `store/site-config.json`。内容覆盖应用本地处理、平台诊断、主动联系支持和网站托管边界，不把网站访问日志混同为应用上传数据。

```sh
# 本地草稿：缺少公开信息时仍可预览，带显著草稿标记和 noindex。
bash scripts/tool.sh store-site --preview
# 完整信息确认后生成可部署文件；缺项会直接失败。
bash scripts/tool.sh store-site
```

输出为 `build/store-site/`，入口 `index.html`。发布 manifest 所列的九个公共文件，manifest 留在本地。页面无脚本、分析代码或登录要求；网站托管边界见网页隐私政策。草稿预览地址不能填写为正式商店 URL。

网页源码与生成产物由本应用仓库维护，发布到上述独立站点目录。正式生成前核对 HTTPS 地址、公开邮箱与实际托管政策；生成器会拒绝缺少公开信息的正式构建。部署按内部运维仓的部署契约执行。

### 当前截图规格

截至核对日，官方页面顶部要求至少一张 **iPhone 灵动岛中尺寸**截图；本项目也支持 iPad，所以还要至少一张 **iPad 13 英寸**截图。每种尺寸允许 1–10 张，无透明度。

- iPhone：1179 × 2556 或 1206 × 2622；本项目准备 1206 × 2622。
- iPad：2064 × 2752 或 2048 × 2732；本项目准备 2064 × 2752。
- 截图与用户将安装的版本和语言一致，保留来源 SHA、设备、运行时、尺寸和画面状态记录。模拟器截图不等于真机触控、震动验收。

自动入口是 Actions 的 `Capture iOS App Store screenshots`。

上传时使用当前 App Asset Library，在版本本地化下分别建立 iPhone medium 与 iPad 13 的截图位置。Apple 已将旧截图集 API 标记为弃用；程序化上传应从 `appAssetLibraryRefData` 读取当前规格与位置组，核对图片处理完成和各语言的顺序，不硬编码旧屏幕枚举。[Apple 资产库迁移说明](https://developer.apple.com/documentation/appstoreconnectapi/migrating-to-the-app-asset-library)

旧文章中的“只准备 6.5/6.9 英寸”清单不能直接套用新页面。Header/Search Results 是可选素材；新 iPhone Duo 截图到 2027 年 4 月才成为必需项。提交当天仍须以后台实际提示和[截图规格](https://developer.apple.com/help/app-store-connect/reference/app-information/screenshot-specifications/)、[素材说明](https://developer.apple.com/help/app-store-connect/manage-app-information/manage-your-app-store-assets)、[发布说明](https://developer.apple.com/help/app-store-connect/release-notes/)核对。

## 全球供应的当前事实与待办

1. 专用支持邮箱设有自动转发并保留原邮件。凭据与转发配置保存在私有凭据库，仓库不保存密码或验证码。
2. 免费发布与 DSA 非商家声明在商业活动变化时须重新评估。
3. 本 App 尚未办理 App 备案，1.0.4 已在中国大陆商店上架。Apple 或主管部门后续提出要求时按实际处理。

个人会员、免费发布都不自动等于欧盟 DSA 的 non-trader。Apple 以实际商业活动和意图自评；trader 的个人联系资料会公开。若 Business 页面出现中国国务院第 810 号令身份信息提示，也须按实际情况处理，这与应用收集用户数据是不同事项。[DSA 自评](https://developer.apple.com/help/app-store-connect/manage-compliance-information/manage-european-union-digital-services-act-trader-requirements)、[中国开发者信息](https://developer.apple.com/help/app-store-connect/manage-compliance-information/manage-information-for-state-council-decree-no-810)

全球供应目标包含中国大陆，按工具类目的实际功能、年龄问卷和后台提示核对地区要求。应用没有账号，适用义务应结合实际功能判断。[Apple 年龄保障说明](https://developer.apple.com/support/age-assurance)

## 社区一手经验：可借鉴与不能推断的部分

以下均读过作者正文或本人评论，日期是发布/补充日期，结果只是作者自述。没有最终结果的案例明确保留“未知”。

| 日期与来源 | 自述结果 | 对本项目的用处与边界 |
| --- | --- | --- |
| 2024-07-28，V2EX jamesdd：[离线 iOS app 也需要 ICP？](https://www.v2ex.com/t/1060673) | 明说不联网、无服务器，送审仍被要求备案号；没有公布解决结果 | 不能把“纯离线”视为后台自动放行；类目和最终判定未知 |
| 2024-12-15/16，V2EX Eyon：[首个计算器上架](https://www.v2ex.com/t/1097690) | 本地计算器已上线；作者走腾讯云域名/实例备案，称约七个工作日；曾因模仿官方界面被拒 | 个人小工具有成功路径；时长和购买实例是该个案，不是所有人的法定要求 |
| 2025-06-28/29，V2EX CL7：[调整定价触发国区缺备案号](https://www.v2ex.com/t/1141681) | 原本可销售，后因备案字段失去大陆供应；有内购，申诉结果未知 | 旧版曾上线不等于以后修改地区/定价仍可免补材料；不是纯离线可比样本 |
| 2024-11-05，Reddit Thin-Ad9372：[多次最低功能拒绝](https://www.reddit.com/r/iOSProgramming/comments/1gk5nle/app_keeps_getting_rejected_for_minimal_use/) | 图书推荐 App 五次被拒，涉及 4.2.2 和截图；没有后续通过结果 | 让审核员能理解实际用途，必要时给视频；不能据此预测抽袜子一定被拒 |
| 2025-04-03，Reddit HamsterBaseMaster：[六次拒绝后获批](https://www.reddit.com/r/iOSProgramming/comments/1jq44yf/the_app_was_rejected_6_times_before_finally/) | 说明原创性、购买路径及被误解的功能后获批 | 对具体问题给证据；作者对框架导致误判的猜测不是 Apple 结论 |
| 2025-04-24—05-03，Reddit obolli：[首次上架讨论](https://www.reddit.com/r/iOSProgramming/comments/1k6so1m/experienced_ios_devs_what_are_your_tips_to_get/)、[本人通过回报](https://www.reddit.com/r/iOSProgramming/comments/1k6so1m/comment/mqdxyza/) | 单一功能、无付费墙的首款 App 后来获批，未披露产品细节 | 功能聚焦不是必然被拒；不能据此保证首次通过 |
| 2024-06-23博客、2025-04-24补充，Kyra：[完整经历](https://www.simplykyra.com/blog/steps-needed-to-get-into-the-apple-app-store/)、[隐私入口补充](https://www.reddit.com/r/iOSProgramming/comments/1k6so1m/comment/mot0y65/) | 处理截图、权限和审核说明后上线；曾因审核员找不到隐私入口被拒 | 隐私入口与审核备注要容易找到；旧文的截图规格不照搬 |
| 2026-01-14，Reddit steve_s0：[Linux 与云端 iOS 构建](https://www.reddit.com/r/FlutterDev/comments/1qbmcf9/comment/nzkysto/) | 日常 Linux，用 Codemagic 构建发布，再找家人朋友用 TestFlight 真机测试 | 无 Mac 有实际交付路径；云端成功不能替代触控/震动实测 |

本项目据此准备明确的使用步骤、可找到的隐私入口、当前规格截图和针对性的审核备注。若收到审核反馈，按实际条款补充证据或修复。
