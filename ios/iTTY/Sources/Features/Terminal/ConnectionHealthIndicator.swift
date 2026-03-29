// iTTY — Connection Health Indicator
//
// A small overlay view that shows the SSH connection's health state.
// Only visible when the connection is degraded or dead.

import SwiftUI

struct ConnectionHealthIndicator: View {
    let health: ConnectionHealth

    var body: some View {
        switch health {
        case .healthy:
            EmptyView()

        case .stale:
            HStack(spacing: 6) {
                Image(systemName: "wifi.exclamationmark")
                    .font(.caption2)
                Text("Connection unstable")
                    .font(.caption2.weight(.medium))
            }
            .foregroundStyle(.orange)
            .padding(.horizontal, 10)
            .padding(.vertical, 5)
            .background(
                Capsule()
                    .fill(.ultraThinMaterial)
            )
            .transition(.move(edge: .top).combined(with: .opacity))

        case .dead(let reason):
            HStack(spacing: 6) {
                Image(systemName: "wifi.slash")
                    .font(.caption2)
                Text(reason.isEmpty ? "Disconnected" : reason)
                    .font(.caption2.weight(.medium))
                    .lineLimit(1)
            }
            .foregroundStyle(.red)
            .padding(.horizontal, 10)
            .padding(.vertical, 5)
            .background(
                Capsule()
                    .fill(.ultraThinMaterial)
            )
            .transition(.move(edge: .top).combined(with: .opacity))
        }
    }
}
