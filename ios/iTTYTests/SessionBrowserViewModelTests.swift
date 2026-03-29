import Foundation
import XCTest
@testable import iTTY

@MainActor
final class SessionBrowserViewModelTests: XCTestCase {
    private final class BodyBox: @unchecked Sendable {
        private let lock = NSLock()
        private var storedValue: [String: String]?

        func set(_ value: [String: String]?) {
            lock.lock()
            storedValue = value
            lock.unlock()
        }

        func get() -> [String: String]? {
            lock.lock()
            let value = storedValue
            lock.unlock()
            return value
        }
    }

    func testNextNewSessionNameIgnoresNonIttySessions() async throws {
        let machine = Machine(name: "Desk", daemonHost: "desk.local")
        let client = DaemonClient(baseURL: URL(string: "http://daemon.test")!) { request in
            switch (request.httpMethod, request.url?.path) {
            case ("GET", "/health"):
                return (Data("""
                {
                  "status": "ok",
                  "version": "1.0",
                  "platform": "linux/amd64",
                  "tmuxInstalled": true,
                  "tmuxVersion": "3.6a"
                }
                """.utf8), Self.response(for: request, statusCode: 200))
            case ("GET", "/sessions"):
                return (Data("""
                [
                  {
                    "name": "main",
                    "windows": 1,
                    "created": "2026-03-28T20:18:00Z",
                    "attached": true,
                    "lastPaneCommand": null,
                    "lastPanePath": null
                  },
                  {
                    "name": "itty-2",
                    "windows": 1,
                    "created": "2026-03-28T20:19:00Z",
                    "attached": false,
                    "lastPaneCommand": null,
                    "lastPanePath": null
                  }
                ]
                """.utf8), Self.response(for: request, statusCode: 200))
            default:
                XCTFail("unexpected request \(request.httpMethod ?? "<nil>") \(request.url?.path ?? "<nil>")")
                return (Data(), Self.response(for: request, statusCode: 500))
            }
        }

        let viewModel = SessionBrowserViewModel(machine: machine, client: client)
        await viewModel.load()

        XCTAssertEqual(viewModel.nextNewSessionName(), "itty-3")
    }

    func testCreateSessionUsesNextFreshIttyName() async throws {
        let machine = Machine(name: "Desk", daemonHost: "desk.local")
        let capturedBody = BodyBox()

        let client = DaemonClient(baseURL: URL(string: "http://daemon.test")!) { request in
            switch (request.httpMethod, request.url?.path) {
            case ("GET", "/health"):
                return (Data("""
                {
                  "status": "ok",
                  "version": "1.0",
                  "platform": "linux/amd64",
                  "tmuxInstalled": true,
                  "tmuxVersion": "3.6a"
                }
                """.utf8), Self.response(for: request, statusCode: 200))
            case ("GET", "/sessions"):
                return (Data("""
                [
                  {
                    "name": "main",
                    "windows": 1,
                    "created": "2026-03-28T20:18:00Z",
                    "attached": true,
                    "lastPaneCommand": null,
                    "lastPanePath": null
                  },
                  {
                    "name": "itty-2",
                    "windows": 1,
                    "created": "2026-03-28T20:19:00Z",
                    "attached": false,
                    "lastPaneCommand": null,
                    "lastPanePath": null
                  }
                ]
                """.utf8), Self.response(for: request, statusCode: 200))
            case ("POST", "/sessions"):
                let payload = try JSONSerialization.jsonObject(with: XCTUnwrap(request.httpBody)) as? [String: String]
                capturedBody.set(payload)
                return (Data("""
                {
                  "name": "itty-3",
                  "windows": 1,
                  "created": "2026-03-28T20:20:00Z",
                  "attached": false,
                  "lastPaneCommand": null,
                  "lastPanePath": null,
                  "windowList": []
                }
                """.utf8), Self.response(for: request, statusCode: 201))
            default:
                XCTFail("unexpected request \(request.httpMethod ?? "<nil>") \(request.url?.path ?? "<nil>")")
                return (Data(), Self.response(for: request, statusCode: 500))
            }
        }

        let viewModel = SessionBrowserViewModel(machine: machine, client: client)
        await viewModel.load()
        let created = try await viewModel.createSession()

        XCTAssertEqual(capturedBody.get()?["name"], "itty-3")
        XCTAssertEqual(created.name, "itty-3")
        XCTAssertTrue(viewModel.sessions.contains(where: { $0.name == "itty-3" }))
    }

    nonisolated private static func response(for request: URLRequest, statusCode: Int) -> HTTPURLResponse {
        HTTPURLResponse(url: request.url!, statusCode: statusCode, httpVersion: nil, headerFields: nil)!
    }
}
