import { DomainShell } from "@platform/shared-components";

export default function App() {
  return (
    <DomainShell
      eyebrow="Planning and publishing workspace"
      title="statuslater.co.uk"
      description="A second domain inside the same platform, with its own public theme, docs repo, and service boundary."
      domain="statuslater.co.uk"
      accent="teal"
      nav={[
        { href: "#product", label: "Product" },
        { href: "#theme", label: "Theme" },
        { href: "#docs", label: "Docs" },
      ]}
    >
      <section class="card-grid" id="product">
        <article class="card">
          <h2>Shared packages</h2>
          <p>Pull UI primitives and service clients from `apps/shared/*` so products stay consistent without forcing one giant app.</p>
        </article>
        <article class="card" id="theme">
          <h2>Domain-first theme</h2>
          <p>Each app owns its public brand tokens inside `apps/&lt;domain&gt;/theme`, while code still lives in `app`.</p>
        </article>
        <article class="card" id="docs">
          <h2>Docs boundary</h2>
          <p>The docs directory can become a submodule without tangling product deploys with marketing or handbook content.</p>
        </article>
      </section>
    </DomainShell>
  );
}
