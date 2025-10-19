
import React, { useState, useEffect, useRef } from 'react';
import './BlacklistPage.mobile.css';
import ExtraCard from './ExtraCard.jsx';
import YoutubePlayer from './YoutubePlayer.jsx';
import Container from './Container.jsx';
import SectionHeader from './SectionHeader.jsx';


function BlacklistPage({ darkMode }) {
  const longArgStringPhrase = "Long argument string detected";
  const longArgStringGroupKey = "Long argument string detected (all)";
  const postprocessingFailedPhrase = "Postprocessing: Conversion failed";
  const postprocessingFailedGroupKey = "Postprocessing: Conversion failed (all)";
  const noDataBlocksPhrase = "Did not get any data blocks";
  const noDataBlocksGroupKey = "Did not get any data blocks (all)";
  const videoUnavailablePhrase = "Video unavailable";
  const videoUnavailableGroupKey = "Video unavailable (all)";
  const signinBotPhrase = "Sign in to confirm you’re not a bot";
  const signinBotGroupKey = "Sign in to confirm you’re not a bot (all)";
  const [blacklist, setBlacklist] = useState(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  const [youtubeModal, setYoutubeModal] = useState({ open: false, videoId: '' });
  // Helper to preload images
  function preloadImages(urls) {
    return Promise.all(
      urls.map(
        url =>
          new Promise(resolve => {
            if (!url) return resolve();
            const img = new window.Image();
            img.onload = img.onerror = () => resolve();
            img.src = url;
          })
      )
    );
  }

  useEffect(() => {
    fetch('/api/blacklist/extras')
      .then(res => {
        if (!res.ok) throw new Error('Failed to fetch blacklist');
        return res.json();
      })
      .then(async data => {
        setBlacklist(data);
        // Collect all image URLs from blacklist items (adjust property as needed)
        let items = Array.isArray(data) ? data : Object.values(data).flat();
        // Try to get thumbnail, poster, or other image field
        const urls = items.map(item => item.thumbnail || item.poster || item.image || null).filter(Boolean);
        if (urls.length > 0) {
          await preloadImages(urls);
        }
        setLoading(false);
      })
      .catch(e => {
        setError(e.message);
        setLoading(false);
      });
  }, []);

  // WebSocket for real-time blacklist status
  const wsRef = useRef(null);
  useEffect(() => {
    const wsUrl = (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host + '/ws/download-queue';
    const ws = new window.WebSocket(wsUrl);
    wsRef.current = ws;
    ws.onopen = () => {
      console.debug('[WebSocket] Connected to download queue (BlacklistPage)');
    };
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'download_queue_update' && Array.isArray(msg.queue)) {
          setBlacklist(prev => {
            if (!prev) return prev;
            // Update status for matching blacklist items
            const update = (arr) => arr.map(item2 => {
              const found = msg.queue.find(q => (q.YouTubeID === (item2.youtubeId)));
              if (found && found.Status && item2.Status !== found.Status) {
                return { ...item2, status: found.Status, Status: found.Status };
              }
              return item2;
            });
            if (Array.isArray(prev)) return update(prev);
            const updated = {};
            for (const k in prev) updated[k] = update(prev[k]);
            return updated;
          });
        }
      } catch (err) {
        console.debug('[WebSocket] Error parsing message', err);
      }
    };
    ws.onerror = (e) => {
      console.debug('[WebSocket] Error', e);
    };
    ws.onclose = () => {
      console.debug('[WebSocket] Closed (BlacklistPage)');
    };
    return () => {
      ws.close();
    };
  }, []);


  if (loading) return <div style={{ padding: 32 }}>Loading blacklist...</div>;
  if (error) return <div style={{ color: 'red', padding: 32 }}>{error}</div>;
  if (!blacklist || (Array.isArray(blacklist) && blacklist.length === 0)) return <div style={{ padding: 32 }}>No blacklisted extras found.</div>;

  // If the blacklist is an object, convert to array for display
  let items;
  if (Array.isArray(blacklist)) {
    items = blacklist;
  } else if (blacklist && typeof blacklist === 'object') {
    items = Object.values(blacklist);
  } else {
    return <div style={{ padding: 32, color: 'red' }}>Unexpected data format<br /><pre>{JSON.stringify(blacklist, null, 2)}</pre></div>;
  }
  if (!Array.isArray(items)) {
    return <div style={{ padding: 32, color: 'red' }}>Unexpected data format<br /><pre>{JSON.stringify(blacklist, null, 2)}</pre></div>;
  }

  // Group items by normalized reason (replace YouTube ID with XXXXXXXX)
  const groups = {};
  // const youtubeIdRegex = /([A-Za-z0-9_-]{8,20})/g;
  const countryPhrase = 'The uploader has not made this video available in your country';
  const countryGroupKey = 'Not available in your country (all)';
  const signinPhrase = "Sign in if you've been granted access to this video";
  const signinAgePhrase = "Sign in to confirm your age";
  const signinGroupKey = "Sign-in required (all)";
  const tooManyRequestsPhrase = 'HTTP Error 429: Too Many Requests';
  const tooManyRequestsGroupKey = 'Too Many Requests (all)';
  items.forEach((item) => {
    let reason = item.reason || item.message || '';
    // Replace YouTube ID in reason with XXXXXXXX if present
    if (item.youtubeId) {
      const ytId = item.youtubeId;
      // Only replace if the ID is present in the reason
      reason = reason.replaceAll(ytId, 'XXXXXXXX');
    }
    // Also replace any likely YouTube ID pattern in the reason
    reason = reason.replace(/([A-Za-z0-9_-]{8,20})/g, (match) => {
      // If the match is the YouTube ID, replace, otherwise leave
      if (item.youtubeId && match === item.youtubeId) return 'XXXXXXXX';
      return match;
    });
    // Group all country restriction, sign-in, and 429 reasons together
    let groupKey = reason;
    if (reason.includes(countryPhrase)) {
      groupKey = countryGroupKey;
    } else if (reason.includes(signinBotPhrase)) {
      groupKey = signinBotGroupKey;
    } else if (reason.includes(signinPhrase) || reason.includes(signinAgePhrase)) {
      groupKey = signinGroupKey;
    } else if (reason.includes(tooManyRequestsPhrase)) {
      groupKey = tooManyRequestsGroupKey;
    } else if (reason.includes(videoUnavailablePhrase)) {
      groupKey = videoUnavailableGroupKey;
    } else if (reason.includes(noDataBlocksPhrase)) {
      groupKey = noDataBlocksGroupKey;
    } else if (reason.includes(postprocessingFailedPhrase)) {
      groupKey = postprocessingFailedGroupKey;
    } else if (reason.includes(longArgStringPhrase)) {
      groupKey = longArgStringGroupKey;
    }
    if (!groups[groupKey]) groups[groupKey] = [];
    groups[groupKey].push(item);
  });

  // If all groups are empty, show a message
  const totalItems = Object.values(groups).reduce((acc, arr) => acc + arr.length, 0);
  if (totalItems === 0) {
    return <div style={{ padding: 32 }}>No blacklisted extras found.</div>;
  }

  const gridStyle = {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, 220px)', // desktop: fixed card width
    gap: 24,
    padding: 32,
    margin: 0,
    width: '100%',
    boxSizing: 'border-box',
    justifyItems: 'start',
    alignItems: 'start',
  };

  return (
    <Container style={{
      minHeight: 'calc(100vh - 64px)',
      padding: 0,
      background: darkMode ? '#18181b' : '#fff',
      color: darkMode ? '#f3f4f6' : '#18181b'
    }}>
      {Object.entries(groups).map(([reason, groupItems], groupIdx) => {
        // Only shrink if reason contains this phrase
        let displayReason = reason;
        if (reason.includes('Did not get any data blocks') && reason.length > 40) {
          displayReason = reason.slice(0, 1000) + '...';
        }
        return (
          <div key={groupIdx} style={{
            marginBottom: 40,
            background: darkMode ? '#23232a' : '#f3f4f6',
            borderRadius: 12,
            boxShadow: darkMode ? '0 2px 8px #0004' : '0 2px 8px #0001',
            padding: 12
          }}>
            <SectionHeader darkMode={darkMode} style={{ fontWeight: 600, fontSize: '1.1em', margin: '0 0 16px 8px', color: '#ef4444', textAlign: 'left', wordBreak: 'break-word' }}>{displayReason}</SectionHeader>
            <div className="BlacklistExtrasGrid" style={{ ...gridStyle, justifyContent: 'start' }}>
              {groupItems.map((item, idx) => {
                const extra = {
                  ExtraTitle: item.extraTitle || '',
                  ExtraType: item.extraType || '',
                  YoutubeId: item.youtubeId || '',
                  reason: item.reason || item.message || '',
                  Status: item.Status || item.status || '',
                };
                const media = {
                  mediaId: item.mediaId || '',
                  mediaTitle: item.mediaTitle || '',
                };
                const mediaType = item.mediaType || '';
                // Unique key for this card
                return (
                  <div key={idx} style={{ display: 'flex', flexDirection: 'column', alignItems: 'stretch' }}>
                    <ExtraCard
                      extra={extra}
                      idx={idx}
                      typeExtras={[]}
                      darkMode={darkMode}
                      media={media}
                      mediaType={mediaType}
                      setExtras={null}
                      setModalMsg={() => {}}
                      setShowModal={() => {}}
                      YoutubeEmbed={null}
                      rejected={true}
                      onPlay={videoId => setYoutubeModal({ open: true, videoId })}
                      onDownloaded={() => {
                        setBlacklist(prev => {
                          if (!prev) return prev;
                          // Update the correct item in the blacklist
                          const update = (arr) => arr.map((item2) => {
                            if (item2.youtubeId === extra.YoutubeId) {
                              return { ...item2, status: 'downloaded', Status: 'downloaded' };
                            }
                            return item2;
                          });
                          if (Array.isArray(prev)) return update(prev);
                          // If object, update all values
                          const updated = {};
                          for (const k in prev) updated[k] = update(prev[k]);
                          return updated;
                        });
                      }}
                    />
                    {media.mediaTitle && media.mediaId && (
                      <a
                        href={
                          mediaType === 'movie'
                            ? `/movies/${media.mediaId}`
                            : mediaType === 'tv'
                              ? `/series/${media.mediaId}`
                              : '#'
                        }
                        style={{
                          marginTop: 8,
                          fontSize: '0.97em',
                          color: darkMode ? '#f3f4f6' : '#23232a',
                          textDecoration: 'none',
                          textAlign: 'center',
                          wordBreak: 'break-word',
                          display: 'block',
                          fontWeight: 500,
                        }}
                      >
                        {media.mediaTitle}
                      </a>
                    )}
                  </div>
                );
              })}
            </div>
          </div>
        );
      })}
      {/* Render YouTube modal only once at the page level */}
      {(youtubeModal.open && youtubeModal.videoId) && (
        <div style={{
          position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh', background: 'rgba(0,0,0,0.7)', zIndex: 99999,
          display: 'flex', alignItems: 'center', justifyContent: 'center',
        }}>
          <div style={{
            position: 'relative',
            background: '#18181b',
            borderRadius: 16,
            boxShadow: '0 2px 24px #000',
            padding: 0,
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            overflow: 'visible',
          }}>
            <button
              onClick={() => setYoutubeModal({ open: false, videoId: '' })}
              style={{ position: 'absolute', top: 8, right: 12, zIndex: 2, fontSize: 28, color: '#fff', background: 'transparent', border: 'none', cursor: 'pointer' }}
              aria-label="Close"
            >×</button>
            <YoutubePlayer videoId={youtubeModal.videoId} />
          </div>
        </div>
      )}
    </Container>
  );
}

export default BlacklistPage;
