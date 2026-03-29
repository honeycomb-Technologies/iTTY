import Security
import XCTest

enum KeychainTestSupport {
    static func requireWritableKeychain() throws {
        let service = "com.itty.tests.keychain-probe"
        let account = UUID().uuidString
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecValueData as String: Data("probe".utf8),
            kSecAttrAccessible as String: kSecAttrAccessibleWhenUnlockedThisDeviceOnly
        ]

        let status = SecItemAdd(query as CFDictionary, nil)
        switch status {
        case errSecSuccess:
            let cleanup: [String: Any] = [
                kSecClass as String: kSecClassGenericPassword,
                kSecAttrService as String: service,
                kSecAttrAccount as String: account
            ]
            SecItemDelete(cleanup as CFDictionary)
        case errSecMissingEntitlement, errSecNotAvailable, errSecInteractionNotAllowed:
            throw XCTSkip("Keychain-backed tests require a runtime with keychain access; current environment returned OSStatus \(status).")
        default:
            throw KeychainProbeError.unexpectedStatus(status)
        }
    }
}

enum KeychainProbeError: LocalizedError {
    case unexpectedStatus(OSStatus)

    var errorDescription: String? {
        switch self {
        case .unexpectedStatus(let status):
            return "Keychain probe failed with unexpected OSStatus \(status)"
        }
    }
}
