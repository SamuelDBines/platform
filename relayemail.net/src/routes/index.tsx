import { createResource, For, Show } from 'solid-js';
import { isServer } from 'solid-js/web';
import { DomainShell } from '@platform/shared-components';
import type { ManagedApp } from '@platform/shared-services';
// import { fetchPlatformApps } from '../lib/platform-api';

async function fetchAppsFromBff(): Promise<ManagedApp[]> {
	// if (isServer) {
	// 	return fetchPlatformApps();
	// }

	const response = await fetch('/api/apps');
	if (!response.ok) {
		throw new Error(`Failed to load relayemail apps (${response.status})`);
	}

	return (await response.json()) as ManagedApp[];
}

export default function HomePage() {
	const [apps] = createResource(fetchAppsFromBff);

	return (
		<DomainShell
			eyebrow='Managed delivery infrastructure'
			title='relayemail.net'
			description='A SolidStart frontend with a BFF edge so the UI can proxy upstream services, hold secrets server-side, and keep product traffic off the docs site.'
			domain='relayemail.net'
			accent='amber'
			nav={[
				{ href: '#services', label: 'Services' },
				{ href: '#bff', label: 'BFF' },
				{ href: '#ops', label: 'Ops' },
			]}
		>
			<section class='card-grid' id='services'>
				<article class='card'>
					<h2>Docs stay static</h2>
					<p>
						Public marketing and docs can keep living in `relayemail.net/docs`
						as a cheap static deployment.
					</p>
				</article>
				<article class='card' id='bff'>
					<h2>App stays dynamic</h2>
					<p>
						This SolidStart app now has server routes, which gives you a proper
						BFF layer for auth, aggregation, cookies, and upstream API calls.
					</p>
				</article>
				<article class='card'>
					<h2>Backend bridge</h2>
					<p>
						The `/api/apps` route already proxies the platform service, which is
						the pattern you can extend for account, billing, and relay
						operations.
					</p>
				</article>
			</section>

			<section class='stack' id='ops'>
				<h2>Managed apps from the relayemail BFF</h2>
				<Show when={apps.error}>
					<p>
						The BFF is up, but the upstream platform API is unavailable. Start
						the Go service on `http://localhost:8080` or set `PLATFORM_API_URL`.
					</p>
				</Show>
				<Show
					when={!apps.loading && !apps.error}
					fallback={<p>Loading through the server boundary...</p>}
				>
					<div class='list'>
						<For each={apps()}>
							{(app) => (
								<article class='list__item'>
									<strong>{app.domain}</strong>
									<span>{app.service}</span>
									<span>web :{app.webPort}</span>
								</article>
							)}
						</For>
					</div>
				</Show>
			</section>
		</DomainShell>
	);
}
