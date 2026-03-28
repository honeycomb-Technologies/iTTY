import SwiftUI

struct MachineListView: View {
    @ObservedObject private var machineStore = MachineStore.shared
    
    @State private var showingAddMachine = false
    
    var onConnect: ((SSHSession) -> Void)? = nil
    
    var body: some View {
        NavigationStack {
            List {
                if !machineStore.favorites.isEmpty {
                    Section("Favorites") {
                        ForEach(machineStore.favorites) { machine in
                            NavigationLink {
                                SessionBrowserView(machine: machine, onConnect: onConnect)
                            } label: {
                                MachineRowView(machine: machine)
                            }
                        }
                    }
                }
                
                Section("Machines") {
                    if machineStore.machines.isEmpty {
                        ContentUnavailableView(
                            "No Machines",
                            systemImage: "desktopcomputer",
                            description: Text("Add a desktop daemon to browse tmux sessions.")
                        )
                    } else {
                        ForEach(machineStore.machines) { machine in
                            NavigationLink {
                                SessionBrowserView(machine: machine, onConnect: onConnect)
                            } label: {
                                MachineRowView(machine: machine)
                            }
                            .contextMenu {
                                Button(machine.isFavorite ? "Remove Favorite" : "Favorite") {
                                    var updated = machine
                                    updated.isFavorite.toggle()
                                    machineStore.update(updated)
                                }
                            }
                        }
                        .onDelete(perform: deleteMachines)
                    }
                }
            }
            .navigationTitle("Machines")
            .toolbar {
                ToolbarItem(placement: .primaryAction) {
                    Button {
                        showingAddMachine = true
                    } label: {
                        Image(systemName: "plus")
                    }
                }
            }
            .sheet(isPresented: $showingAddMachine) {
                AddMachineView { machine in
                    machineStore.add(machine)
                }
            }
        }
    }
    
    private func deleteMachines(at offsets: IndexSet) {
        let sorted = machineStore.machines
        for index in offsets {
            guard sorted.indices.contains(index) else {
                continue
            }
            machineStore.delete(sorted[index])
        }
    }
}

private struct MachineRowView: View {
    let machine: Machine
    
    var body: some View {
        VStack(alignment: .leading, spacing: 6) {
            HStack {
                Text(machine.displayName)
                    .font(.headline)
                
                Spacer()
                
                if machine.isFavorite {
                    Image(systemName: "star.fill")
                        .foregroundStyle(.yellow)
                }
            }
            
            Text(machine.daemonAuthority)
                .font(.subheadline)
                .foregroundStyle(.secondary)
            
            if let lastSeenAt = machine.lastSeenAt {
                Text("Last seen \(lastSeenAt.formatted(date: .abbreviated, time: .shortened))")
                    .font(.caption)
                    .foregroundStyle(.tertiary)
            }
        }
        .padding(.vertical, 4)
    }
}
