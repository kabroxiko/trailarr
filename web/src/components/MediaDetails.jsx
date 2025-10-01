import React, { useState, useEffect } from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faBookmark } from '@fortawesome/free-solid-svg-icons';
import { useParams } from 'react-router-dom';
import { searchExtras } from '../api';

export default function MediaDetails({ mediaItems, loading, mediaType }) {
  const { id } = useParams();
  const media = mediaItems.find(m => String(m.id) === id);
  const [extras, setExtras] = useState([]);
  const [existingExtras, setExistingExtras] = useState([]);
  const [searchLoading, setSearchLoading] = useState(false);
  const [error, setError] = useState('');
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
    searchExtras({ mediaType, id: media.id })
      .then(res => {
        setExtras(res.extras || []);
        if (media.path) {
          let paramName = mediaType === 'tv' ? 'seriesPath' : 'moviePath';
          fetch(`/api/extras/existing?${paramName}=${encodeURIComponent(media.path)}`)
            .then(r => r.json())
            .then(data => setExistingExtras(data.existing || []))
            .catch(() => setExistingExtras([]));
        }
      })
      .catch(() => setError('Failed to search extras'))
      .finally(() => setSearchLoading(false));
  }, [media, mediaType]);

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
      const res = await searchExtras({ mediaType, id: media.id });
      setExtras(res.extras || []);
    } catch (e) {
      setError('Failed to search extras');
    } finally {
      setSearchLoading(false);
    }
  };

  let background;
  if (mediaType === 'tv') {
    background = `url(/api/sonarr/banner/${media.id}) center center/cover no-repeat`;
  } else {
    background = `url(/api/radarr/banner/${media.id}) center center/cover no-repeat`;
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
        minHeight: 210,
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
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
        <div style={{ minWidth: 150, zIndex: 2, display: 'flex', justifyContent: 'flex-start', alignItems: 'center', height: '100%', padding: '0 0 0 32px' }}>
          <img
            src={mediaType === 'tv'
              ? `/api/sonarr/poster/${media.id}`
              : `/api/radarr/poster/${media.id}`}
            style={{ width: 120, height: 180, objectFit: 'cover', borderRadius: 2, background: '#222', boxShadow: '0 1px 4px rgba(0,0,0,0.18)' }}
            onError={e => { e.target.onerror = null; e.target.src = 'https://via.placeholder.com/120x180?text=No+Poster'; }}
          />
        </div>
        <div style={{ flex: 1, zIndex: 2, display: 'flex', flexDirection: 'column', justifyContent: 'center', height: '100%', marginLeft: 32 }}>
          <h2 style={{ color: '#fff', margin: 0, fontSize: 22, fontWeight: 500, textShadow: '0 1px 2px #000', letterSpacing: 0.2, textAlign: 'left', display: 'flex', alignItems: 'center', gap: 8 }}>
            <FontAwesomeIcon icon={faBookmark} color="#eee" style={{ marginRight: 8 }} />
            {media.title}
          </h2>
          <div style={{ marginBottom: 6, color: '#e5e7eb', textAlign: 'left', fontSize: 13, textShadow: '0 1px 2px #000' }}>{media.year} &bull; {media.path}</div>
          {error && <div style={{ color: 'red', marginBottom: 8 }}>{error}</div>}
        </div>
      </div>
      {extras.length > 0 && (
        <div style={{ width: '100%', background: darkMode ? '#23232a' : '#f3e8ff', overflow: 'hidden', padding: 0, margin: 0 }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', background: 'transparent' }}>
            <thead>
              <tr style={{ height: 32 }}>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>Type</th>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>Title</th>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>URL</th>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>Download</th>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>Status</th>
              </tr>
            </thead>
            <tbody>
              {extras.map((extra, idx) => {
                const baseTitle = extra.title || String(extra);
                const totalCount = extras.filter(e => (e.title || String(e)) === baseTitle).length;
                const displayTitle = totalCount > 1 ? `${baseTitle} (${extras.slice(0, idx + 1).filter(e => (e.title || String(e)) === baseTitle).length})` : baseTitle;
                let youtubeID = '';
                if (extra.url) {
                  if (extra.url.includes('youtube.com/watch?v=')) {
                    youtubeID = extra.url.split('v=')[1]?.split('&')[0] || '';
                  } else if (extra.url.includes('youtu.be/')) {
                    youtubeID = extra.url.split('youtu.be/')[1]?.split(/[?&]/)[0] || '';
                  }
                }
                const exists = existingExtras.some(e => e.type === extra.type && e.title === extra.title && e.youtube_id === youtubeID);
                return (
                  <tr key={idx} style={{ height: 32, background: exists ? (darkMode ? '#1e293b' : '#e0e7ff') : undefined }}>
                    <td style={{ padding: '0.5em 1em', textAlign: 'left', color: darkMode ? '#e5e7eb' : '#222', fontSize: 13 }}>{extra.type || ''}</td>
                    <td style={{ padding: '0.5em 1em', textAlign: 'left', color: darkMode ? '#e5e7eb' : '#222', fontSize: 13 }}>{displayTitle}</td>
                    <td style={{ padding: '0.5em 1em', textAlign: 'left', color: darkMode ? '#e5e7eb' : '#6d28d9', fontSize: 13 }}>
                      {extra.url ? (
                        <a href={extra.url} target="_blank" rel="noopener noreferrer" style={{ color: darkMode ? '#e5e7eb' : '#6d28d9', textDecoration: 'underline', fontSize: 13 }}>Link</a>
                      ) : null}
                    </td>
                    <td style={{ padding: '0.5em 1em', textAlign: 'left' }}>
                      {extra.url && (extra.url.includes('youtube.com/watch?v=') || extra.url.includes('youtu.be/')) ? (
                        <button
                          style={{ background: exists ? '#888' : '#a855f7', color: '#fff', border: 'none', borderRadius: 4, padding: '0.25em 0.75em', cursor: exists ? 'not-allowed' : 'pointer', fontWeight: 'bold', fontSize: 13 }}
                          disabled={exists}
                          onClick={async () => {
                            if (exists) return;
                            try {
                              const res = await fetch(`/api/extras/download`, {
                                method: 'POST',
                                headers: { 'Content-Type': 'application/json' },
                                body: JSON.stringify({
                                  moviePath: media.path,
                                  extraType: extra.type,
                                  extraTitle: extra.title,
                                  url: typeof extra.url === 'string' ? extra.url : (extra.url && extra.url.url ? extra.url.url : '')
                                })
                              });
                              if (res.ok) {
                                setExistingExtras(prev => [...prev, { type: extra.type, title: extra.title, youtube_id: youtubeID }]);
                              } else {
                                alert('Download failed');
                              }
                            } catch (e) {
                              alert('Download failed: ' + (e.message || e));
                            }
                          }}
                        >Download</button>
                      ) : null}
                    </td>
                    <td style={{ padding: '0.5em 1em', textAlign: 'left', color: exists ? '#22c55e' : '#ef4444', fontWeight: 'bold', fontSize: 13 }}>{exists ? 'Downloaded' : 'Not downloaded'}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}
