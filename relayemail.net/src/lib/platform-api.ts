import type { ManagedApp } from "@platform/shared-services";

const DEFAULT_PLATFORM_API_URL = "http://localhost:8080";

export async function fetchPlatformApps(): Promise<ManagedApp[]> {
  const apiBaseUrl = process.env.PLATFORM_API_URL ?? DEFAULT_PLATFORM_API_URL;
  const response = await fetch(new URL("/api/apps", apiBaseUrl), {
    headers: {
      accept: "application/json",
    },
  });

  if (!response.ok) {
    throw new Error(`Failed to load platform apps (${response.status})`);
  }

  return (await response.json()) as ManagedApp[];
}
