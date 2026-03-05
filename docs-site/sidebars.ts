import type {SidebarsConfig} from '@docusaurus/plugin-content-docs';

const sidebars: SidebarsConfig = {
  docsSidebar: [
    'intro',
    {
      type: 'category',
      label: 'Getting Started',
      collapsed: false,
      items: [
        'getting-started/quickstart',
        'getting-started/docker',
        'getting-started/configuration',
        'getting-started/cli',
        'getting-started/login-themes',
        'getting-started/config-export',
        'getting-started/ai-skills',
      ],
    },
    {
      type: 'category',
      label: 'Concepts',
      items: [
        'concepts/authentication',
        'concepts/oauth-oidc',
        'concepts/organizations',
        'concepts/roles-permissions',
        'concepts/sessions',
        'concepts/audit-events',
      ],
    },
    {
      type: 'category',
      label: 'API Reference',
      items: [
        'api/overview',
        'api/authentication',
        'api/oauth-endpoints',
        'api/admin-api',
        'api/oidc-discovery',
      ],
    },
    {
      type: 'category',
      label: 'SDK Adapters',
      items: [
        'sdks/overview',
        'sdks/node',
        'sdks/react',
        'sdks/nextjs',
        'sdks/go',
        'sdks/python',
        'sdks/spring-boot',
      ],
    },
    {
      type: 'category',
      label: 'Architecture',
      items: [
        'architecture/overview',
        'architecture/data-model',
        'architecture/security',
      ],
    },
    {
      type: 'category',
      label: 'Comparison',
      items: [
        'comparison/vs-keycloak',
        'comparison/vs-ory',
        'comparison/vs-zitadel',
        'comparison/vs-authentik',
      ],
    },
    'contributing',
  ],
};

export default sidebars;
