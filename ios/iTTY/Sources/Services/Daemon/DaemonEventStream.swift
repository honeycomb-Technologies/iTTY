// iTTY — Daemon Event Stream
//
// Connects to the daemon's WebSocket endpoint and publishes
// real-time session events. Falls back to HTTP polling if the
// WebSocket connection fails.

import Combine
import Foundation
import os.log

private let logger = Logger(subsystem: "com.itty", category: "DaemonEventStream")

/// A real-time event from the daemon's WebSocket stream.
struct DaemonEvent: Codable, Equatable {
    let type: String
    let session: SavedSession
}

/// Connects to a daemon's `/ws` endpoint and streams session events.
@MainActor
final class DaemonEventStream: ObservableObject {
    enum ConnectionState: Equatable {
        case disconnected
        case connecting
        case connected
        case failed(String)
    }

    @Published private(set) var state: ConnectionState = .disconnected
    @Published private(set) var latestEvent: DaemonEvent?

    private let machine: Machine
    private var webSocketTask: URLSessionWebSocketTask?
    private var reconnectTask: Task<Void, Never>?
    private let session: URLSession

    init(machine: Machine, session: URLSession = .shared) {
        self.machine = machine
        self.session = session
    }

    deinit {
        webSocketTask?.cancel(with: .goingAway, reason: nil)
        reconnectTask?.cancel()
    }

    /// Start the WebSocket connection.
    func connect() {
        guard state == .disconnected || state != .connecting else { return }

        guard let url = webSocketURL() else {
            state = .failed("Invalid daemon address")
            return
        }

        state = .connecting
        logger.info("Connecting to WebSocket at \(url.absoluteString)")

        let task = session.webSocketTask(with: url)
        webSocketTask = task
        task.resume()

        state = .connected
        logger.info("WebSocket connected")

        listenForMessages()
    }

    /// Disconnect the WebSocket.
    func disconnect() {
        reconnectTask?.cancel()
        reconnectTask = nil
        webSocketTask?.cancel(with: .normalClosure, reason: nil)
        webSocketTask = nil
        state = .disconnected
    }

    // MARK: - Private

    private func webSocketURL() -> URL? {
        guard var components = URLComponents() as URLComponents? else { return nil }
        components.scheme = machine.daemonScheme == "https" ? "wss" : "ws"
        components.host = machine.daemonHost
        components.port = machine.daemonPort
        components.path = "/ws"
        return components.url
    }

    private func listenForMessages() {
        webSocketTask?.receive { [weak self] result in
            Task { @MainActor [weak self] in
                guard let self else { return }

                switch result {
                case .success(let message):
                    self.handleMessage(message)
                    self.listenForMessages()

                case .failure(let error):
                    logger.error("WebSocket error: \(error.localizedDescription)")
                    self.state = .failed(error.localizedDescription)
                    self.scheduleReconnect()
                }
            }
        }
    }

    private func handleMessage(_ message: URLSessionWebSocketTask.Message) {
        let data: Data
        switch message {
        case .string(let text):
            data = Data(text.utf8)
        case .data(let d):
            data = d
        @unknown default:
            return
        }

        do {
            let decoder = JSONDecoder()
            decoder.dateDecodingStrategy = .custom { decoder in
                let container = try decoder.singleValueContainer()
                let value = try container.decode(String.self)
                let formatter = ISO8601DateFormatter()
                formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
                if let date = formatter.date(from: value) { return date }
                formatter.formatOptions = [.withInternetDateTime]
                if let date = formatter.date(from: value) { return date }
                throw DecodingError.dataCorruptedError(in: container, debugDescription: "unsupported date: \(value)")
            }
            let event = try decoder.decode(DaemonEvent.self, from: data)
            latestEvent = event
            logger.debug("Event: \(event.type) session=\(event.session.name)")
        } catch {
            logger.warning("Failed to decode event: \(error.localizedDescription)")
        }
    }

    private func scheduleReconnect() {
        reconnectTask?.cancel()
        reconnectTask = Task { [weak self] in
            try? await Task.sleep(nanoseconds: 3_000_000_000)
            guard !Task.isCancelled else { return }
            await MainActor.run {
                self?.connect()
            }
        }
    }
}
