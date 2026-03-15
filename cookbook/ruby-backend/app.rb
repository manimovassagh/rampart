# frozen_string_literal: true

require "sinatra/base"
require "json"
require "net/http"
require "jwt"

# Sinatra app that mirrors the Express cookbook backend.
# Verifies Rampart-issued RS256 JWTs using the JWKS endpoint.
class RampartSampleApp < Sinatra::Base
  set :port, Integer(ENV.fetch("PORT", 3001))
  set :bind, "0.0.0.0"

  RAMPART_ISSUER = ENV.fetch("RAMPART_ISSUER", "http://localhost:8080")
  JWKS_URL       = "#{RAMPART_ISSUER}/.well-known/jwks.json"

  # ── CORS ──────────────────────────────────────────────────────────────
  before do
    content_type :json
    headers "Access-Control-Allow-Origin"  => "*",
            "Access-Control-Allow-Methods" => "GET, POST, PUT, DELETE, OPTIONS",
            "Access-Control-Allow-Headers" => "Authorization, Content-Type"
  end

  options "*" do
    200
  end

  # ── JWKS helpers ──────────────────────────────────────────────────────
  @@jwks_cache      = nil
  @@jwks_fetched_at = nil
  JWKS_TTL          = 300 # seconds

  def self.fetch_jwks
    if @@jwks_cache && @@jwks_fetched_at && (Time.now - @@jwks_fetched_at < JWKS_TTL)
      return @@jwks_cache
    end

    uri  = URI(JWKS_URL)
    resp = Net::HTTP.get(uri)
    data = JSON.parse(resp)
    @@jwks_cache      = data["keys"]
    @@jwks_fetched_at = Time.now
    @@jwks_cache
  end

  def self.find_rsa_key(kid)
    keys = fetch_jwks
    jwk  = keys.find { |k| k["kid"] == kid } || keys.first
    return nil unless jwk

    # Use the jwt gem's JWK import — works with OpenSSL 3 immutable pkeys
    jwk_obj = JWT::JWK.new(jwk)
    jwk_obj.public_key
  end

  # ── Auth helper ───────────────────────────────────────────────────────
  def verify_token!
    header = request.env["HTTP_AUTHORIZATION"]
    unless header
      halt 401, { error: "unauthorized", error_description: "Missing authorization header.", status: 401 }.to_json
    end

    parts = header.split(" ")
    unless parts.length == 2 && parts[0].downcase == "bearer"
      halt 401, { error: "unauthorized", error_description: "Invalid authorization header format.", status: 401 }.to_json
    end

    token = parts[1]

    begin
      # Decode header to get kid
      header_segment = Base64.urlsafe_decode64(token.split(".")[0])
      jwt_header     = JSON.parse(header_segment)

      rsa_key = self.class.find_rsa_key(jwt_header["kid"])
      unless rsa_key
        halt 401, { error: "unauthorized", error_description: "Unable to find signing key.", status: 401 }.to_json
      end

      decoded = JWT.decode(token, rsa_key, true, {
        algorithms: ["RS256"],
        iss:        RAMPART_ISSUER,
        verify_iss: true,
      })

      decoded[0] # payload hash
    rescue JWT::DecodeError, JWT::ExpiredSignature, JWT::InvalidIssuerError => e
      halt 401, { error: "unauthorized", error_description: "Invalid or expired access token.", status: 401 }.to_json
    end
  end

  def require_roles!(claims, *roles)
    user_roles = claims["roles"] || []
    missing    = roles.select { |r| !user_roles.include?(r) }
    return if missing.empty?

    halt 403, {
      error:             "forbidden",
      error_description: "Missing required role(s): #{missing.join(', ')}",
      status:            403,
    }.to_json
  end

  # ── Routes ────────────────────────────────────────────────────────────

  # Public
  get "/api/health" do
    { status: "ok", issuer: RAMPART_ISSUER }.to_json
  end

  # Protected — any authenticated user
  get "/api/profile" do
    claims = verify_token!
    {
      message: "Authenticated!",
      user: {
        id:             claims["sub"],
        email:          claims["email"],
        username:       claims["preferred_username"],
        org_id:         claims["org_id"],
        email_verified: claims["email_verified"],
        given_name:     claims["given_name"],
        family_name:    claims["family_name"],
        roles:          claims["roles"] || [],
      },
    }.to_json
  end

  # Protected — raw claims
  get "/api/claims" do
    claims = verify_token!
    claims.to_json
  end

  # RBAC — requires "editor" role
  get "/api/editor/dashboard" do
    claims = verify_token!
    require_roles!(claims, "editor")
    {
      message: "Welcome, Editor!",
      user:    claims["preferred_username"],
      roles:   claims["roles"],
      data:    { drafts: 3, published: 12, pending_review: 2 },
    }.to_json
  end

  # RBAC — requires "manager" role
  get "/api/manager/reports" do
    claims = verify_token!
    require_roles!(claims, "manager")
    {
      message: "Manager Reports",
      user:    claims["preferred_username"],
      roles:   claims["roles"],
      reports: [
        { name: "Q1 Revenue", status: "complete" },
        { name: "User Growth", status: "in_progress" },
      ],
    }.to_json
  end

  # ── Boot ──────────────────────────────────────────────────────────────
  if __FILE__ == $PROGRAM_NAME
    puts "Ruby sample backend running on http://localhost:#{settings.port}"
    puts "Rampart issuer: #{RAMPART_ISSUER}"
    puts ""
    puts "Routes:"
    puts "  GET /api/health            - public"
    puts "  GET /api/profile           - protected (any authenticated user)"
    puts "  GET /api/claims            - protected (any authenticated user)"
    puts "  GET /api/editor/dashboard  - protected (requires \"editor\" role)"
    puts "  GET /api/manager/reports   - protected (requires \"manager\" role)"
    run!
  end
end
