
import { FlutterAbility, FlutterEngine } from '@ohos/flutter_ohos';
import window from '@ohos.window';
import { GeneratedPluginRegistrant } from '../plugins/GeneratedPluginRegistrant';

export default class EntryAbility extends FlutterAbility {
  onWindowStageCreate(windowStage: window.WindowStage) {
    super.onWindowStageCreate(windowStage);
    windowStage.getMainWindowSync().setWindowKeepScreenOn(true);
  }
  configureFlutterEngine(flutterEngine: FlutterEngine) {
    super.configureFlutterEngine(flutterEngine)
    GeneratedPluginRegistrant.registerWith(flutterEngine)
  }
}
