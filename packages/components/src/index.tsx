import type { JSX } from "solid-js";

type NavItem = {
  href: string;
  label: string;
};

type DomainShellProps = {
  eyebrow: string;
  title: string;
  description: string;
  domain: string;
  accent: string;
  nav: NavItem[];
  children?: JSX.Element;
};

export function DomainShell(props: DomainShellProps) {
  return (
    <main class="domain-shell">
      <section class="domain-shell__hero">
        <p class="domain-shell__eyebrow">{props.eyebrow}</p>
        <h1>{props.title}</h1>
        <p class="domain-shell__description">{props.description}</p>
        <div class="domain-shell__meta">
          <span>Domain: {props.domain}</span>
          <span>Accent: {props.accent}</span>
        </div>
      </section>

      <nav class="domain-shell__nav" aria-label={`${props.domain} navigation`}>
        {props.nav.map((item) => (
          <a href={item.href}>{item.label}</a>
        ))}
      </nav>

      <section class="domain-shell__content">{props.children}</section>
    </main>
  );
}
