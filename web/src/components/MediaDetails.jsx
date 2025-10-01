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
    searchExtras(media.title)
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
      const res = await searchExtras(media.title);
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
      {/* ...extras table and other details... */}
    </div>
  );
}
