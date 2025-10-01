export async function deleteExtra({ mediaType, mediaId, extraType, extraTitle }) {
	const payload = { mediaType, mediaId: Number(mediaId), extraType, extraTitle };
	const res = await fetch('/api/extras', {
		method: 'DELETE',
		headers: { 'Content-Type': 'application/json' },
		body: JSON.stringify(payload)
	});
	if (!res.ok) throw new Error('Failed to delete extra');
	return await res.json();
}
export async function getHistory() {
	const res = await fetch('/api/history');
	if (!res.ok) throw new Error('Failed to fetch history');
	const data = await res.json();
	return data.history || [];
}
export async function getSeries() {
	const res = await fetch('/api/series');
	if (!res.ok) throw new Error('Failed to fetch Sonarr series');
	return await res.json();
}

export async function getMovies() {
	const res = await fetch('/api/movies');
	if (!res.ok) throw new Error('Failed to fetch Radarr movies');
	return await res.json();
}

// API functions for Gin backend

export async function getRadarrSettings() {
	const res = await fetch('/api/settings/radarr');
	if (!res.ok) throw new Error('Failed to fetch Radarr settings');
	return await res.json();
}


// New: get extras for a movie or series by id
export async function getExtras({ mediaType, id }) {
	let url;
	if (mediaType === 'movie') {
		url = `/api/movies/${encodeURIComponent(id)}/extras`;
	} else if (mediaType === 'tv') {
		url = `/api/series/${encodeURIComponent(id)}/extras`;
	} else {
		throw new Error('Unknown mediaType: ' + mediaType);
	}
	const res = await fetch(url);
	if (!res.ok) throw new Error('Failed to fetch extras');
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
