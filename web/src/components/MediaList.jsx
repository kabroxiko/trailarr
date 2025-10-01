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
            <div style={{
              width: 200,
              height: 300,
              position: 'relative',
              marginBottom: 10,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              background: darkMode ? '#222' : '#fff',
              borderRadius: 8,
              overflow: 'hidden',
              border: darkMode ? '1px solid #333' : '1px solid #eee',
            }}>
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
                  display: 'block',
                  position: 'absolute',
                  top: 0,
                  left: 0,
                  zIndex: 1,
                }}
                onError={e => {
                  e.target.onerror = null;
                  e.target.src = '';
                  const parent = e.target.parentNode;
                  if (parent && !parent.querySelector('.fallback-logo')) {
                    const logo = document.createElement('img');
                    logo.src = '/logo.svg';
                    logo.width = 80;
                    logo.height = 80;
                    logo.alt = 'No Poster';
                    logo.className = 'fallback-logo';
                    logo.style.position = 'relative';
                    logo.style.zIndex = 2;
                    logo.style.display = 'block';
                    logo.style.margin = 'auto';
                    logo.style.top = '0';
                    logo.style.left = '0';
                    logo.style.right = '0';
                    logo.style.bottom = '0';
                    logo.style.transform = 'none';
                    parent.appendChild(logo);
                  }
                  e.target.style.visibility = 'hidden';
                }}
                alt={item.title}
              />
            </div>
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
