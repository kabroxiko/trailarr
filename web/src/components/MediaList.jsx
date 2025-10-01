import React from 'react';
import { Link } from 'react-router-dom';

export default function MediaList({ items, darkMode, type }) {
  return (
    <div
      style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(200px, 0px))',
        gap: '2rem 1.5rem',
        padding: '1.5rem 1rem',
        width: '100%',
        boxSizing: 'border-box',
        justifyContent: 'flex-start',
        gridAutoFlow: 'unset',
      }}
    >
      {items.map((item) => (
        <div
          key={item.id + '-' + type}
          style={{
            background: darkMode ? '#23232a' : '#fff',
            borderRadius: 12,
            boxShadow: darkMode ? '0 2px 8px #18181b' : '0 2px 8px #e5e7eb',
            padding: '0.85rem',
            display: 'flex',
            flexDirection: 'column',
            alignItems: 'center',
            height: 410, // 300 for image + 120 for text/details
            transition: 'box-shadow 0.2s',
            border: darkMode ? '1px solid #333' : '1px solid #eee',
            overflow: 'hidden',
            width: 220,
            boxSizing: 'border-box',
          }}
        >
          <Link
            to={type === 'series' ? `/series/${item.id}` : `/movies/${item.id}`}
            style={{
              width: '100%',
              display: 'flex',
              flexDirection: 'column',
              alignItems: 'center',
              textDecoration: 'none',
              height: '100%',
            }}
          >
            <img
              key={item.id + '-' + type}
              src={type === 'series'
                ? `/mediacover/Series/${item.id}/poster-500.jpg`
                : `/mediacover/Movies/${item.id}/poster-500.jpg`}
              width={200}
              height={300}
              loading="lazy"
              style={{
                width: 200,
                height: 300,
                objectFit: 'cover',
                borderRadius: 8,
                background: '#222',
                boxShadow: '0 2px 8px rgba(14, 9, 9, 0.18)',
                marginBottom: 10,
                display: 'block',
              }}
              onError={e => { e.target.onerror = null; e.target.src = 'https://via.placeholder.com/200x300?text=No+Poster'; }}
              alt={item.title}
            />
            <div style={{ color: darkMode ? '#e5e7eb' : '#222', fontSize: 14, marginBottom: 2, opacity: 0.85 }}>{item.year ? item.year : (item.airDate || '')}</div>
            <div style={{ flex: 1 }} />
            <div
              style={{
                color: darkMode ? '#fff' : '#222',
                fontWeight: 600,
                fontSize: 16,
                textAlign: 'center',
                marginBottom: 2,
                width: '100%',
                maxWidth: 180,
                wordBreak: 'break-word',
                overflow: 'hidden',
                textOverflow: 'clip',
                whiteSpace: 'normal',
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
