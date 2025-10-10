import { useEffect } from 'react';

/**
 * Polls the backend for live status of all extras by YouTube ID and updates the extras state.
 * @param {Array} extras - The current extras array (flat, not grouped by type).
 * @param {Function} setExtras - The setter to update the extras array.
 * @param {number} intervalMs - Polling interval in ms (default 3000).
 */
export default function useExtrasLiveStatus(extras, setExtras, intervalMs = 3000) {
  useEffect(() => {
    if (!extras || !Array.isArray(extras) || extras.length === 0) return;
    let cancelled = false;
    const youtubeIds = extras.map(e => e.YoutubeId).filter(Boolean);
    if (youtubeIds.length === 0) return;

    const poll = async () => {
      try {
        // Batch endpoint preferred, fallback to individual
        const res = await fetch(`/api/extras/status/batch`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ youtubeIds })
        });
        if (res.ok) {
          const data = await res.json(); // { statuses: { [youtubeId]: { Status: ... } } }
          if (!cancelled && data && data.statuses) {
            setExtras(prev => prev.map(ex => {
              const statusObj = data.statuses[ex.YoutubeId];
              if (statusObj && statusObj.Status && ex.Status !== statusObj.Status) {
                return { ...ex, Status: statusObj.Status };
              }
              return ex;
            }));
          }
        }
      } catch (e) {
        // ignore
      }
    };
    poll();
    const intervalId = setInterval(poll, intervalMs);
    return () => {
      cancelled = true;
      clearInterval(intervalId);
    };
  }, [extras, setExtras, intervalMs]);
}
