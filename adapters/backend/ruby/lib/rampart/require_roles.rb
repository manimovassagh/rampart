# frozen_string_literal: true

module Rampart
  # Rack middleware that enforces role-based access control.
  #
  # Must be used after {Rampart::Middleware} -- requires +env['rampart.claims']+
  # to be set. Returns 403 Forbidden if the user lacks any of the required roles.
  #
  # @example Rails route constraint
  #   # config/application.rb
  #   config.middleware.use Rampart::Middleware, issuer: "http://localhost:8080"
  #
  #   # In a specific controller or mounted Rack app:
  #   admin_app = Rampart::RequireRoles.new(app, "admin")
  class RequireRoles
    # @param app  [#call] The next Rack application in the stack.
    # @param roles [Array<String>] One or more roles the user must possess.
    def initialize(app, *roles)
      @app   = app
      @roles = roles.flatten.map(&:to_s)
    end

    def call(env)
      claims = env["rampart.claims"]

      unless claims
        return json_response(401, "unauthorized", "Authentication required.")
      end

      user_roles = Array(claims.roles)
      missing    = @roles.reject { |r| user_roles.include?(r) }

      unless missing.empty?
        return json_response(
          403, "forbidden",
          "Missing required role(s): #{missing.join(', ')}"
        )
      end

      @app.call(env)
    end

    private

    def json_response(status, error, description)
      body = JSON.generate(
        error: error,
        error_description: description,
        status: status
      )
      [
        status,
        { "content-type" => "application/json" },
        [body]
      ]
    end
  end
end
