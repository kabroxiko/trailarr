// Search YouTube for a query string
export async function searchYoutube(query) {
  const res = await fetch(`/api/youtube/search?q=${encodeURIComponent(query)}`);
  if (!res.ok) throw new Error('Failed to search YouTube');
  return await res.json();
}
