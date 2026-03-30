// iTTY — Daemon Terminal Connection
//
// Connects to the daemon's WebSocket terminal proxy for bidirectional
// terminal I/O. Replaces SSH for machines running the iTTY daemon.
// No credentials needed — the daemon handles tmux attachment directly.

import Foundation
import os.log

private let logger = Logger(subsystem: "com.itty", category: "DaemonTerminal")

/// Delegate for receiving terminal data and lifecycle events from a
/// daemon WebSocket connection. Mirrors the SSH delegate pattern so
/// SSHSession can use either transport interchangeably.
@MainActor
protocol DaemonTerminalConnectionDelegate: AnyObject {
    func daemonTerminalDidConnect()
    func daemonTerminalDidReceiveData(_ data: Data)
    func daemonTerminalDidDisconnect(error: Error?)
}

/// Connects to `ws://<host>:<port>/ws/terminal?session=<name>` and
/// pipes raw bytes bidirectionally. Binary frames carry terminal I/O,
/// text frames carry control messages (resize, ping).
@MainActor
final class DaemonTerminalConnection {
    enum State: Equatable {
        case disconnected
        case connecting
        case connected
    }

    private(set) var state: State = .disconnected
    weak var delegate: DaemonTerminalConnectionDelegate?

    private let machine: Machine
    private let sessionName: String
    private var webSocketTask: URLSessionWebSocketTask?
    private let urlSession: URLSession

    var cols: Int = 80
    var rows: Int = 24

    init(machine: Machine, sessionName: String, urlSession: URLSession = .shared) {
        self.machine = machine
        self.sessionName = sessionName
        self.urlSession = urlSession
    }

    deinit {
        webSocketTask?.cancel(with: .goingAway, reason: nil)
    }

    // MARK: - Connection

    func connect() {
        guard state == .disconnected else { return }

        guard let url = terminalURL() else {
            logger.error("Invalid terminal WebSocket URL for \(self.machine.daemonHost)")
            delegate?.daemonTerminalDidDisconnect(error: nil)
            return
        }

        state = .connecting
        logger.info("Connecting to terminal WebSocket: \(url.absoluteString)")

        let task = urlSession.webSocketTask(with: url)
        webSocketTask = task
        task.resume()

        state = .connected
        delegate?.daemonTerminalDidConnect()

        // Send initial resize
        sendResize(cols: cols, rows: rows)

        // Start reading
        listenForMessages()
    }

    func disconnect() {
        guard state != .disconnected else { return }
        webSocketTask?.cancel(with: .normalClosure, reason: nil)
        webSocketTask = nil
        state = .disconnected
    }

    // MARK: - I/O

    /// Send user input to the terminal.
    func write(_ data: Data) {
        guard state == .connected else { return }
        webSocketTask?.send(.data(data)) { error in
            if let error {
                logger.error("Terminal write error: \(error.localizedDescription)")
            }
        }
    }

    /// Send a PTY resize to the daemon.
    func sendResize(cols: Int, rows: Int) {
        self.cols = cols
        self.rows = rows
        guard state == .connected else { return }

        let json = "{\"type\":\"resize\",\"cols\":\(cols),\"rows\":\(rows)}"
        webSocketTask?.send(.string(json)) { error in
            if let error {
                logger.error("Terminal resize error: \(error.localizedDescription)")
            }
        }
    }

    // MARK: - Private

    private func terminalURL() -> URL? {
        var components = URLComponents()
        components.scheme = machine.daemonScheme == "https" ? "wss" : "ws"
        components.host = machine.daemonHost
        components.port = machine.daemonPort
        components.path = "/ws/terminal"
        components.queryItems = [URLQueryItem(name: "session", value: sessionName)]
        return components.url
    }

    private func listenForMessages() {
        webSocketTask?.receive { [weak self] result in
            Task { @MainActor [weak self] in
                guard let self, self.state == .connected else { return }

                switch result {
                case .success(let message):
                    self.handleMessage(message)
                    self.listenForMessages()

                case .failure(let error):
                    logger.error("Terminal WebSocket error: \(error.localizedDescription)")
                    self.state = .disconnected
                    self.delegate?.daemonTerminalDidDisconnect(error: error)
                }
            }
        }
    }

    private func handleMessage(_ message: URLSessionWebSocketTask.Message) {
        switch message {
        case .data(let data):
            // Raw terminal output
            delegate?.daemonTerminalDidReceiveData(data)

        case .string(let text):
            // Control message — for now just log it
            if text.contains("\"type\":\"pong\"") {
                return // Heartbeat response, ignore
            }
            // Terminal output can also come as text in some cases
            if let data = text.data(using: .utf8) {
                delegate?.daemonTerminalDidReceiveData(data)
            }

        @unknown default:
            break
        }
    }
}
