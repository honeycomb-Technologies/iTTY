import UIKit

@MainActor
struct TailscaleVPNDetector {
    enum Availability: Equatable {
        case installed
        case notInstalled
    }
    
    private static let appURL = URL(string: "tailscale://")!
    private static let appStoreURL = URL(string: "https://apps.apple.com/us/app/tailscale/id1475387142")!
    
    func detectAvailability() -> Availability {
        UIApplication.shared.canOpenURL(Self.appURL) ? .installed : .notInstalled
    }
    
    func openApp() {
        UIApplication.shared.open(Self.appURL)
    }
    
    func openInstallPage() {
        UIApplication.shared.open(Self.appStoreURL)
    }
}
