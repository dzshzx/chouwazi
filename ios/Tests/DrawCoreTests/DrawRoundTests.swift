import XCTest
@testable import DrawCore
final class DrawRoundTests: XCTestCase {
    func testCountdownAndPointerIdentity() {
        var r = DrawRound()
        r.join(id: 17, x: 1, y: 2, now: 0); XCTAssertNil(r.deadline)
        r.join(id: 90, x: 4, y: 5, now: 1); XCTAssertEqual(r.deadline, 4.8)
        r.move(id: 17, x: 5, y: 8); XCTAssertEqual(r.deadline, 4.8)
        XCTAssertFalse(r.tick(now: 4.79))
        r.leave(id: 90, now: 4.79); XCTAssertNil(r.deadline)
        r.join(id: 55, x: 0, y: 0, now: 5)
        XCTAssertTrue(r.tick(now: 8.8, choose: { _ in 1 })); XCTAssertEqual(r.winnerID, 55)
        r.leave(id: 55, now: 9); r.move(id: 55, x: 200, y: 200)
        XCTAssertEqual(r.winner?.x, 0); XCTAssertEqual(r.players.count, 2)
        r.reset(); XCTAssertFalse(r.isFinished); XCTAssertTrue(r.players.isEmpty)
    }
    func testTenContactLimitAndSlotReuse() {
        var r = DrawRound()
        for id in 0..<11 { r.join(id: id, x: 0, y: 0, now: 0) }
        XCTAssertEqual(r.players.count, 10)
        r.leave(id: 4, now: 1); r.join(id: 99, x: 0, y: 0, now: 2)
        XCTAssertEqual(r.players.last?.slot, 4); XCTAssertEqual(r.deadline, 5.8)
        r.join(id: 99, x: 0, y: 0, now: 3); XCTAssertEqual(r.deadline, 5.8)
    }
    func testEveryPlayerCanBeChosen() {
        for selected in 0..<10 {
            var r = DrawRound()
            for id in 0..<10 { r.join(id: id * 7, x: 0, y: 0, now: 0) }
            XCTAssertTrue(r.tick(now: 3.8, choose: { count in XCTAssertEqual(count, 10); return selected }))
            XCTAssertEqual(r.winnerID, selected * 7)
            XCTAssertFalse(r.tick(now: 5))
        }
    }
}
