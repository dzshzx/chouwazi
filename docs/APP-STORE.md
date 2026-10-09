# 抽袜子：App Store 发布与审核

核对日期：2026-10-09。目标为全球供应，包含中国大陆。本文把官方规则、开发者自述和本项目的待办分别记录；社区成功经历不构成审核或备案豁免保证。

## 当前基础与材料入口

- 个人 Apple 开发者会员已验证有效；1.0.4 (5) 已进入 TestFlight 内部测试。云端 macOS 的签名、上传和恢复流程见 [TESTFLIGHT.md](TESTFLIGHT.md)。
- 应用在设备内完成多指随机选择；没有账号、广告、内购、分析 SDK 或业务服务器。触点和结果只在运行时内存中处理。
- 支持与隐私页已发布到独立站点 [chouwazi.646598.xyz](https://chouwazi.646598.xyz/)，公开支持邮箱为 `dzshzxdev@163.com`，已实测收信；本 App 尚未办理 App 备案。
- [三语商店文案](../store/metadata.json)与[审核备注](../store/review-notes.txt)已提交；[中国大陆适用性咨询草稿](../store/mainland-inquiry.txt)保留，尚未发送。
- 已提交版本为 1.0.4 (5)，从 1.0.3 (4) 按 patch 更新三语说明，统一为离线随机选择工具。版权署名为 `dzshzx`，商店版权字段为 `2026 dzshzx`。
- App 内三语本地隐私页已实现，25 项测试与 [1.0.4 (5) 三端构建](https://github.com/dzshzx/chouwazi/actions/runs/37801507597)通过；官网三语支持与隐私页面已部署并完成匿名 HTTPS 验收，六个链接已填入 App Store Connect 并逐项回读。
- App Store Connect 已保存 1.0.4 的三语名称、副标题、描述和关键词，以及版权、Utilities 主类目和无需审核登录的设置；已选中构建 5，并保存用户提供的专用审核联系方式和审核备注。年龄问卷按实际功能完成，常规分级为 4+（地区分级以后台为准）；内容权利声明已保存，Data Not Collected 隐私标签已发布。
- 2026-10-09 08:47（Asia/Shanghai）回读版本状态为 `WAITING_FOR_REVIEW`，页面确认 `1 Item Submitted`；发布方式为 `AFTER_APPROVAL`，获批后自动发布。当前已送审，尚未审核通过或正式上架；大陆供应仍以实际审核及发布结果为准。
- 用户确认免费发布后，已保存 175 个国家和地区的免费价格和供应范围，包含中国大陆；当前大陆状态为 `Available on App Release`，尚不是已上线或备案适用性获批。DSA 已按个人非商家声明保存。iOS 分发设备为 iPhone、iPad；需要实体多指触屏，未启用 Mac 或 Apple Vision Pro 分发。

## 个人离线 App 的通常流程

| 顺序 | 工作 | 完成证据 |
| --- | --- | --- |
| 1 | 确定真实核心用途、免费/收费安排、发布地区与发布主体 | 与实际功能一致的类目；用户确认商业身份和地区 |
| 2 | 核对地区要求，尤其中国大陆 App 备案、欧盟 DSA | 对本 App 适用的资质或 Apple/接入商的明确答复；不沿用别人的个案结论 |
| 3 | 准备隐私政策、支持与联系方式 | 公开 HTTPS 页面可匿名访问，App 内容易找到隐私政策 |
| 4 | 真机与自动验证、生成商店截图 | iPhone/iPad 实测记录；构建/测试回执；当前版本实际截图 |
| 5 | 填写商店资料、隐私标签、年龄分级、审核联系人 | 各必填字段完整；无账号应用关闭审核登录要求 |
| 6 | 选择匹配版本的构建并提交 App Review | App Store Connect 显示已提交/待审核，而不只是已上传 |
| 7 | 按实际审核意见回复或修改，获批后发布 | 对目标地区读回可供下载的商店页面；逐地区核对供应状态 |

Apple 的[必填属性表](https://developer.apple.com/help/app-store-connect/reference/app-information/required-localizable-and-editable-properties)与[版本信息说明](https://developer.apple.com/help/app-store-connect/reference/app-information/platform-version-information/)是字段依据。当前已上传的构建可以作为基础；隐私入口等代码变更后须重新构建并增加构建号。

## 独立支持站点与备案

三语页面位于 `https://chouwazi.646598.xyz/` 的 `zh-Hans/`、`zh-Hant/`、`en/` 子目录下，文件名为 `privacy.html` 和 `support.html`。公司 VPS 的独立静态目录为 `/var/www/chouwazi`；Caddy 路由片段由 fleet-state 的 `deploy/vps/chouwazi/chouwazi.caddy` 维护。网站页脚显示现有网站备案信息，其用途与 App 备案不同。

工信部通知要求在境内从事 App 互联网信息服务的主办者备案，已有网站备案主体可补充 App 信息；网站备案号不能直接当成本 App 已备案的证明。官方通知未直接给出本项目这种纯本地随机选择应用的个案判定。[工信部通知](https://www.gov.cn/zhengce/zhengceku/202308/content_6897341.htm)、[工信部解读](https://wap.miit.gov.cn/zwgk/zcjd/art/2023/art_39b4f1acc36745b98478e0ec3e07128d.html)

Apple 当前说明是“部分 App”需要 ICP 信息；腾讯备案渠道 FAQ 则明确把上应用商店的单机离线 App 纳入办理口径。应向 Apple 确认本 App 的大陆供应要求，再向原接入商或所在地管局核对真实填报方式。备案表的后台域名应反映真实接入；本 App 没有业务服务器，隐私政策网站仅提供网页。[Apple App 信息](https://developer.apple.com/cn/help/app-store-connect/reference/app-information/app-information/)、[腾讯 FAQ](https://cloud.tencent.cn/document/product/243/97691)、[阿里 FAQ](https://help.aliyun.com/zh/icp-filing/basic-icp-service/support/basics-about-icp-filling-for-apps)

本 App 的核心用途是多人同时触屏后随机选出一位。产品定位为离线随机选择工具，商店主类目采用 Utilities（工具）；按这一真实用途准备描述、截图及审核说明。[Apple 类目说明](https://developer.apple.com/app-store/categories/)

若需 App 备案，可先整理下列资料，不把不适用字段编造完整：

- 现有备案主体、证件、省份、接入商、域名及备案号，由用户提供或确认；证件只交给实际办理平台。
- App 名称“抽袜子”、iOS Bundle ID `com.alex.chouwazi`、实际用途、分发平台及运营安排。
- iOS 特征信息按办理平台要求从当前分发证书取得，包括公钥与证书 SHA-1；这不是域名的 HTTPS 证书。证书更新时核对是否需变更备案。[阿里特征信息说明](https://help.aliyun.com/zh/icp-filing/basic-icp-service/user-guide/fill-in-app-feature-information)
- Android 当前只是调试签名试用包，不能把每次 CI 的临时签名当成持续分发的备案签名。

## 商店材料：必需与可选

| 材料 | 本项目准备方式 | 当前状态 |
| --- | --- | --- |
| 名称、描述、关键词 | 简体、繁体、英文；只描述真实随机选择、倒计时和离线功能 | 已随版本送审，与 metadata.json 三语内容逐项核对 |
| 类别、年龄分级、内容权利 | Utilities（工具）；按随机选择功能完成问卷 | 类目、年龄问卷与内容权利声明已保存 |
| 隐私政策、隐私标签 | 公共网页和 App 内入口；核对应用与 SDK 不向设备外发送用户数据 | 本地入口已实现，Data Not Collected 已发布；三语网页已上线，商店隐私链接已回读 |
| 支持 URL、版权、审核联系人 | 版权署名 `dzshzx`；审核联系人与公开邮箱用途分开 | 三语支持链接、版权与专用审核联系人已保存；公开邮箱已实测收信 |
| iPhone/iPad 截图 | 当前应用真实运行画面，三语首页、抽选结果与隐私页 | [云端截图运行](https://github.com/dzshzx/chouwazi/actions/runs/37806273467)成功；18 张 RGB PNG 已逐张检查并上传，六组顺序、尺寸与文件大小均已回读核对 |
| 审核备注与演示 | 清楚说明至少两根手指同时按住；建议补真人多指演示 | 备注已保存到后台；真机视频待录 |
| 价格、全球供应与发布方式 | 175 个国家和地区免费供应，包含中国大陆 | 价格与供应范围已保存，获批后自动发布；待实际审核结果 |
| App Preview、推广文字、Marketing URL、自定义 EULA | 可选，不是首次送审的必填项 | 暂不制作 |
| Product page Header / Search Results 创意素材 | 与截图独立，Apple 明确列为可选 | 暂不制作 |

Apple 要求所有 App 在商店资料和 App 内提供易于访问的隐私政策。仅在设备内处理、不发送到设备外的数据不属于其隐私标签定义的“收集”；是否申报 Data Not Collected 还须结合实际嵌入 SDK 和产物核对。[审核指南 5.1.1](https://developer.apple.com/app-store/review/guidelines/#privacy)、[隐私标签定义](https://developer.apple.com/app-store/app-privacy-details/)

### 支持与隐私网页

源文件在 `store/site/`，公开地址、邮箱与版权署名在 `store/site-config.json`。内容覆盖应用本地处理、平台诊断、主动联系支持和网站托管边界，不把网站访问日志混同为应用上传数据。

```sh
# 本地草稿：缺少公开信息时仍可预览，带显著草稿标记和 noindex。
uv run --locked --script scripts/render-store-site.py --preview
# 完整信息确认后生成可部署文件；缺项会直接失败。
uv run --locked --script scripts/render-store-site.py
```

输出为 `build/store-site/`，入口 `index.html`。发布 manifest 所列的九个公共文件，manifest 留在本地。2026-10-09 已逐文件核对线上 HTTPS 响应与本地产物字节一致，语言切换、联系链接和备案图标检查通过。页面无脚本、分析代码或登录要求；网站托管边界见网页隐私政策。草稿预览地址不能填写为正式商店 URL。

网页源码与生成产物由本应用仓库维护，发布到上述独立站点目录。正式生成前核对 HTTPS 地址、公开邮箱与实际托管政策；生成器会拒绝缺少公开信息的正式构建。部署按 fleet-state 的 chouwazi 部署契约执行，保留源文件摘要和线上验收回执。

### 当前截图规格

截至核对日，官方页面顶部要求至少一张 **iPhone 灵动岛中尺寸**截图；本项目也支持 iPad，所以还要至少一张 **iPad 13 英寸**截图。每种尺寸允许 1–10 张，无透明度。

- iPhone：1179 × 2556 或 1206 × 2622；本项目准备 1206 × 2622。
- iPad：2064 × 2752 或 2048 × 2732；本项目准备 2064 × 2752。
- 截图与用户将安装的版本和语言一致，保留来源 SHA、设备、运行时、尺寸和画面状态记录。模拟器截图不等于真机触控、震动验收。

自动入口是 Actions 的 `Capture iOS App Store screenshots`。2026-10-09 已生成并检查 1.0.4 (5) 的 18 张实际截图：两种设备、三种语言、首页/抽选结果/隐私页各一张。产物 `chouwazi-ios-app-store-screenshots` 包含原始尺寸 PNG 与 manifest；本次截图源码 `3666b60` 的应用代码与 TestFlight 构建 `3f85685` 一致。

18 张图片已随三语版本送审。提交前回读时，图片状态均为 `PREPARE_FOR_SUBMISSION`，截图位置均为 `ACTIVE`。每种语言分别有 iPhone medium 和 iPad 13 两组，各按首页、结果、隐私排序；已核对上传文件大小、服务端尺寸与本地产物一致。

上传时使用当前 App Asset Library，在版本本地化下分别建立 iPhone medium 与 iPad 13 的截图位置。Apple 已将旧截图集 API 标记为弃用；程序化上传应从 `appAssetLibraryRefData` 读取当前规格与位置组，核对图片处理完成和各语言的顺序，不硬编码旧屏幕枚举。[Apple 资产库迁移说明](https://developer.apple.com/documentation/appstoreconnectapi/migrating-to-the-app-asset-library)

旧文章中的“只准备 6.5/6.9 英寸”清单不能直接套用新页面。Header/Search Results 是可选素材；新 iPhone Duo 截图到 2027 年 4 月才成为必需项。提交当天仍须以后台实际提示和[截图规格](https://developer.apple.com/help/app-store-connect/reference/app-information/screenshot-specifications/)、[素材说明](https://developer.apple.com/help/app-store-connect/manage-app-information/manage-your-app-store-assets)、[发布说明](https://developer.apple.com/help/app-store-connect/release-notes/)核对。

## 全球供应的当前事实与待办

1. 独立支持站点已上线，专用邮箱已注册并实测收信；自动转发至用户指定 Gmail 仍需完成网易安全验证与实际转发验收。凭据保存在 Bitwarden，仓库不保存密码或验证码。
2. 用户确认免费身份，已保存免费价格和 DSA 非商家声明；未来商业活动变化时须重新评估。
3. 用户确认本 App 尚未正式备案；当前后台允许选中大陆供应，提审与实际上线仍须核对 Apple 的最终要求。

个人会员、免费发布都不自动等于欧盟 DSA 的 non-trader。Apple 以实际商业活动和意图自评；trader 的个人联系资料会公开。若 Business 页面出现中国国务院第 810 号令身份信息提示，也须按实际情况处理，这与应用收集用户数据是不同事项。[DSA 自评](https://developer.apple.com/help/app-store-connect/manage-compliance-information/manage-european-union-digital-services-act-trader-requirements)、[中国开发者信息](https://developer.apple.com/help/app-store-connect/manage-compliance-information/manage-information-for-state-council-decree-no-810)

全球供应继续覆盖中国大陆，按工具类目的实际功能、年龄问卷和后台提示核对地区要求。应用没有账号，适用义务应结合实际功能判断。[Apple 年龄保障说明](https://developer.apple.com/support/age-assurance)

## 社区一手经验：可借鉴与不能推断的部分

以下均读过作者正文或本人评论，日期是发布/补充日期，结果只是作者自述。没有最终结果的案例明确保留“未知”。

| 日期与来源 | 自述结果 | 对本项目的用处与边界 |
| --- | --- | --- |
| 2024-07-28，V2EX jamesdd：[离线 iOS app 也需要 ICP？](https://www.v2ex.com/t/1060673) | 明说不联网、无服务器，送审仍被要求备案号；没有公布解决结果 | 不能把“纯离线”视为后台自动放行；类目和最终判定未知 |
| 2024-12-15/16，V2EX Eyon：[首个计算器上架](https://www.v2ex.com/t/1097690) | 本地计算器已上线；作者走腾讯云域名/实例备案，称约七个工作日；曾因模仿官方界面被拒 | 个人小工具有成功路径；时长和购买实例是该个案，不是所有人的法定要求 |
| 2025-06-28/29，V2EX CL7：[调整定价触发国区缺备案号](https://www.v2ex.com/t/1141681) | 原本可销售，后因备案字段失去大陆供应；有内购，申诉结局未知 | 旧版曾上线不等于以后修改地区/定价仍可免补材料；不是纯离线可比样本 |
| 2024-11-05，Reddit Thin-Ad9372：[多次最低功能拒绝](https://www.reddit.com/r/iOSProgramming/comments/1gk5nle/app_keeps_getting_rejected_for_minimal_use/) | 图书推荐 App 五次被拒，涉及 4.2.2 和截图；没有后续通过结果 | 让审核员能理解实际用途，必要时给视频；不能据此预测抽袜子一定被拒 |
| 2025-04-03，Reddit HamsterBaseMaster：[六次拒绝后获批](https://www.reddit.com/r/iOSProgramming/comments/1jq44yf/the_app_was_rejected_6_times_before_finally/) | 说明原创性、购买路径及被误解的功能后获批 | 对具体问题给证据；作者对框架导致误判的猜测不是 Apple 结论 |
| 2025-04-24—05-03，Reddit obolli：[首次上架讨论](https://www.reddit.com/r/iOSProgramming/comments/1k6so1m/experienced_ios_devs_what_are_your_tips_to_get/)、[本人通过回报](https://www.reddit.com/r/iOSProgramming/comments/1k6so1m/comment/mqdxyza/) | 单一功能、无付费墙的首款 App 后来获批，未披露产品细节 | 功能聚焦不是必然被拒；不能据此保证首次通过 |
| 2024-06-23博客、2025-04-24补充，Kyra：[完整经历](https://www.simplykyra.com/blog/steps-needed-to-get-into-the-apple-app-store/)、[隐私入口补充](https://www.reddit.com/r/iOSProgramming/comments/1k6so1m/comment/mot0y65/) | 处理截图、权限和审核说明后上线；曾因审核员找不到隐私入口被拒 | 隐私入口与审核备注要容易找到；旧文的截图规格不照搬 |
| 2026-01-14，Reddit steve_s0：[Linux 与云端 iOS 构建](https://www.reddit.com/r/FlutterDev/comments/1qbmcf9/comment/nzkysto/) | 日常 Linux，用 Codemagic 构建发布，再找家人朋友用 TestFlight 真机测试 | 无 Mac 有实际交付路径；云端成功不能替代触控/震动实测 |

本项目据此准备明确的使用步骤、可找到的隐私入口、当前规格截图和针对性的审核备注。若收到审核反馈，按实际条款补充证据或修复。
