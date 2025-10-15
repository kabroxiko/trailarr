import React, { useState } from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faSearch, faHandPointer } from '@fortawesome/free-solid-svg-icons';
import { searchYoutube } from '../api.youtube';

export default function MediaInfoLane({ media, searchLoading, handleSearchExtras, setError }) {
  const [ytLoading, setYtLoading] = useState(false);
  const [ytResults, setYtResults] = useState([]);

  const handleManualSearch = async () => {
    if (!media) return;
    setYtLoading(true);
    setError && setError('');
    setYtResults([]);
    try {
      // Search both title and originalTitle if available
      const queries = [media.title, media.originalTitle].filter(Boolean);
      let allResults = [];
      for (const q of queries) {
        const res = await searchYoutube(q);
        if (Array.isArray(res.items)) allResults = allResults.concat(res.items);
      }
      setYtResults(allResults);
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
      {ytResults.length > 0 && (
        <div style={{ marginTop: 8, width: '100%' }}>
          <div style={{ fontWeight: 600, marginBottom: 4 }}>YouTube Results:</div>
          <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
            {ytResults.map((item, idx) => (
              <li key={item.id?.videoId || idx} style={{ marginBottom: 6 }}>
                <a href={`https://youtube.com/watch?v=${item.id?.videoId}`} target="_blank" rel="noopener noreferrer" style={{ color: '#2563eb', textDecoration: 'underline', fontWeight: 500 }}>
                  {item.snippet?.title || 'Untitled'}
                </a>
                <span style={{ color: '#888', marginLeft: 8, fontSize: 12 }}>{item.snippet?.channelTitle}</span>
              </li>
            ))}
          </ul>
        </div>
      )}
    </div>
  );
}
