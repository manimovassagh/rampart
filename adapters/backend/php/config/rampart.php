<?php

return [

    /*
    |--------------------------------------------------------------------------
    | Rampart Issuer URL
    |--------------------------------------------------------------------------
    |
    | The base URL of your Rampart IAM server. Tokens are verified against
    | the JWKS endpoint at {issuer}/.well-known/jwks.json.
    |
    */
    'issuer' => env('RAMPART_ISSUER', ''),

    /*
    |--------------------------------------------------------------------------
    | Expected Audience
    |--------------------------------------------------------------------------
    |
    | If set, the middleware will reject tokens whose "aud" claim does not
    | match this value. Leave null to skip audience validation.
    |
    */
    'audience' => env('RAMPART_AUDIENCE'),

    /*
    |--------------------------------------------------------------------------
    | JWKS Cache TTL
    |--------------------------------------------------------------------------
    |
    | How long (in seconds) to cache the JWKS keys fetched from the Rampart
    | server. Set to 0 to disable caching (not recommended in production).
    |
    */
    'jwks_cache_ttl' => (int) env('RAMPART_JWKS_CACHE_TTL', 300),

    /*
    |--------------------------------------------------------------------------
    | JWKS Cache Path
    |--------------------------------------------------------------------------
    |
    | Directory for file-based JWKS caching. Defaults to the system temp
    | directory if left null.
    |
    */
    'jwks_cache_path' => env('RAMPART_JWKS_CACHE_PATH'),

];
