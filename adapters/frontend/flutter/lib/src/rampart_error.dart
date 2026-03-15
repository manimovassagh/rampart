/// Error returned by the Rampart server or raised by the client.
class RampartError implements Exception {
  /// Machine-readable error code (e.g. "invalid_callback", "state_mismatch").
  final String error;

  /// Human-readable error description.
  final String errorDescription;

  /// HTTP status code, or 0 for client-side errors.
  final int status;

  const RampartError({
    required this.error,
    required this.errorDescription,
    this.status = 0,
  });

  factory RampartError.fromJson(Map<String, dynamic> json, {int status = 0}) {
    return RampartError(
      error: json['error'] as String? ?? 'unknown_error',
      errorDescription:
          json['error_description'] as String? ?? 'An unknown error occurred.',
      status: status,
    );
  }

  @override
  String toString() => 'RampartError($error): $errorDescription [HTTP $status]';
}
