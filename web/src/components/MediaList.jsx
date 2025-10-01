import React from 'react';
import { Link } from 'react-router-dom';

export default function MediaList({ items, darkMode, type }) {
  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 1fr))',
        gap: '2rem 1.5rem',
        padding: '1.5rem 1rem',
        width: '100%',
        boxSizing: 'border-box',
      }}
    >
      {items.map((item) => (
        <div
          key={item.id + '-' + type}
          style={{
            background: darkMode ? '#23232a' : '#fff',
            borderRadius: 12,
            boxShadow: darkMode ? '0 2px 8px #18181b' : '0 2px 8px #e5e7eb',
            padding: '0.75rem',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            height: 'auto',
            transition: 'box-shadow 0.2s',
            border: darkMode ? '1px solid #333' : '1px solid #eee',
            maxHeight: 370,
            overflow: 'hidden',
          }}
        >
          <Link
            to={type === 'series' ? `/series/${item.id}` : `/movies/${item.id}`}
            style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', textDecoration: 'none' }}
          >
            <img
              key={item.id + '-' + type}
              src={type === 'series' ? `/api/sonarr/poster/${item.id}` : `/api/radarr/poster/${item.id}`}
              width={180}
              height={240}
              loading="lazy"
              style={{
                width: 180,
                height: 240,
                objectFit: 'cover',
                borderRadius: 8,
                background: '#222',
                boxShadow: '0 2px 8px rgba(0,0,0,0.18)',
                marginBottom: 8,
                display: 'block',
              }}
              onError={e => { e.target.onerror = null; e.target.src = 'https://via.placeholder.com/180x270?text=No+Poster'; }}
              alt={item.title}
            />
            <div style={{ color: darkMode ? '#e5e7eb' : '#222', fontSize: 14, marginBottom: 2, opacity: 0.85 }}>{item.year ? item.year : (item.airDate || '')}</div>
            <div
              style={{
                color: darkMode ? '#fff' : '#222',
                fontWeight: 600,
                fontSize: 16,
                textAlign: 'center',
                marginBottom: 2,
                width: '100%',
                maxWidth: 160,
                wordBreak: 'break-word',
              }}
              title={item.title}
            >
              {item.title}
            </div>
          </Link>
        </div>
      ))}
    </div>
  );
}
