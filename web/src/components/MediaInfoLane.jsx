import React, { useState } from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faSearch } from '@fortawesome/free-solid-svg-icons';
import { searchYoutubeStream } from '../api.youtube.sse';

export default function MediaInfoLane({ media, setError, setYtResults, darkMode }) {
  const [ytLoading, setYtLoading] = useState(false);
  const handleManualSearch = () => {
    if (!media) return;
    if (!media.mediaType || !media.id) {
      console.warn('YouTube search: missing mediaType or id', media);
      setError && setError('Missing media info for YouTube search');
      return;
    }
    setYtLoading(true);
    setError && setError('');
    setYtResults([]);
    let results = [];
    searchYoutubeStream({
      mediaType: media.mediaType,
      mediaId: media.id,
      onResult: (item) => {
        results.push(item);
        setYtResults([...results]);
      },
      onDone: () => {
        setYtLoading(false);
      },
      onError: () => {
        setError && setError('YouTube search failed');
        setYtLoading(false);
      }
    });
  };

  const laneBg = darkMode ? '#23232a' : 'var(--save-lane-bg, #f3f4f6)';
  const laneText = darkMode ? '#e5e7eb' : 'var(--save-lane-text, #222)';
  return (
    <div style={{
      position: 'absolute',
      top: 0,
      left: 0,
      width: '100%',
      background: laneBg,
      color: laneText,
      padding: '0.7rem 2rem',
      display: 'flex',
      flexDirection: 'column',
      alignItems: 'flex-start',
      gap: '0.7rem',
      zIndex: 10,
      boxShadow: darkMode ? '0 2px 8px #0008' : '0 2px 8px #0001',
      borderBottom: darkMode ? '1.5px solid #444' : '1.5px solid #e5e7eb',
      transition: 'background 0.2s, color 0.2s',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: '1rem' }}>
        <button
          onClick={handleManualSearch}
          disabled={ytLoading}
          style={{
            background: 'none',
            color: laneText,
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
          <FontAwesomeIcon icon={faSearch} style={{ fontSize: 22, color: laneText }} />
          <span style={{ fontWeight: 500, fontSize: '0.85em', color: laneText, marginTop: 2, display: 'flex', flexDirection: 'column', alignItems: 'center', lineHeight: 1.1 }}>
            <span>{ytLoading ? 'Searching...' : 'Search Trailers'}</span>
          </span>
        </button>
      </div>
    </div>
  );
}
