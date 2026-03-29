import Foundation
import XCTest
@testable import iTTY

@MainActor
final class MachineStoreTests: XCTestCase {
    func testMachineBuildsDaemonBaseURL() {
        let machine = Machine(name: "Desk", daemonScheme: "http", daemonHost: "100.64.0.10", daemonPort: 8080)
        
        XCTAssertEqual(machine.displayName, "Desk")
        XCTAssertEqual(machine.daemonAuthority, "100.64.0.10:8080")
        XCTAssertEqual(machine.daemonBaseURL?.absoluteString, "http://100.64.0.10:8080")
    }
    
    func testMachineDecodingDefaultsLegacyFields() throws {
        let payload = """
        {
          "id": "\(UUID())",
          "name": "Legacy",
          "daemonHost": "desktop.tailnet.ts.net"
        }
        """
        
        let machine = try JSONDecoder().decode(Machine.self, from: Data(payload.utf8))
        
        XCTAssertEqual(machine.daemonScheme, "http")
        XCTAssertEqual(machine.daemonPort, 3420)
        XCTAssertFalse(machine.isFavorite)
    }
    
    func testStorePersistsAndSortsFavoritesFirst() throws {
        let suiteName = "MachineStoreTests.\(UUID().uuidString)"
        let defaults = try XCTUnwrap(UserDefaults(suiteName: suiteName))
        defaults.removePersistentDomain(forName: suiteName)
        
        let store = MachineStore(defaults: defaults, storageKey: "machines.test")
        store.add(Machine(name: "B", daemonHost: "b.local"))
        store.add(Machine(name: "A", daemonHost: "a.local", isFavorite: true))
        
        XCTAssertEqual(store.machines.map(\.displayName), ["A", "B"])
        
        let reloaded = MachineStore(defaults: defaults, storageKey: "machines.test")
        XCTAssertEqual(reloaded.machines.map(\.displayName), ["A", "B"])
        XCTAssertEqual(reloaded.favorites.count, 1)
    }
}
