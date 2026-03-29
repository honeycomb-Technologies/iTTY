import Foundation

enum DaemonClientError: Error, LocalizedError {
    case invalidBaseURL(String)
    case invalidPathComponent(String)
    case invalidResponse
    case httpStatus(Int, String)
    case encodingFailed
    case decodingFailed(String)
    
    var errorDescription: String? {
        switch self {
        case .invalidBaseURL(let address):
            return "Invalid daemon address: \(address)"
        case .invalidPathComponent(let component):
            return "Invalid daemon request path: \(component)"
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

    private static let defaultSession: URLSession = {
        let configuration = URLSessionConfiguration.ephemeral
        configuration.timeoutIntervalForRequest = 5
        configuration.timeoutIntervalForResource = 10
        return URLSession(configuration: configuration)
    }()

    private static let pathComponentAllowedCharacters: CharacterSet = {
        var characters = CharacterSet.urlPathAllowed
        characters.remove(charactersIn: "/")
        return characters
    }()
    
    init(machine: Machine, session: URLSession = DaemonClient.defaultSession) throws {
        guard let baseURL = machine.daemonBaseURL else {
            throw DaemonClientError.invalidBaseURL(machine.daemonAuthority)
        }
        self.init(baseURL: baseURL, session: session)
    }
    
    init(baseURL: URL, session: URLSession = DaemonClient.defaultSession) {
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
        try await request(pathComponents: ["sessions", name], method: "GET")
    }
    
    func sessionContent(name: String) async throws -> SavedSessionContent {
        try await request(pathComponents: ["sessions", name, "content"], method: "GET")
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

    func peers() async throws -> [TailscalePeer] {
        try await request(path: "/peers", method: "GET")
    }

    func createSession(name: String) async throws -> SavedSessionDetail {
        try await request(path: "/sessions", method: "POST", body: ["name": name])
    }

    func registerDevice(token: String) async throws -> [String: String] {
        try await request(path: "/devices", method: "POST", body: ["token": token])
    }
    
    private func request<T: Decodable>(path: String, method: String) async throws -> T {
        try await performRequest(path: path, method: method, body: nil as AutoWrapRequest?)
    }

    private func request<T: Decodable>(pathComponents: [String], method: String) async throws -> T {
        try await performRequest(pathComponents: pathComponents, method: method, body: nil as AutoWrapRequest?)
    }
    
    private func request<T: Decodable, Body: Encodable>(path: String, method: String, body: Body) async throws -> T {
        try await performRequest(path: path, method: method, body: body)
    }
    
    private func performRequest<T: Decodable, Body: Encodable>(path: String, method: String, body: Body?) async throws -> T {
        let url = try endpoint(path: path)
        return try await performRequest(url: url, method: method, body: body)
    }

    private func performRequest<T: Decodable, Body: Encodable>(pathComponents: [String], method: String, body: Body?) async throws -> T {
        let url = try endpoint(pathComponents: pathComponents)
        return try await performRequest(url: url, method: method, body: body)
    }

    private func performRequest<T: Decodable, Body: Encodable>(url: URL, method: String, body: Body?) async throws -> T {
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

    private func endpoint(pathComponents: [String]) throws -> URL {
        guard var components = URLComponents(url: baseURL, resolvingAgainstBaseURL: false),
              components.scheme != nil,
              components.host != nil else {
            throw DaemonClientError.invalidBaseURL(baseURL.absoluteString)
        }

        let basePath = components.percentEncodedPath
            .split(separator: "/")
            .map(String.init)
        let encodedComponents = try pathComponents.map(Self.encodePathComponent)
        components.percentEncodedPath = "/" + (basePath + encodedComponents).joined(separator: "/")

        guard let url = components.url else {
            throw DaemonClientError.invalidBaseURL(baseURL.absoluteString)
        }
        return url
    }

    private static func encodePathComponent(_ component: String) throws -> String {
        guard let encoded = component.addingPercentEncoding(withAllowedCharacters: pathComponentAllowedCharacters) else {
            throw DaemonClientError.invalidPathComponent(component)
        }
        return encoded
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
