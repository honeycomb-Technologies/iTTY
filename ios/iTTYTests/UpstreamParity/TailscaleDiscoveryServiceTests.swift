// iTTY — TailscaleDiscoveryService Tests

import XCTest
@testable import iTTY

@MainActor
final class TailscaleDiscoveryServiceTests: XCTestCase {

    // MARK: - Helpers

    private static let healthOK = DaemonHealth(
        status: "ok",
        version: "1.0",
        platform: "linux/amd64",
        tmuxInstalled: true,
        tmuxVersion: "3.6a"
    )

    private static func makePeer(
        id: String = "p1",
        hostname: String = "desktop",
        dnsName: String = "desktop.tail.ts.net",
        os: String = "linux",
        online: Bool = true,
        isSelf: Bool = false
    ) -> TailscalePeer {
        TailscalePeer(
            id: id,
            hostname: hostname,
            dnsName: dnsName,
            os: os,
            online: online,
            ips: ["100.1.1.2"],
            isSelf: isSelf
        )
    }

    private static func makeMachine(
        name: String = "my-mac",
        host: String = "my-mac.tail.ts.net",
        port: Int = 443,
        scheme: String = "https"
    ) -> Machine {
        Machine(name: name, daemonScheme: scheme, daemonHost: host, daemonPort: port)
    }

    // MARK: - Tests

    func testNoReachableDaemonSkipsPeerDiscovery() async {
        var addedMachines: [Machine] = []

        let service = TailscaleDiscoveryService(
            detector: TailscaleVPNDetector(),
            refreshIntervalNanoseconds: UInt64.max,
            machinesProvider: { [Self.makeMachine()] },
            markSeen: { _, _ in },
            probeMachine: { _ in
                .init(health: nil, errorMessage: "timeout")
            },
            fetchPeers: { _ in
                XCTFail("fetchPeers should not be called when no daemon is reachable")
                return []
            },
            addMachine: { addedMachines.append($0) }
        )

        await service.refresh()

        XCTAssertTrue(service.discoveredPeers.isEmpty)
        XCTAssertTrue(addedMachines.isEmpty)
    }

    func testReachableDaemonNoPeersReturnsEmpty() async {
        let service = TailscaleDiscoveryService(
            detector: TailscaleVPNDetector(),
            refreshIntervalNanoseconds: UInt64.max,
            machinesProvider: { [Self.makeMachine()] },
            markSeen: { _, _ in },
            probeMachine: { _ in
                .init(health: Self.healthOK, errorMessage: nil)
            },
            fetchPeers: { _ in [] },
            addMachine: { _ in }
        )

        await service.refresh()

        XCTAssertTrue(service.discoveredPeers.isEmpty)
    }

    func testDiscoversPeerWithDaemon() async {
        var addedMachines: [Machine] = []

        let peer = Self.makePeer(hostname: "desktop", dnsName: "desktop.tail.ts.net")

        let service = TailscaleDiscoveryService(
            detector: TailscaleVPNDetector(),
            refreshIntervalNanoseconds: UInt64.max,
            machinesProvider: { [Self.makeMachine(host: "my-mac.tail.ts.net")] },
            markSeen: { _, _ in },
            probeMachine: { _ in
                .init(health: Self.healthOK, errorMessage: nil)
            },
            fetchPeers: { _ in [peer] },
            addMachine: { addedMachines.append($0) }
        )

        await service.refresh()

        XCTAssertEqual(service.discoveredPeers.count, 1)
        XCTAssertTrue(service.discoveredPeers[0].isReachable)
        XCTAssertEqual(addedMachines.count, 1)
        XCTAssertEqual(addedMachines[0].daemonHost, "desktop.tail.ts.net")
        XCTAssertEqual(addedMachines[0].daemonScheme, "https")
        XCTAssertEqual(addedMachines[0].daemonPort, 443)
    }

    func testPeerAlreadyInStoreNotDuplicated() async {
        var addedMachines: [Machine] = []

        let peer = Self.makePeer(hostname: "desktop", dnsName: "desktop.tail.ts.net")

        let service = TailscaleDiscoveryService(
            detector: TailscaleVPNDetector(),
            refreshIntervalNanoseconds: UInt64.max,
            machinesProvider: {
                [
                    Self.makeMachine(name: "my-mac", host: "my-mac.tail.ts.net"),
                    Self.makeMachine(name: "desktop", host: "desktop.tail.ts.net"),
                ]
            },
            markSeen: { _, _ in },
            probeMachine: { _ in
                .init(health: Self.healthOK, errorMessage: nil)
            },
            fetchPeers: { _ in [peer] },
            addMachine: { addedMachines.append($0) }
        )

        await service.refresh()

        XCTAssertTrue(service.discoveredPeers.isEmpty, "Peer already in store should be skipped")
        XCTAssertTrue(addedMachines.isEmpty)
    }

    func testSelfPeerSkipped() async {
        var addedMachines: [Machine] = []

        let selfPeer = Self.makePeer(hostname: "my-mac", dnsName: "other.tail.ts.net", isSelf: true)

        let service = TailscaleDiscoveryService(
            detector: TailscaleVPNDetector(),
            refreshIntervalNanoseconds: UInt64.max,
            machinesProvider: { [Self.makeMachine(host: "my-mac.tail.ts.net")] },
            markSeen: { _, _ in },
            probeMachine: { _ in
                .init(health: Self.healthOK, errorMessage: nil)
            },
            fetchPeers: { _ in [selfPeer] },
            addMachine: { addedMachines.append($0) }
        )

        await service.refresh()

        XCTAssertTrue(service.discoveredPeers.isEmpty, "Self peer should be skipped")
        XCTAssertTrue(addedMachines.isEmpty)
    }

    func testOfflinePeerSkipped() async {
        var addedMachines: [Machine] = []

        let offlinePeer = Self.makePeer(hostname: "sleepy", dnsName: "sleepy.tail.ts.net", online: false)

        let service = TailscaleDiscoveryService(
            detector: TailscaleVPNDetector(),
            refreshIntervalNanoseconds: UInt64.max,
            machinesProvider: { [Self.makeMachine(host: "my-mac.tail.ts.net")] },
            markSeen: { _, _ in },
            probeMachine: { _ in
                .init(health: Self.healthOK, errorMessage: nil)
            },
            fetchPeers: { _ in [offlinePeer] },
            addMachine: { addedMachines.append($0) }
        )

        await service.refresh()

        XCTAssertTrue(service.discoveredPeers.isEmpty, "Offline peer should be skipped")
        XCTAssertTrue(addedMachines.isEmpty)
    }

    func testPeerProbeTimeoutNotAdded() async {
        var addedMachines: [Machine] = []
        var probeCount = 0

        let peer = Self.makePeer(hostname: "noresponse", dnsName: "noresponse.tail.ts.net")

        let service = TailscaleDiscoveryService(
            detector: TailscaleVPNDetector(),
            refreshIntervalNanoseconds: UInt64.max,
            machinesProvider: { [Self.makeMachine(host: "my-mac.tail.ts.net")] },
            markSeen: { _, _ in },
            probeMachine: { machine in
                probeCount += 1
                // First probe (saved machine) succeeds, second (discovered peer) fails
                if machine.daemonHost == "my-mac.tail.ts.net" {
                    return .init(health: Self.healthOK, errorMessage: nil)
                }
                return .init(health: nil, errorMessage: "timeout")
            },
            fetchPeers: { _ in [peer] },
            addMachine: { addedMachines.append($0) }
        )

        await service.refresh()

        XCTAssertEqual(service.discoveredPeers.count, 1)
        XCTAssertFalse(service.discoveredPeers[0].isReachable)
        XCTAssertTrue(addedMachines.isEmpty, "Unreachable peer should not be auto-added")
    }

    func testPeersFetchedFromFirstReachableOnly() async {
        var fetchCount = 0

        let service = TailscaleDiscoveryService(
            detector: TailscaleVPNDetector(),
            refreshIntervalNanoseconds: UInt64.max,
            machinesProvider: {
                [
                    Self.makeMachine(name: "mac1", host: "mac1.tail.ts.net"),
                    Self.makeMachine(name: "mac2", host: "mac2.tail.ts.net"),
                ]
            },
            markSeen: { _, _ in },
            probeMachine: { _ in
                .init(health: Self.healthOK, errorMessage: nil)
            },
            fetchPeers: { machine in
                fetchCount += 1
                return []
            },
            addMachine: { _ in }
        )

        await service.refresh()

        XCTAssertEqual(fetchCount, 1, "Peers should be fetched from only the first reachable daemon")
    }

    func testBonjourDuplicatesAreOnlyAutoAddedOnce() async {
        let browser = BonjourBrowser()
        browser.upsertDiscoveredDaemon(name: "Desk", host: "desk.local.", port: 3420)
        browser.upsertDiscoveredDaemon(name: "Desk Again", host: "DESK.local", port: 3420)

        var addedMachines: [Machine] = []

        let service = TailscaleDiscoveryService(
            detector: TailscaleVPNDetector(),
            bonjourBrowser: browser,
            refreshIntervalNanoseconds: UInt64.max,
            machinesProvider: { [] },
            markSeen: { _, _ in },
            probeMachine: { _ in
                .init(health: Self.healthOK, errorMessage: nil)
            },
            fetchPeers: { _ in [] },
            addMachine: { addedMachines.append($0) },
            linkProfile: { _ in }
        )

        await service.refresh()

        XCTAssertEqual(addedMachines.count, 1)
        XCTAssertEqual(addedMachines[0].daemonHost, "desk.local")
    }

    func testDuplicatePeerHostsAreOnlyAddedOnce() async {
        var addedMachines: [Machine] = []

        let duplicatePeers = [
            Self.makePeer(id: "peer-1", hostname: "desktop", dnsName: "desktop.tail.ts.net"),
            Self.makePeer(id: "peer-2", hostname: "desktop-alt", dnsName: "DESKTOP.tail.ts.net"),
        ]

        let service = TailscaleDiscoveryService(
            detector: TailscaleVPNDetector(),
            refreshIntervalNanoseconds: UInt64.max,
            machinesProvider: { [Self.makeMachine(host: "my-mac.tail.ts.net")] },
            markSeen: { _, _ in },
            probeMachine: { _ in
                .init(health: Self.healthOK, errorMessage: nil)
            },
            fetchPeers: { _ in duplicatePeers },
            addMachine: { addedMachines.append($0) },
            linkProfile: { _ in }
        )

        await service.refresh()

        XCTAssertEqual(service.discoveredPeers.count, 1)
        XCTAssertEqual(addedMachines.count, 1)
        XCTAssertEqual(addedMachines[0].daemonHost, "desktop.tail.ts.net")
    }
}
