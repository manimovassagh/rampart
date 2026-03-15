<?php

declare(strict_types=1);

namespace Rampart\Laravel;

use Illuminate\Contracts\Http\Kernel;
use Illuminate\Routing\Router;
use Illuminate\Support\ServiceProvider;

/**
 * Laravel service provider for automatic Rampart middleware registration.
 *
 * Auto-discovered via the "extra.laravel.providers" key in composer.json.
 * Registers two route middleware aliases:
 *   - 'rampart'       => RampartMiddleware (JWT verification)
 *   - 'rampart.roles' => RequireRoles (RBAC)
 *
 * Configuration is read from environment variables:
 *   - RAMPART_ISSUER   (required) — Base URL of the Rampart server
 *   - RAMPART_AUDIENCE (optional) — Expected audience claim
 */
class RampartServiceProvider extends ServiceProvider
{
    /**
     * Register bindings in the container.
     */
    public function register(): void
    {
        $this->app->singleton(RampartMiddleware::class, function ($app) {
            return new RampartMiddleware(
                issuer: (string) config('rampart.issuer', env('RAMPART_ISSUER', '')),
                audience: config('rampart.audience', env('RAMPART_AUDIENCE')),
                cacheTtl: (int) config('rampart.jwks_cache_ttl', 300),
                cachePath: config('rampart.jwks_cache_path'),
            );
        });

        $this->app->singleton(RequireRoles::class, function () {
            return new RequireRoles();
        });

        // Publish config file
        $this->mergeConfigFrom($this->configPath(), 'rampart');
    }

    /**
     * Bootstrap services.
     */
    public function boot(): void
    {
        // Publish configuration
        $this->publishes([
            $this->configPath() => $this->app->configPath('rampart.php'),
        ], 'rampart-config');

        // Register route middleware aliases
        /** @var Router $router */
        $router = $this->app->make(Router::class);
        $router->aliasMiddleware('rampart', RampartMiddleware::class);
        $router->aliasMiddleware('rampart.roles', RequireRoles::class);
    }

    /**
     * Path to the package config file.
     */
    private function configPath(): string
    {
        return dirname(__DIR__) . '/config/rampart.php';
    }
}
