// iTTY — Session Browser
//
// Shows a clean list of terminal sessions on a machine.
// Tap a session → enter terminal. Tap + → new terminal.

import SwiftUI

@MainActor
final class SessionBrowserViewModel: ObservableObject {
    enum SessionBrowserError: LocalizedError {
        case missingClient(String)

        var errorDescription: String? {
            switch self {
            case .missingClient(let message):
                return message
            }
        }
    }

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

    func createSession() async throws -> SavedSession {
        guard let client else {
            throw SessionBrowserError.missingClient(clientError ?? "Missing daemon client")
        }

        let detail = try await client.createSession(name: nextNewSessionName())
        let session = SavedSession(
            name: detail.name,
            windows: detail.windows,
            created: detail.created,
            attached: detail.attached,
            lastPaneCommand: detail.lastPaneCommand,
            lastPanePath: detail.lastPanePath
        )

        if let existingIndex = sessions.firstIndex(where: { $0.name == session.name }) {
            sessions[existingIndex] = session
        } else {
            sessions.append(session)
        }
        sessions.sort { $0.name.localizedCaseInsensitiveCompare($1.name) == .orderedAscending }
        lastUpdatedAt = Date()
        loadState = .loaded

        return session
    }

    func nextNewSessionName() -> String {
        let entries = sessions.map {
            TmuxSessionNameResolver.SessionEntry(
                name: $0.name,
                attachedCount: $0.attached ? 1 : 0
            )
        }
        return TmuxSessionNameResolver.nextNewSessionName(from: entries)
    }
}

struct SessionBrowserView: View {
    @StateObject private var viewModel: SessionBrowserViewModel
    @State private var connectingSessionID: String?
    @State private var showingSSHSetup = false
    @State private var errorMessage: String?

    private let onConnect: ((SSHSession) -> Void)?

    init(machine: Machine, onConnect: ((SSHSession) -> Void)? = nil) {
        _viewModel = StateObject(wrappedValue: SessionBrowserViewModel(machine: machine))
        self.onConnect = onConnect
    }

    var body: some View {
        ZStack {
            iTTYColors.background.ignoresSafeArea()

            Group {
                if viewModel.isLoading && viewModel.sessions.isEmpty {
                    ProgressView("Loading sessions…")
                        .foregroundStyle(iTTYColors.textSecondary)
                } else if viewModel.sessions.isEmpty {
                    emptyState
                } else {
                    sessionList
                }
            }
        }
        .navigationTitle(viewModel.machine.displayName)
        .toolbar {
            ToolbarItem(placement: .primaryAction) {
                Button {
                    Task { await createAndConnect() }
                } label: {
                    Image(systemName: "plus")
                        .foregroundStyle(iTTYColors.accent)
                }
                .accessibilityIdentifier("NewSessionButton")
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
        .overlay {
            if connectingSessionID != nil {
                ZStack {
                    Color.black.opacity(0.4).ignoresSafeArea()
                    ProgressView("Connecting…")
                        .padding(24)
                        .background(iTTYColors.surfaceElevated)
                        .clipShape(RoundedRectangle(cornerRadius: 16, style: .continuous))
                        .foregroundStyle(iTTYColors.textPrimary)
                }
            }
        }
        .alert("Connection Error", isPresented: Binding(
            get: { errorMessage != nil },
            set: { if !$0 { errorMessage = nil } }
        )) {
            Button("OK", role: .cancel) {}
        } message: {
            Text(errorMessage ?? "")
        }
        .sheet(isPresented: $showingSSHSetup) {
            NavigationStack {
                ConnectionEditorView(
                    profile: linkedProfile != nil ? linkedProfileOrNew() : nil,
                    onSave: { savedProfile in
                        if linkedProfile == nil {
                            ConnectionProfileManager.shared.addProfile(savedProfile)
                        } else {
                            ConnectionProfileManager.shared.updateProfile(savedProfile)
                        }
                        var machine = viewModel.machine
                        machine.linkedProfileID = savedProfile.id
                        MachineStore.shared.update(machine)
                        showingSSHSetup = false
                    }
                )
            }
        }
    }

    // MARK: - Views

    private var sessionList: some View {
        ScrollView {
            VStack(spacing: 2) {
                ForEach(viewModel.sessions) { session in
                    Button {
                        connectToSession(session)
                    } label: {
                        HStack(spacing: 14) {
                            Image(systemName: "terminal")
                                .font(.title3)
                                .foregroundStyle(iTTYColors.accent)
                                .frame(width: 32)

                            VStack(alignment: .leading, spacing: 3) {
                                Text(session.name)
                                    .font(.body.weight(.medium))
                                    .foregroundStyle(iTTYColors.textPrimary)

                                Text(sessionSubtitle(session))
                                    .font(.caption)
                                    .foregroundStyle(iTTYColors.textSecondary)
                            }

                            Spacer()

                            Image(systemName: "chevron.right")
                                .font(.caption.weight(.semibold))
                                .foregroundStyle(iTTYColors.textSecondary.opacity(0.5))
                        }
                        .padding(.horizontal, 16)
                        .padding(.vertical, 14)
                        .background(iTTYColors.surface)
                    }
                    .buttonStyle(.plain)
                }
            }
            .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
            .padding(20)
        }
    }

    private var emptyState: some View {
        VStack(spacing: 16) {
            Image(systemName: "terminal")
                .font(.system(size: 44))
                .foregroundStyle(iTTYColors.textSecondary)

            Text("No terminals open")
                .font(.headline)
                .foregroundStyle(iTTYColors.textPrimary)

            Text("Tap + to open a new terminal session.")
                .font(.subheadline)
                .foregroundStyle(iTTYColors.textSecondary)

            Button {
                Task { await createAndConnect() }
            } label: {
                HStack {
                    Image(systemName: "plus.circle.fill")
                    Text("New Terminal")
                }
                .font(.body.weight(.medium))
                .foregroundStyle(.white)
                .padding(.horizontal, 24)
                .padding(.vertical, 12)
                .background(iTTYColors.accent)
                .clipShape(Capsule())
            }
        }
        .padding(40)
    }

    // MARK: - Helpers

    private func sessionSubtitle(_ session: SavedSession) -> String {
        var parts: [String] = []
        if let cmd = session.lastPaneCommand, !cmd.isEmpty {
            parts.append(cmd)
        }
        parts.append("\(session.windows) window\(session.windows == 1 ? "" : "s")")
        return parts.joined(separator: " · ")
    }

    private var linkedProfile: ConnectionProfile? {
        MachineStore.shared.linkedProfile(for: viewModel.machine)
    }

    private func linkedProfileOrNew() -> ConnectionProfile {
        if let existing = linkedProfile { return existing }
        return ConnectionProfile(
            name: viewModel.machine.displayName,
            host: viewModel.machine.daemonHost,
            port: 22,
            username: "",
            authMethod: .password,
            useTmux: true
        )
    }

    // MARK: - Actions

    private func connectToSession(_ session: SavedSession) {
        connectingSessionID = session.id

        Task {
            do {
                let sshSession = SSHSession()
                try await sshSession.connectViaDaemon(
                    machine: viewModel.machine,
                    sessionName: session.name
                )

                await MainActor.run {
                    connectingSessionID = nil
                    onConnect?(sshSession)
                }
            } catch {
                await MainActor.run {
                    connectingSessionID = nil
                    errorMessage = error.localizedDescription
                }
            }
        }
    }

    private func createAndConnect() async {
        do {
            let session = try await viewModel.createSession()
            connectToSession(session)
        } catch {
            errorMessage = error.localizedDescription
        }
    }
}
