// iTTY — Notification Manager
//
// Handles APNs device token registration, notification permission
// requests, and notification tap routing.

import Foundation
import UIKit
import UserNotifications
import os.log

private let logger = Logger(subsystem: "com.itty", category: "NotificationManager")

@MainActor
final class NotificationManager: NSObject, ObservableObject {
    static let shared = NotificationManager()

    @Published private(set) var isAuthorized = false
    @Published private(set) var deviceToken: String?

    private var registeredDaemons: Set<String> = []

    private override init() {
        super.init()
    }

    /// Request notification permission and register for remote notifications.
    func requestPermission() async {
        let center = UNUserNotificationCenter.current()

        do {
            let granted = try await center.requestAuthorization(options: [.alert, .badge, .sound])
            isAuthorized = granted

            if granted {
                await MainActor.run {
                    UIApplication.shared.registerForRemoteNotifications()
                }
                logger.info("Notification permission granted")
            } else {
                logger.info("Notification permission denied")
            }
        } catch {
            logger.error("Notification permission error: \(error.localizedDescription)")
        }
    }

    /// Called by the app delegate when APNs returns a device token.
    func didRegisterForRemoteNotifications(deviceToken data: Data) {
        let token = data.map { String(format: "%02x", $0) }.joined()
        self.deviceToken = token
        logger.info("APNs device token: \(token.prefix(8))...\(token.suffix(4))")
    }

    /// Called by the app delegate when APNs registration fails.
    func didFailToRegisterForRemoteNotifications(error: Error) {
        logger.error("APNs registration failed: \(error.localizedDescription)")
    }

    /// Register the device token with a daemon for push notifications.
    func registerWithDaemon(_ machine: Machine) async {
        guard let token = deviceToken else {
            logger.debug("No device token — skipping daemon registration")
            return
        }

        let key = machine.daemonHost
        guard !registeredDaemons.contains(key) else { return }

        do {
            let client = try DaemonClient(machine: machine)
            try await client.registerDevice(token: token)
            registeredDaemons.insert(key)
            logger.info("Registered device token with daemon at \(machine.daemonHost)")
        } catch {
            logger.warning("Failed to register with daemon at \(machine.daemonHost): \(error.localizedDescription)")
        }
    }
}

// MARK: - UNUserNotificationCenterDelegate

extension NotificationManager: UNUserNotificationCenterDelegate {
    nonisolated func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        willPresent notification: UNNotification
    ) async -> UNNotificationPresentationOptions {
        // Show notification even when app is in foreground.
        [.banner, .badge, .sound]
    }

    nonisolated func userNotificationCenter(
        _ center: UNUserNotificationCenter,
        didReceive response: UNNotificationResponse
    ) async {
        let userInfo = response.notification.request.content.userInfo
        if let sessionName = userInfo["sessionName"] as? String {
            await MainActor.run {
                logger.info("Notification tapped for session: \(sessionName)")
                // Navigation to the session will be wired in the UI phase.
            }
        }
    }
}
