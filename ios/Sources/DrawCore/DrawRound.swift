import Foundation

/// Each active contact gets a stable ID; joins/leaves restart the countdown.
public struct DrawRound {
    public struct Player: Equatable {
        public let id: Int
        public let slot: Int
        public var x: Double
        public var y: Double
    }
    public static let limit = 10
    public static let delay: TimeInterval = 3.8
    public private(set) var players: [Player] = []
    public private(set) var deadline: TimeInterval?
    public private(set) var winnerID: Int?
    public var isFinished: Bool { winnerID != nil }
    public var winner: Player? { players.first { $0.id == winnerID } }
    public init() {}

    public mutating func join(id: Int, x: Double, y: Double, now: TimeInterval) {
        guard !isFinished, players.count < Self.limit, !players.contains(where: { $0.id == id }) else { return }
        let used = Set(players.map(\.slot))
        guard let slot = (0..<Self.limit).first(where: { !used.contains($0) }) else { return }
        players.append(Player(id: id, slot: slot, x: x, y: y))
        restart(now)
    }
    public mutating func move(id: Int, x: Double, y: Double) {
        guard !isFinished, let i = players.firstIndex(where: { $0.id == id }) else { return }
        players[i].x = x; players[i].y = y
    }
    public mutating func leave(id: Int, now: TimeInterval) {
        guard !isFinished, players.contains(where: { $0.id == id }) else { return }
        players.removeAll { $0.id == id }; restart(now)
    }
    private mutating func restart(_ now: TimeInterval) {
        deadline = players.count >= 2 ? now + Self.delay : nil
    }
    @discardableResult public mutating func tick(now: TimeInterval, choose: (Int) -> Int = { Int.random(in: 0..<$0) }) -> Bool {
        guard !isFinished, let end = deadline, now >= end, players.count >= 2 else { return false }
        let index = choose(players.count)
        precondition(players.indices.contains(index))
        winnerID = players[index].id; deadline = nil; return true
    }
    public mutating func reset() { self = DrawRound() }
}
