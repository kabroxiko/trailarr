// Search YouTube for a media item by mediaType and mediaId
export async function searchYoutube({ mediaType, mediaId }) {
  const res = await fetch("/api/youtube/search", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ mediaType, mediaId }),
  });
  if (!res.ok) throw new Error("Failed to search YouTube");
  return await res.json();
}
