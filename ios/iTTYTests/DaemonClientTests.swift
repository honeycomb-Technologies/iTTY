import Foundation
import XCTest
@testable import iTTY

final class DaemonClientTests: XCTestCase {
    func testHealthDecodesImplementedDaemonShape() async throws {
        let client = DaemonClient(baseURL: URL(string: "http://daemon.test:8080")!) { request in
            XCTAssertEqual(request.url?.path, "/health")
            let body = """
            {
              "status": "ok",
              "version": "0.1.0",
              "platform": "linux/amd64",
              "tmuxInstalled": true,
              "tmuxVersion": "3.6a"
            }
            """
            return (Data(body.utf8), Self.response(for: request, statusCode: 200))
        }
        
        let health = try await client.health()
        
        XCTAssertEqual(health.status, "ok")
        XCTAssertEqual(health.version, "0.1.0")
        XCTAssertEqual(health.platform, "linux/amd64")
        XCTAssertTrue(health.tmuxInstalled)
        XCTAssertEqual(health.tmuxVersion, "3.6a")
    }
    
    func testListSessionsDecodesRFC3339Dates() async throws {
        let client = DaemonClient(baseURL: URL(string: "http://daemon.test")!) { request in
            XCTAssertEqual(request.url?.path, "/sessions")
            let body = """
            [
              {
                "name": "main",
                "windows": 2,
                "created": "2026-03-28T20:18:00Z",
                "attached": true,
                "lastPaneCommand": "nvim",
                "lastPanePath": "/work/project"
              }
            ]
            """
            return (Data(body.utf8), Self.response(for: request, statusCode: 200))
        }
        
        let sessions = try await client.listSessions()
        
        XCTAssertEqual(sessions.count, 1)
        XCTAssertEqual(sessions[0].name, "main")
        XCTAssertEqual(sessions[0].windows, 2)
        XCTAssertTrue(sessions[0].attached)
        XCTAssertEqual(sessions[0].lastPaneCommand, "nvim")
        XCTAssertEqual(sessions[0].lastPanePath, "/work/project")
    }
    
    func testSetAutoWrapEncodesBooleanBody() async throws {
        var capturedBody: Data?
        let client = DaemonClient(baseURL: URL(string: "http://daemon.test")!) { request in
            XCTAssertEqual(request.httpMethod, "PUT")
            XCTAssertEqual(request.url?.path, "/config/auto")
            capturedBody = request.httpBody
            
            let body = """
            {
              "listenAddr": ":8080",
              "tmuxPath": "tmux",
              "autoWrap": true,
              "tailscaleServe": true,
              "apnsKeyPath": "",
              "apnsKeyID": "",
              "apnsTeamID": ""
            }
            """
            return (Data(body.utf8), Self.response(for: request, statusCode: 200))
        }
        
        let config = try await client.setAutoWrap(enabled: true)
        
        XCTAssertTrue(config.autoWrap)
        XCTAssertNotNil(capturedBody)
        let json = try XCTUnwrap(try JSONSerialization.jsonObject(with: XCTUnwrap(capturedBody)) as? [String: Bool])
        XCTAssertEqual(json["enabled"], true)
    }
    
    func testHTTPErrorUsesDaemonEnvelope() async {
        let client = DaemonClient(baseURL: URL(string: "http://daemon.test")!) { request in
            let body = #"{"error":"session not found: missing"}"#
            return (Data(body.utf8), Self.response(for: request, statusCode: 404))
        }
        
        do {
            _ = try await client.sessionDetail(name: "missing")
            XCTFail("expected request to fail")
        } catch {
            XCTAssertEqual(error.localizedDescription, "session not found: missing")
        }
    }
    
    private static func response(for request: URLRequest, statusCode: Int) -> HTTPURLResponse {
        HTTPURLResponse(url: request.url!, statusCode: statusCode, httpVersion: nil, headerFields: nil)!
    }
}
