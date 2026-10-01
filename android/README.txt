抽袜子 / chouwazi 1.0.1

安装：在 Android 6.0 或更高版本手机打开 chouwazi-1.0.1.apk，按系统提示允许此次安装。
玩法：至少两根手指按住屏幕，参与者不再增减后，经过短暂准备和三秒倒计时，等概率选中一根手指。保持结果，全部松开后点击“再来一轮”。
上限：10 根手指；实际识别数量取决于手机硬件和系统手势设置。若三指截图等手势打断触控，可在系统中关闭相应手势再试。
隐私：完全离线，只申请震动权限。没有网络权限、广告、账户或数据上传。

工程：Java 原生 Activity + Canvas；不依赖第三方运行库。app/src/main 为完整源码。
Android Studio 可打开本目录，使用 JDK 17、Android SDK 35、Gradle 8.9 同步并构建 app。工程未附 Gradle wrapper。
也可使用 build-local.sh 通过 Android 官方 build-tools 35.0.0 直接构建；设置 ANDROID_JAR、BUILD_TOOLS 指向本机 SDK。需要 Java 17、Python 3。
此安装包为个人使用自签名版本，非应用商店发布版本。

验证：Java 源码编译成功；APK v1/v2/v3 签名校验通过；DEX 中主 Activity 完整。
核心状态测试涵盖 pointer ID 跟踪、十指上限、增减参与者重新计时、移动不重置计时、结果冻结、重开及随机分布冒烟测试。
未进行 Android 模拟器或真机触控测试，实际十指触控、显示效果及振动需在设备上确认。
