import {themes as prismThemes} from 'prism-react-renderer';
import type {Config} from '@docusaurus/types';
import type * as Preset from '@docusaurus/preset-classic';

const config: Config = {
  title: 'Rampart',
  tagline: 'Modern Identity & Access Management — lightweight, fast, and easy to deploy',
  favicon: 'img/favicon.ico',
  future: { v4: true },

  url: 'https://manimovassagh.github.io',
  baseUrl: '/rampart/',

  organizationName: 'manimovassagh',
  projectName: 'rampart',

  onBrokenLinks: 'warn',

  markdown: {
    hooks: {
      onBrokenMarkdownLinks: 'warn',
    },
  },

  i18n: { defaultLocale: 'en', locales: ['en'] },

  presets: [
    [
      'classic',
      {
        docs: {
          sidebarPath: './sidebars.ts',
          editUrl: 'https://github.com/manimovassagh/rampart/tree/main/docs-site/',
        },
        blog: false,
        theme: {
          customCss: './src/css/custom.css',
        },
      } satisfies Preset.Options,
    ],
  ],

  themeConfig: {
    // image: 'img/rampart-social.png',  // TODO: add a real social preview image
    colorMode: {
      defaultMode: 'dark',
      respectPrefersColorScheme: true,
    },
    announcementBar: {
      id: 'v3_release',
      content: '🏰 Rampart v3.2.0 is here — 8 SDK adapters, social login, pentest-verified security — <a href="https://github.com/manimovassagh/rampart/releases">Download now</a>',
      backgroundColor: '#8b5cf6',
      textColor: '#fff',
      isCloseable: true,
    },
    navbar: {
      title: 'Rampart',
      logo: {
        alt: 'Rampart Logo',
        src: 'img/logo.svg',
      },
      items: [
        { type: 'docSidebar', sidebarId: 'docsSidebar', position: 'left', label: 'Docs' },
        { to: '/docs/getting-started/quickstart', label: 'Quick Start', position: 'left' },
        { to: '/docs/api/overview', label: 'API', position: 'left' },
        { to: '/docs/sdks/overview', label: 'SDKs', position: 'left' },
        {
          href: 'https://github.com/manimovassagh/rampart',
          label: 'GitHub',
          position: 'right',
        },
      ],
    },
    footer: {
      style: 'dark',
      links: [
        {
          title: 'Documentation',
          items: [
            { label: 'Getting Started', to: '/docs/getting-started/quickstart' },
            { label: 'Architecture', to: '/docs/architecture/overview' },
            { label: 'API Reference', to: '/docs/api/overview' },
            { label: 'SDK Adapters', to: '/docs/sdks/overview' },
          ],
        },
        {
          title: 'Community',
          items: [
            { label: 'GitHub Discussions', href: 'https://github.com/manimovassagh/rampart/discussions' },
            { label: 'Issues', href: 'https://github.com/manimovassagh/rampart/issues' },
            { label: 'Contributing', to: '/docs/contributing' },
          ],
        },
        {
          title: 'Resources',
          items: [
            { label: 'CI (GitHub Actions)', href: 'https://github.com/manimovassagh/rampart/actions' },
            { label: 'GitHub', href: 'https://github.com/manimovassagh/rampart' },
            { label: 'Releases', href: 'https://github.com/manimovassagh/rampart/releases' },
          ],
        },
      ],
      copyright: `Copyright © ${new Date().getFullYear()} Rampart Project. Licensed under AGPL-3.0.`,
    },
    prism: {
      theme: prismThemes.github,
      darkTheme: prismThemes.dracula,
      additionalLanguages: ['bash', 'json', 'yaml', 'go', 'java', 'python', 'typescript'],
    },
  } satisfies Preset.ThemeConfig,
};

export default config;
