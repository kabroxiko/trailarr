import React from 'react';
import IconButton from './IconButton.jsx';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faBookmark } from '@fortawesome/free-regular-svg-icons';

export default function MediaCard({ media, mediaType, darkMode, error }) {
  if (!media) return null;

  let background;
  if (mediaType === 'tv') {
    background = `url(/mediacover/Series/${media.id}/fanart-1280.jpg) center center/cover no-repeat`;
  } else {
    background = `url(/mediacover/Movies/${media.id}/fanart-1280.jpg) center center/cover no-repeat`;
  }

  return (
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
          onError={e => { e.target.onerror = null; e.target.src = '/logo.svg'; }}
        />
      </div>
      <div style={{ flex: 1, zIndex: 2, display: 'flex', flexDirection: 'column', justifyContent: 'flex-start', height: '100%', marginLeft: 32, marginTop: 32 }}>
        <h2 style={{ color: '#fff', margin: 0, fontSize: 32, fontWeight: 600, textShadow: '0 1px 2px #000', letterSpacing: 0.2, textAlign: 'left', display: 'flex', alignItems: 'center', gap: 8 }}>
          <IconButton icon={<FontAwesomeIcon icon={faBookmark} color="#eee" style={{ marginLeft: -10 }} />} disabled style={{ background: 'none', border: 'none', padding: 0, margin: 0 }} />
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
  );
}
