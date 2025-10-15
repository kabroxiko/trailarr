import React, { useState, useEffect, useRef } from 'react';
import MediaInfoLane from './MediaInfoLane.jsx';
import MediaCard from './MediaCard.jsx';
import ExtrasList from './ExtrasList';
import YoutubePlayer from './YoutubePlayer.jsx';
import Container from './Container.jsx';
import Toast from './Toast.jsx';
import { useParams } from 'react-router-dom';

// Spinner and YouTubeEmbed component
function Spinner() {
  return (
    <div style={{
      position: 'absolute',
      top: '50%',
      left: '50%',
      transform: 'translate(-50%, -50%)',
      zIndex: 10,
      display: 'flex',
      alignItems: 'center',
      justifyContent: 'center',
      background: 'rgba(0,0,0,0.2)',
      borderRadius: 8,
      padding: 16,
    }}>
      <svg width="48" height="48" viewBox="0 0 48 48" fill="none" xmlns="http://www.w3.org/2000/svg">
        <circle cx="24" cy="24" r="20" stroke="#a855f7" strokeWidth="4" opacity="0.2" />
        <path d="M44 24c0-11.046-8.954-20-20-20" stroke="#a855f7" strokeWidth="4" strokeLinecap="round" />
      </svg>
    </div>
  );
}

function YoutubeEmbed({ videoId }) {
  const [loading, setLoading] = useState(true);
  useEffect(() => {
    setLoading(true);
    console.log('YoutubeEmbed mounted', videoId);
    return () => {
      console.log('YoutubeEmbed unmounted', videoId);
    };
  }, [videoId]);
  return (
    <div style={{ width: '100%', height: '100%', position: 'relative', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      {loading && <Spinner />}
      <iframe
        src={`https://www.youtube.com/embed/${videoId}`}
        title="YouTube video player"
        frameBorder="0"
        allow="accelerometer; autoplay; clipboard-write; encrypted-media; gyroscope; picture-in-picture"
        allowFullScreen
        loading="lazy"
        style={{
          borderRadius: 8,
          background: '#000',
          width: '100%',
          height: '100%',
          position: 'absolute',
          top: 0,
          left: 0,
        }}
        onLoad={() => setLoading(false)}
      />
    </div>
  );
}

export default function MediaDetails({ mediaItems, loading, mediaType }) {
  const [youtubeModal, setYoutubeModal] = useState({ open: false, videoId: '' });
  // Store YouTube search results for merging into Trailers group
  const [ytResults, setYtResults] = useState([]);

  // Close modal on outside click or Escape
  useEffect(() => {
    if (!youtubeModal.open) return;
    const handleKey = (e) => { if (e.key === 'Escape') setYoutubeModal({ open: false, videoId: '' }); };
    const handleClick = (e) => {
      if (e.target.classList.contains('youtube-modal-backdrop')) setYoutubeModal({ open: false, videoId: '' });
    };
    window.addEventListener('keydown', handleKey);
    window.addEventListener('mousedown', handleClick);
    return () => {
      window.removeEventListener('keydown', handleKey);
      window.removeEventListener('mousedown', handleClick);
    };
  }, [youtubeModal.open]);
  const { id } = useParams();
  const media = mediaItems.find(m => String(m.id) === id);
  const [extras, setExtras] = useState([]);
  const [searchLoading, setSearchLoading] = useState(false);
  const [error, setError] = useState('');
  const [modalMsg, setModalMsg] = useState('');
  const [showModal, setShowModal] = useState(false);
  const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
  const [darkMode, setDarkMode] = useState(prefersDark);
  useEffect(() => {
    const listener = e => setDarkMode(e.matches);
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', listener);
    return () => window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', listener);
  }, []);

  useEffect(() => {
    if (!media) return;
    setSearchLoading(true);
    setError('');
    import('../api').then(({ getExtras }) => {
      getExtras({ mediaType, id: media.id })
        .then(res => {
          setExtras(res.extras || []);
        })
        .catch(() => setError('Failed to fetch extras'))
        .finally(() => setSearchLoading(false));
    });
  }, [media, mediaType]);

  // WebSocket for real-time extras status
  const wsRef = useRef(null);
  useEffect(() => {
    if (!media) return;
    const wsUrl = (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host + '/ws/download-queue';
    const ws = new window.WebSocket(wsUrl);
    wsRef.current = ws;
    ws.onopen = () => {
      console.debug('[WebSocket] Connected to download queue (MediaDetails)');
    };
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'download_queue_update' && Array.isArray(msg.queue)) {
          setExtras(prev => prev.map(ex => {
            const found = msg.queue.find(q => q.MediaId == media.id && q.YouTubeID === ex.YoutubeId);
            if (found && found.Status) {
              // Only show toast if status transitions to 'failed' or 'rejected'
              if ((found.Status === 'failed' || found.Status === 'rejected') &&
                  (found.reason || found.Reason) &&
                  ex.Status !== found.Status) {
                setError(found.reason || found.Reason);
              }
              return {
                ...ex,
                Status: found.Status,
                reason: found.reason || found.Reason,
                Reason: found.reason || found.Reason,
              };
            }
            return ex;
          }));
        }
      } catch (err) {
        console.debug('[WebSocket] Error parsing message', err);
      }
    };
    ws.onerror = (e) => {
      console.debug('[WebSocket] Error', e);
    };
    ws.onclose = () => {
      console.debug('[WebSocket] Closed (MediaDetails)');
    };
    return () => {
      ws.close();
    };
  }, [media]);

  useEffect(() => {
    if (showModal && modalMsg) {
      const timer = setTimeout(() => {
        setShowModal(false);
        setModalMsg('');
      }, 3500);
      return () => clearTimeout(timer);
    }
  }, [showModal, modalMsg]);

  if (loading) return <div>Loading media details...</div>;
  if (!media) {
    return (
      <div>
        Media not found
        <pre style={{ background: '#eee', color: '#222', padding: 8, marginTop: 12, fontSize: 13 }}>
          Debug info:
          id: {String(id)}
          mediaItems.length: {mediaItems ? mediaItems.length : 'undefined'}
          mediaItems: {JSON.stringify(mediaItems, null, 2)}
        </pre>
      </div>
    );
  }

  const handleSearchExtras = async () => {
    setSearchLoading(true);
    setError('');
    try {
      const api = await import('../api');
      const res = await api.getExtras({ mediaType, id: media.id });
      setExtras(res.extras || []);
    } catch {
      setError('Failed to fetch extras');
    } finally {
      setSearchLoading(false);
    }
  };

  // (removed duplicate declaration; merged version below)

  // Helper to convert YouTube search results to extras format for Trailers
  function ytResultsToExtras(ytResults) {
    return ytResults.map(item => ({
      YoutubeId: item.id?.videoId || '',
      ExtraType: 'Trailers',
      ExtraTitle: item.snippet?.title || 'YouTube Trailer',
      Status: '', // Not downloaded yet
      Thumb: item.snippet?.thumbnails?.medium?.url || '',
      ChannelTitle: item.snippet?.channelTitle || '',
      PublishedAt: item.snippet?.publishedAt || '',
      Description: item.snippet?.description || '',
      reason: '',
      Reason: '',
      Source: 'YouTubeSearch',
      // Add all fields that ExtraCard expects, with safe defaults
      Downloaded: false,
      Exists: false,
      // ...add more if needed
    })).filter(e => e.YoutubeId);
  }

  // Group extras by type, merging YouTube search results into 'Trailers'
  const extrasByType = extras.reduce((acc, extra) => {
    const type = extra.ExtraType || 'Other';
    if (!acc[type]) acc[type] = [];
    acc[type].push(extra);
    return acc;
  }, {});

  // Merge YouTube search results into 'Trailers', but always prefer backend extras for same YoutubeId
  if (ytResults.length > 0) {
    const ytExtras = ytResultsToExtras(ytResults);
    const existing = extrasByType['Trailers'] || [];
    // Build a map for quick lookup
    const existingMap = Object.fromEntries(existing.map(e => [e.YoutubeId, e]));
    // Start with all backend extras
    const all = [...existing];
    // Add only search results not present in backend
    ytExtras.forEach(yt => {
      if (!existingMap[yt.YoutubeId]) {
        all.push(yt);
      }
    });
    extrasByType['Trailers'] = all;
  }

  return (
    <Container style={{ minHeight: '100vh', background: darkMode ? '#18181b' : '#f7f8fa', fontFamily: 'Roboto, Arial, sans-serif', padding: 0 }}>
      {/* Floating Modal for Download Error */}
      {showModal && (
        <div style={{
          position: 'fixed',
          top: 24,
          left: '50%',
          transform: 'translateX(-50%)',
          background: '#ef4444',
          color: '#fff',
          padding: '12px 32px',
          borderRadius: 8,
          boxShadow: '0 2px 12px rgba(0,0,0,0.18)',
          zIndex: 9999,
          fontWeight: 500,
          fontSize: 16,
          minWidth: 260,
          textAlign: 'center',
        }}>
          {modalMsg}
        </div>
      )}
  <MediaInfoLane
    media={{ ...media, mediaType }}
    searchLoading={searchLoading}
    handleSearchExtras={handleSearchExtras}
    setError={setError}
    ytResults={ytResults}
    setYtResults={setYtResults}
  />
      <div style={{ marginTop: '4.5rem' }}>
        <MediaCard media={media} mediaType={mediaType} darkMode={darkMode} error={error} />
      </div>
      <Toast message={error} onClose={() => setError('')} darkMode={darkMode} />
      {/* Grouped extras by type, with 'Trailers' first */}
      {Object.keys(extrasByType).length > 0 && (
        <div style={{ width: '100%', background: darkMode ? '#23232a' : '#f3e8ff', overflow: 'hidden', padding: '24px 10px', margin: 0 }}>
          <ExtrasList
            extrasByType={extrasByType}
            darkMode={darkMode}
            media={media}
            mediaType={mediaType}
            setExtras={setExtras}
            setModalMsg={setModalMsg}
            setShowModal={setShowModal}
            YoutubeEmbed={YoutubeEmbed}
            setYoutubeModal={setYoutubeModal}
          />
        </div>
      )}
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
