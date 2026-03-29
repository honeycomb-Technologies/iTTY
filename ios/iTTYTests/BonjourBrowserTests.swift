import XCTest
@testable import iTTY

@MainActor
final class BonjourBrowserTests: XCTestCase {
    func testUpsertDiscoveredDaemonDeduplicatesHostCaseInsensitively() {
        let browser = BonjourBrowser()

        browser.upsertDiscoveredDaemon(name: "Desk", host: "desk.local.", port: 3420)
        browser.upsertDiscoveredDaemon(name: "Desk Updated", host: "DESK.local", port: 3420)

        XCTAssertEqual(browser.discoveredDaemons.count, 1)
        XCTAssertEqual(browser.discoveredDaemons[0].id, "desk.local:3420")
        XCTAssertEqual(browser.discoveredDaemons[0].host, "desk.local")
        XCTAssertEqual(browser.discoveredDaemons[0].name, "Desk Updated")
    }

    func testStopBrowsingClearsDiscoveredDaemons() {
        let browser = BonjourBrowser()
        browser.upsertDiscoveredDaemon(name: "Desk", host: "desk.local", port: 3420)

        browser.stopBrowsing()

        XCTAssertTrue(browser.discoveredDaemons.isEmpty)
    }
}
