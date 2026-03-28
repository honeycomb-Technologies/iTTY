import SwiftUI

struct AddMachineView: View {
    @Environment(\.dismiss) private var dismiss
    
    @State private var name = ""
    @State private var daemonHost = ""
    @State private var daemonPort = 8080
    @State private var daemonScheme = "http"
    @State private var linkedProfileID: UUID?
    
    let onSave: (Machine) -> Void
    
    @ObservedObject private var profileManager = ConnectionProfileManager.shared
    
    var body: some View {
        NavigationStack {
            Form {
                Section("Desktop") {
                    TextField("Name", text: $name)
                    TextField("Daemon Host", text: $daemonHost)
                        .textInputAutocapitalization(.never)
                        .keyboardType(.URL)
                    
                    Picker("Scheme", selection: $daemonScheme) {
                        Text("http").tag("http")
                        Text("https").tag("https")
                    }
                    
                    HStack {
                        Text("Port")
                        Spacer()
                        TextField("8080", value: $daemonPort, format: .number)
                            .keyboardType(.numberPad)
                            .multilineTextAlignment(.trailing)
                            .frame(width: 80)
                    }
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
            .navigationTitle("Add Machine")
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
