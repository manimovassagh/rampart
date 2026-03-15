# frozen_string_literal: true

module Rampart
  # Verified JWT claims extracted from a Rampart access token.
  #
  # After successful authentication by {Rampart::Middleware}, an instance of
  # this class is stored in +env['rampart.claims']+.
  class Claims
    ATTRIBUTES = %i[
      iss sub iat exp org_id preferred_username
      email email_verified given_name family_name roles
    ].freeze

    attr_reader(*ATTRIBUTES)

    # @param payload [Hash] Decoded JWT payload (string keys).
    def initialize(payload)
      @iss                = payload["iss"]
      @sub                = payload["sub"]
      @iat                = payload["iat"]
      @exp                = payload["exp"]
      @org_id             = payload["org_id"]
      @preferred_username = payload["preferred_username"]
      @email              = payload["email"]
      @email_verified     = !!payload["email_verified"]
      @given_name         = payload["given_name"]
      @family_name        = payload["family_name"]
      @roles              = Array(payload["roles"]).map(&:to_s)
    end

    # Returns the claims as a plain Hash with string keys.
    def to_h
      ATTRIBUTES.each_with_object({}) do |attr, hash|
        hash[attr.to_s] = public_send(attr)
      end
    end
  end
end
