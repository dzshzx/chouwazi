import UIKit

private func tr(_ key: String) -> String { NSLocalizedString(key, comment: "") }
private extension UIColor {
    convenience init(hex: UInt32) { self.init(red: CGFloat((hex >> 16) & 255)/255, green: CGFloat((hex >> 8) & 255)/255, blue: CGFloat(hex & 255)/255, alpha: 1) }
}

final class GameView: UIView {
    private var round = DrawRound()
    private var contacts: [ObjectIdentifier: Int] = [:]
    private var nextID = 0
    private var timer: Timer?
    private var haptics = true
    private var lastSecond = -1
    private let feedback = UIImpactFeedbackGenerator(style: .light)
    private let notification = UINotificationFeedbackGenerator()
    private let colors: [UIColor] = [0xB5F36D,0x8CB8FF,0xFFADCA,0xFFD478,0xC3A0FF,0x69DDCA,0xFF967F,0x98E2FF,0xE7A7EC,0xE5EA8B].map { UIColor(hex: $0) }
    private let retry = UIButton(type: .system)
    private let vibration = UIButton(type: .system)
    private var now: TimeInterval { ProcessInfo.processInfo.systemUptime }

    init() {
        super.init(frame: .zero)
        isMultipleTouchEnabled = true
        backgroundColor = UIColor(hex: 0x0D111B)
        retry.backgroundColor = colors[0]; retry.setTitleColor(UIColor(hex: 0x152010), for: .normal)
        retry.titleLabel?.font = .systemFont(ofSize: 17, weight: .semibold)
        retry.layer.cornerRadius = 18
        retry.addTarget(self, action: #selector(restart), for: .touchUpInside)
        addSubview(retry)
        vibration.titleLabel?.font = .systemFont(ofSize: 11)
        vibration.setTitleColor(UIColor(hex: 0x8B96AA), for: .normal)
        vibration.addTarget(self, action: #selector(toggleVibration), for: .touchUpInside)
        addSubview(vibration)
        updateControls()
    }
    required init?(coder: NSCoder) { fatalError("init(coder:) is unsupported") }
    override func layoutSubviews() {
        super.layoutSubviews()
        retry.frame = CGRect(x: 28, y: bounds.height-90, width: max(0,bounds.width-56), height: 54)
        vibration.frame = CGRect(x: bounds.width-108, y: 0, width: 100, height: 44)
    }
    func resume() {
        guard timer == nil else { return }
        let t = Timer(timeInterval: 1.0/30, repeats: true) { [weak self] _ in self?.frame() }
        RunLoop.main.add(t, forMode: .common); timer = t
    }
    func pause() {
        timer?.invalidate(); timer = nil; contacts.removeAll()
        if !round.isFinished { round.reset() }
        updateControls(); setNeedsDisplay()
    }
    deinit { timer?.invalidate() }
    @objc private func toggleVibration() { haptics.toggle(); updateControls() }
    @objc private func restart() {
        guard contacts.isEmpty else { return }
        round.reset(); lastSecond = -1; updateControls(); setNeedsDisplay()
    }
    private func updateControls() {
        retry.isHidden = !round.isFinished
        retry.isEnabled = contacts.isEmpty
        retry.alpha = retry.isEnabled ? 1 : 0.45
        retry.setTitle(tr(contacts.isEmpty ? "again" : "release"), for: .normal)
        vibration.isEnabled = contacts.isEmpty
        vibration.setTitle(tr(haptics ? "haptics_on" : "haptics_off"), for: .normal)
    }
    private func frame() {
        if round.tick(now: now) {
            if haptics { notification.notificationOccurred(.success) }
            if let winner = round.winner { UIAccessibility.post(notification: .announcement, argument: String(format: tr("winner"), winner.slot+1)) }
            updateControls()
        }
        if let end = round.deadline {
            let remaining = end-now
            let second = Int(ceil(min(3, remaining)))
            if remaining <= 3 && second != lastSecond {
                lastSecond = second
                if haptics { feedback.impactOccurred() }
            }
        }
        if !round.players.isEmpty { setNeedsDisplay() }
    }
    override func touchesBegan(_ touches: Set<UITouch>, with event: UIEvent?) {
        for touch in touches where touch.type == .direct {
            let key = ObjectIdentifier(touch)
            guard contacts[key] == nil else { continue }
            nextID += 1; contacts[key] = nextID
            let p = touch.location(in: self)
            round.join(id: nextID, x: Double(p.x), y: Double(p.y), now: now)
        }
        lastSecond = -1; if haptics && !round.isFinished { feedback.impactOccurred() }
        updateControls(); setNeedsDisplay()
    }
    override func touchesMoved(_ touches: Set<UITouch>, with event: UIEvent?) {
        for touch in touches {
            guard let id = contacts[ObjectIdentifier(touch)] else { continue }
            let p = touch.location(in: self); round.move(id: id, x: Double(p.x), y: Double(p.y))
        }
        setNeedsDisplay()
    }
    override func touchesEnded(_ touches: Set<UITouch>, with event: UIEvent?) { end(touches) }
    override func touchesCancelled(_ touches: Set<UITouch>, with event: UIEvent?) { end(touches) }
    private func end(_ touches: Set<UITouch>) {
        for touch in touches {
            if let id = contacts.removeValue(forKey: ObjectIdentifier(touch)) { round.leave(id: id, now: now) }
        }
        lastSecond = -1; updateControls(); setNeedsDisplay()
    }
    private func label(_ string: String, x: CGFloat, y: CGFloat, size: CGFloat, color: UIColor, bold: Bool = false) {
        let style = NSMutableParagraphStyle(); style.alignment = .center
        let font = UIFont.systemFont(ofSize: size, weight: bold ? .bold : .regular)
        let width = min(bounds.width-24, CGFloat(500))
        (string as NSString).draw(in: CGRect(x: x-width/2, y: y, width: width, height: size*2.5), withAttributes: [.font: font, .foregroundColor: color, .paragraphStyle: style])
    }
    private func ring(_ center: CGPoint, radius: CGFloat, color: UIColor, width: CGFloat) {
        let path = UIBezierPath(arcCenter: center, radius: radius, startAngle: 0, endAngle: .pi*2, clockwise: true)
        color.setStroke(); path.lineWidth = width; path.stroke()
    }
    private func disk(_ center: CGPoint, radius: CGFloat, color: UIColor) {
        color.setFill(); UIBezierPath(ovalIn: CGRect(x: center.x-radius, y: center.y-radius, width: radius*2, height: radius*2)).fill()
    }
    override func draw(_ rect: CGRect) {
        let cx = bounds.midX, h = bounds.height, muted = UIColor(hex: 0x8B96AA)
        for x in stride(from: CGFloat(20), to: bounds.width, by: CGFloat(28)) {
            for y in stride(from: CGFloat(155), to: max(155,h-115), by: CGFloat(28)) { disk(CGPoint(x:x,y:y), radius: 0.8, color: UIColor(hex: 0x18202C)) }
        }
        label("CHOUWAZI", x:cx, y:12, size:10, color:muted, bold:true)
        label(tr("app_name"), x:cx, y:43, size:28, color:.white, bold:true)
        label(tr("tagline"), x:cx, y:83, size:12, color:muted)
        let status = round.winner.map { String(format: tr("winner"), $0.slot+1) } ?? String(format: tr("joined"), round.players.count)
        label(status, x:cx, y:121, size:15, color:round.winner.map { colors[$0.slot] } ?? colors[0], bold:true)
        let mid = max(210, h*0.46)
        if !round.isFinished && round.players.count < 2 {
            ring(CGPoint(x:cx,y:mid), radius:48, color:UIColor(hex:0x283C38), width:2)
            ring(CGPoint(x:cx,y:mid), radius:35, color:UIColor(hex:0x536D49), width:2)
            label("+", x:cx, y:mid-23, size:36, color:colors[0])
            label(tr(round.players.isEmpty ? "place" : "one_more"), x:cx, y:mid+63, size:20, color:.white, bold:true)
            label(tr("range"), x:cx, y:mid+94, size:13, color:muted)
        }
        if let end = round.deadline {
            let remain = end-now
            label(remain > 3 ? tr("ready") : String(max(1,Int(ceil(remain)))), x:cx, y:mid-52, size:remain > 3 ? 40 : 76, color:UIColor(hex:0xE7EDE5), bold:true)
            label(tr("hold"), x:cx, y:mid+42, size:13, color:muted)
        }
        for player in round.players {
            let center = CGPoint(x:CGFloat(player.x),y:CGFloat(player.y)), selected = round.winnerID == player.id
            let dim = round.isFinished && !selected, color = colors[player.slot]
            let r = CGFloat(selected ? 49 : 36) + CGFloat(sin(now*4.3 + Double(player.slot)))*2
            disk(center, radius:r+12, color:color.withAlphaComponent(dim ? 0.05 : 0.14))
            ring(center, radius:r, color:dim ? UIColor(hex:0x384050) : color, width:selected ? 5 : 3)
            disk(center, radius:23, color:dim ? UIColor(hex:0x202735) : color)
            label(String(player.slot+1), x:center.x, y:center.y-11, size:17, color:UIColor(hex:0x0D111B), bold:true)
            if selected {
                label(tr("you"), x:center.x, y:max(151,center.y-r-44), size:19, color:color, bold:true)
                for i in 0..<12 {
                    let angle = Double(i)*Double.pi/6
                    disk(CGPoint(x:center.x+CGFloat(cos(angle))*(r+22),y:center.y+CGFloat(sin(angle))*(r+22)), radius:2, color:color)
                }
            }
        }
        if !round.isFinished { label(tr("restart_hint"), x:cx, y:h-73, size:12, color:muted) }
        label(tr("footer"), x:cx, y:h-25, size:10, color:UIColor(hex:0x687488))
    }
}
