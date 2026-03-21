// import { fetchPlatformApps } from '../../lib/platform-api';

export async function GET() {
	try {
		// const apps = await fetchPlatformApps();
		// return Response.json(apps);
		return Response.json(
			{ message: 'success' },
			{
				status: 200,
			},
		);
	} catch (error) {
		const message =
			error instanceof Error ? error.message : 'Unknown upstream error';

		return Response.json(
			{
				error: message,
			},
			{
				status: 502,
			},
		);
	}
}
