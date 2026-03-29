import Combine
import Foundation

@MainActor
final class TailscaleDiscoveryService: ObservableObject {
    struct ProbeResult: Equatable {
        let health: DaemonHealth?
        let errorMessage: String?
    }
    
    struct MachineStatus: Identifiable, Equatable {
        let machine: Machine
        let health: DaemonHealth?
        let errorMessage: String?
        
        var id: UUID { machine.id }
        var isReachable: Bool { health != nil }
    }
    
    struct DiscoveredPeer: Identifiable, Equatable {
        let peer: TailscalePeer
        let health: DaemonHealth?

        var id: String { peer.id }
        var isReachable: Bool { health != nil }
    }

    @Published private(set) var availability: TailscaleVPNDetector.Availability = .notInstalled
    @Published private(set) var machineStatuses: [MachineStatus] = []
    @Published private(set) var discoveredPeers: [DiscoveredPeer] = []
    @Published private(set) var isRefreshing = false
    @Published private(set) var lastRefreshAt: Date?

    private let detector: TailscaleVPNDetector
    private let bonjourBrowser: BonjourBrowser
    private let machinesProvider: @MainActor () -> [Machine]
    private let markSeen: @MainActor (UUID, Date) -> Void
    private let probeMachine: (Machine) async -> ProbeResult
    private let fetchPeers: (Machine) async -> [TailscalePeer]
    private let addMachine: @MainActor (Machine) -> Void
    private let linkProfile: @MainActor (Machine) -> Void
    private let refreshIntervalNanoseconds: UInt64
    private var refreshTask: Task<Void, Never>?

    init(
        detector: TailscaleVPNDetector? = nil,
        bonjourBrowser: BonjourBrowser? = nil,
        refreshIntervalNanoseconds: UInt64 = 5_000_000_000,
        machinesProvider: @escaping @MainActor () -> [Machine] = { MachineStore.shared.machines },
        markSeen: @escaping @MainActor (UUID, Date) -> Void = { machineID, date in
            MachineStore.shared.markSeen(machineID, at: date)
        },
        probeMachine: ((Machine) async -> ProbeResult)? = nil,
        fetchPeers: ((Machine) async -> [TailscalePeer])? = nil,
        addMachine: (@MainActor (Machine) -> Void)? = nil,
        linkProfile: (@MainActor (Machine) -> Void)? = nil
    ) {
        self.detector = detector ?? TailscaleVPNDetector()
        self.bonjourBrowser = bonjourBrowser ?? BonjourBrowser()
        self.refreshIntervalNanoseconds = refreshIntervalNanoseconds
        self.machinesProvider = machinesProvider
        self.markSeen = markSeen
        self.probeMachine = probeMachine ?? Self.probe(machine:)
        self.fetchPeers = fetchPeers ?? Self.defaultFetchPeers(machine:)
        self.addMachine = addMachine ?? { machine in MachineStore.shared.add(machine) }
        self.linkProfile = linkProfile ?? Self.defaultLinkProfile(machine:)
    }
    
    deinit {
        refreshTask?.cancel()
    }
    
    var activeMachines: [MachineStatus] {
        machineStatuses.filter(\.isReachable)
    }
    
    var inactiveMachines: [MachineStatus] {
        machineStatuses.filter { !$0.isReachable }
    }
    
    func startPolling() {
        guard refreshTask == nil else { return }

        bonjourBrowser.startBrowsing()

        refreshTask = Task { [weak self] in
            guard let self else { return }

            while !Task.isCancelled {
                await self.refresh()
                try? await Task.sleep(nanoseconds: self.refreshIntervalNanoseconds)
            }
        }
    }

    func stopPolling() {
        refreshTask?.cancel()
        refreshTask = nil
        bonjourBrowser.stopBrowsing()
    }
    
    func refresh() async {
        isRefreshing = true
        defer { isRefreshing = false }

        availability = detector.detectAvailability()

        // Auto-add any daemons discovered via Bonjour on the local network.
        let bonjourDaemons = bonjourBrowser.discoveredDaemons
        let existingMachines = machinesProvider()
        var knownMachineHosts = Set(existingMachines.map { Self.normalizedHost($0.daemonHost) })

        for daemon in bonjourDaemons {
            let host = Self.normalizedHost(daemon.host)
            guard !host.isEmpty, !knownMachineHosts.contains(host) else { continue }

            let candidate = Machine(
                name: daemon.name,
                daemonScheme: "http",
                daemonHost: daemon.host,
                daemonPort: daemon.port
            )

            let result = await probeMachine(candidate)
            if result.health != nil {
                addMachine(candidate)
                linkProfile(candidate)
                knownMachineHosts.insert(host)
            }
        }

        let machines = machinesProvider()
        let refreshDate = Date()
        var nextStatuses: [MachineStatus] = []
        
        for machine in machines {
            let result = await probeMachine(machine)
            if result.health != nil {
                markSeen(machine.id, refreshDate)
            }
            
            nextStatuses.append(
                MachineStatus(
                    machine: machine,
                    health: result.health,
                    errorMessage: result.errorMessage
                )
            )
        }
        
        machineStatuses = nextStatuses.sorted { lhs, rhs in
            if lhs.isReachable != rhs.isReachable {
                return lhs.isReachable && !rhs.isReachable
            }

            return lhs.machine.displayName.localizedCaseInsensitiveCompare(rhs.machine.displayName) == .orderedAscending
        }

        // Autodiscovery: query peers from the first reachable daemon
        var nextDiscovered: [DiscoveredPeer] = []
        if let reachable = nextStatuses.first(where: \.isReachable) {
            let peers = await fetchPeers(reachable.machine)
            var knownHosts = Set(machines.map { Self.normalizedHost($0.daemonHost) })
            var seenPeerHosts = Set<String>()

            for peer in peers where peer.online && !peer.isSelf {
                let peerHost = Self.normalizedHost(peer.dnsName)
                guard !peerHost.isEmpty else { continue }
                guard seenPeerHosts.insert(peerHost).inserted else { continue }
                guard !knownHosts.contains(peerHost) else { continue }

                let candidate = Machine(
                    name: peer.hostname,
                    daemonScheme: "https",
                    daemonHost: peer.dnsName,
                    daemonPort: 443
                )
                let probeResult = await probeMachine(candidate)
                nextDiscovered.append(DiscoveredPeer(peer: peer, health: probeResult.health))

                if probeResult.health != nil {
                    addMachine(candidate)
                    linkProfile(candidate)
                    knownHosts.insert(peerHost)
                }
            }
        }
        discoveredPeers = nextDiscovered

        lastRefreshAt = refreshDate
    }
    
    func openTailscaleApp() {
        detector.openApp()
    }
    
    func openInstallPage() {
        detector.openInstallPage()
    }
    
    private static func probe(machine: Machine) async -> ProbeResult {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.timeoutIntervalForRequest = 3
        configuration.timeoutIntervalForResource = 3

        do {
            let client = try DaemonClient(machine: machine, session: URLSession(configuration: configuration))
            let health = try await client.health()
            return ProbeResult(health: health, errorMessage: nil)
        } catch {
            return ProbeResult(health: nil, errorMessage: error.localizedDescription)
        }
    }

    private static func defaultLinkProfile(machine: Machine) {
        let manager = ConnectionProfileManager.shared
        let store = MachineStore.shared
        let targetHost = normalizedHost(machine.daemonHost)

        // Skip if this machine already has a linked profile
        guard let stored = store.machines.first(where: { normalizedHost($0.daemonHost) == targetHost }),
              stored.linkedProfileID == nil else {
            return
        }

        // Check if a profile already exists for this host
        if let existing = manager.profiles.first(where: { normalizedHost($0.host) == targetHost }) {
            var updated = stored
            updated.linkedProfileID = existing.id
            store.update(updated)
            return
        }

        // Create a new profile with sensible defaults
        let profile = ConnectionProfile(
            name: machine.displayName,
            host: machine.daemonHost,
            port: 22,
            username: "",
            authMethod: .password,
            useTmux: true
        )
        manager.addProfile(profile)

        var updated = stored
        updated.linkedProfileID = profile.id
        store.update(updated)
    }

    private static func defaultFetchPeers(machine: Machine) async -> [TailscalePeer] {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.timeoutIntervalForRequest = 5
        configuration.timeoutIntervalForResource = 5

        do {
            let client = try DaemonClient(machine: machine, session: URLSession(configuration: configuration))
            return try await client.peers()
        } catch {
            return []
        }
    }

    private static func normalizedHost(_ host: String) -> String {
        host
            .trimmingCharacters(in: .whitespacesAndNewlines)
            .trimmingCharacters(in: CharacterSet(charactersIn: "."))
            .lowercased()
    }
}
