// Progressive YouTube search using SSE
export function searchYoutubeStream({
  mediaType,
  mediaId,
  onResult,
  onDone,
  onError,
}) {
  const controller = new AbortController();
  const url = `/api/youtube/search/stream?mediaType=${encodeURIComponent(mediaType)}&mediaId=${encodeURIComponent(mediaId)}`;
  fetch(url, {
    method: "GET",
    signal: controller.signal,
  })
    .then(async (res) => {
      if (!res.body) throw new Error("No response body");
      const reader = res.body.getReader();
      let buffer = "";
      const decoder = new TextDecoder();
      let done = false;
      while (!done) {
        const { value, done: streamDone } = await reader.read();
        if (value) buffer += decoder.decode(value, { stream: true });
        done = streamDone;
        let idx;
        while ((idx = buffer.indexOf("\n\n")) !== -1) {
          const chunk = buffer.slice(0, idx);
          buffer = buffer.slice(idx + 2);
          if (chunk.startsWith("data: ")) {
            try {
              const json = JSON.parse(chunk.slice(6));
              if (onResult) onResult(json);
            } catch {
              // ignore parse error
            }
          } else if (chunk.startsWith("event: done")) {
            if (onDone) onDone();
          }
        }
      }
      if (onDone) onDone();
    })
    .catch((err) => {
      if (onError) onError(err);
    });
  return controller;
}
