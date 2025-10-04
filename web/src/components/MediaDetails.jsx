import React, { useState, useEffect } from 'react';
import MediaInfoLane from './MediaInfoLane.jsx';
import MediaCard from './MediaCard.jsx';
import ExtrasList from './ExtrasList';
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
  }, [videoId]);
  return (
    <div style={{ width: '100%', height: '100%', position: 'relative', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
      {loading && <Spinner />}
      <iframe
        src={`https://www.youtube-nocookie.com/embed/${videoId}?autoplay=1&rel=0&modestbranding=1`}
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
    console.log('[MediaDetails] mediaType:', mediaType, 'id:', media.id);
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
      const res = await getExtras({ mediaType, id: media.id });
      setExtras(res.extras || []);
    } catch (e) {
      setError('Failed to fetch extras');
    } finally {
      setSearchLoading(false);
    }
  };

  // Group extras by type
  const extrasByType = extras.reduce((acc, extra) => {
    const type = extra.Type || 'Other';
    if (!acc[type]) acc[type] = [];
    acc[type].push(extra);
    return acc;
  }, {});

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      minHeight: '100vh',
      background: darkMode ? '#18181b' : '#f7f8fa',
      fontFamily: 'Roboto, Arial, sans-serif',
      margin: 0,
      padding: 0,
      width: '100%',
      boxSizing: 'border-box',
    }}>
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
      <MediaInfoLane searchLoading={searchLoading} handleSearchExtras={handleSearchExtras} />
      <MediaCard media={media} mediaType={mediaType} darkMode={darkMode} error={error} />
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
            youtubeModal={youtubeModal}
            setYoutubeModal={setYoutubeModal}
            YoutubeEmbed={YoutubeEmbed}
          />
        </div>
      )}
    </div>
  );
}
