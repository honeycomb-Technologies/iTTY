import SwiftUI

struct SessionRowView: View {
    let session: SavedSession
    
    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack(alignment: .firstTextBaseline) {
                Text(session.name)
                    .font(.headline)
                
                Spacer()
                
                Text(session.statusLabel)
                    .font(.caption.weight(.medium))
                    .foregroundStyle(session.attached ? .green : .secondary)
            }
            
            Text(session.subtitle)
                .font(.subheadline)
                .foregroundStyle(.secondary)
                .lineLimit(1)
            
            HStack(spacing: 12) {
                Label("\(session.windows)", systemImage: "square.split.2x1")
                if let lastPanePath = session.lastPanePath, !lastPanePath.isEmpty {
                    Label(lastPanePath, systemImage: "folder")
                        .lineLimit(1)
                }
            }
            .font(.caption)
            .foregroundStyle(.tertiary)
        }
        .padding(.vertical, 4)
    }
}
