enum RuntimeEnvironment {
    static let simulatorTerminalUnavailableMessage =
        "Terminal sessions are not available in the iOS simulator yet. Use a physical device to open a live terminal."
    
    static var supportsLiveTerminalSessions: Bool {
        #if targetEnvironment(simulator)
        return false
        #else
        return true
        #endif
    }
}
