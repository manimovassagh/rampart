<?php

declare(strict_types=1);

require __DIR__ . '/../vendor/autoload.php';

use Firebase\JWT\JWT;
use Firebase\JWT\JWK;
use GuzzleHttp\Client;
use Psr\Http\Message\ResponseInterface as Response;
use Psr\Http\Message\ServerRequestInterface as Request;
use Psr\Http\Server\RequestHandlerInterface as RequestHandler;
use Slim\Factory\AppFactory;

// ── Configuration ────────────────────────────────────────────────────────────

$RAMPART_ISSUER = getenv('RAMPART_ISSUER') ?: 'http://localhost:8080';
$JWKS_URL       = $RAMPART_ISSUER . '/.well-known/jwks.json';

// ── JWKS Cache ───────────────────────────────────────────────────────────────

$jwksCache     = null;
$jwksFetchedAt = 0;
$JWKS_TTL      = 300; // seconds

function fetchJwks(string $jwksUrl, ?array &$jwksCache, int &$jwksFetchedAt, int $ttl): array
{
    if ($jwksCache !== null && (time() - $jwksFetchedAt) < $ttl) {
        return $jwksCache;
    }

    $client   = new Client(['timeout' => 5]);
    $response = $client->get($jwksUrl);
    $data     = json_decode((string) $response->getBody(), true);

    $jwksCache     = $data['keys'] ?? [];
    $jwksFetchedAt = time();

    return $jwksCache;
}

function parseJwks(array $keys): array
{
    // Build the key set structure that firebase/php-jwt expects
    return JWK::parseKeySet(['keys' => $keys]);
}

// ── Helper: JSON response ────────────────────────────────────────────────────

function jsonResponse(Response $response, array $data, int $status = 200): Response
{
    $response->getBody()->write(json_encode($data, JSON_UNESCAPED_SLASHES));
    return $response
        ->withHeader('Content-Type', 'application/json')
        ->withStatus($status);
}

// ── Helper: Verify JWT ───────────────────────────────────────────────────────

function verifyToken(Request $request, string $jwksUrl, string $issuer, ?array &$jwksCache, int &$jwksFetchedAt, int $ttl): object|array
{
    $authHeader = $request->getHeaderLine('Authorization');
    if (empty($authHeader)) {
        throw new \RuntimeException('Missing authorization header.', 401);
    }

    $parts = explode(' ', $authHeader);
    if (count($parts) !== 2 || strtolower($parts[0]) !== 'bearer') {
        throw new \RuntimeException('Invalid authorization header format.', 401);
    }

    $token = $parts[1];

    try {
        $keys     = fetchJwks($jwksUrl, $jwksCache, $jwksFetchedAt, $ttl);
        $keySet   = parseJwks($keys);
        $decoded  = JWT::decode($token, $keySet);

        // Verify issuer
        if (!isset($decoded->iss) || $decoded->iss !== $issuer) {
            throw new \RuntimeException('Invalid issuer.', 401);
        }

        return $decoded;
    } catch (\RuntimeException $e) {
        throw $e;
    } catch (\Exception $e) {
        throw new \RuntimeException('Invalid or expired access token.', 401);
    }
}

// ── Create Slim App ──────────────────────────────────────────────────────────

$app = AppFactory::create();

// ── CORS Middleware ──────────────────────────────────────────────────────────

$app->add(function (Request $request, RequestHandler $handler): Response {
    if ($request->getMethod() === 'OPTIONS') {
        $response = new \Slim\Psr7\Response();
    } else {
        $response = $handler->handle($request);
    }

    return $response
        ->withHeader('Access-Control-Allow-Origin', '*')
        ->withHeader('Access-Control-Allow-Methods', 'GET, POST, PUT, DELETE, OPTIONS')
        ->withHeader('Access-Control-Allow-Headers', 'Authorization, Content-Type');
});

$app->addRoutingMiddleware();

// ── Error Middleware ─────────────────────────────────────────────────────────

$errorMiddleware = $app->addErrorMiddleware(false, true, true);

// ── Routes ───────────────────────────────────────────────────────────────────

// Public — health check
$app->get('/api/health', function (Request $request, Response $response) use ($RAMPART_ISSUER) {
    return jsonResponse($response, [
        'status' => 'ok',
        'issuer' => $RAMPART_ISSUER,
    ]);
});

// Protected — authenticated user profile
$app->get('/api/profile', function (Request $request, Response $response) use ($JWKS_URL, $RAMPART_ISSUER, &$jwksCache, &$jwksFetchedAt, $JWKS_TTL) {
    try {
        $claims = verifyToken($request, $JWKS_URL, $RAMPART_ISSUER, $jwksCache, $jwksFetchedAt, $JWKS_TTL);
    } catch (\RuntimeException $e) {
        $status = $e->getCode() ?: 401;
        return jsonResponse($response, [
            'error'             => 'unauthorized',
            'error_description' => $e->getMessage(),
            'status'            => $status,
        ], $status);
    }

    return jsonResponse($response, [
        'message' => 'Authenticated!',
        'user'    => [
            'id'             => $claims->sub ?? null,
            'email'          => $claims->email ?? null,
            'username'       => $claims->preferred_username ?? null,
            'org_id'         => $claims->org_id ?? null,
            'email_verified' => $claims->email_verified ?? null,
            'given_name'     => $claims->given_name ?? null,
            'family_name'    => $claims->family_name ?? null,
            'roles'          => $claims->roles ?? [],
        ],
    ]);
});

// Protected — raw claims
$app->get('/api/claims', function (Request $request, Response $response) use ($JWKS_URL, $RAMPART_ISSUER, &$jwksCache, &$jwksFetchedAt, $JWKS_TTL) {
    try {
        $claims = verifyToken($request, $JWKS_URL, $RAMPART_ISSUER, $jwksCache, $jwksFetchedAt, $JWKS_TTL);
    } catch (\RuntimeException $e) {
        $status = $e->getCode() ?: 401;
        return jsonResponse($response, [
            'error'             => 'unauthorized',
            'error_description' => $e->getMessage(),
            'status'            => $status,
        ], $status);
    }

    $body = json_encode($claims, JSON_UNESCAPED_SLASHES);
    $response->getBody()->write($body);
    return $response->withHeader('Content-Type', 'application/json');
});

// RBAC — requires "editor" role
$app->get('/api/editor/dashboard', function (Request $request, Response $response) use ($JWKS_URL, $RAMPART_ISSUER, &$jwksCache, &$jwksFetchedAt, $JWKS_TTL) {
    try {
        $claims = verifyToken($request, $JWKS_URL, $RAMPART_ISSUER, $jwksCache, $jwksFetchedAt, $JWKS_TTL);
    } catch (\RuntimeException $e) {
        $status = $e->getCode() ?: 401;
        return jsonResponse($response, [
            'error'             => 'unauthorized',
            'error_description' => $e->getMessage(),
            'status'            => $status,
        ], $status);
    }

    $roles = $claims->roles ?? [];
    if (!in_array('editor', (array) $roles, true)) {
        return jsonResponse($response, [
            'error'             => 'forbidden',
            'error_description' => 'Missing required role(s): editor',
            'status'            => 403,
        ], 403);
    }

    return jsonResponse($response, [
        'message' => 'Welcome, Editor!',
        'user'    => $claims->preferred_username ?? null,
        'roles'   => $roles,
        'data'    => [
            'drafts'         => 3,
            'published'      => 12,
            'pending_review' => 2,
        ],
    ]);
});

// RBAC — requires "manager" role
$app->get('/api/manager/reports', function (Request $request, Response $response) use ($JWKS_URL, $RAMPART_ISSUER, &$jwksCache, &$jwksFetchedAt, $JWKS_TTL) {
    try {
        $claims = verifyToken($request, $JWKS_URL, $RAMPART_ISSUER, $jwksCache, $jwksFetchedAt, $JWKS_TTL);
    } catch (\RuntimeException $e) {
        $status = $e->getCode() ?: 401;
        return jsonResponse($response, [
            'error'             => 'unauthorized',
            'error_description' => $e->getMessage(),
            'status'            => $status,
        ], $status);
    }

    $roles = $claims->roles ?? [];
    if (!in_array('manager', (array) $roles, true)) {
        return jsonResponse($response, [
            'error'             => 'forbidden',
            'error_description' => 'Missing required role(s): manager',
            'status'            => 403,
        ], 403);
    }

    return jsonResponse($response, [
        'message' => 'Manager Reports',
        'user'    => $claims->preferred_username ?? null,
        'roles'   => $roles,
        'reports' => [
            ['name' => 'Q1 Revenue', 'status' => 'complete'],
            ['name' => 'User Growth', 'status' => 'in_progress'],
        ],
    ]);
});

// ── Slim requires explicit routing for OPTIONS on each path ──────────────────

$app->options('/{routes:.+}', function (Request $request, Response $response) {
    return $response;
});

// ── Boot ─────────────────────────────────────────────────────────────────────

$port = (int) (getenv('PORT') ?: 3001);
echo "PHP sample backend running on http://localhost:{$port}\n";
echo "Rampart issuer: {$RAMPART_ISSUER}\n";
echo "\nRoutes:\n";
echo "  GET /api/health            - public\n";
echo "  GET /api/profile           - protected (any authenticated user)\n";
echo "  GET /api/claims            - protected (any authenticated user)\n";
echo "  GET /api/editor/dashboard  - protected (requires \"editor\" role)\n";
echo "  GET /api/manager/reports   - protected (requires \"manager\" role)\n";

$app->run();
