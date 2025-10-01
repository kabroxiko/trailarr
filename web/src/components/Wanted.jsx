import React, { useEffect, useState } from 'react';
import MediaList from './MediaList';

export default function Wanted({ darkMode }) {
  const [movies, setMovies] = useState([]);
  const [series, setSeries] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState('');

  useEffect(() => {
    async function fetchWanted() {
      setLoading(true);
      setError('');
      try {
        const [moviesRes, seriesRes] = await Promise.all([
          fetch('/api/movies/no_trailer_extra'),
          fetch('/api/series/no_trailer_extra'),
        ]);
        const moviesData = await moviesRes.json();
        const seriesData = await seriesRes.json();
        const sortedMovies = (moviesData.movies || []).slice().sort((a, b) => {
          if (!a.title) return 1;
          if (!b.title) return -1;
          return a.title.localeCompare(b.title);
        });
        const sortedSeries = (seriesData.series || []).slice().sort((a, b) => {
          if (!a.title) return 1;
          if (!b.title) return -1;
          return a.title.localeCompare(b.title);
        });
        setMovies(sortedMovies);
        setSeries(sortedSeries);
      } catch (e) {
        setError('Failed to fetch wanted items');
      }
      setLoading(false);
    }
    fetchWanted();
  }, []);

  return (
    <div style={{ padding: '0em 0em', width: '100%' }}>
      {loading && <div>Loading...</div>}
      {error && <div style={{ color: 'red' }}>{error}</div>}
      {!loading && (
        <div style={{ width: '100%' }}>
          <h2 style={{ fontSize: 32, padding: '15px', marginBottom: '0', textAlign: 'left' }}>Movies</h2>
          <MediaList items={movies} darkMode={darkMode} type="movie" />
          <h2 style={{ fontSize: 32, padding: '15px', margin: '0', textAlign: 'left' }}>Series</h2>
          <MediaList items={series} darkMode={darkMode} type="series" />
        </div>
      )}
    </div>
  );
}
