export type ManagedApp = {
  id: string;
  domain: string;
  webPort: number;
  docsPath: string;
  service: string;
};

export const API_BASE_URL =
  import.meta.env.VITE_PLATFORM_API_URL ?? "http://localhost:8080";

export async function fetchManagedApps(): Promise<ManagedApp[]> {
  const response = await fetch(`${API_BASE_URL}/api/apps`);
  if (!response.ok) {
    throw new Error(`Failed to load managed apps (${response.status})`);
  }

  return (await response.json()) as ManagedApp[];
}
