// Fetch Plex items from backend
export async function fetchPlexItems() {
	const res = await fetch('/api/plex');
	if (!res.ok) throw new Error('Failed to fetch Plex items');
	return await res.json();
}

// API functions for Gin backend

export async function getRadarrSettings() {
	const res = await fetch('/api/settings/radarr');
	if (!res.ok) throw new Error('Failed to fetch Radarr settings');
	return await res.json();
}

export async function searchExtras(movieTitle) {
	const res = await fetch(`/api/extras/search?movie=${encodeURIComponent(movieTitle)}`);
	if (!res.ok) throw new Error('Failed to search extras');
	return await res.json();
}

export async function downloadExtra(url) {
	const res = await fetch(`/api/extras/download`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify({ url })
	});
	if (!res.ok) throw new Error('Failed to start download');
	return await res.json();
}
