export async function getRadarrMovies() {
	const res = await fetch('/api/radarr/movies');
	if (!res.ok) throw new Error('Failed to fetch Radarr movies');
	return await res.json();
}
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

export async function searchExtras({ mediaType, id }) {
	const res = await fetch(`/api/extras/search?mediaType=${encodeURIComponent(mediaType)}&id=${encodeURIComponent(id)}`);
	if (!res.ok) throw new Error('Failed to search extras');
	return await res.json();
}

export async function downloadExtra({ moviePath, extraType, extraTitle, url }) {
	const payload = { moviePath, extraType, extraTitle, url };
	console.log('downloadExtra payload:', payload);
	const res = await fetch(`/api/extras/download`, {
		method: 'POST',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(payload)
	});
	if (!res.ok) throw new Error('Failed to start download');
	return await res.json();
}
