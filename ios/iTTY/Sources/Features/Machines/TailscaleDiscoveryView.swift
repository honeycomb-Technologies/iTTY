import SwiftUI

struct TailscaleDiscoveryView: View {
    @Environment(\.dismiss) private var dismiss
    
    @StateObject private var discoveryService: TailscaleDiscoveryService
    @State private var showingAddMachine = false
    @State private var showingManualSetup = false
    
    private let onConnect: ((SSHSession) -> Void)?
    
    @MainActor
    init(
        onConnect: ((SSHSession) -> Void)? = nil,
        discoveryService: TailscaleDiscoveryService? = nil
    ) {
        self.onConnect = onConnect
        _discoveryService = StateObject(
            wrappedValue: discoveryService ?? TailscaleDiscoveryService()
        )
    }
    
    var body: some View {
        NavigationStack {
            ScrollView {
                VStack(alignment: .leading, spacing: 20) {
                    TailscaleStatusCard(
                        availability: discoveryService.availability,
                        hasSavedComputers: !discoveryService.machineStatuses.isEmpty,
                        activeCount: discoveryService.activeMachines.count,
                        lastRefreshAt: discoveryService.lastRefreshAt,
                        openTailscale: openTailscale,
                        installTailscale: installTailscale
                    )
                    
                    if !discoveryService.activeMachines.isEmpty {
                        DiscoverySectionHeader(
                            title: "Ready Now",
                            subtitle: "These computers answered the iTTY daemon health check."
                        )
                        
                        VStack(spacing: 12) {
                            ForEach(discoveryService.activeMachines) { status in
                                NavigationLink {
                                    SessionBrowserView(machine: status.machine, onConnect: onConnect)
                                } label: {
                                    DiscoveryMachineCard(
                                        status: status,
                                        statusText: "Daemon online",
                                        statusColor: .green,
                                        detailText: status.health?.platform ?? "Ready for desktop sessions"
                                    )
                                }
                                .buttonStyle(.plain)
                            }
                        }
                    }
                    
                    if !discoveryService.discoveredPeers.isEmpty {
                        DiscoverySectionHeader(
                            title: "Discovered on Tailnet",
                            subtitle: "Found via a reachable daemon. Probing for iTTY daemons..."
                        )

                        VStack(spacing: 12) {
                            ForEach(discoveryService.discoveredPeers) { discovered in
                                DiscoveredPeerCard(discovered: discovered)
                            }
                        }
                    }

                    if !discoveryService.inactiveMachines.isEmpty {
                        DiscoverySectionHeader(
                            title: "Known Computers",
                            subtitle: "We keep checking these saved daemon addresses every 5 seconds."
                        )
                        
                        VStack(spacing: 12) {
                            ForEach(discoveryService.inactiveMachines) { status in
                                DiscoveryMachineCard(
                                    status: status,
                                    statusText: "Waiting for daemon",
                                    statusColor: .orange,
                                    detailText: status.errorMessage ?? "Start the iTTY daemon, then we will pick it up automatically."
                                )
                            }
                        }
                    }
                    
                    ManualSetupCard(
                        tailscaleAvailable: discoveryService.availability == .installed,
                        hasSavedComputers: !discoveryService.machineStatuses.isEmpty,
                        addComputer: { showingAddMachine = true },
                        openManualSetup: { showingManualSetup = true }
                    )
                }
                .padding(20)
            }
            .background(Color(.systemGroupedBackground))
            .navigationTitle("Find Computers")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Done") {
                        dismiss()
                    }
                }
                
                ToolbarItemGroup(placement: .primaryAction) {
                    Button {
                        Task { await discoveryService.refresh() }
                    } label: {
                        if discoveryService.isRefreshing {
                            ProgressView()
                        } else {
                            Image(systemName: "arrow.clockwise")
                        }
                    }
                    .accessibilityIdentifier("TailscaleRefreshButton")
                    
                    Button {
                        showingAddMachine = true
                    } label: {
                        Image(systemName: "plus")
                    }
                    .accessibilityIdentifier("TailscaleAddComputerButton")
                }
            }
            .task {
                discoveryService.startPolling()
            }
            .onDisappear {
                discoveryService.stopPolling()
            }
            .sheet(isPresented: $showingAddMachine) {
                AddMachineView { machine in
                    MachineStore.shared.add(machine)
                }
            }
            .sheet(isPresented: $showingManualSetup) {
                ConnectionListView(onConnect: onConnect)
            }
        }
    }
    
    private func openTailscale() {
        discoveryService.openTailscaleApp()
    }
    
    private func installTailscale() {
        discoveryService.openInstallPage()
    }
}

private struct TailscaleStatusCard: View {
    let availability: TailscaleVPNDetector.Availability
    let hasSavedComputers: Bool
    let activeCount: Int
    let lastRefreshAt: Date?
    let openTailscale: () -> Void
    let installTailscale: () -> Void
    
    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            HStack(alignment: .top, spacing: 14) {
                ZStack {
                    RoundedRectangle(cornerRadius: 18, style: .continuous)
                        .fill(iconBackground)
                        .frame(width: 54, height: 54)
                    
                    Image(systemName: iconName)
                        .font(.system(size: 24, weight: .semibold))
                        .foregroundStyle(iconColor)
                }
                
                VStack(alignment: .leading, spacing: 6) {
                    Text(title)
                        .font(.title3.weight(.semibold))
                        .accessibilityIdentifier("TailscaleDiscoveryTitle")
                    
                    Text(detail)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
            }
            
            if let lastRefreshAt {
                Text("Last checked \(lastRefreshAt.formatted(date: .omitted, time: .standard))")
                    .font(.caption)
                    .foregroundStyle(.secondary)
            }
            
            switch availability {
            case .installed:
                Button("Open Tailscale", action: openTailscale)
                    .buttonStyle(.borderedProminent)
                    .accessibilityIdentifier("OpenTailscaleButton")
            case .notInstalled:
                Button("Install Tailscale", action: installTailscale)
                    .buttonStyle(.borderedProminent)
                    .accessibilityIdentifier("InstallTailscaleButton")
            }
        }
        .padding(20)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(cardBackground)
    }
    
    private var title: String {
        switch (availability, activeCount, hasSavedComputers) {
        case (.notInstalled, _, _):
            return "Install Tailscale for the easiest setup"
        case (.installed, let count, _) where count > 0:
            return count == 1 ? "1 computer is ready" : "\(count) computers are ready"
        case (.installed, _, false):
            return "Tailscale is ready to look for your computers"
        case (.installed, _, true):
            return "Looking for a running iTTY daemon"
        }
    }
    
    private var detail: String {
        switch (availability, activeCount, hasSavedComputers) {
        case (.notInstalled, _, _):
            return "Install Tailscale on your phone and your computer, then start the iTTY daemon. Manual setup is still available below."
        case (.installed, let count, _) where count > 0:
            return count == 1
                ? "We found one reachable daemon over your saved Tailscale or manual addresses."
                : "We found reachable daemons over your saved Tailscale or manual addresses."
        case (.installed, _, false):
            return "Add a computer running the iTTY daemon once, and we will keep checking it every 5 seconds."
        case (.installed, _, true):
            return "Tailscale is available, but none of your saved computers are answering yet. Start the daemon and keep this screen open."
        }
    }
    
    private var iconName: String {
        switch availability {
        case .installed:
            return activeCount > 0 ? "desktopcomputer.and.arrow.down" : "network.badge.shield.half.filled"
        case .notInstalled:
            return "arrow.down.app"
        }
    }
    
    private var iconColor: Color {
        switch availability {
        case .installed:
            return activeCount > 0 ? .green : .blue
        case .notInstalled:
            return .orange
        }
    }
    
    private var iconBackground: Color {
        switch availability {
        case .installed:
            return activeCount > 0 ? Color.green.opacity(0.15) : Color.blue.opacity(0.15)
        case .notInstalled:
            return Color.orange.opacity(0.15)
        }
    }
    
    private var cardBackground: some View {
        RoundedRectangle(cornerRadius: 24, style: .continuous)
            .fill(
                LinearGradient(
                    colors: [
                        Color(.secondarySystemBackground),
                        Color(.tertiarySystemBackground)
                    ],
                    startPoint: .topLeading,
                    endPoint: .bottomTrailing
                )
            )
            .overlay(
                RoundedRectangle(cornerRadius: 24, style: .continuous)
                    .strokeBorder(Color.primary.opacity(0.06), lineWidth: 1)
            )
    }
}

private struct DiscoverySectionHeader: View {
    let title: String
    let subtitle: String
    
    var body: some View {
        VStack(alignment: .leading, spacing: 4) {
            Text(title)
                .font(.headline)
            Text(subtitle)
                .font(.footnote)
                .foregroundStyle(.secondary)
        }
    }
}

private struct DiscoveryMachineCard: View {
    let status: TailscaleDiscoveryService.MachineStatus
    let statusText: String
    let statusColor: Color
    let detailText: String
    
    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(status.machine.displayName)
                        .font(.headline)
                        .foregroundStyle(.primary)
                    Text(status.machine.daemonAuthority)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }
                
                Spacer()
                
                Label(statusText, systemImage: "circle.fill")
                    .font(.caption.weight(.semibold))
                    .labelStyle(.titleAndIcon)
                    .foregroundStyle(statusColor)
            }
            
            Text(detailText)
                .font(.footnote)
                .foregroundStyle(.secondary)
            
            if let lastSeenAt = status.machine.lastSeenAt {
                Text("Last seen \(lastSeenAt.formatted(date: .abbreviated, time: .shortened))")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            RoundedRectangle(cornerRadius: 20, style: .continuous)
                .fill(Color(.secondarySystemBackground))
        )
    }
}

private struct DiscoveredPeerCard: View {
    let discovered: TailscaleDiscoveryService.DiscoveredPeer

    var body: some View {
        VStack(alignment: .leading, spacing: 10) {
            HStack(spacing: 12) {
                VStack(alignment: .leading, spacing: 4) {
                    Text(discovered.peer.hostname)
                        .font(.headline)
                        .foregroundStyle(.primary)
                    Text(discovered.peer.dnsName)
                        .font(.subheadline)
                        .foregroundStyle(.secondary)
                }

                Spacer()

                if discovered.isReachable {
                    Label("Daemon found", systemImage: "circle.fill")
                        .font(.caption.weight(.semibold))
                        .labelStyle(.titleAndIcon)
                        .foregroundStyle(.green)
                } else {
                    Label("No daemon", systemImage: "circle.fill")
                        .font(.caption.weight(.semibold))
                        .labelStyle(.titleAndIcon)
                        .foregroundStyle(.secondary)
                }
            }

            Text(discovered.isReachable
                 ? "iTTY daemon detected — added to your computers."
                 : "Online on tailnet but no iTTY daemon running (\(discovered.peer.os)).")
                .font(.footnote)
                .foregroundStyle(.secondary)
        }
        .padding(16)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            RoundedRectangle(cornerRadius: 20, style: .continuous)
                .fill(Color(.secondarySystemBackground))
        )
    }
}

private struct ManualSetupCard: View {
    let tailscaleAvailable: Bool
    let hasSavedComputers: Bool
    let addComputer: () -> Void
    let openManualSetup: () -> Void
    
    var body: some View {
        VStack(alignment: .leading, spacing: 14) {
            Text("Manual Setup")
                .font(.headline)
            
            Text(description)
                .font(.footnote)
                .foregroundStyle(.secondary)
            
            Button("Add Computer Manually", action: addComputer)
                .buttonStyle(.borderedProminent)
                .accessibilityIdentifier("AddComputerManuallyButton")
            
            Button("Manual SSH Setup", action: openManualSetup)
                .buttonStyle(.bordered)
                .accessibilityIdentifier("ManualSSHSetupButton")
        }
        .padding(18)
        .frame(maxWidth: .infinity, alignment: .leading)
        .background(
            RoundedRectangle(cornerRadius: 20, style: .continuous)
                .fill(Color(.secondarySystemBackground))
        )
    }
    
    private var description: String {
        if hasSavedComputers {
            return "Use this if you want to add another daemon address or fall back to a direct SSH connection."
        }
        
        if tailscaleAvailable {
            return "If discovery has nothing to check yet, add the hostname for a computer running the iTTY daemon or use a direct SSH connection."
        }
        
        return "You can still add a daemon hostname or use a direct SSH connection without Tailscale."
    }
}
