import UIKit

@main
final class AppDelegate: UIResponder, UIApplicationDelegate {
    var window: UIWindow?
    func application(_ application: UIApplication, didFinishLaunchingWithOptions options: [UIApplication.LaunchOptionsKey: Any]?) -> Bool {
        let window = UIWindow(frame: UIScreen.main.bounds)
        window.rootViewController = GameViewController()
        window.makeKeyAndVisible(); self.window = window
        return true
    }
    func applicationWillResignActive(_ application: UIApplication) {
        (window?.rootViewController as? GameViewController)?.game.pause()
        application.isIdleTimerDisabled = false
    }
    func applicationDidBecomeActive(_ application: UIApplication) {
        (window?.rootViewController as? GameViewController)?.game.resume()
        application.isIdleTimerDisabled = true
    }
}

final class GameViewController: UIViewController {
    let game = GameView()
    override func viewDidLoad() {
        super.viewDidLoad()
        view.backgroundColor = UIColor(red: 13/255, green: 17/255, blue: 27/255, alpha: 1)
        view.addSubview(game); game.translatesAutoresizingMaskIntoConstraints = false
        NSLayoutConstraint.activate([
            game.leadingAnchor.constraint(equalTo: view.safeAreaLayoutGuide.leadingAnchor),
            game.trailingAnchor.constraint(equalTo: view.safeAreaLayoutGuide.trailingAnchor),
            game.topAnchor.constraint(equalTo: view.safeAreaLayoutGuide.topAnchor),
            game.bottomAnchor.constraint(equalTo: view.safeAreaLayoutGuide.bottomAnchor)
        ])
    }
    override var preferredStatusBarStyle: UIStatusBarStyle { .lightContent }
    override var supportedInterfaceOrientations: UIInterfaceOrientationMask { .portrait }
}
