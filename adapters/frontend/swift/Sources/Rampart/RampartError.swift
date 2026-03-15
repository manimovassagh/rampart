import Foundation

/// Error returned by the Rampart server or raised by the client.
public enum RampartError: LocalizedError, Sendable {

    /// The authorization URL could not be opened.
    case launchFailed(String)

    /// The callback URL is missing required parameters.
    case invalidCallback(String)

    /// The OAuth state parameter does not match (possible CSRF).
    case stateMismatch

    /// The PKCE code verifier was not found in storage.
    case missingVerifier

    /// No refresh token is available.
    case noRefreshToken

    /// The server returned an OAuth/HTTP error.
    case serverError(error: String, description: String, status: Int)

    /// An unexpected error occurred.
    case unknown(String)

    public var errorDescription: String? {
        switch self {
        case .launchFailed(let msg):
            return "Launch failed: \(msg)"
        case .invalidCallback(let msg):
            return "Invalid callback: \(msg)"
        case .stateMismatch:
            return "State parameter does not match. Possible CSRF attack."
        case .missingVerifier:
            return "Code verifier not found in Keychain."
        case .noRefreshToken:
            return "No refresh token available."
        case .serverError(let error, let description, let status):
            return "\(error): \(description) [HTTP \(status)]"
        case .unknown(let msg):
            return msg
        }
    }

    /// Parse an error from a Rampart server JSON response.
    static func fromResponse(data: Data, status: Int) -> RampartError {
        if let json = try? JSONSerialization.jsonObject(with: data) as? [String: Any] {
            let error = json["error"] as? String ?? "unknown_error"
            let description = json["error_description"] as? String ?? "An unknown error occurred."
            return .serverError(error: error, description: description, status: status)
        }
        return .serverError(
            error: "unknown_error",
            description: "HTTP \(status)",
            status: status
        )
    }
}
