
import React, { useState, useEffect } from 'react';
import ExtraCard from './ExtraCard.jsx';
import Container from './Container.jsx';
import SectionHeader from './SectionHeader.jsx';


function BlacklistPage() {
  const [blacklist, setBlacklist] = useState(null);
  const [error, setError] = useState('');
  const [loading, setLoading] = useState(true);
  // Track unbanned cards by a unique key (mediaType|mediaId|extraType|extraTitle|youtubeId)
  const [unbanned, setUnbanned] = useState({});

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
    if (!groups[reason]) groups[reason] = [];
    groups[reason].push(item);
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
    <Container style={{ minHeight: 'calc(100vh - 64px)', padding: 0 }}>
      {Object.entries(groups).map(([reason, groupItems], groupIdx) => {
        // Only shrink if reason contains this phrase
        let displayReason = reason;
        if (reason.includes('Did not get any data blocks') && reason.length > 40) {
          displayReason = reason.slice(0, 1000) + '...';
        }
        return (
          <div key={groupIdx} style={{ marginBottom: 40 }}>
            <div style={{ fontWeight: 600, fontSize: '1.1em', margin: '0 0 16px 8px', color: '#ef4444', textAlign: 'left', wordBreak: 'break-word' }}>{displayReason}</div>
            <div style={{ ...gridStyle, justifyContent: 'start' }}>
              {groupItems.map((item, idx) => {
                const extra = {
                  Title: item.extra_title || item.extraTitle || '',
                  Type: item.extra_type || item.extraType || '',
                  YoutubeId: item.youtube_id || item.youtubeId || '',
                  reason: item.reason || item.message || '',
                };
                const media = {
                  id: item.media_id || item.mediaId || '',
                  title: item.media_title || item.mediaTitle || '',
                };
                const mediaType = item.media_type || item.mediaType || '';
                // Unique key for this card
                const cardKey = `${mediaType}|${media.id}|${extra.Type}|${extra.Title}|${extra.YoutubeId}`;
                const isUnbanned = !!unbanned[cardKey];
                return (
                  <ExtraCard
                    key={idx}
                    extra={extra}
                    idx={idx}
                    typeExtras={[]}
                    darkMode={true}
                    media={media}
                    mediaType={mediaType}
                    setExtras={() => {}}
                    setModalMsg={() => {}}
                    setShowModal={() => {}}
                    youtubeModal={{ open: false, videoId: '' }}
                    setYoutubeModal={() => {}}
                    YoutubeEmbed={null}
                    rejected={!isUnbanned}
                    onRemoveBan={async () => {
                      await fetch('/api/blacklist/extras/remove', {
                        method: 'POST',
                        headers: { 'Content-Type': 'application/json' },
                        body: JSON.stringify({
                          mediaType,
                          mediaId: media.id,
                          extraType: extra.Type,
                          extraTitle: extra.Title,
                          youtubeId: extra.YoutubeId
                        })
                      });
                      setUnbanned(prev => ({ ...prev, [cardKey]: true }));
                    }}
                  />
                );
              })}
            </div>
          </div>
        );
      })}
    </Container>
  );
}

export default BlacklistPage;
