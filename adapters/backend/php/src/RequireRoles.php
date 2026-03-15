<?php

declare(strict_types=1);

namespace Rampart\Laravel;

use Closure;
use Illuminate\Http\JsonResponse;
use Illuminate\Http\Request;
use Psr\Http\Message\ResponseInterface;
use Psr\Http\Message\ServerRequestInterface;
use Psr\Http\Server\MiddlewareInterface;
use Psr\Http\Server\RequestHandlerInterface;

/**
 * Role-based access control middleware for Rampart.
 *
 * Must be applied after RampartMiddleware so that the 'rampart' attribute is set.
 *
 * Laravel usage:
 *   Route::middleware(['rampart', 'rampart.roles:admin,editor'])->group(function () { ... });
 *
 * PSR-15 usage:
 *   $rbac = new RequireRoles('admin', 'editor');
 *   // Stack after RampartMiddleware in your middleware pipeline.
 */
class RequireRoles implements MiddlewareInterface
{
    /** @var string[] */
    private array $requiredRoles;

    /**
     * @param string ...$roles Role names that the token must contain
     */
    public function __construct(string ...$roles)
    {
        $this->requiredRoles = $roles;
    }

    // ---------------------------------------------------------------
    // Laravel middleware interface
    // ---------------------------------------------------------------

    /**
     * Handle an incoming Laravel request.
     *
     * Laravel passes route middleware parameters as extra arguments after $next.
     * Usage: Route::middleware('rampart.roles:admin,editor')
     *
     * @param Request  $request
     * @param Closure  $next
     * @param string   ...$roles  Roles passed via route middleware definition
     * @return mixed
     */
    public function handle(Request $request, Closure $next, string ...$roles): mixed
    {
        // Use roles from route definition if provided, otherwise use constructor roles
        $required = !empty($roles) ? $roles : $this->requiredRoles;

        /** @var RampartClaims|null $claims */
        $claims = $request->attributes->get('rampart');

        if ($claims === null) {
            return new JsonResponse(
                ['detail' => 'Authentication required before role check'],
                401
            );
        }

        $missing = $claims->missingRoles(...$required);
        if (!empty($missing)) {
            return new JsonResponse(
                ['detail' => 'Missing required roles: ' . implode(', ', $missing)],
                403
            );
        }

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
        /** @var RampartClaims|null $claims */
        $claims = $request->getAttribute('rampart');

        if ($claims === null) {
            return self::psrErrorResponse('Authentication required before role check', 401);
        }

        $missing = $claims->missingRoles(...$this->requiredRoles);
        if (!empty($missing)) {
            return self::psrErrorResponse(
                'Missing required roles: ' . implode(', ', $missing),
                403
            );
        }

        return $handler->handle($request);
    }

    /**
     * Return a PSR-7 JSON error response.
     */
    private static function psrErrorResponse(string $detail, int $status): ResponseInterface
    {
        $body = json_encode(['detail' => $detail]);
        return new \GuzzleHttp\Psr7\Response($status, ['Content-Type' => 'application/json'], $body);
    }
}
