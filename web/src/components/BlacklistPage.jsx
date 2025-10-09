
import React, { useState, useEffect } from 'react';
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

  useEffect(() => {
    fetch('/api/blacklist/extras')
      .then(res => {
        if (!res.ok) throw new Error('Failed to fetch blacklist');
        return res.json();
      })
      .then(data => {
        setBlacklist(data);
        setLoading(false);
      })
      .catch(e => {
        setError(e.message);
        setLoading(false);
      });
  }, []);

  if (loading) return <div style={{ padding: 32 }}>Loading blacklist...</div>;
  if (error) return <div style={{ color: 'red', padding: 32 }}>{error}</div>;
  if (!blacklist || (Array.isArray(blacklist) && blacklist.length === 0)) return <div style={{ padding: 32 }}>No rejected extras found.</div>;

  // If the blacklist is an object, convert to array for display
  let items = Array.isArray(blacklist) ? blacklist : Object.values(blacklist);
  if (!Array.isArray(items)) {
    return <div style={{ padding: 32, color: 'red' }}>Unexpected data format<br /><pre>{JSON.stringify(blacklist, null, 2)}</pre></div>;
  }

  // Group items by normalized reason (replace YouTube ID with XXXXXXXX)
  const groups = {};
  const youtubeIdRegex = /([A-Za-z0-9_-]{8,20})/g;
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
    if (item.youtube_id || item.youtubeId) {
      const ytId = item.youtube_id || item.youtubeId;
      // Only replace if the ID is present in the reason
      reason = reason.replaceAll(ytId, 'XXXXXXXX');
    }
    // Also replace any likely YouTube ID pattern in the reason
    reason = reason.replace(/([A-Za-z0-9_-]{8,20})/g, (match) => {
      // If the match is the YouTube ID, replace, otherwise leave
      if (item.youtube_id && match === item.youtube_id) return 'XXXXXXXX';
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

  const gridStyle = {
    display: 'grid',
    gridTemplateColumns: 'repeat(auto-fill, 220px)', // fixed card width
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
            <div style={{ ...gridStyle, justifyContent: 'start' }}>
              {groupItems.map((item, idx) => {
                console.log('Blacklist item:', item);
                const extra = {
                  Title: item.extra_title || item.extraTitle || '',
                  Type: item.extra_type || item.extraType || '',
                  YoutubeId: item.youtube_id || item.youtubeId || '',
                  reason: item.reason || item.message || '',
                  Status: item.Status || item.status || '',
                };
                const media = {
                  id: item.media_id || item.mediaId || '',
                  title: item.media_title || item.mediaTitle || '',
                };
                console.log('Mapped media:', media);
                const mediaType = item.media_type || item.mediaType || '';
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
                            if ((item2.extra_title || item2.extraTitle) === extra.Title &&
                                (item2.extra_type || item2.extraType) === extra.Type &&
                                (item2.youtube_id || item2.youtubeId) === extra.YoutubeId) {
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
                    {media.title && media.id && (
                      <a
                        href={
                          mediaType === 'movie'
                            ? `/movies/${media.id}`
                            : mediaType === 'tv'
                              ? `/series/${media.id}`
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
                        {media.title}
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
