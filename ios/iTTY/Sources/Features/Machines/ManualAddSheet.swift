// iTTY — Manual Add Sheet
//
// Simple bottom sheet for adding a server by hostname and port.

import SwiftUI

struct ManualAddSheet: View {
    @Environment(\.dismiss) private var dismiss

    @State private var hostname = ""
    @State private var port = "3420"
    @State private var name = ""

    let onSave: (Machine) -> Void

    var body: some View {
        NavigationStack {
            ZStack {
                iTTYColors.background.ignoresSafeArea()

                VStack(spacing: 20) {
                    VStack(alignment: .leading, spacing: 6) {
                        Text("Hostname or IP")
                            .font(.caption.weight(.medium))
                            .foregroundStyle(iTTYColors.textSecondary)
                        TextField("192.168.1.100 or my-pc.tail.ts.net", text: $hostname)
                            .textInputAutocapitalization(.never)
                            .keyboardType(.URL)
                            .padding(12)
                            .background(iTTYColors.surface)
                            .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                            .foregroundStyle(iTTYColors.textPrimary)
                    }

                    HStack(spacing: 16) {
                        VStack(alignment: .leading, spacing: 6) {
                            Text("Port")
                                .font(.caption.weight(.medium))
                                .foregroundStyle(iTTYColors.textSecondary)
                            TextField("3420", text: $port)
                                .keyboardType(.numberPad)
                                .padding(12)
                                .background(iTTYColors.surface)
                                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                                .foregroundStyle(iTTYColors.textPrimary)
                        }
                        .frame(width: 100)

                        VStack(alignment: .leading, spacing: 6) {
                            Text("Name (optional)")
                                .font(.caption.weight(.medium))
                                .foregroundStyle(iTTYColors.textSecondary)
                            TextField("My Server", text: $name)
                                .padding(12)
                                .background(iTTYColors.surface)
                                .clipShape(RoundedRectangle(cornerRadius: 10, style: .continuous))
                                .foregroundStyle(iTTYColors.textPrimary)
                        }
                    }

                    Button {
                        save()
                    } label: {
                        Text("Add Server")
                            .font(.body.weight(.semibold))
                            .foregroundStyle(.white)
                            .frame(maxWidth: .infinity)
                            .padding(.vertical, 14)
                            .background(isValid ? iTTYColors.accent : iTTYColors.accent.opacity(0.4))
                            .clipShape(RoundedRectangle(cornerRadius: 12, style: .continuous))
                    }
                    .disabled(!isValid)

                    Spacer()
                }
                .padding(20)
            }
            .navigationTitle("Add Server")
            .navigationBarTitleDisplayMode(.inline)
            .toolbarColorScheme(.dark, for: .navigationBar)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") { dismiss() }
                        .foregroundStyle(iTTYColors.accent)
                }
            }
        }
        .presentationDetents([.medium])
    }

    private var isValid: Bool {
        let trimmed = hostname.trimmingCharacters(in: .whitespacesAndNewlines)
        guard !trimmed.isEmpty else { return false }
        guard let p = Int(port), p >= 1, p <= 65535 else { return false }
        return true
    }

    private func save() {
        let trimmed = hostname.trimmingCharacters(in: .whitespacesAndNewlines)
        let portNum = Int(port) ?? 3420
        let displayName = name.trimmingCharacters(in: .whitespacesAndNewlines)

        let machine = Machine(
            name: displayName.isEmpty ? trimmed : displayName,
            daemonScheme: "http",
            daemonHost: trimmed,
            daemonPort: portNum
        )
        onSave(machine)
        dismiss()
    }
}
