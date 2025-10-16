import React, { useEffect, useState } from 'react';
import MediaList from './MediaList';

export default function Wanted({ darkMode, type }) {
  const [items, setItems] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    async function fetchWanted() {
      setLoading(true);
      setError('');
      try {
        const endpoint = type === 'movie' ? '/api/movies/wanted' : '/api/series/wanted';
        const res = await fetch(endpoint);
        const data = await res.json();
        const sorted = (data.items || []).slice().sort((a, b) => {
          if (!a.title) return 1;
          if (!b.title) return -1;
          return a.title.localeCompare(b.title);
        });
        setItems(sorted);
      } catch {
        setError('Failed to fetch wanted ' + (type === 'movie' ? 'movies' : 'series'));
      }
      setLoading(false);
    }
    fetchWanted();
  }, [type]);

  return (
    <div style={{ padding: '0em 0em', width: '100%' }}>
      {loading && <div>Loading...</div>}
      {error && <div style={{ color: 'red' }}>{error}</div>}
      {!loading && (
        <MediaList
          items={items}
          darkMode={darkMode}
          type={type}
          basePath={type === 'series' ? '/wanted/series' : '/wanted/movies'}
        />
      )}
    </div>
  );
}
