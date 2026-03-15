# frozen_string_literal: true

require "json"
require "net/http"
require "uri"
require "jwt"

require_relative "rampart/claims"
require_relative "rampart/require_roles"

module Rampart
  # Rack middleware that validates RS256 JWTs issued by a Rampart IAM server.
  #
  # It extracts the Bearer token from the Authorization header, fetches the
  # JWKS from +{issuer}/.well-known/jwks.json+ (with in-memory caching), and
  # verifies the token signature, issuer, and expiration.
  #
  # On success it stores a {Rampart::Claims} instance in +env['rampart.claims']+.
  # On failure it returns a 401 JSON response and does not call the next app.
  #
  # Works with Rails, Sinatra, and any Rack-compatible framework.
  #
  # @example Rails
  #   # config/application.rb
  #   config.middleware.use Rampart::Middleware, issuer: "http://localhost:8080"
  #
  # @example Sinatra
  #   use Rampart::Middleware, issuer: "http://localhost:8080"
  class Middleware
    JWKS_CACHE_TTL = 600 # seconds (10 minutes)

    # @param app    [#call]  The next Rack application.
    # @param issuer [String] Rampart server URL (e.g. "http://localhost:8080").
    def initialize(app, issuer:)
      @app    = app
      @issuer = issuer.chomp("/")
      @jwks_uri = URI("#{@issuer}/.well-known/jwks.json")
      @jwks       = nil
      @jwks_fetched_at = 0
      @mutex = Mutex.new
    end

    def call(env)
      header = env["HTTP_AUTHORIZATION"]

      unless header
        return unauthorized("Missing authorization header.")
      end

      parts = header.split(" ", 2)
      unless parts.length == 2 && parts[0].downcase == "bearer"
        return unauthorized("Invalid authorization header format.")
      end

      token = parts[1]

      begin
        jwks = fetch_jwks
        decoded = JWT.decode(
          token, nil, true,
          algorithms: ["RS256"],
          iss: @issuer,
          verify_iss: true,
          jwks: jwks
        )
        payload = decoded.first
      rescue JWT::DecodeError, JWT::ExpiredSignature, JWT::InvalidIssuerError => _e
        return unauthorized("Invalid or expired access token.")
      end

      env["rampart.claims"] = Claims.new(payload)
      @app.call(env)
    end

    private

    # Fetches the JWKS from the issuer, caching for JWKS_CACHE_TTL seconds.
    # Thread-safe via Mutex.
    def fetch_jwks
      @mutex.synchronize do
        now = Process.clock_gettime(Process::CLOCK_MONOTONIC).to_i
        if @jwks.nil? || (now - @jwks_fetched_at) > JWKS_CACHE_TTL
          response = Net::HTTP.get(@jwks_uri)
          @jwks = JWT::JWK::Set.new(JSON.parse(response))
          @jwks_fetched_at = now
        end
        @jwks
      end
    end

    def unauthorized(description)
      body = JSON.generate(
        error: "unauthorized",
        error_description: description,
        status: 401
      )
      [
        401,
        { "content-type" => "application/json" },
        [body]
      ]
    end
  end
end
