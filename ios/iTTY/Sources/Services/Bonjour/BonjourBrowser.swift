// iTTY — Bonjour Browser
//
// Browses the local network for iTTY daemons advertised via DNS-SD.
// Discovered services are resolved to hostnames and ports, then
// handed to TailscaleDiscoveryService for health probing.

import Foundation

@MainActor
final class BonjourBrowser: NSObject, ObservableObject {
    struct DiscoveredDaemon: Identifiable, Equatable {
        let id: String
        let name: String
        let host: String
        let port: Int
    }

    @Published private(set) var discoveredDaemons: [DiscoveredDaemon] = []

    private let browserFactory: () -> NetServiceBrowser
    private var browser: NetServiceBrowser?
    private var services: [ObjectIdentifier: NetService] = [:]
    private var resolving: Set<ObjectIdentifier> = []
    private var daemonIDsByService: [ObjectIdentifier: String] = [:]
    private var daemonsByID: [String: DiscoveredDaemon] = [:]

    init(browserFactory: @escaping () -> NetServiceBrowser = { NetServiceBrowser() }) {
        self.browserFactory = browserFactory
        super.init()
    }

    func startBrowsing() {
        guard browser == nil else { return }

        clearState(clearPublishedDaemons: true)

        let nextBrowser = browserFactory()
        nextBrowser.delegate = self
        nextBrowser.searchForServices(ofType: "_itty._tcp", inDomain: "local.")
        browser = nextBrowser
    }

    func stopBrowsing() {
        browser?.stop()
        browser?.delegate = nil
        browser = nil
        clearState(clearPublishedDaemons: true)
    }

    deinit {
        browser?.stop()
        browser?.delegate = nil
    }

    func upsertDiscoveredDaemon(name: String, host: String, port: Int) {
        upsertDiscoveredDaemon(name: name, host: host, port: port, service: nil)
    }

    private func upsertDiscoveredDaemon(name: String, host: String, port: Int, service: NetService?) {
        let cleanHost = normalizeHost(host)
        guard port > 0, !cleanHost.isEmpty else { return }

        let daemonID = Self.daemonID(host: cleanHost, port: port)
        daemonsByID[daemonID] = DiscoveredDaemon(
            id: daemonID,
            name: name,
            host: cleanHost,
            port: port
        )

        if let service {
            daemonIDsByService[ObjectIdentifier(service)] = daemonID
        }

        publishDaemons()
    }

    private func removeDiscoveredDaemon(for service: NetService) {
        let key = ObjectIdentifier(service)
        if let daemonID = daemonIDsByService.removeValue(forKey: key) {
            daemonsByID.removeValue(forKey: daemonID)
            publishDaemons()
        }
    }

    private func clearState(clearPublishedDaemons: Bool) {
        for service in services.values {
            service.stop()
            service.delegate = nil
        }

        services.removeAll()
        resolving.removeAll()
        daemonIDsByService.removeAll()

        if clearPublishedDaemons {
            daemonsByID.removeAll()
            discoveredDaemons = []
        }
    }

    private func publishDaemons() {
        discoveredDaemons = daemonsByID.values.sorted { lhs, rhs in
            lhs.id.localizedCaseInsensitiveCompare(rhs.id) == .orderedAscending
        }
    }

    private func normalizeHost(_ host: String) -> String {
        let trimmed = host.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return "" }

        if trimmed.hasSuffix(".") {
            return String(trimmed.dropLast()).lowercased()
        }

        return trimmed.lowercased()
    }

    private static func daemonID(host: String, port: Int) -> String {
        "\(host.lowercased()):\(port)"
    }
}

extension BonjourBrowser: NetServiceBrowserDelegate {
    nonisolated func netServiceBrowser(_ browser: NetServiceBrowser, didFind service: NetService, moreComing: Bool) {
        Task { @MainActor in
            let key = ObjectIdentifier(service)
            guard services[key] == nil else { return }

            services[key] = service
            service.delegate = self
            resolving.insert(key)
            service.resolve(withTimeout: 5)
        }
    }

    nonisolated func netServiceBrowser(_ browser: NetServiceBrowser, didRemove service: NetService, moreComing: Bool) {
        Task { @MainActor in
            let key = ObjectIdentifier(service)
            services.removeValue(forKey: key)
            resolving.remove(key)
            removeDiscoveredDaemon(for: service)
            service.stop()
            service.delegate = nil
        }
    }

    nonisolated func netServiceBrowser(_ browser: NetServiceBrowser, didNotSearch errorDict: [String: NSNumber]) {
        // Browse failed — not fatal, just means Bonjour unavailable.
    }
}

extension BonjourBrowser: NetServiceDelegate {
    nonisolated func netServiceDidResolveAddress(_ sender: NetService) {
        Task { @MainActor in
            resolving.remove(ObjectIdentifier(sender))

            let host = sender.hostName ?? sender.name
            upsertDiscoveredDaemon(name: sender.name, host: host, port: sender.port, service: sender)
        }
    }

    nonisolated func netService(_ sender: NetService, didNotResolve errorDict: [String: NSNumber]) {
        Task { @MainActor in
            resolving.remove(ObjectIdentifier(sender))
        }
    }
}
