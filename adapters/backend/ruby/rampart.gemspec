# frozen_string_literal: true

Gem::Specification.new do |spec|
  spec.name          = "rampart-ruby"
  spec.version       = "0.1.0"
  spec.authors       = ["Rampart Contributors"]
  spec.email         = ["info@rampart-auth.io"]

  spec.summary       = "Rack middleware for verifying Rampart JWTs"
  spec.description   = "RS256 JWT verification middleware for Ruby web frameworks. " \
                        "Works with Rails, Sinatra, and any Rack-compatible application. " \
                        "Fetches JWKS automatically, validates tokens, and provides " \
                        "role-based access control."
  spec.homepage      = "https://github.com/manimovassagh/rampart"
  spec.license       = "MIT"

  spec.required_ruby_version = ">= 3.0"

  spec.metadata = {
    "homepage_uri"    => spec.homepage,
    "source_code_uri" => "https://github.com/manimovassagh/rampart/tree/main/adapters/backend/ruby",
    "bug_tracker_uri" => "https://github.com/manimovassagh/rampart/issues"
  }

  spec.files = Dir["lib/**/*.rb", "README.md", "LICENSE"]
  spec.require_paths = ["lib"]

  spec.add_dependency "jwt", "~> 2.7"

  spec.add_development_dependency "rake", "~> 13.0"
  spec.add_development_dependency "rspec", "~> 3.12"
  spec.add_development_dependency "rack-test", "~> 2.1"
end
