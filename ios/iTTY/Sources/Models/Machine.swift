import Combine
import Foundation

struct Machine: Identifiable, Codable, Hashable {
    let id: UUID
    var name: String
    var daemonScheme: String
    var daemonHost: String
    var daemonPort: Int
    var linkedProfileID: UUID?
    var createdAt: Date
    var lastSeenAt: Date?
    var isFavorite: Bool
    
    init(
        id: UUID = UUID(),
        name: String,
        daemonScheme: String = "http",
        daemonHost: String,
        daemonPort: Int = 8080,
        linkedProfileID: UUID? = nil,
        createdAt: Date = Date(),
        lastSeenAt: Date? = nil,
        isFavorite: Bool = false
    ) {
        self.id = id
        self.name = name
        self.daemonScheme = daemonScheme
        self.daemonHost = daemonHost
        self.daemonPort = daemonPort
        self.linkedProfileID = linkedProfileID
        self.createdAt = createdAt
        self.lastSeenAt = lastSeenAt
        self.isFavorite = isFavorite
    }
    
    enum CodingKeys: String, CodingKey {
        case id
        case name
        case daemonScheme
        case daemonHost
        case daemonPort
        case linkedProfileID
        case createdAt
        case lastSeenAt
        case isFavorite
    }
    
    init(from decoder: Decoder) throws {
        let container = try decoder.container(keyedBy: CodingKeys.self)
        id = try container.decode(UUID.self, forKey: .id)
        name = try container.decode(String.self, forKey: .name)
        daemonScheme = try container.decodeIfPresent(String.self, forKey: .daemonScheme) ?? "http"
        daemonHost = try container.decode(String.self, forKey: .daemonHost)
        daemonPort = try container.decodeIfPresent(Int.self, forKey: .daemonPort) ?? 8080
        linkedProfileID = try container.decodeIfPresent(UUID.self, forKey: .linkedProfileID)
        createdAt = try container.decodeIfPresent(Date.self, forKey: .createdAt) ?? Date()
        lastSeenAt = try container.decodeIfPresent(Date.self, forKey: .lastSeenAt)
        isFavorite = try container.decodeIfPresent(Bool.self, forKey: .isFavorite) ?? false
    }
    
    var displayName: String {
        let trimmed = name.trimmingCharacters(in: .whitespacesAndNewlines)
        return trimmed.isEmpty ? daemonHost : trimmed
    }
    
    var daemonAuthority: String {
        switch (daemonScheme.lowercased(), daemonPort) {
        case ("http", 80), ("https", 443):
            return daemonHost
        default:
            return "\(daemonHost):\(daemonPort)"
        }
    }
    
    var daemonBaseURL: URL? {
        var components = URLComponents()
        components.scheme = daemonScheme.lowercased()
        components.host = daemonHost
        components.port = daemonPort
        return components.url
    }
}

struct DaemonHealth: Codable, Hashable {
    let status: String
    let version: String
    let platform: String
    let tmuxInstalled: Bool
    let tmuxVersion: String?
    
    var tmuxSummary: String {
        guard tmuxInstalled else {
            return "tmux unavailable"
        }
        if let tmuxVersion, !tmuxVersion.isEmpty {
            return "tmux \(tmuxVersion)"
        }
        return "tmux installed"
    }
}

struct DaemonConfig: Codable, Hashable {
    let listenAddr: String
    let tmuxPath: String
    let autoWrap: Bool
    let tailscaleServe: Bool
    let apnsKeyPath: String
    let apnsKeyID: String
    let apnsTeamID: String
}

struct DesktopWindow: Codable, Identifiable, Hashable {
    let id: String
    let title: String
    let app: String
    let focused: Bool
}

@MainActor
final class MachineStore: ObservableObject {
    static let shared = MachineStore()
    
    @Published private(set) var machines: [Machine] = []
    
    private let defaults: UserDefaults
    private let storageKey: String
    private let encoder = JSONEncoder()
    private let decoder = JSONDecoder()
    
    init(defaults: UserDefaults = .standard, storageKey: String = "machines.v1") {
        self.defaults = defaults
        self.storageKey = storageKey
        load()
    }
    
    var favorites: [Machine] {
        machines.filter(\.isFavorite)
    }
    
    func add(_ machine: Machine) {
        machines.append(machine)
        sortAndPersist()
    }
    
    func update(_ machine: Machine) {
        guard let index = machines.firstIndex(where: { $0.id == machine.id }) else {
            add(machine)
            return
        }
        machines[index] = machine
        sortAndPersist()
    }
    
    func upsert(_ machine: Machine) {
        update(machine)
    }
    
    func delete(_ machine: Machine) {
        machines.removeAll { $0.id == machine.id }
        persist()
    }
    
    func linkedProfile(for machine: Machine) -> ConnectionProfile? {
        guard let linkedProfileID = machine.linkedProfileID else {
            return nil
        }
        return ConnectionProfileManager.shared.profiles.first { $0.id == linkedProfileID }
    }
    
    func markSeen(_ machineID: UUID, at date: Date = Date()) {
        guard let index = machines.firstIndex(where: { $0.id == machineID }) else {
            return
        }
        machines[index].lastSeenAt = date
        persist()
    }
    
    private func sortAndPersist() {
        machines.sort {
            if $0.isFavorite != $1.isFavorite {
                return $0.isFavorite && !$1.isFavorite
            }
            return $0.displayName.localizedCaseInsensitiveCompare($1.displayName) == .orderedAscending
        }
        persist()
    }
    
    private func load() {
        guard let data = defaults.data(forKey: storageKey) else {
            machines = []
            return
        }
        
        do {
            machines = try decoder.decode([Machine].self, from: data)
            sortAndPersist()
        } catch {
            machines = []
        }
    }
    
    private func persist() {
        do {
            let data = try encoder.encode(machines)
            defaults.set(data, forKey: storageKey)
        } catch {
            return
        }
    }
}
