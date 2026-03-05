import React from 'react';
import Layout from '@theme/Layout';
import Link from '@docusaurus/Link';
import useDocusaurusContext from '@docusaurus/useDocusaurusContext';
import {useColorMode} from '@docusaurus/theme-common';

const METRICS = [
  {value: '<1s', label: 'Startup'},
  {value: '~30MB', label: 'Memory'},
  {value: '1', label: 'Binary'},
  {value: '7', label: 'SDK Adapters'},
];

const FEATURES = [
  {
    icon: '\uD83D\uDD10',
    title: 'OAuth 2.0 + OIDC',
    description:
      'Full RFC 6749 / OpenID Connect compliant. Authorization code + PKCE, client credentials, refresh tokens, device flow.',
  },
  {
    icon: '\uD83D\uDC65',
    title: 'User Management',
    description:
      'Registration, login, password reset, email verification. TOTP and WebAuthn MFA built in.',
  },
  {
    icon: '\uD83C\uDFE2',
    title: 'Multi-Tenancy',
    description:
      'Organizations and realms by default. Isolate users, clients, and config per tenant.',
  },
  {
    icon: '\uD83D\uDDA5\uFE0F',
    title: 'Admin Console',
    description:
      'Beautiful admin dashboard built with htmx + Tailwind. Manage users, roles, clients, sessions, audit logs, and login themes from one place.',
  },
  {
    icon: '\uD83D\uDCDC',
    title: 'Audit Trail',
    description:
      'Event-sourced audit log for every security-relevant action. Login, token issuance, permission changes.',
  },
  {
    icon: '\uD83D\uDD11',
    title: 'OAuth Clients',
    description:
      'Register and manage OAuth clients with scopes, redirect URIs, and grant types. Public and confidential clients.',
  },
  {
    icon: '\u23F1\uFE0F',
    title: 'Session Management',
    description:
      'Redis-backed sessions with token blacklisting. View and revoke active sessions per user.',
  },
  {
    icon: '\u2328\uFE0F',
    title: 'CLI Tool',
    description:
      'rampart-cli for server management, user provisioning, and developer auth via device flow.',
  },
  {
    icon: '\uD83D\uDCE6',
    title: 'Single Binary',
    description:
      'One binary, zero external dependencies at runtime. UI embedded. Deploy anywhere in seconds.',
  },
];

const COMPARISON = {
  headers: ['Feature', 'Rampart', 'Keycloak', 'Ory Hydra', 'Zitadel', 'Authentik'],
  rows: [
    ['Startup time', '<1s', '~30s', '~2s', '~3s', '~15s'],
    ['Memory usage', '~30MB', '~512MB+', '~50MB', '~100MB', '~300MB'],
    ['Single binary', 'Yes', 'No (JVM)', 'Yes', 'Yes', 'No (Python)'],
    ['Admin UI', 'Built-in', 'Built-in', 'None', 'Built-in', 'Built-in'],
    ['Login theming', '10+ themes', 'FreeMarker', 'BYO', 'Limited', 'Limited'],
    ['PKCE support', 'Yes', 'Yes', 'Yes', 'Yes', 'Yes'],
    ['Multi-tenant', 'Native', 'Realms', 'No', 'Yes', 'Tenants'],
    ['CLI tool', 'Yes', 'kcadm.sh', 'Yes', 'Yes', 'No'],
    ['Database', 'PostgreSQL', 'Many DBs', 'PostgreSQL', 'CockroachDB', 'PostgreSQL'],
    ['SDK adapters', 'Node/React/Go', 'Java-first', 'REST only', 'Go/gRPC', 'Python-first'],
  ],
};

const SDKS = [
  {name: 'Node.js', icon: '\uD83D\uDFE9'},
  {name: 'React', icon: '\u269B\uFE0F'},
  {name: 'Next.js', icon: '\u25B2'},
  {name: 'Go', icon: '\uD83D\uDC39'},
  {name: 'Python', icon: '\uD83D\uDC0D'},
  {name: 'Spring Boot', icon: '\uD83C\uDF31'},
  {name: 'Web / JS', icon: '\uD83C\uDF10'},
  {name: 'Docker', icon: '\uD83D\uDC33'},
];

function getRampartCellClass(rampart: string, others: string[]): string {
  const rampartLower = rampart.toLowerCase();
  if (rampartLower === 'yes' || rampartLower.includes('<1') || rampartLower.includes('~30mb') || rampartLower.includes('native') || rampartLower.includes('10+') || rampartLower.includes('built-in')) {
    return 'win';
  }
  return '';
}

function getCompetitorCellClass(value: string, rampartValue: string): string {
  const v = value.toLowerCase();
  if (v === 'no' || v === 'none' || v === 'byo' || v.includes('no (')) {
    return 'lose';
  }
  if (v === 'limited' || v === 'rest only' || v.includes('-first')) {
    return 'meh';
  }
  return '';
}

function HeroSection(): React.JSX.Element {
  const {colorMode} = useColorMode();
  const isDark = colorMode === 'dark';

  return (
    <section
      style={{
        padding: '5rem 1.5rem 3rem',
        textAlign: 'center',
        maxWidth: 900,
        margin: '0 auto',
      }}>
      <h1
        className="hero__title"
        style={{
          marginBottom: '1.25rem',
          lineHeight: 1.1,
        }}>
        Identity management{' '}
        <span
          style={{
            background: 'linear-gradient(135deg, #8b5cf6, #ec4899)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
            backgroundClip: 'text',
          }}>
          that actually works
        </span>
      </h1>
      <p
        style={{
          fontSize: '1.25rem',
          color: isDark ? '#a1a1aa' : '#52525b',
          maxWidth: 680,
          margin: '0 auto 2rem',
          lineHeight: 1.6,
        }}>
        Modern, lightweight IAM server. Full OAuth 2.0 + OIDC in a single binary.
        30MB memory. Sub-second startup.
      </p>
      <div style={{display: 'flex', gap: '1rem', justifyContent: 'center', flexWrap: 'wrap'}}>
        <Link
          className="button button--primary button--lg"
          to="/docs/getting-started/quickstart"
          style={{
            borderRadius: 8,
            padding: '0.75rem 2rem',
            fontWeight: 700,
          }}>
          Get Started
        </Link>
        <Link
          className="button button--outline button--lg"
          href="https://github.com/manimovassagh/rampart"
          style={{
            borderRadius: 8,
            padding: '0.75rem 2rem',
            fontWeight: 700,
            borderColor: isDark ? '#3f3f46' : '#d4d4d8',
            color: isDark ? '#e4e4e7' : '#27272a',
          }}>
          GitHub
        </Link>
      </div>
    </section>
  );
}

function MetricsSection(): React.JSX.Element {
  return (
    <section className="metrics-row">
      {METRICS.map((m) => (
        <div key={m.label} className="metric">
          <div className="value">{m.value}</div>
          <div className="label">{m.label}</div>
        </div>
      ))}
    </section>
  );
}

function FeaturesSection(): React.JSX.Element {
  return (
    <section style={{padding: '3rem 1.5rem', maxWidth: 1100, margin: '0 auto'}}>
      <h2 style={{textAlign: 'center', fontSize: '2rem', fontWeight: 800, marginBottom: '0.5rem'}}>
        Everything you need. Nothing you don't.
      </h2>
      <p style={{textAlign: 'center', color: 'var(--ifm-color-emphasis-600)', marginBottom: '2rem'}}>
        Production-grade IAM features, built from the ground up.
      </p>
      <div className="features-grid">
        {FEATURES.map((f) => (
          <div key={f.title} className="feature-card">
            <h3>
              <span style={{marginRight: '0.5rem'}}>{f.icon}</span>
              {f.title}
            </h3>
            <p>{f.description}</p>
          </div>
        ))}
      </div>
    </section>
  );
}

function ComparisonSection(): React.JSX.Element {
  const {colorMode} = useColorMode();
  const isDark = colorMode === 'dark';

  return (
    <section style={{padding: '3rem 1.5rem', maxWidth: 1100, margin: '0 auto'}}>
      <h2 style={{textAlign: 'center', fontSize: '2rem', fontWeight: 800, marginBottom: '0.5rem'}}>
        How Rampart stacks up
      </h2>
      <p style={{textAlign: 'center', color: 'var(--ifm-color-emphasis-600)', marginBottom: '2rem'}}>
        A fair comparison with the alternatives.
      </p>
      <div style={{overflowX: 'auto'}}>
        <table
          style={{
            width: '100%',
            borderCollapse: 'collapse',
            fontSize: '0.9rem',
          }}>
          <thead>
            <tr>
              {COMPARISON.headers.map((h, i) => (
                <th
                  key={h}
                  style={{
                    textAlign: i === 0 ? 'left' : 'center',
                    padding: '0.75rem 1rem',
                    borderBottom: `2px solid ${isDark ? '#27272a' : '#e4e4e7'}`,
                    fontWeight: 700,
                    fontSize: '0.8rem',
                    textTransform: 'uppercase',
                    letterSpacing: '0.05em',
                    color: i === 1 ? 'var(--ifm-color-primary)' : undefined,
                  }}>
                  {h}
                </th>
              ))}
            </tr>
          </thead>
          <tbody>
            {COMPARISON.rows.map((row) => (
              <tr key={row[0]}>
                {row.map((cell, i) => {
                  let className = '';
                  if (i === 1) {
                    className = getRampartCellClass(cell, row.slice(2));
                  } else if (i > 1) {
                    className = getCompetitorCellClass(cell, row[1]);
                  }
                  return (
                    <td
                      key={`${row[0]}-${i}`}
                      className={className}
                      style={{
                        textAlign: i === 0 ? 'left' : 'center',
                        padding: '0.65rem 1rem',
                        borderBottom: `1px solid ${isDark ? '#1e1e2a' : '#f4f4f5'}`,
                        fontWeight: i === 1 ? 600 : 400,
                      }}>
                      {cell}
                    </td>
                  );
                })}
              </tr>
            ))}
          </tbody>
        </table>
      </div>
    </section>
  );
}

function SDKSection(): React.JSX.Element {
  const {colorMode} = useColorMode();
  const isDark = colorMode === 'dark';

  return (
    <section style={{padding: '3rem 1.5rem', maxWidth: 900, margin: '0 auto'}}>
      <h2 style={{textAlign: 'center', fontSize: '2rem', fontWeight: 800, marginBottom: '0.5rem'}}>
        Works with your stack
      </h2>
      <p style={{textAlign: 'center', color: 'var(--ifm-color-emphasis-600)', marginBottom: '2rem'}}>
        First-class SDK adapters and integration guides.
      </p>
      <div
        style={{
          display: 'grid',
          gridTemplateColumns: 'repeat(auto-fit, minmax(120px, 1fr))',
          gap: '1rem',
          maxWidth: 700,
          margin: '0 auto',
        }}>
        {SDKS.map((sdk) => (
          <div
            key={sdk.name}
            style={{
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              gap: '0.5rem',
              padding: '1.25rem 0.75rem',
              borderRadius: 12,
              border: `1px solid ${isDark ? '#27272a' : '#e4e4e7'}`,
              background: isDark ? '#16161f' : '#fafafa',
              transition: 'border-color 0.2s, transform 0.2s',
            }}
            onMouseEnter={(e) => {
              (e.currentTarget as HTMLElement).style.borderColor = '#8b5cf6';
              (e.currentTarget as HTMLElement).style.transform = 'translateY(-2px)';
            }}
            onMouseLeave={(e) => {
              (e.currentTarget as HTMLElement).style.borderColor = isDark ? '#27272a' : '#e4e4e7';
              (e.currentTarget as HTMLElement).style.transform = 'translateY(0)';
            }}>
            <span style={{fontSize: '2rem'}}>{sdk.icon}</span>
            <span
              style={{
                fontSize: '0.8rem',
                fontWeight: 600,
                color: isDark ? '#a1a1aa' : '#52525b',
              }}>
              {sdk.name}
            </span>
          </div>
        ))}
      </div>
    </section>
  );
}

function CTASection(): React.JSX.Element {
  const {colorMode} = useColorMode();
  const isDark = colorMode === 'dark';

  return (
    <section
      style={{
        padding: '4rem 1.5rem',
        textAlign: 'center',
        maxWidth: 700,
        margin: '0 auto',
      }}>
      <h2
        style={{
          fontSize: '2.25rem',
          fontWeight: 900,
          marginBottom: '1rem',
          lineHeight: 1.2,
        }}>
        Ready to ditch{' '}
        <span
          style={{
            background: 'linear-gradient(135deg, #8b5cf6, #ec4899)',
            WebkitBackgroundClip: 'text',
            WebkitTextFillColor: 'transparent',
            backgroundClip: 'text',
          }}>
          Keycloak
        </span>
        ?
      </h2>
      <p
        style={{
          fontSize: '1.1rem',
          color: isDark ? '#a1a1aa' : '#52525b',
          marginBottom: '2rem',
          lineHeight: 1.6,
        }}>
        Deploy Rampart in under a minute. One binary, no JVM, no YAML nightmares.
      </p>
      <div style={{display: 'flex', gap: '1rem', justifyContent: 'center', flexWrap: 'wrap'}}>
        <Link
          className="button button--primary button--lg"
          href="https://github.com/manimovassagh/rampart/releases"
          style={{
            borderRadius: 8,
            padding: '0.75rem 2rem',
            fontWeight: 700,
          }}>
          Download Rampart
        </Link>
        <Link
          className="button button--outline button--lg"
          to="/docs/getting-started/quickstart"
          style={{
            borderRadius: 8,
            padding: '0.75rem 2rem',
            fontWeight: 700,
            borderColor: isDark ? '#3f3f46' : '#d4d4d8',
            color: isDark ? '#e4e4e7' : '#27272a',
          }}>
          Read the Docs
        </Link>
      </div>
    </section>
  );
}

export default function Home(): React.JSX.Element {
  const {siteConfig} = useDocusaurusContext();

  return (
    <Layout
      title={`${siteConfig.title} - Modern IAM Server`}
      description="Modern, lightweight identity and access management server. The open-source Keycloak alternative with OAuth 2.0, OIDC, single binary deployment, and beautiful admin UI.">
      <main>
        <HeroSection />
        <MetricsSection />
        <div
          style={{
            width: 80,
            height: 2,
            background: 'linear-gradient(90deg, #8b5cf6, #ec4899)',
            margin: '2rem auto',
            borderRadius: 1,
          }}
        />
        <FeaturesSection />
        <ComparisonSection />
        <SDKSection />
        <CTASection />
      </main>
    </Layout>
  );
}
