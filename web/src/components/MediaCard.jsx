import React from 'react';

// Compact MediaCard for use in MediaList (tiles)
export default function MediaCard({ media, mediaType, darkMode = false }) {
  if (!media) return null;
  const poster = mediaType === 'series' ? `/mediacover/Series/${media.id}/poster-500.jpg` : `/mediacover/Movies/${media.id}/poster-500.jpg`;
  return (
    <div style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
      <div style={{ width: 200, height: 300, position: 'relative', display: 'flex', alignItems: 'center', justifyContent: 'center', background: darkMode ? '#222' : '#fff', borderRadius: 8, overflow: 'hidden', border: darkMode ? '1px solid #333' : '1px solid #eee' }}>
        <img
          src={poster}
          width={200}
          height={300}
          loading="lazy"
          style={{ width: 200, height: 300, objectFit: 'cover', borderRadius: 8, display: 'block' }}
          onError={e => { e.target.onerror = null; e.target.src = '/logo.svg'; }}
          alt={media.title}
        />
      </div>
      <div style={{ marginTop: 8, textAlign: 'center', width: '100%', maxWidth: 180 }} title={media.title}>
        <div style={{ color: darkMode ? '#fff' : '#222', fontWeight: 600, fontSize: 14 }}>{media.title}</div>
        <div style={{ color: darkMode ? '#ddd' : '#666', fontSize: 12 }}>{media.year || media.airDate || ''}</div>
      </div>
    </div>
  );
}
