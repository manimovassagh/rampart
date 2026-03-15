<?php

declare(strict_types=1);

namespace Rampart\Laravel;

use Closure;
use Firebase\JWT\JWK;
use Firebase\JWT\JWT;
use Firebase\JWT\ExpiredException;
use Firebase\JWT\SignatureInvalidException;
use GuzzleHttp\Client;
use GuzzleHttp\Exception\GuzzleException;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;

/**
 * JWT verification middleware for Rampart IAM.
 *
 * Works as both a Laravel middleware and a PSR-15 middleware.
 *
 * Laravel usage:
 *   Register in your kernel or route middleware, then:
 *   Route::middleware('rampart')->group(function () { ... });
 *
 * PSR-15 usage:
 *   $middleware = new RampartMiddleware('https://auth.example.com');
 *   $response = $middleware->process($request, $handler);
 */
class RampartMiddleware implements MiddlewareInterface
{
    private string $issuer;
    private ?string $audience;
    private string $jwksUri;
    private string $cachePath;
    private int $cacheTtl;

    /** @var array<string, mixed>|null */
    private ?array $cachedKeys = null;

    private Client $httpClient;

    /**
     * @param string      $issuer    Base URL of the Rampart server
     * @param string|null $audience  Expected audience claim (optional)
     * @param int         $cacheTtl  JWKS cache lifetime in seconds (default: 300)
     * @param string|null $cachePath Path for file-based JWKS cache (default: sys_get_temp_dir())
     */
    public function __construct(
        ?string $issuer = null,
        ?string $audience = null,
        int $cacheTtl = 300,
        ?string $cachePath = null,
        ?Client $httpClient = null,
    ) {
        $this->issuer = rtrim($issuer ?? (string) env('RAMPART_ISSUER', ''), '/');
        $this->audience = $audience ?? env('RAMPART_AUDIENCE');
        $this->cacheTtl = $cacheTtl;
        $this->jwksUri = $this->issuer . '/.well-known/jwks.json';
        $this->cachePath = ($cachePath ?? sys_get_temp_dir()) . '/rampart_jwks_' . md5($this->issuer) . '.json';
        $this->httpClient = $httpClient ?? new Client(['timeout' => 10]);
    }

    // ---------------------------------------------------------------
    // Laravel middleware interface
    // ---------------------------------------------------------------

    /**
     * Handle an incoming Laravel request.
     *
     * @param Request  $request
     * @param Closure  $next
     * @return mixed
     */
    public function handle(Request $request, Closure $next): mixed
    {
        $authHeader = $request->header('Authorization', '');
        if (!str_starts_with($authHeader, 'Bearer ')) {
            return self::errorResponse('Missing or invalid Authorization header', 401);
        }

        $token = substr($authHeader, 7);

        try {
            $claims = $this->verifyToken($token);
        } catch (ExpiredException) {
            return self::errorResponse('Token has expired', 401);
        } catch (\UnexpectedValueException | SignatureInvalidException $e) {
            return self::errorResponse('Invalid token: ' . $e->getMessage(), 401);
        } catch (\Throwable $e) {
            return self::errorResponse('Invalid token: ' . $e->getMessage(), 401);
        }

        $request->attributes->set('rampart', $claims);

        return $next($request);
    }

    // ---------------------------------------------------------------
    // PSR-15 MiddlewareInterface
    // ---------------------------------------------------------------

    /**
     * Process a PSR-15 server request.
     */
    public function process(ServerRequestInterface $request, RequestHandlerInterface $handler): ResponseInterface
    {
        $authHeader = $request->getHeaderLine('Authorization');
        if (!str_starts_with($authHeader, 'Bearer ')) {
            return self::psrErrorResponse('Missing or invalid Authorization header', 401);
        }

        $token = substr($authHeader, 7);

        try {
            $claims = $this->verifyToken($token);
        } catch (ExpiredException) {
            return self::psrErrorResponse('Token has expired', 401);
        } catch (\UnexpectedValueException | SignatureInvalidException $e) {
            return self::psrErrorResponse('Invalid token: ' . $e->getMessage(), 401);
        } catch (\Throwable $e) {
            return self::psrErrorResponse('Invalid token: ' . $e->getMessage(), 401);
        }

        $request = $request->withAttribute('rampart', $claims);

        return $handler->handle($request);
    }

    // ---------------------------------------------------------------
    // Token verification
    // ---------------------------------------------------------------

    /**
     * Verify a JWT token and return parsed claims.
     *
     * @param string $token Raw JWT string (without "Bearer " prefix)
     * @return RampartClaims
     * @throws ExpiredException
     * @throws SignatureInvalidException
     * @throws \UnexpectedValueException
     */
    public function verifyToken(string $token): RampartClaims
    {
        $keys = $this->getJwks();

        $decoded = JWT::decode($token, $keys);

        // Validate issuer
        $payload = (array) $decoded;
        if (($payload['iss'] ?? '') !== $this->issuer) {
            throw new \UnexpectedValueException(
                sprintf('Issuer mismatch: expected "%s", got "%s"', $this->issuer, $payload['iss'] ?? '')
            );
        }

        // Validate audience if configured
        if ($this->audience !== null) {
            $aud = $payload['aud'] ?? null;
            $audiences = is_array($aud) ? $aud : [$aud];
            if (!in_array($this->audience, $audiences, true)) {
                throw new \UnexpectedValueException(
                    sprintf('Audience mismatch: expected "%s"', $this->audience)
                );
            }
        }

        return RampartClaims::fromToken($decoded);
    }

    // ---------------------------------------------------------------
    // JWKS fetching and caching
    // ---------------------------------------------------------------

    /**
     * Fetch JWKS keys with file-based caching.
     *
     * @return array<string, \Firebase\JWT\Key>
     */
    private function getJwks(): array
    {
        // In-memory cache
        if ($this->cachedKeys !== null) {
            return JWK::parseKeySet($this->cachedKeys, 'RS256');
        }

        // File-based cache
        if (file_exists($this->cachePath)) {
            $cacheData = json_decode((string) file_get_contents($this->cachePath), true);
            if (
                is_array($cacheData)
                && isset($cacheData['fetched_at'], $cacheData['keys'])
                && (time() - (int) $cacheData['fetched_at']) < $this->cacheTtl
            ) {
                $this->cachedKeys = $cacheData['keys'];
                return JWK::parseKeySet($this->cachedKeys, 'RS256');
            }
        }

        // Fetch from remote
        $jwks = $this->fetchJwks();
        $this->cachedKeys = $jwks;

        // Write to file cache
        $cachePayload = json_encode([
            'fetched_at' => time(),
            'keys' => $jwks,
        ]);
        if ($cachePayload !== false) {
            file_put_contents($this->cachePath, $cachePayload, LOCK_EX);
        }

        return JWK::parseKeySet($jwks, 'RS256');
    }

    /**
     * Fetch the JWKS document from the Rampart server.
     *
     * @return array<string, mixed>
     * @throws \RuntimeException
     */
    private function fetchJwks(): array
    {
        try {
            $response = $this->httpClient->get($this->jwksUri);
            $body = (string) $response->getBody();
        } catch (GuzzleException $e) {
            throw new \RuntimeException('Failed to fetch JWKS from ' . $this->jwksUri . ': ' . $e->getMessage(), 0, $e);
        }

        $data = json_decode($body, true);
        if (!is_array($data) || !isset($data['keys'])) {
            throw new \RuntimeException('Invalid JWKS response from ' . $this->jwksUri);
        }

        return $data;
    }

    /**
     * Force-clear the cached JWKS keys (useful for key rotation).
     */
    public function clearCache(): void
    {
        $this->cachedKeys = null;
        if (file_exists($this->cachePath)) {
            @unlink($this->cachePath);
        }
    }

    // ---------------------------------------------------------------
    // Error response helpers
    // ---------------------------------------------------------------

    /**
     * Return a Laravel JSON error response matching Rampart error format.
     */
    private static function errorResponse(string $detail, int $status): JsonResponse
    {
        return new JsonResponse(['detail' => $detail], $status);
    }

    /**
     * Return a PSR-7 JSON error response.
     */
    private static function psrErrorResponse(string $detail, int $status): ResponseInterface
    {
        $body = json_encode(['detail' => $detail]);
        $response = new \GuzzleHttp\Psr7\Response($status, ['Content-Type' => 'application/json'], $body);
        return $response;
    }
}
