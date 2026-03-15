import 'dart:convert';
import 'dart:math';

import 'package:crypto/crypto.dart';
import 'package:flutter_secure_storage/flutter_secure_storage.dart';
import 'package:http/http.dart' as http;
import 'package:url_launcher/url_launcher.dart';

import 'rampart_claims.dart';
import 'rampart_error.dart';

/// Configuration for [RampartClient].
class RampartClientConfig {
  /// Rampart server URL (e.g. "http://localhost:8080").
  final String issuer;

  /// OAuth 2.0 client ID registered with the Rampart server.
  final String clientId;

  /// OAuth 2.0 redirect URI -- must exactly match a registered redirect URI.
  /// For mobile, use a custom scheme (e.g. "com.example.app://callback").
  /// For web, use a standard HTTPS URL.
  final String redirectUri;

  /// OAuth 2.0 scopes (default: "openid").
  final String scope;

  /// Optional HTTP client for testing or custom configuration.
  final http.Client? httpClient;

  const RampartClientConfig({
    required this.issuer,
    required this.clientId,
    required this.redirectUri,
    this.scope = 'openid',
    this.httpClient,
  });
}

/// Token set returned by login and refresh operations.
class RampartTokens {
  final String accessToken;
  final String refreshToken;
  final String tokenType;
  final int expiresIn;

  const RampartTokens({
    required this.accessToken,
    required this.refreshToken,
    required this.tokenType,
    required this.expiresIn,
  });

  factory RampartTokens.fromJson(Map<String, dynamic> json) {
    return RampartTokens(
      accessToken: json['access_token'] as String,
      refreshToken: json['refresh_token'] as String,
      tokenType: json['token_type'] as String? ?? 'Bearer',
      expiresIn: json['expires_in'] as int? ?? 3600,
    );
  }

  Map<String, dynamic> toJson() => {
        'access_token': accessToken,
        'refresh_token': refreshToken,
        'token_type': tokenType,
        'expires_in': expiresIn,
      };
}

/// Flutter authentication client for Rampart IAM.
///
/// Implements OAuth 2.0 Authorization Code flow with PKCE.
/// Supports mobile (iOS/Android) via custom URI schemes and web via standard
/// redirect URLs. Tokens are persisted using flutter_secure_storage.
class RampartClient {
  final String _issuer;
  final String _clientId;
  final String _redirectUri;
  final String _scope;
  final http.Client _httpClient;
  final FlutterSecureStorage _storage;

  RampartTokens? _tokens;

  static const _storageKeyTokens = 'rampart_tokens';
  static const _storageKeyCodeVerifier = 'rampart_pkce_code_verifier';
  static const _storageKeyState = 'rampart_pkce_state';

  /// Creates a new [RampartClient].
  ///
  /// Call [loadStoredTokens] after construction to restore any previously
  /// persisted session.
  RampartClient(RampartClientConfig config)
      : _issuer = config.issuer.replaceAll(RegExp(r'/+$'), ''),
        _clientId = config.clientId,
        _redirectUri = config.redirectUri,
        _scope = config.scope,
        _httpClient = config.httpClient ?? http.Client(),
        _storage = const FlutterSecureStorage();

  /// Whether the client holds a non-expired access token.
  bool get isAuthenticated {
    final token = _tokens?.accessToken;
    if (token == null) return false;
    try {
      final parts = token.split('.');
      if (parts.length != 3) return false;
      // Normalize base64url to base64.
      final payload = _decodeBase64Url(parts[1]);
      final claims = jsonDecode(payload) as Map<String, dynamic>;
      final exp = claims['exp'] as int?;
      if (exp == null) return false;
      return DateTime.fromMillisecondsSinceEpoch(exp * 1000)
          .isAfter(DateTime.now());
    } catch (_) {
      return false;
    }
  }

  /// Returns the current access token, or null.
  String? get accessToken => _tokens?.accessToken;

  /// Returns a copy of the current tokens, or null.
  RampartTokens? get tokens => _tokens;

  /// Load tokens from secure storage. Call this once at app startup to
  /// restore a previous session.
  Future<void> loadStoredTokens() async {
    final stored = await _storage.read(key: _storageKeyTokens);
    if (stored != null) {
      try {
        _tokens = RampartTokens.fromJson(
          jsonDecode(stored) as Map<String, dynamic>,
        );
      } catch (_) {
        await _storage.delete(key: _storageKeyTokens);
      }
    }
  }

  /// Start the OAuth login flow by opening the authorization URL in the
  /// system browser.
  ///
  /// On mobile, the browser will redirect back to your app via the custom
  /// URI scheme configured in [RampartClientConfig.redirectUri].
  /// On web, the page will navigate to the redirect URL.
  ///
  /// After the redirect, call [handleCallback] with the full callback URI.
  Future<void> loginWithRedirect() async {
    final codeVerifier = _generateCodeVerifier();
    final codeChallenge = _generateCodeChallenge(codeVerifier);
    final state = _generateState();

    // Persist PKCE values so they survive the browser redirect.
    await _storage.write(key: _storageKeyCodeVerifier, value: codeVerifier);
    await _storage.write(key: _storageKeyState, value: state);

    final params = {
      'response_type': 'code',
      'client_id': _clientId,
      'redirect_uri': _redirectUri,
      'scope': _scope,
      'state': state,
      'code_challenge': codeChallenge,
      'code_challenge_method': 'S256',
    };

    final authorizeUrl =
        Uri.parse('$_issuer/oauth/authorize').replace(queryParameters: params);

    if (!await launchUrl(authorizeUrl, mode: LaunchMode.externalApplication)) {
      throw const RampartError(
        error: 'launch_failed',
        errorDescription: 'Could not open the authorization URL in a browser.',
      );
    }
  }

  /// Handle the OAuth callback after the browser redirects back to the app.
  ///
  /// Pass the full callback URI (including query parameters). This method
  /// validates the state parameter, exchanges the authorization code for
  /// tokens, and persists them in secure storage.
  Future<RampartTokens> handleCallback(Uri uri) async {
    final code = uri.queryParameters['code'];
    final state = uri.queryParameters['state'];
    final error = uri.queryParameters['error'];

    if (error != null) {
      final description =
          uri.queryParameters['error_description'] ?? error;
      throw RampartError(error: error, errorDescription: description);
    }

    if (code == null || state == null) {
      throw const RampartError(
        error: 'invalid_callback',
        errorDescription: 'Missing code or state parameter in callback URL.',
      );
    }

    // Validate state.
    final storedState = await _storage.read(key: _storageKeyState);
    if (storedState == null || storedState != state) {
      await _cleanupPkceStorage();
      throw const RampartError(
        error: 'state_mismatch',
        errorDescription:
            'State parameter does not match. Possible CSRF attack.',
      );
    }

    final codeVerifier = await _storage.read(key: _storageKeyCodeVerifier);
    if (codeVerifier == null) {
      await _cleanupPkceStorage();
      throw const RampartError(
        error: 'missing_verifier',
        errorDescription: 'Code verifier not found in secure storage.',
      );
    }

    // Exchange code for tokens.
    final body = {
      'grant_type': 'authorization_code',
      'code': code,
      'client_id': _clientId,
      'redirect_uri': _redirectUri,
      'code_verifier': codeVerifier,
    };

    final response = await _httpClient.post(
      Uri.parse('$_issuer/oauth/token'),
      headers: {'Content-Type': 'application/x-www-form-urlencoded'},
      body: body,
    );

    await _cleanupPkceStorage();

    if (response.statusCode != 200) {
      throw _parseError(response);
    }

    final tokens = RampartTokens.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
    await _setTokens(tokens);
    return tokens;
  }

  /// Fetch the authenticated user profile from the /me endpoint.
  Future<RampartClaims> getUser() async {
    final response = await authFetch(Uri.parse('$_issuer/me'));

    if (response.statusCode != 200) {
      throw _parseError(response);
    }

    return RampartClaims.fromJson(
      jsonDecode(response.body) as Map<String, dynamic>,
    );
  }

  /// Make an authenticated HTTP GET request with automatic Bearer token.
  ///
  /// On a 401 response, attempts one silent token refresh and retries.
  /// Pass [method], [headers], and [body] for non-GET requests.
  Future<http.Response> authFetch(
    Uri url, {
    String method = 'GET',
    Map<String, String>? headers,
    Object? body,
  }) async {
    Future<http.Response> doFetch() {
      final mergedHeaders = {
        if (headers != null) ...headers,
        'Authorization': 'Bearer ${_tokens?.accessToken}',
      };

      final request = http.Request(method, url)..headers.addAll(mergedHeaders);
      if (body is String) {
        request.body = body;
      } else if (body is Map) {
        request.bodyFields = body.cast<String, String>();
      }

      return _httpClient.send(request).then(http.Response.fromStream);
    }

    var response = await doFetch();

    if (response.statusCode == 401 && _tokens?.refreshToken != null) {
      try {
        await refresh();
        response = await doFetch();
      } catch (_) {
        // Refresh failed -- return the original 401.
      }
    }

    return response;
  }

  /// Refresh the access token using the stored refresh token.
  Future<RampartTokens> refresh() async {
    final refreshToken = _tokens?.refreshToken;
    if (refreshToken == null) {
      throw const RampartError(
        error: 'no_refresh_token',
        errorDescription: 'No refresh token available.',
      );
    }

    final response = await _httpClient.post(
      Uri.parse('$_issuer/token/refresh'),
      headers: {'Content-Type': 'application/json'},
      body: jsonEncode({'refresh_token': refreshToken}),
    );

    if (response.statusCode != 200) {
      await _setTokens(null);
      throw _parseError(response);
    }

    final data = jsonDecode(response.body) as Map<String, dynamic>;
    final updated = RampartTokens(
      accessToken: data['access_token'] as String,
      refreshToken: refreshToken,
      tokenType: data['token_type'] as String? ?? 'Bearer',
      expiresIn: data['expires_in'] as int? ?? 3600,
    );
    await _setTokens(updated);
    return updated;
  }

  /// Logout -- invalidates the refresh token on the server and clears local
  /// tokens.
  Future<void> logout() async {
    final refreshToken = _tokens?.refreshToken;
    if (refreshToken != null) {
      try {
        await _httpClient.post(
          Uri.parse('$_issuer/logout'),
          headers: {'Content-Type': 'application/json'},
          body: jsonEncode({'refresh_token': refreshToken}),
        );
      } catch (_) {
        // Best-effort server logout.
      }
    }

    await _setTokens(null);
  }

  // ---------------------------------------------------------------------------
  // Private helpers
  // ---------------------------------------------------------------------------

  Future<void> _setTokens(RampartTokens? tokens) async {
    _tokens = tokens;
    if (tokens != null) {
      await _storage.write(
        key: _storageKeyTokens,
        value: jsonEncode(tokens.toJson()),
      );
    } else {
      await _storage.delete(key: _storageKeyTokens);
    }
  }

  Future<void> _cleanupPkceStorage() async {
    await _storage.delete(key: _storageKeyCodeVerifier);
    await _storage.delete(key: _storageKeyState);
  }

  RampartError _parseError(http.Response response) {
    try {
      final json = jsonDecode(response.body) as Map<String, dynamic>;
      return RampartError.fromJson(json, status: response.statusCode);
    } catch (_) {
      return RampartError(
        error: 'unknown_error',
        errorDescription:
            'HTTP ${response.statusCode}: ${response.reasonPhrase}',
        status: response.statusCode,
      );
    }
  }

  /// Generate a random 64-character code verifier (base64url-encoded).
  String _generateCodeVerifier() {
    final random = Random.secure();
    final bytes = List<int>.generate(48, (_) => random.nextInt(256));
    return _base64UrlEncode(bytes);
  }

  /// Compute S256 code challenge: BASE64URL(SHA256(verifier)).
  String _generateCodeChallenge(String verifier) {
    final bytes = utf8.encode(verifier);
    final digest = sha256.convert(bytes);
    return _base64UrlEncode(digest.bytes);
  }

  /// Generate a random state parameter for CSRF protection.
  String _generateState() {
    final random = Random.secure();
    final bytes = List<int>.generate(32, (_) => random.nextInt(256));
    return _base64UrlEncode(bytes);
  }

  /// URL-safe base64 encoding without padding.
  String _base64UrlEncode(List<int> bytes) {
    return base64Url.encode(bytes).replaceAll('=', '');
  }

  /// Decode a base64url-encoded string to a UTF-8 string.
  String _decodeBase64Url(String input) {
    // Add padding if necessary.
    var padded = input;
    switch (padded.length % 4) {
      case 2:
        padded += '==';
        break;
      case 3:
        padded += '=';
        break;
    }
    final normalized = padded.replaceAll('-', '+').replaceAll('_', '/');
    return utf8.decode(base64.decode(normalized));
  }
}
