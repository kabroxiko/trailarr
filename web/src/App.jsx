
import React, { useState, useEffect } from 'react';
import { Routes, Route, Link, useParams } from 'react-router-dom';
import './App.css';
import { searchExtras, downloadExtra, fetchPlexItems, getRadarrSettings, getRadarrMovies } from './api';

function MovieDetails({ movies, loading }) {
  const { id } = useParams();
  const movie = movies.find(m => String(m.id) === id);
  const [extras, setExtras] = useState([]);
  const [existingExtras, setExistingExtras] = useState([]);
  const [searchLoading, setSearchLoading] = useState(false);
  const [error, setError] = useState('');
  const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
  const [darkMode, setDarkMode] = useState(prefersDark);
  useEffect(() => {
    const listener = e => setDarkMode(e.matches);
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', listener);
    return () => window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', listener);
  }, []);

  useEffect(() => {
    if (!movie) return;
    setSearchLoading(true);
    setError('');
    searchExtras(movie.title)
      .then(res => {
        setExtras(res.extras || []);
        if (movie.path) {
          fetch(`/api/extras/existing?moviePath=${encodeURIComponent(movie.path)}`)
            .then(r => r.json())
            .then(data => setExistingExtras(data.existing || []))
            .catch(() => setExistingExtras([]));
        }
      })
      .catch(() => setError('Failed to search extras'))
      .finally(() => setSearchLoading(false));
  }, [movie]);

  if (loading) return <div>Loading movie details...</div>;
  if (!movie) {
    return (
      <div>
        Movie not found
        <pre style={{ background: '#eee', color: '#222', padding: 8, marginTop: 12, fontSize: 13 }}>
          Debug info:
          id: {String(id)}
          movies.length: {movies ? movies.length : 'undefined'}
          movies: {JSON.stringify(movies, null, 2)}
        </pre>
      </div>
    );
  }

  const handleSearchExtras = async () => {
    setSearchLoading(true);
    setError('');
    try {
      const res = await searchExtras(movie.title);
  const [selectedSection, setSelectedSection] = useState('');
  const [selectedSettingsSub, setSelectedSettingsSub] = useState('General');
  const [plexItems, setPlexItems] = useState([]);
  const [plexError, setPlexError] = useState('');
  const [radarrMovies, setRadarrMovies] = useState([]);
  const [radarrMoviesError, setRadarrMoviesError] = useState('');
  const [radarrMoviesLoading, setRadarrMoviesLoading] = useState(true);
  const [radarrUrl, setRadarrUrl] = useState('');
  const [radarrApiKey, setRadarrApiKey] = useState('');
  const [radarrStatus, setRadarrStatus] = useState('');

  useEffect(() => {
    fetchPlexItems()
      .then(res => setPlexItems(res.items || []))
      .catch(e => setPlexError(e.message));
    getRadarrSettings()
      .then(res => {
        setRadarrUrl(res.url || '');
        setRadarrApiKey(res.apiKey || '');
      })
      .catch(() => {
        setRadarrUrl('');
        setRadarrApiKey('');
      });
  }, []);

  useEffect(() => {
    setRadarrMoviesLoading(true);
    getRadarrMovies()
      .then(res => {
        const sorted = (res.movies || []).slice().sort((a, b) => {
          if (!a.title) return 1;
          if (!b.title) return -1;
          return a.title.localeCompare(b.title);
        });
        setRadarrMovies(sorted);
        setRadarrMoviesLoading(false);
      })
      .catch(e => {
        setRadarrMoviesError(e.message);
        setRadarrMoviesLoading(false);
      });
  }, []);

    } catch (e) {
      setError('Failed to search extras');
    } finally {
      setSearchLoading(false);
    }
  };

  return (
    <div style={{
      display: 'flex',
      flexDirection: 'column',
      minHeight: '100vh',
      background: darkMode ? '#18181b' : '#f7f8fa',
      fontFamily: 'Roboto, Arial, sans-serif',
      margin: 0,
      padding: 0,
      width: '100%',
      boxSizing: 'border-box',
    }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-start', margin: '0px 0 0 0', padding: 0, width: '100%' }}>
        <div
          style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer', fontWeight: 'bold', color: '#e5e7eb', fontSize: 18 }}
          onClick={handleSearchExtras}
        >
          <span style={{ fontSize: 20, display: 'flex', alignItems: 'center' }}>
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
              <circle cx="9" cy="9" r="7" stroke="#e5e7eb" strokeWidth="2" />
              <line x1="15" y1="15" x2="19" y2="19" stroke="#e5e7eb" strokeWidth="2" strokeLinecap="round" />
            </svg>
          </span>
          <span>{searchLoading ? 'Searching...' : 'Search'}</span>
        </div>
      </div>
      <div style={{
        width: '100%',
        position: 'relative',
        background: `url(/mediacover/${movie.id}/fanart-1280.jpg) center center/cover no-repeat`,
        minHeight: 210,
        display: 'flex',
        flexDirection: 'row',
        alignItems: 'center',
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
        <div style={{ minWidth: 150, zIndex: 2, display: 'flex', justifyContent: 'flex-start', alignItems: 'center', height: '100%', padding: '0 0 0 32px' }}>
          <img
            src={`/mediacover/${movie.id}/poster-500.jpg`}
            style={{ width: 120, height: 180, objectFit: 'cover', borderRadius: 2, background: '#222', boxShadow: '0 1px 4px rgba(0,0,0,0.18)' }}
            onError={e => { e.target.onerror = null; e.target.src = 'https://via.placeholder.com/120x180?text=No+Poster'; }}
          />
        </div>
        <div style={{ flex: 1, zIndex: 2, display: 'flex', flexDirection: 'column', justifyContent: 'center', height: '100%', marginLeft: 32 }}>
          <h2 style={{ color: '#fff', margin: 0, fontSize: 22, fontWeight: 500, textShadow: '0 1px 2px #000', letterSpacing: 0.2 }}>{movie.title}</h2>
          <div style={{ marginBottom: 6, color: '#e5e7eb', textAlign: 'left', fontSize: 13, textShadow: '0 1px 2px #000' }}>{movie.year} &bull; {movie.path}</div>
          <div style={{ marginBottom: 10, color: '#f3e8ff', fontSize: 13, textShadow: '0 1px 2px #000' }}>Movie extras would be listed here.</div>
          {error && <div style={{ color: 'red', marginBottom: 8 }}>{error}</div>}
        </div>
      </div>
      {extras.length > 0 && (
        <div style={{ width: '100%', background: darkMode ? '#23232a' : '#f3e8ff', overflow: 'hidden', padding: 0, margin: 0 }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', background: 'transparent' }}>
            <thead>
              <tr style={{ height: 32 }}>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>Type</th>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>Title</th>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>URL</th>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>Download</th>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>Status</th>
              </tr>
            </thead>
            <tbody>
              {extras.map((extra, idx) => {
                const baseTitle = extra.title || String(extra);
                const totalCount = extras.filter(e => (e.title || String(e)) === baseTitle).length;
                const displayTitle = totalCount > 1 ? `${baseTitle} (${extras.slice(0, idx + 1).filter(e => (e.title || String(e)) === baseTitle).length})` : baseTitle;
                let youtubeID = '';
                if (extra.url) {
                  if (extra.url.includes('youtube.com/watch?v=')) {
                    youtubeID = extra.url.split('v=')[1]?.split('&')[0] || '';
                  } else if (extra.url.includes('youtu.be/')) {
                    youtubeID = extra.url.split('youtu.be/')[1]?.split(/[?&]/)[0] || '';
                  }
                }
                const exists = existingExtras.some(e => e.type === extra.type && e.title === extra.title && e.youtube_id === youtubeID);
                return (
                  <tr key={idx} style={{ height: 32, background: exists ? (darkMode ? '#1e293b' : '#e0e7ff') : undefined }}>
                    <td style={{ padding: '0.5em 1em', textAlign: 'left', color: darkMode ? '#e5e7eb' : '#222', fontSize: 13 }}>{extra.type || ''}</td>
                    <td style={{ padding: '0.5em 1em', textAlign: 'left', color: darkMode ? '#e5e7eb' : '#222', fontSize: 13 }}>{displayTitle}</td>
                    <td style={{ padding: '0.5em 1em', textAlign: 'left', color: darkMode ? '#e5e7eb' : '#6d28d9', fontSize: 13 }}>
                      {extra.url ? (
                        <a href={extra.url} target="_blank" rel="noopener noreferrer" style={{ color: darkMode ? '#e5e7eb' : '#6d28d9', textDecoration: 'underline', fontSize: 13 }}>Link</a>
                      ) : null}
                    </td>
                    <td style={{ padding: '0.5em 1em', textAlign: 'left' }}>
                      {extra.url && (extra.url.includes('youtube.com/watch?v=') || extra.url.includes('youtu.be/')) ? (
                        <button
                          style={{ background: exists ? '#888' : '#a855f7', color: '#fff', border: 'none', borderRadius: 4, padding: '0.25em 0.75em', cursor: exists ? 'not-allowed' : 'pointer', fontWeight: 'bold', fontSize: 13 }}
                          disabled={exists}
                          onClick={async () => {
                            if (exists) return;
                            try {
                              const res = await downloadExtra({
                                moviePath: movie.path,
                                extraType: extra.type,
                                extraTitle: extra.title,
                                url: typeof extra.url === 'string' ? extra.url : (extra.url && extra.url.url ? extra.url.url : '')
                              });
                              setExistingExtras(prev => [...prev, { type: extra.type, title: extra.title, youtube_id: youtubeID }]);
                            } catch (e) {
                              alert('Download failed: ' + (e.message || e));
                            }
                          }}
                        >Download</button>
                      ) : null}
                    </td>
                    <td style={{ padding: '0.5em 1em', textAlign: 'left', color: exists ? '#22c55e' : '#ef4444', fontWeight: 'bold', fontSize: 13 }}>{exists ? 'Downloaded' : 'Not downloaded'}</td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function App() {
  const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
  const [darkMode, setDarkMode] = useState(prefersDark);
  useEffect(() => {
    const listener = e => setDarkMode(e.matches);
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', listener);
    return () => window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', listener);
  }, []);
  const [selectedSection, setSelectedSection] = useState('Movies');
  const [selectedSettingsSub, setSelectedSettingsSub] = useState('General');

  // Sync sidebar state with route on mount/refresh
  useEffect(() => {
    const path = window.location.pathname;
    if (path.startsWith('/settings/')) {
      setSelectedSection('Settings');
      const sub = path.split('/')[2];
      if (sub) {
        // Capitalize first letter for matching submenu
        setSelectedSettingsSub(sub.charAt(0).toUpperCase() + sub.slice(1));
      }
    } else if (path.startsWith('/settings')) {
      setSelectedSection('Settings');
      setSelectedSettingsSub('General');
    } else if (path.startsWith('/movies')) {
      setSelectedSection('Movies');
    }
  }, []);
  const [plexItems, setPlexItems] = useState([]);
  const [plexError, setPlexError] = useState('');
  const [radarrMovies, setRadarrMovies] = useState([]);
  const [radarrMoviesError, setRadarrMoviesError] = useState('');
  const [radarrMoviesLoading, setRadarrMoviesLoading] = useState(true);
  const [radarrUrl, setRadarrUrl] = useState('');
  const [radarrApiKey, setRadarrApiKey] = useState('');
  const [radarrStatus, setRadarrStatus] = useState('');

  useEffect(() => {
    fetchPlexItems()
      .then(res => setPlexItems(res.items || []))
      .catch(e => setPlexError(e.message));
    getRadarrSettings()
      .then(res => {
        setRadarrUrl(res.url || '');
        setRadarrApiKey(res.apiKey || '');
      })
      .catch(() => {
        setRadarrUrl('');
        setRadarrApiKey('');
      });
  }, []);

  useEffect(() => {
    setRadarrMoviesLoading(true);
    getRadarrMovies()
      .then(res => {
        const sorted = (res.movies || []).slice().sort((a, b) => {
          if (!a.title) return 1;
          if (!b.title) return -1;
          return a.title.localeCompare(b.title);
        });
        setRadarrMovies(sorted);
        setRadarrMoviesLoading(false);
      })
      .catch(e => {
        setRadarrMoviesError(e.message);
        setRadarrMoviesLoading(false);
      });
  }, []);

  return (
    <div style={{ width: '100vw', height: '100vh', fontFamily: 'sans-serif', background: darkMode ? '#18181b' : '#f7f8fa', color: darkMode ? '#e5e7eb' : '#222', overflow: 'hidden', position: 'fixed', left: 0, top: 0 }}>
      <header style={{ width: '100%', height: 64, background: darkMode ? '#23232a' : '#fff', display: 'flex', alignItems: 'center', justifyContent: 'space-between', boxShadow: darkMode ? '0 1px 4px #222' : '0 1px 4px #e5e7eb', padding: '0 32px', position: 'relative', zIndex: 10 }}>
        <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
          <img src="/logo.svg" alt="Logo" style={{ width: 40, height: 40, marginRight: 12 }} />
          <span style={{ fontWeight: 'bold', fontSize: 22, color: '#e5e7eb', letterSpacing: 0.5 }}>Trailarr</span>
        </div>
        <nav style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
          <input type="search" placeholder="Search movies" style={{ padding: '0.5em', borderRadius: 6, border: '1px solid #e5e7eb', width: 200, textAlign: 'left' }} />
          <span style={{ fontSize: 20, display: 'flex', alignItems: 'center' }}>
            <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
              <circle cx="9" cy="9" r="7" stroke="#e5e7eb" strokeWidth="2" />
              <line x1="15" y1="15" x2="19" y2="19" stroke="#e5e7eb" strokeWidth="2" strokeLinecap="round" />
            </svg>
          </span>
        </nav>
      </header>
      <div style={{ display: 'flex', width: '100%', height: 'calc(100vh - 64px)' }}>
        <aside style={{ width: 220, background: darkMode ? '#23232a' : '#fff', borderRight: darkMode ? '1px solid #333' : '1px solid #e5e7eb', padding: '0em 0', height: '100%', boxSizing: 'border-box' }}>
          <nav>
            <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
              {[
                { name: 'Series', icon: (
                  <svg width="18" height="18" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <rect x="3" y="5" width="14" height="10" rx="2" fill={darkMode ? '#e5e7eb' : '#333'} />
                    <rect x="7" y="15" width="6" height="2" rx="1" fill={darkMode ? '#e5e7eb' : '#333'} />
                  </svg>
                ) },
                { name: 'Movies', icon: (
                  <svg width="18" height="18" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <rect x="4" y="5" width="12" height="10" rx="2" fill={darkMode ? '#e5e7eb' : '#333'} />
                    <circle cx="10" cy="10" r="3" fill={darkMode ? '#e5e7eb' : '#333'} />
                  </svg>
                ) },
                { name: 'History', icon: (
                  <svg width="18" height="18" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <circle cx="10" cy="10" r="8" stroke={darkMode ? '#e5e7eb' : '#333'} strokeWidth="2" />
                    <path d="M10 6v4l3 3" stroke={darkMode ? '#e5e7eb' : '#333'} strokeWidth="2" strokeLinecap="round" />
                  </svg>
                ) },
                { name: 'Wanted', icon: (
                  <svg width="18" height="18" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <polygon points="10,3 12,8 17,8 13,11 15,16 10,13 5,16 7,11 3,8 8,8" fill={darkMode ? '#e5e7eb' : '#333'} />
                  </svg>
                ) },
                { name: 'Blacklist', icon: (
                  <svg width="18" height="18" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <rect x="4" y="4" width="12" height="12" rx="2" fill={darkMode ? '#e5e7eb' : '#333'} />
                    <line x1="6" y1="6" x2="14" y2="14" stroke="#e5e7eb" strokeWidth="2" />
                  </svg>
                ) },
                { name: 'Settings', icon: (
                  <svg width="18" height="18" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <circle cx="10" cy="10" r="7" stroke={darkMode ? '#e5e7eb' : '#333'} strokeWidth="2" />
                    <circle cx="10" cy="10" r="3" fill={darkMode ? '#e5e7eb' : '#333'} />
                  </svg>
                ) },
                { name: 'System', icon: (
                  <svg width="18" height="18" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
                    <rect x="3" y="5" width="14" height="10" rx="2" fill={darkMode ? '#e5e7eb' : '#333'} />
                    <rect x="7" y="15" width="6" height="2" rx="1" fill={darkMode ? '#e5e7eb' : '#333'} />
                  </svg>
                ) }
              ].map(({ name, icon }) => (
                <li key={name} style={{ marginBottom: 16 }}>
                  {name === 'Settings' ? (
                    <div
                      style={{
                        textDecoration: 'none',
                        background: selectedSection === name ? (darkMode ? '#d6b4f7' : '#f3e8ff') : 'none',
                        border: 'none',
                        color: selectedSection === name ? (darkMode ? '#6d28d9' : '#e5e7eb') : (darkMode ? '#e5e7eb' : '#333'),
                        fontWeight: selectedSection === name ? 'bold' : 'normal',
                        width: '100%',
                        textAlign: 'left',
                        padding: '0.5em 1em',
                        borderRadius: 6,
                        cursor: 'pointer',
                        display: 'flex',
                        alignItems: 'center',
                        gap: '0.75em',
                      }}
                      onClick={() => setSelectedSection(name)}
                    >
                      <span style={{ fontSize: 18, display: 'flex', alignItems: 'center' }}>{icon}</span>
                      {name}
                    </div>
                  ) : (
                    <Link
                      to={name === 'Movies' ? '/movies' : '/'}
                      style={{ textDecoration: 'none', background: 'none', border: 'none', color: selectedSection === name ? (darkMode ? '#a855f7' : '#6d28d9') : (darkMode ? '#e5e7eb' : '#333'), fontWeight: selectedSection === name ? 'bold' : 'normal', width: '100%', textAlign: 'left', padding: '0.5em 1em', borderRadius: 6, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '0.75em' }}
                      onClick={() => setSelectedSection(name)}
                    >
                      <span style={{ fontSize: 18, display: 'flex', alignItems: 'center' }}>{icon}</span>
                      {name}
                    </Link>
                  )}
                  {name === 'Settings' && selectedSection === 'Settings' && (
                    <ul style={{ listStyle: 'none', padding: 0, margin: '8px 0 0 0', background: darkMode ? '#23232a' : '#f3f4f6', borderRadius: 6, color: darkMode ? '#e5e7eb' : '#222', textAlign: 'left' }}>
                      {['General', 'Languages', 'Providers', 'Subtitles', 'Sonarr', 'Radarr', 'Plex', 'Notifications', 'Scheduler', 'UI'].map((submenu, idx) => (
                        <li key={submenu} style={{ padding: '0.5em 1em', borderLeft: selectedSettingsSub === submenu ? '3px solid #a855f7' : '3px solid transparent', background: 'none', color: selectedSettingsSub === submenu ? (darkMode ? '#a855f7' : '#6d28d9') : (darkMode ? '#e5e7eb' : '#333'), fontWeight: selectedSettingsSub === submenu ? 'bold' : 'normal', cursor: 'pointer', textAlign: 'left' }}>
                          <Link
                            to={`/settings/${submenu.toLowerCase()}`}
                            style={{ color: 'inherit', textDecoration: 'none', display: 'block', width: '100%', textAlign: 'left' }}
                            onClick={() => setSelectedSettingsSub(submenu)}
                          >{submenu}</Link>
                        </li>
                      ))}
                    </ul>
                  )}
                </li>
              ))}
            </ul>
          </nav>
        </aside>
        <main style={{ flex: 1, padding: '0em', height: '100%', boxSizing: 'border-box', overflowY: 'auto', overflowX: 'hidden', display: 'flex', flexDirection: 'column', alignItems: 'flex-start', justifyContent: 'stretch', maxWidth: 'calc(100vw - 220px)', background: darkMode ? '#18181b' : '#fff', color: darkMode ? '#e5e7eb' : '#222' }}>
          {/* Removed content title (Movies, Settings, etc) */}
          {/* Radarr Connection block is now rendered via a dedicated route below */}
          <div style={{ background: darkMode ? '#23232a' : '#fff', borderRadius: 8, boxShadow: darkMode ? '0 1px 4px #222' : '0 1px 4px #e5e7eb', padding: '0em', width: '100%', maxWidth: '100%', flex: 1, overflowY: 'auto', overflowX: 'hidden', color: darkMode ? '#e5e7eb' : '#222' }}>
            <Routes>
              <Route path="/movies" element={
                <>
                  <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                    <thead>
                      <tr style={{ background: darkMode ? '#23232a' : '#f3e8ff' }}>
                        <th style={{ textAlign: 'left', padding: '0.5em', color: darkMode ? '#e5e7eb' : '#6d28d9' }}>Title</th>
                        <th style={{ textAlign: 'left', padding: '0.5em', color: darkMode ? '#e5e7eb' : '#6d28d9' }}>Year</th>
                        <th style={{ textAlign: 'left', padding: '0.5em', color: darkMode ? '#e5e7eb' : '#6d28d9' }}>Path</th>
                      </tr>
                    </thead>
                    <tbody>
                      {radarrMovies.map((movie, idx) => (
                        <tr key={idx} style={{ borderBottom: '1px solid #f3e8ff' }}>
                          <td style={{ padding: '0.5em', textAlign: 'left' }}>
                            <Link to={`/movies/${movie.id}`} style={{ color: '#a855f7', textDecoration: 'underline', cursor: 'pointer', fontWeight: 'bold', textAlign: 'left', display: 'block' }}>{movie.title}</Link>
                          </td>
                          <td style={{ padding: '0.5em', textAlign: 'left' }}>{movie.year}</td>
                          <td style={{ padding: '0.5em', textAlign: 'left' }}>{movie.path}</td>
                        </tr>
                      ))}
                    </tbody>
                  </table>
                  {radarrMoviesError && <div style={{ color: 'red', marginTop: '1em' }}>{radarrMoviesError}</div>}
                </>
              } />
              <Route path="/movies/:id" element={<MovieDetails movies={radarrMovies} loading={radarrMoviesLoading} />} />
              <Route path="/settings/radarr" element={
                <div style={{
                  background: darkMode ? '#23232a' : '#fff',
                  borderRadius: 8,
                  boxShadow: darkMode ? '0 1px 4px #222' : '0 1px 4px #e5e7eb',
                  padding: '0em',
                  width: 400,
                  marginBottom: '0em',
                  color: darkMode ? '#e5e7eb' : '#222',
                  border: darkMode ? '1px solid #333' : 'none',
                }}>
                  <h3 style={{ color: '#e5e7eb', marginTop: 0 }}>Radarr Connection</h3>
                  <div style={{ marginBottom: '1em' }}>
                    <label style={{ display: 'block', marginBottom: 4, color: darkMode ? '#e5e7eb' : '#222' }}>Radarr URL</label>
                    <input
                      type="text"
                      value={radarrUrl}
                      onChange={e => setRadarrUrl(e.target.value)}
                      style={{
                        width: '100%',
                        padding: '0.5em',
                        borderRadius: 6,
                        border: darkMode ? '1px solid #333' : '1px solid #e5e7eb',
                        background: darkMode ? '#18181b' : '#fff',
                        color: darkMode ? '#e5e7eb' : '#222',
                      }}
                      placeholder="http://localhost:7878"
                    />
                  </div>
                  <div style={{ marginBottom: '1em' }}>
                    <label style={{ display: 'block', marginBottom: 4, color: darkMode ? '#e5e7eb' : '#222' }}>API Key</label>
                    <input
                      type="text"
                      value={radarrApiKey}
                      onChange={e => setRadarrApiKey(e.target.value)}
                      style={{
                        width: '100%',
                        padding: '0.5em',
                        borderRadius: 6,
                        border: darkMode ? '1px solid #333' : '1px solid #e5e7eb',
                        background: darkMode ? '#18181b' : '#fff',
                        color: darkMode ? '#e5e7eb' : '#222',
                      }}
                      placeholder="Your Radarr API Key"
                    />
                  </div>
                  <button
                    style={{
                      background: '#a855f7',
                      color: '#fff',
                      border: 'none',
                      borderRadius: 6,
                      padding: '0.5em 1em',
                      cursor: 'pointer',
                      fontWeight: 'bold',
                      boxShadow: darkMode ? '0 1px 4px #222' : '0 1px 4px #e5e7eb',
                    }}
                    onClick={async () => {
                      setRadarrStatus('');
                      try {
                        const res = await fetch('/api/settings/radarr', {
                          method: 'POST',
                          headers: { 'Content-Type': 'application/json' },
                          body: JSON.stringify({ url: radarrUrl, apiKey: radarrApiKey })
                        });
                        if (!res.ok) throw new Error('Failed to save');
                        setRadarrStatus('Saved!');
                      } catch {
                        setRadarrStatus('Error saving');
                      }
                    }}
                  >Save</button>
                  {radarrStatus && <div style={{ marginTop: '1em', color: '#22c55e' }}>{radarrStatus}</div>}
                </div>
              } />
              <Route path="/settings" element={
                selectedSettingsSub === 'Radarr' ? null : (
                  <>
                    <table style={{ width: '100%', borderCollapse: 'collapse' }}>
                      <thead>
                        <tr style={{ background: darkMode ? '#23232a' : '#f3e8ff' }}>
                          <th style={{ textAlign: 'left', padding: '0.5em', color: darkMode ? '#e5e7eb' : '#6d28d9' }}>Name</th>
                          <th style={{ textAlign: 'left', padding: '0.5em', color: darkMode ? '#e5e7eb' : '#6d28d9' }}>Language</th>
                          <th style={{ textAlign: 'left', padding: '0.5em', color: darkMode ? '#e5e7eb' : '#6d28d9' }}>Extras</th>
                        </tr>
                      </thead>
                      <tbody>
                        {plexItems.map((item, idx) => (
                          <tr key={idx} style={{ borderBottom: '1px solid #f3e8ff' }}>
                            <td style={{ padding: '0.5em' }}>{item.Title}</td>
                            <td style={{ padding: '0.5em' }}>{item.Language}</td>
                            <td style={{ padding: '0.5em' }}>{item.Extras.length > 0 ? item.Extras.join(', ') : <span style={{ color: '#bbb' }}>None</span>}</td>
                          </tr>
                        ))}
                      </tbody>
                    </table>
                    {plexError && <div style={{ color: 'red', marginTop: '1em' }}>{plexError}</div>}
                  </>
                )
              } />
              <Route path="/" element={<div>Welcome to Trailarr</div>} />
            </Routes>
          </div>
        </main>
      </div>
    </div>
  );
}

export default App;
