import AuthenticationServices
import CryptoKit
import Foundation
import Security

/// Configuration for ``RampartClient``.
public struct RampartClientConfig: Sendable {

    /// Rampart server URL (e.g. "https://auth.example.com").
    public let issuer: String

    /// OAuth 2.0 client ID registered with the Rampart server.
    public let clientId: String

    /// OAuth 2.0 redirect URI using a custom URL scheme (e.g. "com.example.app://callback").
    public let redirectUri: String

    /// OAuth 2.0 scopes (default: "openid").
    public let scope: String

    /// Custom `URLSession` for testing or advanced configuration.
    public let urlSession: URLSession

    public init(
        issuer: String,
        clientId: String,
        redirectUri: String,
        scope: String = "openid",
        urlSession: URLSession = .shared
    ) {
        // Strip trailing slashes from issuer.
        var trimmed = issuer
        while trimmed.hasSuffix("/") {
            trimmed = String(trimmed.dropLast())
        }
        self.issuer = trimmed
        self.clientId = clientId
        self.redirectUri = redirectUri
        self.scope = scope
        self.urlSession = urlSession
    }
}

/// Token set returned by login and refresh operations.
public struct RampartTokens: Codable, Sendable {
    public let accessToken: String
    public let refreshToken: String
    public let tokenType: String
    public let expiresIn: Int

    enum CodingKeys: String, CodingKey {
        case accessToken = "access_token"
        case refreshToken = "refresh_token"
        case tokenType = "token_type"
        case expiresIn = "expires_in"
    }
}

// MARK: - RampartClient

/// Native Swift authentication client for Rampart IAM.
///
/// Implements OAuth 2.0 Authorization Code flow with PKCE using
/// `ASWebAuthenticationSession`. Tokens are stored in the system Keychain.
///
/// Usage:
/// ```swift
/// let client = RampartClient(config: .init(
///     issuer: "https://auth.example.com",
///     clientId: "my-ios-app",
///     redirectUri: "com.example.app://callback"
/// ))
///
/// try await client.loadStoredTokens()
/// if !client.isAuthenticated {
///     try await client.loginWithRedirect()
/// }
/// let user = try await client.getUser()
/// ```
@MainActor
public final class RampartClient {

    private let config: RampartClientConfig
    private var tokens: RampartTokens?

    // Keychain service keys
    private static let keychainService = "com.rampart.auth"
    private static let keychainTokensKey = "rampart_tokens"
    private static let keychainVerifierKey = "rampart_pkce_verifier"
    private static let keychainStateKey = "rampart_pkce_state"

    /// Whether the client holds a non-expired access token.
    public var isAuthenticated: Bool {
        guard let token = tokens?.accessToken else { return false }
        return !isTokenExpired(token)
    }

    /// The current access token, or `nil`.
    public var accessToken: String? { tokens?.accessToken }

    /// Creates a new ``RampartClient``.
    ///
    /// Call ``loadStoredTokens()`` after construction to restore a previous session.
    public init(config: RampartClientConfig) {
        self.config = config
    }

    // MARK: - Public API

    /// Load tokens from the Keychain. Call once at app startup to restore a
    /// previous session.
    public func loadStoredTokens() {
        guard let data = Keychain.read(
            service: Self.keychainService,
            account: Self.keychainTokensKey
        ) else { return }

        do {
            tokens = try JSONDecoder().decode(RampartTokens.self, from: data)
        } catch {
            Keychain.delete(service: Self.keychainService, account: Self.keychainTokensKey)
        }
    }

    /// Start the OAuth login flow using `ASWebAuthenticationSession`.
    ///
    /// This presents a system authentication sheet, handles the browser
    /// redirect, and exchanges the authorization code for tokens automatically.
    ///
    /// - Parameter contextProvider: An object that conforms to
    ///   `ASWebAuthenticationPresentationContextProviding`. On iOS this is
    ///   typically your root view controller or a SwiftUI wrapper.
    /// - Returns: The token set received from the server.
    @discardableResult
    public func loginWithRedirect(
        contextProvider: ASWebAuthenticationPresentationContextProviding? = nil
    ) async throws -> RampartTokens {
        let verifier = generateCodeVerifier()
        let challenge = generateCodeChallenge(verifier)
        let state = generateState()

        // Persist PKCE values so they survive across the auth session.
        Keychain.write(
            Data(verifier.utf8),
            service: Self.keychainService,
            account: Self.keychainVerifierKey
        )
        Keychain.write(
            Data(state.utf8),
            service: Self.keychainService,
            account: Self.keychainStateKey
        )

        var components = URLComponents(string: "\(config.issuer)/oauth/authorize")!
        components.queryItems = [
            URLQueryItem(name: "response_type", value: "code"),
            URLQueryItem(name: "client_id", value: config.clientId),
            URLQueryItem(name: "redirect_uri", value: config.redirectUri),
            URLQueryItem(name: "scope", value: config.scope),
            URLQueryItem(name: "state", value: state),
            URLQueryItem(name: "code_challenge", value: challenge),
            URLQueryItem(name: "code_challenge_method", value: "S256"),
        ]

        guard let authorizeURL = components.url else {
            throw RampartError.launchFailed("Could not construct authorization URL.")
        }

        // Extract the custom scheme from the redirect URI for the callback.
        guard let scheme = URL(string: config.redirectUri)?.scheme else {
            throw RampartError.launchFailed("Invalid redirect URI scheme.")
        }

        let callbackURL = try await withCheckedThrowingContinuation { (continuation: CheckedContinuation<URL, Error>) in
            let session = ASWebAuthenticationSession(
                url: authorizeURL,
                callbackURLScheme: scheme
            ) { url, error in
                if let error = error {
                    continuation.resume(throwing: RampartError.launchFailed(error.localizedDescription))
                } else if let url = url {
                    continuation.resume(returning: url)
                } else {
                    continuation.resume(throwing: RampartError.launchFailed("No callback URL received."))
                }
            }
            session.prefersEphemeralWebBrowserSession = false

            if let provider = contextProvider {
                session.presentationContextProvider = provider
            }

            session.start()
        }

        return try await handleCallback(callbackURL)
    }

    /// Handle the OAuth callback URL.
    ///
    /// Validates the state parameter, exchanges the authorization code for
    /// tokens, and persists them in the Keychain.
    ///
    /// - Parameter url: The full callback URL including query parameters.
    /// - Returns: The token set received from the server.
    @discardableResult
    public func handleCallback(_ url: URL) async throws -> RampartTokens {
        guard let components = URLComponents(url: url, resolvingAgainstBaseURL: false) else {
            throw RampartError.invalidCallback("Could not parse callback URL.")
        }

        let params = Dictionary(
            uniqueKeysWithValues: (components.queryItems ?? []).compactMap { item in
                item.value.map { (item.name, $0) }
            }
        )

        if let error = params["error"] {
            let description = params["error_description"] ?? error
            cleanupPkceStorage()
            throw RampartError.serverError(error: error, description: description, status: 0)
        }

        guard let code = params["code"], let state = params["state"] else {
            cleanupPkceStorage()
            throw RampartError.invalidCallback("Missing code or state parameter in callback URL.")
        }

        // Validate state.
        let storedStateData = Keychain.read(
            service: Self.keychainService,
            account: Self.keychainStateKey
        )
        let storedState = storedStateData.flatMap { String(data: $0, encoding: .utf8) }

        guard storedState == state else {
            cleanupPkceStorage()
            throw RampartError.stateMismatch
        }

        // Retrieve code verifier.
        let verifierData = Keychain.read(
            service: Self.keychainService,
            account: Self.keychainVerifierKey
        )
        guard let verifier = verifierData.flatMap({ String(data: $0, encoding: .utf8) }) else {
            cleanupPkceStorage()
            throw RampartError.missingVerifier
        }

        cleanupPkceStorage()

        // Exchange code for tokens.
        let body = [
            "grant_type": "authorization_code",
            "code": code,
            "client_id": config.clientId,
            "redirect_uri": config.redirectUri,
            "code_verifier": verifier,
        ]

        let tokenURL = URL(string: "\(config.issuer)/oauth/token")!
        var request = URLRequest(url: tokenURL)
        request.httpMethod = "POST"
        request.setValue("application/x-www-form-urlencoded", forHTTPHeaderField: "Content-Type")
        request.httpBody = body
            .map { "\($0.key)=\(urlEncode($0.value))" }
            .joined(separator: "&")
            .data(using: .utf8)

        let (data, response) = try await config.urlSession.data(for: request)
        let httpResponse = response as! HTTPURLResponse

        guard httpResponse.statusCode == 200 else {
            throw RampartError.fromResponse(data: data, status: httpResponse.statusCode)
        }

        let newTokens = try JSONDecoder().decode(RampartTokens.self, from: data)
        setTokens(newTokens)
        return newTokens
    }

    /// Fetch the authenticated user profile from the `/me` endpoint.
    public func getUser() async throws -> RampartClaims {
        let (data, response) = try await authFetch(
            URL(string: "\(config.issuer)/me")!
        )
        let httpResponse = response as! HTTPURLResponse

        guard httpResponse.statusCode == 200 else {
            throw RampartError.fromResponse(data: data, status: httpResponse.statusCode)
        }

        return try JSONDecoder().decode(RampartClaims.self, from: data)
    }

    /// Make an authenticated HTTP request with automatic Bearer token.
    ///
    /// On a 401 response, attempts one silent token refresh and retries.
    ///
    /// - Parameters:
    ///   - url: The request URL.
    ///   - method: HTTP method (default: "GET").
    ///   - headers: Additional HTTP headers.
    ///   - body: Optional request body data.
    /// - Returns: A tuple of response data and `URLResponse`.
    public func authFetch(
        _ url: URL,
        method: String = "GET",
        headers: [String: String]? = nil,
        body: Data? = nil
    ) async throws -> (Data, URLResponse) {
        func doFetch() async throws -> (Data, URLResponse) {
            var request = URLRequest(url: url)
            request.httpMethod = method
            request.httpBody = body

            if let token = tokens?.accessToken {
                request.setValue("Bearer \(token)", forHTTPHeaderField: "Authorization")
            }
            headers?.forEach { request.setValue($0.value, forHTTPHeaderField: $0.key) }

            return try await config.urlSession.data(for: request)
        }

        let (data, response) = try await doFetch()
        let httpResponse = response as! HTTPURLResponse

        if httpResponse.statusCode == 401, tokens?.refreshToken != nil {
            do {
                try await refresh()
                return try await doFetch()
            } catch {
                // Refresh failed -- return the original 401.
                return (data, response)
            }
        }

        return (data, response)
    }

    /// Refresh the access token using the stored refresh token.
    @discardableResult
    public func refresh() async throws -> RampartTokens {
        guard let refreshToken = tokens?.refreshToken else {
            throw RampartError.noRefreshToken
        }

        let refreshURL = URL(string: "\(config.issuer)/token/refresh")!
        var request = URLRequest(url: refreshURL)
        request.httpMethod = "POST"
        request.setValue("application/json", forHTTPHeaderField: "Content-Type")
        request.httpBody = try JSONEncoder().encode(["refresh_token": refreshToken])

        let (data, response) = try await config.urlSession.data(for: request)
        let httpResponse = response as! HTTPURLResponse

        guard httpResponse.statusCode == 200 else {
            setTokens(nil)
            throw RampartError.fromResponse(data: data, status: httpResponse.statusCode)
        }

        let json = try JSONSerialization.jsonObject(with: data) as? [String: Any] ?? [:]
        let updated = RampartTokens(
            accessToken: json["access_token"] as? String ?? "",
            refreshToken: refreshToken,
            tokenType: json["token_type"] as? String ?? "Bearer",
            expiresIn: json["expires_in"] as? Int ?? 3600
        )
        setTokens(updated)
        return updated
    }

    /// Logout -- invalidates the refresh token on the server and clears local tokens.
    public func logout() async {
        if let refreshToken = tokens?.refreshToken {
            let logoutURL = URL(string: "\(config.issuer)/logout")!
            var request = URLRequest(url: logoutURL)
            request.httpMethod = "POST"
            request.setValue("application/json", forHTTPHeaderField: "Content-Type")
            request.httpBody = try? JSONEncoder().encode(["refresh_token": refreshToken])

            // Best-effort server logout.
            _ = try? await config.urlSession.data(for: request)
        }

        setTokens(nil)
    }

    // MARK: - Private Helpers

    private func setTokens(_ newTokens: RampartTokens?) {
        tokens = newTokens
        if let newTokens = newTokens, let data = try? JSONEncoder().encode(newTokens) {
            Keychain.write(data, service: Self.keychainService, account: Self.keychainTokensKey)
        } else {
            Keychain.delete(service: Self.keychainService, account: Self.keychainTokensKey)
        }
    }

    private func cleanupPkceStorage() {
        Keychain.delete(service: Self.keychainService, account: Self.keychainVerifierKey)
        Keychain.delete(service: Self.keychainService, account: Self.keychainStateKey)
    }

    private func isTokenExpired(_ token: String) -> Bool {
        let parts = token.split(separator: ".")
        guard parts.count == 3 else { return true }

        var base64 = String(parts[1])
            .replacingOccurrences(of: "-", with: "+")
            .replacingOccurrences(of: "_", with: "/")
        // Pad to multiple of 4.
        while base64.count % 4 != 0 { base64.append("=") }

        guard let payloadData = Data(base64Encoded: base64),
              let json = try? JSONSerialization.jsonObject(with: payloadData) as? [String: Any],
              let exp = json["exp"] as? TimeInterval
        else { return true }

        return Date(timeIntervalSince1970: exp) <= Date()
    }

    /// Generate a cryptographically random 64-character code verifier.
    private func generateCodeVerifier() -> String {
        var bytes = [UInt8](repeating: 0, count: 32)
        _ = SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes)
        return Data(bytes).base64URLEncodedString()
    }

    /// Compute S256 code challenge: BASE64URL(SHA256(verifier)).
    private func generateCodeChallenge(_ verifier: String) -> String {
        let hash = SHA256.hash(data: Data(verifier.utf8))
        return Data(hash).base64URLEncodedString()
    }

    /// Generate a random state parameter for CSRF protection.
    private func generateState() -> String {
        var bytes = [UInt8](repeating: 0, count: 32)
        _ = SecRandomCopyBytes(kSecRandomDefault, bytes.count, &bytes)
        return Data(bytes).base64URLEncodedString()
    }

    /// Percent-encode a string for use in form-urlencoded bodies.
    private func urlEncode(_ string: String) -> String {
        string.addingPercentEncoding(withAllowedCharacters: .urlQueryAllowed) ?? string
    }
}

// MARK: - Base64URL Extension

extension Data {
    /// Base64url encoding without padding, per RFC 7636.
    func base64URLEncodedString() -> String {
        base64EncodedString()
            .replacingOccurrences(of: "+", with: "-")
            .replacingOccurrences(of: "/", with: "_")
            .replacingOccurrences(of: "=", with: "")
    }
}

// MARK: - Keychain Helpers

enum Keychain {

    @discardableResult
    static func write(_ data: Data, service: String, account: String) -> Bool {
        // Delete any existing item first.
        delete(service: service, account: account)

        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecValueData as String: data,
            kSecAttrAccessible as String: kSecAttrAccessibleAfterFirstUnlockThisDeviceOnly,
        ]
        return SecItemAdd(query as CFDictionary, nil) == errSecSuccess
    }

    static func read(service: String, account: String) -> Data? {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
            kSecReturnData as String: true,
            kSecMatchLimit as String: kSecMatchLimitOne,
        ]
        var result: AnyObject?
        let status = SecItemCopyMatching(query as CFDictionary, &result)
        return status == errSecSuccess ? result as? Data : nil
    }

    @discardableResult
    static func delete(service: String, account: String) -> Bool {
        let query: [String: Any] = [
            kSecClass as String: kSecClassGenericPassword,
            kSecAttrService as String: service,
            kSecAttrAccount as String: account,
        ]
        return SecItemDelete(query as CFDictionary) == errSecSuccess
    }
}
