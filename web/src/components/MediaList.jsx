import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import MediaCard from './MediaCard.jsx';
import PropTypes from 'prop-types';

// basePath: e.g. '/wanted/movies' or '/movies'
export default function MediaList({ items, darkMode, type, basePath }) {
  const [showEmpty, setShowEmpty] = useState(false);

  // Prepare a sorted copy of items by sortTitle (case-insensitive). Fall back to title when sortTitle is missing.
  const sortedItems = (items || []).slice().sort((a, b) => {
    const aKey = (a.sortTitle || a.title || '').toString().toLowerCase();
    const bKey = (b.sortTitle || b.title || '').toString().toLowerCase();
    if (aKey < bKey) return -1;
    if (aKey > bKey) return 1;
    return 0;
  });

  useEffect(() => {
    let t;
    if (!items || items.length === 0) {
      // wait briefly before showing empty state to avoid flash while loading
      t = setTimeout(() => setShowEmpty(true), 500);
    } else {
      setShowEmpty(false);
    }
    return () => clearTimeout(t);
  }, [items]);

  return (
    <div style={{ width: '100%' }}>
  {(items.length === 0 && showEmpty) ? (
        <div
          style={{
            minHeight: 'calc(100vh - 120px)',
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            padding: '1.5rem',
            boxSizing: 'border-box',
          }}
        >
          <div
            style={{
              textAlign: 'center',
              color: darkMode ? '#ddd' : '#333',
              background: darkMode ? '#121214' : '#fbfbfb',
              border: darkMode ? '1px solid #222' : '1px solid #eee',
              padding: '1.25rem 1.5rem',
              borderRadius: 10,
              maxWidth: 800,
              width: 'auto',
              margin: '0 auto',
            }}
          >
            <div style={{ fontSize: 18, fontWeight: 600, marginBottom: 6 }}>No media found</div>
            <div style={{ fontSize: 14, opacity: 0.85 }}>
              There are no items to show here. Try scanning your libraries, check your path mappings, or adjust filters.
            </div>
          </div>
        </div>
      ) : (
        <div
          style={{
            display: 'grid',
            gridTemplateColumns: (items && items.length > 0)
              ? 'repeat(auto-fill, 220px)'
              : 'repeat(auto-fit, minmax(200px, 1fr))',
            gridAutoRows: '1fr',
            justifyContent: (items && items.length > 0) ? 'start' : 'center',
            gap: '2rem 1.5rem',
            padding: '1.5rem 1rem',
            width: '100%',
            boxSizing: 'border-box',
            alignItems: 'start',
          }}
        >
          {sortedItems.map((item) => {
            let linkTo;
            if (basePath) {
              linkTo = `${basePath}/${item.id}`;
            } else if (type === 'series') {
              linkTo = `/series/${item.id}`;
            } else {
              linkTo = `/movies/${item.id}`;
            }
            return (
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
              height: 410,
              transition: 'box-shadow 0.2s',
              border: darkMode ? '1px solid #333' : '1px solid #eee',
              overflow: 'hidden',
              width: 220,
              boxSizing: 'border-box',
            }}
          >
            <Link
              to={linkTo}
              style={{
                width: '100%',
                display: 'flex',
                flexDirection: 'column',
                alignItems: 'center',
                textDecoration: 'none',
                height: '100%',
              }}
            >
              <div style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'center', height: '100%' }}>
                <MediaCard media={item} mediaType={type} darkMode={darkMode} />
                <div style={{ flex: 1 }} />
              </div>
            </Link>
          </div>
            );
          })}
        </div>
      )}
    </div>
  );
}

MediaList.propTypes = {
  items: PropTypes.arrayOf(PropTypes.object),
  darkMode: PropTypes.bool,
  type: PropTypes.string,
  basePath: PropTypes.string,
};

MediaList.defaultProps = {
  items: [],
  darkMode: false,
  type: 'movies',
  basePath: '',
};
