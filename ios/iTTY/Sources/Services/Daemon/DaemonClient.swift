import Foundation

enum DaemonClientError: Error, LocalizedError {
    case invalidBaseURL(String)
    case invalidResponse
    case httpStatus(Int, String)
    case encodingFailed
    case decodingFailed(String)
    
    var errorDescription: String? {
        switch self {
        case .invalidBaseURL(let address):
            return "Invalid daemon address: \(address)"
        case .invalidResponse:
            return "The daemon returned an invalid response."
        case .httpStatus(let statusCode, let message):
            return message.isEmpty ? "Daemon request failed with status \(statusCode)." : message
        case .encodingFailed:
            return "Unable to encode the daemon request."
        case .decodingFailed(let message):
            return "Unable to decode daemon response: \(message)"
        }
    }
}

private struct AutoWrapRequest: Encodable {
    let enabled: Bool
}

struct DaemonClient {
    typealias DataLoader = @Sendable (URLRequest) async throws -> (Data, URLResponse)
    
    private let baseURL: URL
    private let load: DataLoader
    private let decoder: JSONDecoder
    private let encoder: JSONEncoder
    
    init(machine: Machine, session: URLSession = .shared) throws {
        guard let baseURL = machine.daemonBaseURL else {
            throw DaemonClientError.invalidBaseURL(machine.daemonAuthority)
        }
        self.init(baseURL: baseURL, session: session)
    }
    
    init(baseURL: URL, session: URLSession = .shared) {
        self.init(baseURL: baseURL) { request in
            try await session.data(for: request)
        }
    }
    
    init(baseURL: URL, load: @escaping DataLoader) {
        self.baseURL = baseURL
        self.load = load
        self.decoder = DaemonClient.makeDecoder()
        self.encoder = JSONEncoder()
    }
    
    func health() async throws -> DaemonHealth {
        try await request(path: "/health", method: "GET")
    }
    
    func listSessions() async throws -> [SavedSession] {
        try await request(path: "/sessions", method: "GET")
    }
    
    func sessionDetail(name: String) async throws -> SavedSessionDetail {
        try await request(path: "/sessions/\(name.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? name)", method: "GET")
    }
    
    func sessionContent(name: String) async throws -> SavedSessionContent {
        try await request(path: "/sessions/\(name.addingPercentEncoding(withAllowedCharacters: .urlPathAllowed) ?? name)/content", method: "GET")
    }
    
    func config() async throws -> DaemonConfig {
        try await request(path: "/config", method: "GET")
    }
    
    func setAutoWrap(enabled: Bool) async throws -> DaemonConfig {
        try await request(path: "/config/auto", method: "PUT", body: AutoWrapRequest(enabled: enabled))
    }
    
    func windows() async throws -> [DesktopWindow] {
        try await request(path: "/windows", method: "GET")
    }
    
    private func request<T: Decodable>(path: String, method: String) async throws -> T {
        try await performRequest(path: path, method: method, body: nil as AutoWrapRequest?)
    }
    
    private func request<T: Decodable, Body: Encodable>(path: String, method: String, body: Body) async throws -> T {
        try await performRequest(path: path, method: method, body: body)
    }
    
    private func performRequest<T: Decodable, Body: Encodable>(path: String, method: String, body: Body?) async throws -> T {
        let url = try endpoint(path: path)
        var request = URLRequest(url: url)
        request.httpMethod = method
        request.setValue("application/json", forHTTPHeaderField: "Accept")
        
        if let body {
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            do {
                request.httpBody = try encoder.encode(body)
            } catch {
                throw DaemonClientError.encodingFailed
            }
        }
        
        let (data, response) = try await load(request)
        guard let httpResponse = response as? HTTPURLResponse else {
            throw DaemonClientError.invalidResponse
        }
        
        guard (200...299).contains(httpResponse.statusCode) else {
            let message: String
            if let envelope = try? decoder.decode(DaemonErrorEnvelope.self, from: data) {
                message = envelope.error
            } else {
                message = String(data: data, encoding: .utf8) ?? ""
            }
            throw DaemonClientError.httpStatus(httpResponse.statusCode, message)
        }
        
        do {
            return try decoder.decode(T.self, from: data)
        } catch {
            throw DaemonClientError.decodingFailed(String(describing: error))
        }
    }
    
    private func endpoint(path: String) throws -> URL {
        guard var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false),
              components.scheme != nil,
              components.host != nil else {
            throw DaemonClientError.invalidBaseURL(baseURL.absoluteString)
        }
        
        let basePath = components.path.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        let suffix = path.trimmingCharacters(in: CharacterSet(charactersIn: "/"))
        components.path = [basePath, suffix].filter { !$0.isEmpty }.joined(separator: "/")
        components.path = "/" + components.path
        
        guard let url = components.url else {
            throw DaemonClientError.invalidBaseURL(baseURL.absoluteString)
        }
        return url
    }
    
    private static func makeDecoder() -> JSONDecoder {
        let decoder = JSONDecoder()
        decoder.dateDecodingStrategy = .custom { decoder in
            let container = try decoder.singleValueContainer()
            let value = try container.decode(String.self)
            
            if let date = fractionalSecondsFormatter.date(from: value) ?? internetDateFormatter.date(from: value) {
                return date
            }
            
            throw DecodingError.dataCorruptedError(in: container, debugDescription: "unsupported date: \(value)")
        }
        return decoder
    }
}

private let fractionalSecondsFormatter: ISO8601DateFormatter = {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime, .withFractionalSeconds]
    return formatter
}()

private let internetDateFormatter: ISO8601DateFormatter = {
    let formatter = ISO8601DateFormatter()
    formatter.formatOptions = [.withInternetDateTime]
    return formatter
}()
