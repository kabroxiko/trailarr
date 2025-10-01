import React, { useState, useEffect } from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faTrashCan, faBookmark, faCheckSquare } from '@fortawesome/free-regular-svg-icons';
import { faPlay, faDownload } from '@fortawesome/free-solid-svg-icons';
import { useParams } from 'react-router-dom';
import { getExtras } from '../api';

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

  // Group extras by type
  const extrasByType = extras.reduce((acc, extra) => {
    const type = extra.type || 'Other';
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
      {/* Grouped extras by type, with 'Trailers' first */}
      {Object.keys(extrasByType).length > 0 && (
        <div style={{ width: '100%', background: darkMode ? '#23232a' : '#f3e8ff', overflow: 'hidden', padding: '24px 10px', margin: 0 }}>
          {/* Render 'Trailers' group first if present */}
          {extrasByType['Trailers'] && (
            <div key="Trailers" style={{ marginBottom: 32 }}>
              <h3 style={{
                color: '#111',
                fontSize: 20,
                fontWeight: 700,
                margin: '0 0 18px 8px',
                textTransform: 'capitalize',
                letterSpacing: 0.5,
                textAlign: 'left',
              }}>Trailers</h3>
              <div style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 0px))',
                gap: '32px',
                justifyItems: 'start',
                alignItems: 'start',
                width: '100%',
                justifyContent: 'start',
              }}>
                {extrasByType['Trailers'].map((extra, idx) => {
                  // ...existing code for rendering extras card...
                  // Copy the card rendering logic from below
                  const baseTitle = extra.title || String(extra);
                  const totalCount = extrasByType['Trailers'].filter(e => (e.title || String(e)) === baseTitle).length;
                  let displayTitle = totalCount > 1 ? `${baseTitle} (${extrasByType['Trailers'].slice(0, idx + 1).filter(e => (e.title || String(e)) === baseTitle).length})` : baseTitle;
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
                  let posterUrl = extra.poster;
                  if (!posterUrl && youtubeID) {
                    posterUrl = `https://img.youtube.com/vi/${youtubeID}/hqdefault.jpg`;
                  }
                  let titleFontSize = 16;
                  if (displayTitle.length > 22) titleFontSize = 14;
                  if (displayTitle.length > 32) titleFontSize = 12;
                  const downloaded = extra.downloaded === 'true';
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
                        setExtras(prev => prev.map((e) =>
                          e.title === extra.title && e.type === extra.type ? { ...e, downloaded: 'true' } : e
                        ));
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
                      height: 210,
                      background: darkMode ? '#18181b' : '#fff',
                      borderRadius: 12,
                      boxShadow: darkMode ? '0 2px 12px rgba(0,0,0,0.22)' : '0 2px 12px rgba(0,0,0,0.10)',
                      overflow: 'hidden',
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'center',
                      padding: '0 0 0 0',
                      position: 'relative',
                      border: downloaded ? '2px solid #22c55e' : '2px solid transparent',
                    }}>
                      <div style={{ width: '100%', background: '#222', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                        <div style={{position: 'relative', width: '100%'}}>
                          {/* Play (YouTube) icon at center over poster */}
                          {extra.url && (extra.url.includes('youtube.com/watch?v=') || extra.url.includes('youtu.be/')) && (
                            <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', zIndex: 2 }}>
                              <FontAwesomeIcon
                                icon={faPlay}
                                color="#fff"
                                size="lg"
                                style={{ cursor: 'pointer', filter: 'drop-shadow(0 2px 8px #000)' }}
                                title="Play"
                                onClick={() => {
                                  let youtubeID = '';
                                  if (extra.url.includes('youtube.com/watch?v=')) {
                                    youtubeID = extra.url.split('v=')[1]?.split('&')[0] || '';
                                  } else if (extra.url.includes('youtu.be/')) {
                                    youtubeID = extra.url.split('youtu.be/')[1]?.split(/[?&]/)[0] || '';
                                  }
                                  if (youtubeID) setYoutubeModal({ open: true, videoId: youtubeID });
                                }}
                              />
                            </div>
                          )}
                          {posterUrl ? (
                            <img src={posterUrl} alt={displayTitle} style={{ width: '100%', height: 'auto', objectFit: 'contain', maxHeight: 260, background: '#222' }} />
                          ) : (
                            <div style={{ color: '#fff', fontSize: 18, textAlign: 'center', padding: 12 }}>No Image</div>
                          )}
                          {/* Download icon at upper right over poster */}
                          {extra.url && (extra.url.includes('youtube.com/watch?v=') || extra.url.includes('youtu.be/')) && !downloaded && (
                            <div style={{ position: 'absolute', top: 8, right: downloaded ? 36 : 8, zIndex: 2 }}>
                              <FontAwesomeIcon
                                icon={faDownload}
                                color="#fff"
                                size="lg"
                                style={{ cursor: 'pointer' }}
                                title="Download"
                                onClick={handleDownloadClick}
                              />
                            </div>
                          )}
                          {downloaded && (
                            <div style={{ position: 'absolute', top: 8, right: 8, zIndex: 2 }}>
                              <FontAwesomeIcon icon={faCheckSquare} color="#22c55e" size="lg" title="Downloaded" />
                            </div>
                          )}
                          {downloaded && (
                            <div style={{ position: 'absolute', bottom: 8, right: 8, zIndex: 2 }}>
                              <FontAwesomeIcon
                                icon={faTrashCan}
                                color="#ef4444"
                                size="md"
                                style={{ cursor: 'pointer' }}
                                title="Delete"
                                onClick={async () => {
                                  if (!window.confirm('Delete this extra?')) return;
                                  try {
                                    const { deleteExtra } = await import('../api');
                                    const payload = {
                                      mediaType,
                                      mediaId: media.id,
                                      extraType: extra.type,
                                      extraTitle: extra.title
                                    };
                                    await deleteExtra(payload);
                                    setExtras(prev => prev.map((e) =>
                                      e.title === extra.title && e.type === extra.type ? { ...e, downloaded: 'false' } : e
                                    ));
                                  } catch (e) {
                                    let msg = e?.message || e;
                                    if (e?.detail) msg += `\n${e.detail}`;
                                    console.error('Failed to delete extra:', e);
                                    setModalMsg(msg || 'Delete failed');
                                    setShowModal(true);
                                  }
                                }}
                              />
                            </div>
                          )}
                        </div>
                      </div>
                      <div style={{ width: '100%', padding: '12px 10px 0 10px', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                        <div style={{ fontWeight: 600, fontSize: titleFontSize, color: darkMode ? '#e5e7eb' : '#222', textAlign: 'center', marginBottom: 4, height: 50, display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden', width: '100%' }}>{displayTitle}</div>
                        <div style={{ fontSize: 13, color: '#888', marginBottom: 2 }}>{extra.year || ''}</div>
                        <div style={{ width: '100%', display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: 18, position: 'absolute', bottom: 12, left: 0 }}>
                        </div>
                        {youtubeModal.open && (
                          <div className="youtube-modal-backdrop" style={{
                            position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh', background: 'rgba(0,0,0,0.7)', zIndex: 99999, display: 'flex', alignItems: 'center', justifyContent: 'center',
                          }}>
                            <div style={{
                              position: 'relative',
                              background: '#18181b',
                              borderRadius: 12,
                              boxShadow: '0 2px 24px #000',
                              padding: 0,
                              width: '90vw',
                              maxWidth: 800,
                              aspectRatio: '16/9',
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'center',
                              overflow: 'hidden',
                            }}>
                              <button onClick={() => setYoutubeModal({ open: false, videoId: '' })} style={{ position: 'absolute', top: 8, right: 12, background: 'transparent', color: '#fff', border: 'none', fontSize: 28, cursor: 'pointer', zIndex: 2 }} title="Close">×</button>
                              <YoutubeEmbed videoId={youtubeModal.videoId} />
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
          {/* Render other groups except 'Trailers' and 'Others' */}
          {Object.entries(extrasByType)
            .filter(([type]) => type !== 'Trailers' && type !== 'Others')
            .map(([type, typeExtras]) => (
              <div key={type} style={{ marginBottom: 32 }}>
                {/* ...existing code for rendering extras card... */}
                <h3 style={{
                  color: '#111',
                  fontSize: 20,
                  fontWeight: 700,
                  margin: '0 0 18px 8px',
                  textTransform: 'capitalize',
                  letterSpacing: 0.5,
                  textAlign: 'left',
                }}>{type}</h3>
                <div style={{
                  display: 'grid',
                  gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 0px))',
                  gap: '32px',
                  justifyItems: 'start',
                  alignItems: 'start',
                  width: '100%',
                  justifyContent: 'start',
                }}>
                  {typeExtras.map((extra, idx) => {
                    // ...existing code for rendering extras card...
                    const baseTitle = extra.title || String(extra);
                    const totalCount = typeExtras.filter(e => (e.title || String(e)) === baseTitle).length;
                    let displayTitle = totalCount > 1 ? `${baseTitle} (${typeExtras.slice(0, idx + 1).filter(e => (e.title || String(e)) === baseTitle).length})` : baseTitle;
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
                    let posterUrl = extra.poster;
                    if (!posterUrl && youtubeID) {
                      posterUrl = `https://img.youtube.com/vi/${youtubeID}/hqdefault.jpg`;
                    }
                    let titleFontSize = 16;
                    if (displayTitle.length > 22) titleFontSize = 14;
                    if (displayTitle.length > 32) titleFontSize = 12;
                    const downloaded = extra.downloaded === 'true';
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
                          setExtras(prev => prev.map((e, i) => i === idx && e.type === type ? { ...e, downloaded: 'true' } : e));
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
                        height: 210,
                        background: darkMode ? '#18181b' : '#fff',
                        borderRadius: 12,
                        boxShadow: darkMode ? '0 2px 12px rgba(0,0,0,0.22)' : '0 2px 12px rgba(0,0,0,0.10)',
                        overflow: 'hidden',
                        display: 'flex',
                        flexDirection: 'column',
                        alignItems: 'center',
                        padding: '0 0 0 0',
                        position: 'relative',
                        border: downloaded ? '2px solid #22c55e' : '2px solid transparent',
                      }}>
                        <div style={{ width: '100%', background: '#222', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                          <div style={{position: 'relative', width: '100%'}}>
                            {/* Play (YouTube) icon at center over poster */}
                            {extra.url && (extra.url.includes('youtube.com/watch?v=') || extra.url.includes('youtu.be/')) && (
                              <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', zIndex: 2 }}>
                                <FontAwesomeIcon
                                  icon={faPlay}
                                  color="#fff"
                                  size="lg"
                                  style={{ cursor: 'pointer', filter: 'drop-shadow(0 2px 8px #000)' }}
                                  title="Play"
                                  onClick={() => {
                                    let youtubeID = '';
                                    if (extra.url.includes('youtube.com/watch?v=')) {
                                      youtubeID = extra.url.split('v=')[1]?.split('&')[0] || '';
                                    } else if (extra.url.includes('youtu.be/')) {
                                      youtubeID = extra.url.split('youtu.be/')[1]?.split(/[?&]/)[0] || '';
                                    }
                                    if (youtubeID) setYoutubeModal({ open: true, videoId: youtubeID });
                                  }}
                                />
                              </div>
                            )}
                            {posterUrl ? (
                              <img src={posterUrl} alt={displayTitle} style={{ width: '100%', height: 'auto', objectFit: 'contain', maxHeight: 260, background: '#222' }} />
                            ) : (
                              <div style={{ color: '#fff', fontSize: 18, textAlign: 'center', padding: 12 }}>No Image</div>
                            )}
                            {/* Download icon at upper right over poster */}
                            {extra.url && (extra.url.includes('youtube.com/watch?v=') || extra.url.includes('youtu.be/')) && !downloaded && (
                              <div style={{ position: 'absolute', top: 8, right: downloaded ? 36 : 8, zIndex: 2 }}>
                                <FontAwesomeIcon
                                  icon={faDownload}
                                  color="#fff"
                                  size="lg"
                                  style={{ cursor: 'pointer' }}
                                  title="Download"
                                  onClick={handleDownloadClick}
                                />
                              </div>
                            )}
                            {downloaded && (
                              <div style={{ position: 'absolute', top: 8, right: 8, zIndex: 2 }}>
                                <FontAwesomeIcon icon={faCheckSquare} color="#22c55e" size="lg" title="Downloaded" />
                              </div>
                            )}
                            {downloaded && (
                              <div style={{ position: 'absolute', bottom: 8, right: 8, zIndex: 2 }}>
                                <FontAwesomeIcon
                                  icon={faTrashCan}
                                  color="#ef4444"
                                  size="md"
                                  style={{ cursor: 'pointer' }}
                                  title="Delete"
                                  onClick={async () => {
                                    if (!window.confirm('Delete this extra?')) return;
                                    try {
                                      const { deleteExtra } = await import('../api');
                                      const payload = {
                                        mediaType,
                                        mediaId: media.id,
                                        extraType: extra.type,
                                        extraTitle: extra.title
                                      };
                                      await deleteExtra(payload);
                                      setExtras(prev => prev.map((e, i) => i === idx && e.type === type ? { ...e, downloaded: 'false' } : e));
                                    } catch (e) {
                                      let msg = e?.message || e;
                                      if (e?.detail) msg += `\n${e.detail}`;
                                      console.error('Failed to delete extra:', e);
                                      setModalMsg(msg || 'Delete failed');
                                      setShowModal(true);
                                    }
                                  }}
                                />
                              </div>
                            )}
                          </div>
                        </div>
                        <div style={{ width: '100%', padding: '12px 10px 0 10px', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                          <div style={{ fontWeight: 600, fontSize: titleFontSize, color: darkMode ? '#e5e7eb' : '#222', textAlign: 'center', marginBottom: 4, height: 50, display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden', width: '100%' }}>{displayTitle}</div>
                          <div style={{ fontSize: 13, color: '#888', marginBottom: 2 }}>{extra.year || ''}</div>
                          <div style={{ width: '100%', display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: 18, position: 'absolute', bottom: 12, left: 0 }}>
                          </div>
                          {youtubeModal.open && (
                            <div className="youtube-modal-backdrop" style={{
                              position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh', background: 'rgba(0,0,0,0.7)', zIndex: 99999, display: 'flex', alignItems: 'center', justifyContent: 'center',
                            }}>
                              <div style={{
                                position: 'relative',
                                background: '#18181b',
                                borderRadius: 12,
                                boxShadow: '0 2px 24px #000',
                                padding: 0,
                                width: '90vw',
                                maxWidth: 800,
                                aspectRatio: '16/9',
                                display: 'flex',
                                alignItems: 'center',
                                justifyContent: 'center',
                                overflow: 'hidden',
                              }}>
                                <button onClick={() => setYoutubeModal({ open: false, videoId: '' })} style={{ position: 'absolute', top: 8, right: 12, background: 'transparent', color: '#fff', border: 'none', fontSize: 28, cursor: 'pointer', zIndex: 2 }} title="Close">×</button>
                                <YoutubeEmbed videoId={youtubeModal.videoId} />
                              </div>
                            </div>
                          )}
                        </div>
                      </div>
                    );
                  })}
                </div>
              </div>
            ))}
          {/* Render 'Others' group last if present */}
          {extrasByType['Others'] && (
            <div key="Others" style={{ marginBottom: 32 }}>
              <h3 style={{
                color: '#111',
                fontSize: 20,
                fontWeight: 700,
                margin: '0 0 18px 8px',
                textTransform: 'capitalize',
                letterSpacing: 0.5,
                textAlign: 'left',
              }}>Others</h3>
              <div style={{
                display: 'grid',
                gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 0px))',
                gap: '32px',
                justifyItems: 'start',
                alignItems: 'start',
                width: '100%',
                justifyContent: 'start',
              }}>
                {extrasByType['Others'].map((extra, idx) => {
                  // ...existing code for rendering extras card...
                  const baseTitle = extra.title || String(extra);
                  const totalCount = extrasByType['Others'].filter(e => (e.title || String(e)) === baseTitle).length;
                  let displayTitle = totalCount > 1 ? `${baseTitle} (${extrasByType['Others'].slice(0, idx + 1).filter(e => (e.title || String(e)) === baseTitle).length})` : baseTitle;
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
                  let posterUrl = extra.poster;
                  if (!posterUrl && youtubeID) {
                    posterUrl = `https://img.youtube.com/vi/${youtubeID}/hqdefault.jpg`;
                  }
                  let titleFontSize = 16;
                  if (displayTitle.length > 22) titleFontSize = 14;
                  if (displayTitle.length > 32) titleFontSize = 12;
                  const downloaded = extra.downloaded === 'true';
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
                        setExtras(prev => prev.map((e, i) => i === idx && e.type === 'Others' ? { ...e, downloaded: 'true' } : e));
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
                      height: 210,
                      background: darkMode ? '#18181b' : '#fff',
                      borderRadius: 12,
                      boxShadow: darkMode ? '0 2px 12px rgba(0,0,0,0.22)' : '0 2px 12px rgba(0,0,0,0.10)',
                      overflow: 'hidden',
                      display: 'flex',
                      flexDirection: 'column',
                      alignItems: 'center',
                      padding: '0 0 0 0',
                      position: 'relative',
                      border: downloaded ? '2px solid #22c55e' : '2px solid transparent',
                    }}>
                      <div style={{ width: '100%', background: '#222', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                        <div style={{position: 'relative', width: '100%'}}>
                          {/* Play (YouTube) icon at center over poster */}
                          {extra.url && (extra.url.includes('youtube.com/watch?v=') || extra.url.includes('youtu.be/')) && (
                            <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', zIndex: 2 }}>
                              <FontAwesomeIcon
                                icon={faPlay}
                                color="#fff"
                                size="lg"
                                style={{ cursor: 'pointer', filter: 'drop-shadow(0 2px 8px #000)' }}
                                title="Play"
                                onClick={() => {
                                  let youtubeID = '';
                                  if (extra.url.includes('youtube.com/watch?v=')) {
                                    youtubeID = extra.url.split('v=')[1]?.split('&')[0] || '';
                                  } else if (extra.url.includes('youtu.be/')) {
                                    youtubeID = extra.url.split('youtu.be/')[1]?.split(/[?&]/)[0] || '';
                                  }
                                  if (youtubeID) setYoutubeModal({ open: true, videoId: youtubeID });
                                }}
                              />
                            </div>
                          )}
                          {posterUrl ? (
                            <img src={posterUrl} alt={displayTitle} style={{ width: '100%', height: 'auto', objectFit: 'contain', maxHeight: 260, background: '#222' }} />
                          ) : (
                            <div style={{ color: '#fff', fontSize: 18, textAlign: 'center', padding: 12 }}>No Image</div>
                          )}
                          {/* Download icon at upper right over poster */}
                          {extra.url && (extra.url.includes('youtube.com/watch?v=') || extra.url.includes('youtu.be/')) && !downloaded && (
                            <div style={{ position: 'absolute', top: 8, right: downloaded ? 36 : 8, zIndex: 2 }}>
                              <FontAwesomeIcon
                                icon={faDownload}
                                color="#fff"
                                size="lg"
                                style={{ cursor: 'pointer' }}
                                title="Download"
                                onClick={handleDownloadClick}
                              />
                            </div>
                          )}
                          {downloaded && (
                            <div style={{ position: 'absolute', top: 8, right: 8, zIndex: 2 }}>
                              <FontAwesomeIcon icon={faCheckSquare} color="#22c55e" size="lg" title="Downloaded" />
                            </div>
                          )}
                          {downloaded && (
                            <div style={{ position: 'absolute', bottom: 8, right: 8, zIndex: 2 }}>
                              <FontAwesomeIcon
                                icon={faTrashCan}
                                color="#ef4444"
                                size="md"
                                style={{ cursor: 'pointer' }}
                                title="Delete"
                                onClick={async () => {
                                  if (!window.confirm('Delete this extra?')) return;
                                  try {
                                    const { deleteExtra } = await import('../api');
                                    const payload = {
                                      mediaType,
                                      mediaId: media.id,
                                      extraType: extra.type,
                                      extraTitle: extra.title
                                    };
                                    await deleteExtra(payload);
                                    setExtras(prev => prev.map((e, i) => i === idx && e.type === 'Others' ? { ...e, downloaded: 'false' } : e));
                                  } catch (e) {
                                    let msg = e?.message || e;
                                    if (e?.detail) msg += `\n${e.detail}`;
                                    console.error('Failed to delete extra:', e);
                                    setModalMsg(msg || 'Delete failed');
                                    setShowModal(true);
                                  }
                                }}
                              />
                            </div>
                          )}
                        </div>
                      </div>
                      <div style={{ width: '100%', padding: '12px 10px 0 10px', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
                        <div style={{ fontWeight: 600, fontSize: titleFontSize, color: darkMode ? '#e5e7eb' : '#222', textAlign: 'center', marginBottom: 4, height: 50, display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden', width: '100%' }}>{displayTitle}</div>
                        <div style={{ fontSize: 13, color: '#888', marginBottom: 2 }}>{extra.year || ''}</div>
                        <div style={{ width: '100%', display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: 18, position: 'absolute', bottom: 12, left: 0 }}>
                        </div>
                        {youtubeModal.open && (
                          <div className="youtube-modal-backdrop" style={{
                            position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh', background: 'rgba(0,0,0,0.7)', zIndex: 99999, display: 'flex', alignItems: 'center', justifyContent: 'center',
                          }}>
                            <div style={{
                              position: 'relative',
                              background: '#18181b',
                              borderRadius: 12,
                              boxShadow: '0 2px 24px #000',
                              padding: 0,
                              width: '90vw',
                              maxWidth: 800,
                              aspectRatio: '16/9',
                              display: 'flex',
                              alignItems: 'center',
                              justifyContent: 'center',
                              overflow: 'hidden',
                            }}>
                              <button onClick={() => setYoutubeModal({ open: false, videoId: '' })} style={{ position: 'absolute', top: 8, right: 12, background: 'transparent', color: '#fff', border: 'none', fontSize: 28, cursor: 'pointer', zIndex: 2 }} title="Close">×</button>
                              <YoutubeEmbed videoId={youtubeModal.videoId} />
                            </div>
                          </div>
                        )}
                      </div>
                    </div>
                  );
                })}
              </div>
            </div>
          )}
        </div>
      )}
    </div>
  );
}
