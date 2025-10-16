import { useEffect } from 'react';

/**
 * Polls the backend for live status of all blacklist extras by YouTube ID and updates the blacklist state.
 * @param {Array} blacklist - The current blacklist array (flat).
 * @param {Function} setBlacklist - The setter to update the blacklist array.
 * @param {number} intervalMs - Polling interval in ms (default 3000).
 */
export default function useBlacklistLiveStatus(blacklist, setBlacklist, intervalMs = 3000) {
  useEffect(() => {
    if (!blacklist || !Array.isArray(blacklist) || blacklist.length === 0) return;
    let cancelled = false;
    const youtubeIds = blacklist.map(e => e.youtubeId).filter(Boolean);
    if (youtubeIds.length === 0) return;

    const poll = async () => {
      try {
        const res = await fetch(`/api/extras/status/batch`, {
          method: 'POST',
          headers: { 'Content-Type': 'application/json' },
          body: JSON.stringify({ youtubeIds })
        });
        if (res.ok) {
          const data = await res.json(); // { statuses: { [youtubeId]: { Status: ... } } }
          if (!cancelled && data && data.statuses) {
            setBlacklist(prev => prev && prev.map(ex => {
              const ytId = ex.youtubeId;
              const statusObj = data.statuses[ytId];
              if (statusObj && statusObj.Status && ex.Status !== statusObj.Status) {
                return { ...ex, Status: statusObj.Status, status: statusObj.Status };
              }
              return ex;
            }));
          }
        }
      } catch {
        // ignore
      }
    };
    poll();
    const intervalId = setInterval(poll, intervalMs);
    return () => {
      cancelled = true;
      clearInterval(intervalId);
    };
  }, [blacklist, setBlacklist, intervalMs]);
}
