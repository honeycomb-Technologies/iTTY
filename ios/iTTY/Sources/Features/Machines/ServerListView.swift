// iTTY — Server List View
//
// The home screen. Shows all known and discovered servers in a flat list.
// Tap a server → straight into terminal. Manual add at the bottom.

import SwiftUI

struct ServerListView: View {
    @StateObject private var discoveryService: TailscaleDiscoveryService
    @State private var showingManualAdd = false

    let onConnect: (SSHSession) -> Void

    @MainActor
    init(onConnect: @escaping (SSHSession) -> Void) {
        self.onConnect = onConnect
        _discoveryService = StateObject(wrappedValue: TailscaleDiscoveryService())
    }

    var body: some View {
        ZStack {
            iTTYColors.background.ignoresSafeArea()

            ScrollView {
                VStack(spacing: 0) {
                    // Header
                    headerSection
                        .padding(.top, 16)
                        .padding(.bottom, 24)

                    // Server list
                    if !allMachines.isEmpty {
                        serverList
                    } else if !discoveryService.isRefreshing {
                        emptyState
                    }

                    // Tailscale recommendation
                    if discoveryService.availability == .notInstalled {
                        tailscaleHint
                            .padding(.top, 16)
                    }

                    // Manual add button
                    manualAddButton
                        .padding(.top, 24)
                        .padding(.bottom, 40)
                }
                .padding(.horizontal, 20)
            }
        }
        .task {
            discoveryService.startPolling()
        }
        .onDisappear {
            discoveryService.stopPolling()
        }
        .sheet(isPresented: $showingManualAdd) {
            ManualAddSheet { machine in
                MachineStore.shared.add(machine)
            }
        }
    }

    // MARK: - Computed

    private var allMachines: [TailscaleDiscoveryService.MachineStatus] {
        discoveryService.machineStatuses
    }

    // MARK: - Sections

    private var headerSection: some View {
        HStack {
            VStack(alignment: .leading, spacing: 4) {
                Text("iTTY")
                    .font(.largeTitle.weight(.bold))
                    .foregroundStyle(iTTYColors.textPrimary)
                Text("Your terminals")
                    .font(.subheadline)
                    .foregroundStyle(iTTYColors.textSecondary)
            }
            Spacer()
            if discoveryService.isRefreshing {
                ProgressView()
                    .tint(iTTYColors.accent)
            }
        }
    }

    private var serverList: some View {
        VStack(spacing: 2) {
            ForEach(allMachines) { status in
                ServerRow(
                    status: status,
                    onTap: { quickConnect(machine: status.machine) }
                )
            }
        }
        .clipShape(RoundedRectangle(cornerRadius: 14, style: .continuous))
    }

    private var emptyState: some View {
        VStack(spacing: 12) {
            Image(systemName: "desktopcomputer.trianglebadge.exclamationmark")
                .font(.system(size: 40))
                .foregroundStyle(iTTYColors.textSecondary)
            Text("No servers found")
                .font(.headline)
                .foregroundStyle(iTTYColors.textPrimary)
            Text("Start the iTTY daemon on a computer, or add one manually below.")
                .font(.subheadline)
                .foregroundStyle(iTTYColors.textSecondary)
                .multilineTextAlignment(.center)
        }
        .padding(.vertical, 40)
    }

    private var tailscaleHint: some View {
        HStack(spacing: 10) {
            Image(systemName: "network.badge.shield.half.filled")
                .foregroundStyle(iTTYColors.accent)
            VStack(alignment: .leading, spacing: 2) {
                Text("Tailscale recommended")
                    .font(.footnote.weight(.medium))
                    .foregroundStyle(iTTYColors.textPrimary)
                Text("Install Tailscale for zero-config access from anywhere.")
                    .font(.caption)
                    .foregroundStyle(iTTYColors.textSecondary)
            }
            Spacer()
        }
        .padding(14)
        .background(iTTYColors.surface)
        .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
    }

    private var manualAddButton: some View {
        Button {
            showingManualAdd = true
        } label: {
            HStack {
                Image(systemName: "plus.circle.fill")
                Text("Add Server Manually")
            }
            .font(.body.weight(.medium))
            .foregroundStyle(iTTYColors.accent)
            .frame(maxWidth: .infinity)
            .padding(.vertical, 14)
            .background(iTTYColors.surface)
            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
        }
    }

    // MARK: - Actions

    private func quickConnect(machine: Machine) {
        guard let linkedProfile = MachineStore.shared.linkedProfile(for: machine) else {
            // No SSH profile linked — need to set one up first
            // For now, open the session browser which handles this
            return
        }

        Task {
            do {
                let credential = try await CredentialManager.shared.getCredentials(for: linkedProfile)
                var attachProfile = linkedProfile
                attachProfile.useTmux = true

                let sshSession = SSHSession()
                try await sshSession.connect(profile: attachProfile, credential: credential)

                await MainActor.run {
                    onConnect(sshSession)
                }
            } catch {
                // Connection failed — could show an alert
            }
        }
    }
}

// MARK: - Server Row

private struct ServerRow: View {
    let status: TailscaleDiscoveryService.MachineStatus
    let onTap: () -> Void

    var body: some View {
        Button(action: onTap) {
            HStack(spacing: 14) {
                // Status dot
                Circle()
                    .fill(status.isReachable ? iTTYColors.online : iTTYColors.offline)
                    .frame(width: 10, height: 10)

                // Machine info
                VStack(alignment: .leading, spacing: 3) {
                    Text(status.machine.displayName)
                        .font(.body.weight(.medium))
                        .foregroundStyle(iTTYColors.textPrimary)

                    Text(statusText)
                        .font(.caption)
                        .foregroundStyle(iTTYColors.textSecondary)
                }

                Spacer()

                // Platform badge
                if let platform = status.health?.platform {
                    Text(platformLabel(platform))
                        .font(.caption2.weight(.medium))
                        .foregroundStyle(iTTYColors.textSecondary)
                        .padding(.horizontal, 8)
                        .padding(.vertical, 4)
                        .background(iTTYColors.surfaceElevated)
                        .clipShape(Capsule())
                }

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

    private var statusText: String {
        if status.isReachable {
            return status.health?.tmuxSummary ?? "Online"
        }
        return status.errorMessage ?? "Waiting for daemon"
    }

    private func platformLabel(_ platform: String) -> String {
        if platform.contains("darwin") { return "macOS" }
        if platform.contains("linux") { return "Linux" }
        if platform.contains("windows") { return "Windows" }
        return platform
    }
}
