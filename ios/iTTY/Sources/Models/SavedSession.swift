import Foundation

struct SavedSession: Identifiable, Codable, Hashable {
    let name: String
    let windows: Int
    let created: Date
    let attached: Bool
    let lastPaneCommand: String?
    let lastPanePath: String?
    
    var id: String { name }
    
    var statusLabel: String {
        attached ? "Attached" : "Detached"
    }
    
    var subtitle: String {
        if let lastPaneCommand, !lastPaneCommand.isEmpty {
            return lastPaneCommand
        }
        if let lastPanePath, !lastPanePath.isEmpty {
            return lastPanePath
        }
        return "\(windows) window\(windows == 1 ? "" : "s")"
    }
}

struct SavedSessionDetail: Identifiable, Codable, Hashable {
    let name: String
    let windows: Int
    let created: Date
    let attached: Bool
    let lastPaneCommand: String?
    let lastPanePath: String?
    let windowList: [SavedWindow]
    
    var id: String { name }
    
    var activeWindow: SavedWindow? {
        windowList.first(where: \.active)
    }
}

struct SavedWindow: Identifiable, Codable, Hashable {
    let index: Int
    let name: String
    let active: Bool
    let panes: [SavedPane]
    
    var id: Int { index }
    
    var activePane: SavedPane? {
        panes.first(where: \.active)
    }
}

struct SavedPane: Codable, Hashable {
    let id: String
    let index: Int
    let active: Bool
    let command: String
    let path: String
    let width: Int
    let height: Int
}

struct SavedSessionContent: Codable, Hashable {
    let content: String
}

struct DaemonErrorEnvelope: Codable, Hashable {
    let error: String
}
