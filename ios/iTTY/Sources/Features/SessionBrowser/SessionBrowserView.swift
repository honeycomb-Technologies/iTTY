import SwiftUI

@MainActor
final class SessionBrowserViewModel: ObservableObject {
    enum LoadState: Equatable {
        case idle
        case loading
        case loaded
        case failed(String)
    }
    
    @Published private(set) var health: DaemonHealth?
    @Published private(set) var sessions: [SavedSession] = []
    @Published private(set) var loadState: LoadState = .idle
    @Published private(set) var lastUpdatedAt: Date?
    @Published var selectedSessionDetail: SavedSessionDetail?
    @Published var selectedSessionPreview: String?
    
    let machine: Machine
    
    private let client: DaemonClient?
    private let clientError: String?
    
    init(machine: Machine, client: DaemonClient? = nil) {
        self.machine = machine
        
        if let client {
            self.client = client
            self.clientError = nil
            return
        }
        
        do {
            self.client = try DaemonClient(machine: machine)
            self.clientError = nil
        } catch {
            self.client = nil
            self.clientError = error.localizedDescription
        }
    }
    
    var isLoading: Bool {
        loadState == .loading
    }
    
    var errorMessage: String? {
        guard case .failed(let message) = loadState else {
            return nil
        }
        return message
    }
    
    func clearError() {
        guard case .failed = loadState else {
            return
        }
        loadState = sessions.isEmpty && health == nil ? .idle : .loaded
    }
    
    func reportError(_ message: String) {
        loadState = .failed(message)
    }
    
    func load() async {
        guard let client else {
            loadState = .failed(clientError ?? "Missing daemon client")
            return
        }
        
        loadState = .loading
        
        do {
            async let daemonHealth = client.health()
            async let daemonSessions = client.listSessions()
            let (health, sessions) = try await (daemonHealth, daemonSessions)
            
            self.health = health
            self.sessions = sessions.sorted { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
            self.lastUpdatedAt = Date()
            self.loadState = .loaded
            MachineStore.shared.markSeen(machine.id)
        } catch {
            self.loadState = .failed(error.localizedDescription)
        }
    }
    
    func inspect(_ session: SavedSession) async {
        guard let client else {
            loadState = .failed(clientError ?? "Missing daemon client")
            return
        }
        
        do {
            async let detail = client.sessionDetail(name: session.name)
            async let content = client.sessionContent(name: session.name)
            let (detailResult, contentResult) = try await (detail, content)
            selectedSessionDetail = detailResult
            selectedSessionPreview = contentResult.content
        } catch {
            loadState = .failed(error.localizedDescription)
        }
    }
}

struct SessionBrowserView: View {
    @StateObject private var viewModel: SessionBrowserViewModel
    @State private var attachingSessionID: String?
    
    private let onConnect: ((SSHSession) -> Void)?
    
    init(machine: Machine, onConnect: ((SSHSession) -> Void)? = nil) {
        _viewModel = StateObject(wrappedValue: SessionBrowserViewModel(machine: machine))
        self.onConnect = onConnect
    }
    
    var body: some View {
        List {
            Section("Machine") {
                LabeledContent("Name", value: viewModel.machine.displayName)
                LabeledContent("Daemon", value: viewModel.machine.daemonAuthority)
                if let linkedProfile = MachineStore.shared.linkedProfile(for: viewModel.machine) {
                    LabeledContent("Attach Profile", value: linkedProfile.displayString)
                }
            }
            
            if let health = viewModel.health {
                Section("Daemon") {
                    LabeledContent("Status", value: health.status)
                    LabeledContent("Version", value: health.version)
                    LabeledContent("Platform", value: health.platform)
                    LabeledContent("tmux", value: health.tmuxSummary)
                }
            }
            
            Section("Sessions") {
                if viewModel.sessions.isEmpty && !viewModel.isLoading {
                    ContentUnavailableView(
                        "No Sessions",
                        systemImage: "terminal",
                        description: Text("The daemon is reachable, but it did not report any tmux sessions.")
                    )
                } else {
                    ForEach(viewModel.sessions) { session in
                        Button {
                            Task {
                                await viewModel.inspect(session)
                            }
                        } label: {
                            SessionRowView(session: session)
                        }
                        .buttonStyle(.plain)
                        .swipeActions(edge: .trailing, allowsFullSwipe: false) {
                            if canAttach {
                                Button("Attach") {
                                    attach(to: session)
                                }
                                .tint(.blue)
                            }
                        }
                    }
                }
            }
            
            if let lastUpdatedAt = viewModel.lastUpdatedAt {
                Section {
                    LabeledContent("Last Refresh", value: lastUpdatedAt.formatted(date: .abbreviated, time: .shortened))
                }
            }
        }
        .navigationTitle(viewModel.machine.displayName)
        .overlay {
            if viewModel.isLoading {
                ProgressView("Loading daemon state…")
            } else if attachingSessionID != nil {
                ProgressView("Opening session…")
            }
        }
        .task {
            if viewModel.loadState == .idle {
                await viewModel.load()
            }
        }
        .refreshable {
            await viewModel.load()
        }
        .sheet(item: $viewModel.selectedSessionDetail) { detail in
            SessionInspectorView(detail: detail, preview: viewModel.selectedSessionPreview)
        }
        .alert(
            "Daemon Error",
            isPresented: Binding(
                get: { viewModel.errorMessage != nil },
                set: { _ in
                    viewModel.clearError()
                }
            )
        ) {
            Button("OK", role: .cancel) {}
        } message: {
            Text(viewModel.errorMessage ?? "Unknown daemon error")
        }
    }
    
    private var canAttach: Bool {
        MachineStore.shared.linkedProfile(for: viewModel.machine) != nil && onConnect != nil
    }
    
    private func attach(to session: SavedSession) {
        guard let onConnect,
              let linkedProfile = MachineStore.shared.linkedProfile(for: viewModel.machine) else {
            return
        }
        
        attachingSessionID = session.id
        
        Task {
            do {
                let credential = try await CredentialManager.shared.getCredentials(for: linkedProfile)
                var attachProfile = linkedProfile
                attachProfile.useTmux = true
                attachProfile.tmuxSessionName = session.name
                
                let sshSession = SSHSession()
                try await sshSession.connect(profile: attachProfile, credential: credential)
                
                await MainActor.run {
                    attachingSessionID = nil
                    onConnect(sshSession)
                }
            } catch {
                await MainActor.run {
                    attachingSessionID = nil
                    viewModel.reportError(error.localizedDescription)
                }
            }
        }
    }
}

private struct SessionInspectorView: View {
    let detail: SavedSessionDetail
    let preview: String?
    
    var body: some View {
        NavigationStack {
            List {
                Section("Session") {
                    LabeledContent("Name", value: detail.name)
                    LabeledContent("Status", value: detail.attached ? "Attached" : "Detached")
                    LabeledContent("Windows", value: "\(detail.windows)")
                    if let lastPaneCommand = detail.lastPaneCommand, !lastPaneCommand.isEmpty {
                        LabeledContent("Last Command", value: lastPaneCommand)
                    }
                    if let lastPanePath = detail.lastPanePath, !lastPanePath.isEmpty {
                        LabeledContent("Last Path", value: lastPanePath)
                    }
                }
                
                Section("Windows") {
                    ForEach(detail.windowList) { window in
                        VStack(alignment: .leading, spacing: 8) {
                            HStack {
                                Text("\(window.index): \(window.name)")
                                    .font(.headline)
                                if window.active {
                                    Spacer()
                                    Text("Active")
                                        .font(.caption.weight(.medium))
                                        .foregroundStyle(.green)
                                }
                            }
                            
                            ForEach(window.panes, id: \.id) { pane in
                                VStack(alignment: .leading, spacing: 4) {
                                    Text("%\(pane.id) • \(pane.command)")
                                        .font(.subheadline)
                                    Text(pane.path)
                                        .font(.caption)
                                        .foregroundStyle(.secondary)
                                    Text("\(pane.width)×\(pane.height)")
                                        .font(.caption2)
                                        .foregroundStyle(.tertiary)
                                }
                                .padding(.vertical, 2)
                            }
                        }
                    }
                }
                
                if let preview, !preview.isEmpty {
                    Section("Active Pane Preview") {
                        ScrollView(.horizontal) {
                            Text(preview)
                                .font(.system(.footnote, design: .monospaced))
                                .textSelection(.enabled)
                        }
                    }
                }
            }
            .navigationTitle(detail.name)
            .navigationBarTitleDisplayMode(.inline)
        }
    }
}
