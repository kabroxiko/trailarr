import React, { useState } from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faSearch, faHandPointer } from '@fortawesome/free-solid-svg-icons';
import { searchYoutube } from '../api.youtube';

export default function MediaInfoLane({ media, searchLoading, handleSearchExtras, setError, ytResults, setYtResults }) {
  const [ytLoading, setYtLoading] = useState(false);

  const handleManualSearch = async () => {
    if (!media) return;
    if (!media.mediaType || !media.id) {
      console.warn('YouTube search: missing mediaType or id', media);
      setError && setError('Missing media info for YouTube search');
      return;
    }
    setYtLoading(true);
    setError && setError('');
    setYtResults([]);
    try {
      const res = await searchYoutube({ mediaType: media.mediaType, mediaId: media.id });
      if (Array.isArray(res.items)) setYtResults(res.items);
      else setYtResults([]);
    } catch (e) {
      setError && setError('YouTube search failed');
    } finally {
      setYtLoading(false);
    }
  };

  return (
    <div style={{
      position: 'absolute',
      top: 0,
      left: 0,
      width: '100%',
      background: 'var(--save-lane-bg, #f3f4f6)',
      color: 'var(--save-lane-text, #222)',
      padding: '0.7rem 2rem',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'flex-start',
      gap: '0.7rem',
      zIndex: 10,
      boxShadow: '0 2px 8px #0001',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
        <button
          onClick={handleSearchExtras}
          disabled={searchLoading}
          style={{
            background: 'none',
            color: 'var(--save-lane-text, #222)',
            border: 'none',
            padding: '0.3rem 1rem',
            cursor: searchLoading ? 'not-allowed' : 'pointer',
            opacity: searchLoading ? 0.7 : 1,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '0.2rem',
          }}
        >
          <FontAwesomeIcon icon={faSearch} style={{ fontSize: 22, color: 'var(--save-lane-text, #222)' }} />
          <span style={{ fontWeight: 500, fontSize: '0.85em', color: 'var(--save-lane-text, #222)', marginTop: 2, display: 'flex', flexDirection: 'column', alignItems: 'center', lineHeight: 1.1 }}>
            <span>Search</span>
            <span>Extras</span>
          </span>
        </button>
        <button
          onClick={handleManualSearch}
          disabled={ytLoading}
          style={{
            background: 'none',
            color: 'var(--save-lane-text, #222)',
            border: 'none',
            padding: '0.3rem 1rem',
            cursor: ytLoading ? 'not-allowed' : 'pointer',
            opacity: ytLoading ? 0.7 : 1,
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            gap: '0.2rem',
          }}
        >
          <FontAwesomeIcon icon={faHandPointer} style={{ fontSize: 22, color: 'var(--save-lane-text, #222)' }} />
          <span style={{ fontWeight: 500, fontSize: '0.85em', color: 'var(--save-lane-text, #222)', marginTop: 2, display: 'flex', flexDirection: 'column', alignItems: 'center', lineHeight: 1.1 }}>
            <span>{ytLoading ? 'Searching...' : 'Manual'}</span>
            <span>Search</span>
          </span>
        </button>
      </div>
      {/* YouTube search results are now shown as cards in the Trailers group, not as a text list here. */}
    </div>
  );
}
