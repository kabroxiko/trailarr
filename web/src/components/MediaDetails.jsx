import React, { useState, useEffect } from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faBookmark } from '@fortawesome/free-solid-svg-icons';
import { useParams } from 'react-router-dom';
import { getExtras } from '../api';

export default function MediaDetails({ mediaItems, loading, mediaType }) {
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
    getExtras({ mediaType, id: media.id })
      .then(res => {
        setExtras(res.extras || []);
      })
      .catch(() => setError('Failed to fetch extras'))
      .finally(() => setSearchLoading(false));
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

  let background;
  if (mediaType === 'tv') {
    background = `url(/mediacover/Series/${media.id}/fanart-1280.jpg) center center/cover no-repeat`;
  } else {
    background = `url(/mediacover/Movies/${media.id}/fanart-1280.jpg) center center/cover no-repeat`;
  }

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
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-start', margin: '0px 0 0 0', padding: 0, width: '100%' }}>
        <div
          style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer', fontWeight: 'bold', color: '#e5e7eb', fontSize: 18 }}
          onClick={handleSearchExtras}
        >
          <span style={{ fontSize: 20, display: 'flex', alignItems: 'center' }}>
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
              <circle cx="9" cy="9" r="7" stroke="#e5e7eb" strokeWidth="2" />
              <line x1="15" y1="15" x2="19" y2="19" stroke="#e5e7eb" strokeWidth="2" strokeLinecap="round" />
            </svg>
          </span>
          <span>{searchLoading ? 'Searching...' : 'Search'}</span>
        </div>
      </div>
      <div style={{
        width: '100%',
        position: 'relative',
        background,
        minHeight: 420,
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'flex-start',
        boxSizing: 'border-box',
        padding: 0,
      }}>
        <div style={{
          position: 'absolute',
          top: 0,
          left: 0,
          width: '100%',
          height: '100%',
          background: 'rgba(0,0,0,0.55)',
          zIndex: 1,
        }} />
        <div style={{ minWidth: 150, zIndex: 2, display: 'flex', justifyContent: 'flex-start', alignItems: 'flex-start', height: '100%', padding: '32px 32px' }}>
          <img
            src={mediaType === 'tv'
              ? `/mediacover/Series/${media.id}/poster-500.jpg`
              : `/mediacover/Movies/${media.id}/poster-500.jpg`}
            style={{ height: 370, objectFit: 'cover', borderRadius: 4, background: '#222', boxShadow: '0 2px 8px rgba(0,0,0,0.22)' }}
            onError={e => { e.target.onerror = null; e.target.src = 'https://via.placeholder.com/180x270?text=No+Poster'; }}
          />
        </div>
        <div style={{ flex: 1, zIndex: 2, display: 'flex', flexDirection: 'column', justifyContent: 'flex-start', height: '100%', marginLeft: 32, marginTop: 32 }}>
          <h2 style={{ color: '#fff', margin: 0, fontSize: 32, fontWeight: 600, textShadow: '0 1px 2px #000', letterSpacing: 0.2, textAlign: 'left', display: 'flex', alignItems: 'center', gap: 8 }}>
            <FontAwesomeIcon icon={faBookmark} color="#eee" style={{ marginLeft: -10 }} />
            {media.title}
          </h2>
          {media.overview && (
            <div style={{ color: '#e5e7eb', fontSize: 15, margin: '10px 0 6px 0', textShadow: '0 1px 2px #000', textAlign: 'left', lineHeight: 1.5, maxWidth: 700 }}>
              {media.overview}
            </div>
          )}
          <div style={{ marginBottom: 6, color: '#e5e7eb', textAlign: 'left', fontSize: 13, textShadow: '0 1px 2px #000' }}>{media.year} &bull; {media.path}</div>
          {error && <div style={{ color: 'red', marginBottom: 8 }}>{error}</div>}
        </div>
      </div>
      {extras.length > 0 && (
        <div style={{ width: '100%', background: darkMode ? '#23232a' : '#f3e8ff', overflow: 'hidden', padding: '24px 10px', margin: 0 }}>
          <div style={{
            display: 'grid',
            gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 0px))',
            gap: '32px',
            justifyItems: 'start',
            alignItems: 'start',
            width: '100%',
            justifyContent: 'start',
          }}>
            {extras.map((extra, idx) => {
              const baseTitle = extra.title || String(extra);
              const totalCount = extras.filter(e => (e.title || String(e)) === baseTitle).length;
              let displayTitle = totalCount > 1 ? `${baseTitle} (${extras.slice(0, idx + 1).filter(e => (e.title || String(e)) === baseTitle).length})` : baseTitle;
              // Truncate and add ellipsis if too long
              const maxLen = 40;
              if (displayTitle.length > maxLen) {
                displayTitle = displayTitle.slice(0, maxLen - 3) + '...';
              }
              let youtubeID = '';
              if (extra.url) {
                if (extra.url.includes('youtube.com/watch?v=')) {
                  youtubeID = extra.url.split('v=')[1]?.split('&')[0] || '';
                } else if (extra.url.includes('youtu.be/')) {
                  youtubeID = extra.url.split('youtu.be/')[1]?.split(/[?&]/)[0] || '';
                }
              }
              // Use YouTube thumbnail if available
              let posterUrl = extra.poster;
              if (!posterUrl && youtubeID) {
                posterUrl = `https://img.youtube.com/vi/${youtubeID}/hqdefault.jpg`;
              }
              // Adjust font size for long titles
              let titleFontSize = 16;
              if (displayTitle.length > 22) titleFontSize = 14;
              if (displayTitle.length > 32) titleFontSize = 12;
              const downloaded = extra.downloaded === 'true';

              // Extracted button click handler
              const handleDownloadClick = async () => {
                if (downloaded) return;
                try {
                  const getExtraUrl = extra => typeof extra.url === 'string' ? extra.url : extra.url?.url ?? '';
                  const res = await fetch(`/api/extras/download`, {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                      moviePath: media.path,
                      extraType: extra.type,
                      extraTitle: extra.title,
                      url: getExtraUrl(extra)
                    })
                  });
                  if (res.ok) {
                    setExtras(prev => prev.map((e, i) => i === idx ? { ...e, downloaded: 'true' } : e));
                  } else {
                    const data = await res.json();
                    let msg = data?.error || 'Download failed';
                    if (msg.includes('UNPLAYABLE') || msg.includes('no se encuentra disponible')) {
                      msg = 'This YouTube video is unavailable and cannot be downloaded.';
                    }
                    setModalMsg(msg);
                    setShowModal(true);
                  }
                } catch (e) {
                  let msg = (e.message || e);
                  if (msg.includes('UNPLAYABLE') || msg.includes('no se encuentra disponible')) {
                    msg = 'This YouTube video is unavailable and cannot be downloaded.';
                  }
                  setModalMsg(msg);
                  setShowModal(true);
                }
              };

              return (
                <div key={idx} style={{
                  width: 180,
                  height: 280,
                  background: darkMode ? '#18181b' : '#fff',
                  borderRadius: 12,
                  boxShadow: darkMode ? '0 2px 12px rgba(0,0,0,0.22)' : '0 2px 12px rgba(0,0,0,0.10)',
                  overflow: 'hidden',
                  display: 'flex',
                  flexDirection: 'column',
                  alignItems: 'center',
                  padding: '0 0 18px 0',
                  position: 'relative',
                  border: downloaded ? '2px solid #22c55e' : '2px solid transparent',
                }}>
                  <div style={{ width: '100%', background: '#222', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                    {posterUrl ? (
                      <img src={posterUrl} alt={displayTitle} style={{ width: '100%', height: 'auto', objectFit: 'contain', maxHeight: 260, background: '#222' }} />
                    ) : (
                      <div style={{ color: '#fff', fontSize: 18, textAlign: 'center', padding: 12 }}>No Image</div>
                    )}
                  </div>
                  <div style={{ width: '100%', padding: '12px 10px 0 10px', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                    <div style={{ fontWeight: 600, fontSize: titleFontSize, color: darkMode ? '#e5e7eb' : '#222', textAlign: 'center', marginBottom: 4, height: 50, display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden', width: '100%' }}>{displayTitle}</div>
                    <div style={{ fontSize: 13, color: '#888', marginBottom: 2 }}>{extra.year || ''}</div>
                    <div style={{ fontSize: 13, color: downloaded ? '#22c55e' : '#ef4444', fontWeight: 'bold', marginBottom: 6 }}>{downloaded ? 'Downloaded' : 'Not downloaded'}</div>
                    {extra.url ? (
                      <a href={extra.url} target="_blank" rel="noopener noreferrer" style={{ color: darkMode ? '#e5e7eb' : '#6d28d9', textDecoration: 'underline', fontSize: 13, marginBottom: 8 }}>View</a>
                    ) : null}
                    {extra.url && (extra.url.includes('youtube.com/watch?v=') || extra.url.includes('youtu.be/')) ? (
                      <button
                        style={{ background: downloaded ? '#888' : '#a855f7', color: '#fff', border: 'none', borderRadius: 4, padding: '0.25em 0.75em', cursor: downloaded ? 'not-allowed' : 'pointer', fontWeight: 'bold', fontSize: 13, marginTop: 4 }}
                        disabled={downloaded}
                        onClick={handleDownloadClick}
                      >Download</button>
                    ) : null}
                  </div>
                </div>
              );
            })}
          </div>
        </div>
      )}
    </div>
  );
}
