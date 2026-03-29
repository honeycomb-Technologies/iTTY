import SwiftUI

struct AddMachineView: View {
    @Environment(\.dismiss) private var dismiss
    
    @State private var name = ""
    @State private var daemonHost = ""
    @State private var daemonPort = 443
    @State private var daemonScheme = "https"
    @State private var linkedProfileID: UUID?
    
    let onSave: (Machine) -> Void
    
    @ObservedObject private var profileManager = ConnectionProfileManager.shared
    
    var body: some View {
        NavigationStack {
            Form {
                Section {
                    TextField("Name", text: $name)
                    TextField("Daemon Hostname", text: $daemonHost)
                        .textInputAutocapitalization(.never)
                        .keyboardType(.URL)
                    
                    Picker("Scheme", selection: $daemonScheme) {
                        Text("https").tag("https")
                        Text("http").tag("http")
                    }
                    
                    HStack {
                        Text("Port")
                        Spacer()
                        TextField("443", value: $daemonPort, format: .number)
                            .keyboardType(.numberPad)
                            .multilineTextAlignment(.trailing)
                            .frame(width: 80)
                    }
                } header: {
                    Text("Computer")
                } footer: {
                    Text("Use `https` + `443` for Tailscale Serve. Switch to `http` + `3420` for a local or LAN daemon.")
                }
                
                Section("Attach Flow") {
                    Picker("Linked Connection Profile", selection: $linkedProfileID) {
                        Text("None").tag(Optional<UUID>.none)
                        ForEach(profileManager.profiles) { profile in
                            Text(profile.name).tag(Optional(profile.id))
                        }
                    }
                    
                    Text("Linking a saved SSH profile lets the session browser know which connection to reuse when attach flows are wired on macOS.")
                        .font(.footnote)
                        .foregroundStyle(.secondary)
                }
                
                Section {
                    Button("Save") {
                        onSave(
                            Machine(
                                name: name,
                                daemonScheme: daemonScheme,
                                daemonHost: daemonHost,
                                daemonPort: daemonPort,
                                linkedProfileID: linkedProfileID
                            )
                        )
                        dismiss()
                    }
                    .disabled(!isValid)
                }
            }
            .navigationTitle("Add Computer")
            .navigationBarTitleDisplayMode(.inline)
            .toolbar {
                ToolbarItem(placement: .cancellationAction) {
                    Button("Cancel") {
                        dismiss()
                    }
                }
            }
        }
    }
    
    private var isValid: Bool {
        !daemonHost.trimmingCharacters(in: .whitespacesAndNewlines).isEmpty &&
        daemonPort >= 1 &&
        daemonPort <= 65535
    }
}
