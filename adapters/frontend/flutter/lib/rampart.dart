/// Flutter authentication SDK for Rampart IAM.
///
/// Implements OAuth 2.0 Authorization Code flow with PKCE for mobile
/// (iOS/Android) and web platforms.
///
/// ```dart
/// import 'package:rampart_flutter/rampart.dart';
///
/// final client = RampartClient(RampartClientConfig(
///   issuer: 'http://localhost:8080',
///   clientId: 'my-app',
///   redirectUri: 'com.example.app://callback',
/// ));
/// ```
library rampart;

export 'src/rampart_claims.dart' show RampartClaims;
export 'src/rampart_client.dart'
    show RampartClient, RampartClientConfig, RampartTokens;
export 'src/rampart_error.dart' show RampartError;
