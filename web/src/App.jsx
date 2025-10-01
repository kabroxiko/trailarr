
import { useState, useEffect } from 'react';
import { Routes, Route, Link, useNavigate, useParams, useLocation } from 'react-router-dom';
import './App.css';
import { searchExtras, downloadExtra, fetchPlexItems, getRadarrSettings, getRadarrMovies } from './api';

function MovieDetails({ movies, loading }) {
  const { id } = useParams();
  const movie = movies.find(m => String(m.id) === id);
  const navigate = useNavigate();
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

  // Fetch extras from TMDB when movie details are loaded
  useEffect(() => {
    if (!movie) return;
    setSearchLoading(true);
    setError('');
    searchExtras(movie.title)
      .then(res => {
        setExtras(res.extras || []);
        // Check which extras exist on disk
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
  if (!movie) return <div>Movie not found</div>;

  const handleSearchExtras = async () => {
    setSearchLoading(true);
    setError('');
    try {
      const res = await searchExtras(movie.title);
      setExtras(res.extras || []);
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
      {/* Top bar */}
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', margin: '0px 0 0 0', padding: 0, width: '100%' }}>
        <div
          style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer', fontWeight: 'bold', color: '#a855f7', fontSize: 18 }}
          onClick={handleSearchExtras}
        >
          <span style={{ fontSize: 20, display: 'inline-block' }}>🔎</span>
          <span>{searchLoading ? 'Searching...' : 'Search'}</span>
        </div>
        <button style={{ background: '#eee', border: 'none', borderRadius: 6, padding: '0.5em 1em', cursor: 'pointer', fontWeight: 'bold' }} onClick={() => navigate('/movies')}>Back to list</button>
      </div>
      {/* Movie info card with background and overlay */}
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
        {/* Overlay for readability, Bazarr style */}
        <div style={{
          position: 'absolute',
          top: 0,
          left: 0,
          width: '100%',
          height: '100%',
          background: 'rgba(0,0,0,0.55)',
          zIndex: 1,
        }} />
        {/* Poster flush left, Bazarr size */}
        <div style={{ minWidth: 150, zIndex: 2, display: 'flex', justifyContent: 'flex-start', alignItems: 'center', height: '100%', padding: '0 0 0 32px' }}>
          <img
            src={`/mediacover/${movie.id}/poster-500.jpg`}
            alt={movie.title}
            style={{ width: 120, height: 180, objectFit: 'cover', borderRadius: 2, background: '#222', boxShadow: '0 1px 4px rgba(0,0,0,0.18)' }}
            onError={e => { e.target.onerror = null; e.target.src = 'https://via.placeholder.com/120x180?text=No+Poster'; }}
          />
        </div>
        {/* Details vertically centered next to poster, Bazarr font/colors */}
        <div style={{ flex: 1, zIndex: 2, display: 'flex', flexDirection: 'column', justifyContent: 'center', height: '100%', marginLeft: 32 }}>
          <h2 style={{ color: '#fff', margin: 0, fontSize: 22, fontWeight: 500, textShadow: '0 1px 2px #000', letterSpacing: 0.2 }}>{movie.title}</h2>
          <div style={{ marginBottom: 6, color: '#e5e7eb', textAlign: 'left', fontSize: 13, textShadow: '0 1px 2px #000' }}>{movie.year} &bull; {movie.path}</div>
          <div style={{ marginBottom: 10, color: '#f3e8ff', fontSize: 13, textShadow: '0 1px 2px #000' }}>Movie extras would be listed here.</div>
          {error && <div style={{ color: 'red', marginBottom: 8 }}>{error}</div>}
        </div>
      </div>
      {/* Extras table below info card, Bazarr style */}
      {extras.length > 0 && (
        <div style={{ width: '100%', background: darkMode ? '#23232a' : '#f3e8ff', overflow: 'hidden', padding: 0, margin: 0 }}>
          <table style={{ width: '100%', borderCollapse: 'collapse', background: 'transparent' }}>
            <thead>
              <tr style={{ height: 32 }}>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>Type</th>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>Title</th>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>URL</th>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>Download</th>
                <th style={{ textAlign: 'left', padding: '0.5em 1em', color: darkMode ? '#e5e7eb' : '#6d28d9', background: 'transparent', fontSize: 13, fontWeight: 500 }}>Exists</th>
              </tr>
            </thead>
            <tbody>
              {(() => {
                // Track counts and indexes for repeated titles
                const titleCounts = {};
                return extras.map((extra, idx) => {
                  const baseTitle = extra.title || String(extra);
                  titleCounts[baseTitle] = (titleCounts[baseTitle] || 0) + 1;
                  // Only show incremental if there are duplicates
                  const totalCount = extras.filter(e => (e.title || String(e)) === baseTitle).length;
                  const displayTitle = totalCount > 1 ? `${baseTitle} (${titleCounts[baseTitle]})` : baseTitle;
                  // Extract YouTubeID from extra.url
                  let youtubeID = '';
                  if (extra.url) {
                    if (extra.url.includes('youtube.com/watch?v=')) {
                      youtubeID = extra.url.split('v=')[1]?.split('&')[0] || '';
                    } else if (extra.url.includes('youtu.be/')) {
                      youtubeID = extra.url.split('youtu.be/')[1]?.split(/[?&]/)[0] || '';
                    }
                  }
                  // Determine if this extra exists (by type, title, and youtube_id)
                  const exists = existingExtras.some(e => e.type === extra.type && e.title === extra.title && e.youtube_id === youtubeID);
                  return (
                    <tr key={idx} style={{ height: 32, background: exists ? (darkMode ? '#1e293b' : '#e0e7ff') : undefined }}>
                      <td style={{ padding: '0.5em 1em', textAlign: 'left', color: darkMode ? '#e5e7eb' : '#222', fontSize: 13 }}>{extra.type || ''}</td>
                      <td style={{ padding: '0.5em 1em', textAlign: 'left', color: darkMode ? '#e5e7eb' : '#222', fontSize: 13 }}>{displayTitle}</td>
                      <td style={{ padding: '0.5em 1em', textAlign: 'left', color: darkMode ? '#a855f7' : '#6d28d9', fontSize: 13 }}>
                        {extra.url ? (
                          <a href={extra.url} target="_blank" rel="noopener noreferrer" style={{ color: darkMode ? '#a855f7' : '#6d28d9', textDecoration: 'underline', fontSize: 13 }}>Link</a>
                        ) : ''}
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
                        ) : ''}
                      </td>
                      <td style={{ padding: '0.5em 1em', textAlign: 'left', color: exists ? '#22c55e' : '#ef4444', fontWeight: 'bold', fontSize: 13 }}>{exists ? 'Yes' : 'No'}</td>
                    </tr>
                  );
                });
              })()}
            </tbody>
          </table>
        </div>
      )}
    </div>
  );
}

function App() {
  // Night mode detection
  const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
  const [darkMode, setDarkMode] = useState(prefersDark);
  useEffect(() => {
    const listener = e => setDarkMode(e.matches);
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', listener);
    return () => window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', listener);
  }, []);
  const [selectedMovie, setSelectedMovie] = useState(null);
  const [plexItems, setPlexItems] = useState([]);
  const [plexError, setPlexError] = useState('');
  const [radarrMovies, setRadarrMovies] = useState([]);
  const [radarrMoviesError, setRadarrMoviesError] = useState('');
  const [radarrMoviesLoading, setRadarrMoviesLoading] = useState(true);
  const [selectedSection, setSelectedSection] = useState('');
  const [selectedSettingsSub, setSelectedSettingsSub] = useState('General');
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
    <div style={{
      display: 'flex',
      width: '100vw',
      height: '100vh',
      fontFamily: 'sans-serif',
      background: darkMode ? '#18181b' : '#f7f8fa',
      color: darkMode ? '#e5e7eb' : '#222',
      overflowX: 'hidden',
      overflowY: 'hidden',
      position: 'fixed',
      left: 0,
      top: 0
    }}>
    {/* Sidebar */}
  <aside style={{
    width: 220,
    background: darkMode ? '#23232a' : '#fff',
    borderRight: darkMode ? '1px solid #333' : '1px solid #e5e7eb',
    padding: '0em 0',
    height: '100vh',
    boxSizing: 'border-box'
  }}>
        <div style={{ textAlign: 'center', marginBottom: '0em' }}>
          <span style={{ fontWeight: 'bold', color: '#d6b4f7', fontSize: 18 }}>EXTRAZARR</span>
        </div>
        <nav>
          <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
            {[
              { name: 'Series', icon: '📺' },
              { name: 'Movies', icon: '🎬' },
              { name: 'History', icon: '🕑' },
              { name: 'Wanted', icon: '⭐' },
              { name: 'Blacklist', icon: '🚫' },
              { name: 'Settings', icon: '⚙️' },
              { name: 'System', icon: '🖥️' }
            ].map(({ name, icon }) => (
              <li key={name} style={{ marginBottom: 16 }}>
                <Link
                  to={name === 'Movies' ? '/movies' : name === 'Settings' ? '/settings' : '/'}
                  style={{
                    textDecoration: 'none',
                    background: selectedSection === name ? (darkMode ? '#d6b4f7' : '#f3e8ff') : 'none',
                    border: 'none',
                    color: selectedSection === name
                      ? (darkMode ? '#6d28d9' : '#a855f7')
                      : (darkMode ? '#e5e7eb' : '#333'),
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
                  <span style={{ fontSize: 18 }}>{icon}</span>
                  {name}
                </Link>
                {/* Settings submenu */}
                {name === 'Settings' && selectedSection === 'Settings' && (
                  <ul style={{
                    listStyle: 'none',
                    padding: 0,
                    margin: '8px 0 0 0',
                    background: darkMode ? '#23232a' : '#f3f4f6',
                    borderRadius: 6,
                    color: darkMode ? '#e5e7eb' : '#222',
                  }}>
                    {['General', 'Languages', 'Providers', 'Subtitles', 'Sonarr', 'Radarr', 'Plex', 'Notifications', 'Scheduler', 'UI'].map((submenu, idx) => (
                      <li key={submenu} style={{
                        padding: '0.5em 1em',
                        borderLeft: selectedSettingsSub === submenu ? '3px solid #d6b4f7' : '3px solid transparent',
                        background: selectedSettingsSub === submenu ? (darkMode ? '#d6b4f7' : '#fff') : 'none',
                        color: selectedSettingsSub === submenu
                          ? (darkMode ? '#6d28d9' : '#a855f7')
                          : (darkMode ? '#e5e7eb' : '#333'),
                        fontWeight: selectedSettingsSub === submenu ? 'bold' : 'normal',
                        cursor: 'pointer',
                      }}
                        onClick={() => setSelectedSettingsSub(submenu)}
                      >{submenu}</li>
                    ))}
                  </ul>
                )}
              </li>
            ))}
          </ul>
        </nav>
      </aside>

  {/* Main content */}
  <main style={{
    flex: 1,
    padding: '0em',
    height: '100vh',
    boxSizing: 'border-box',
    overflowY: 'auto',
    overflowX: 'hidden',
    display: 'flex',
    flexDirection: 'column',
    alignItems: 'flex-start',
    justifyContent: 'stretch',
    maxWidth: 'calc(100vw - 220px)',
    background: darkMode ? '#18181b' : '#fff',
    color: darkMode ? '#e5e7eb' : '#222'
  }}>
  <div style={{ marginBottom: '0em', display: 'flex', width: '100%', justifyContent: 'space-between', alignItems: 'center' }}>
          <h2 style={{ color: '#a855f7', margin: 0 }}>{selectedSection}</h2>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, flex: 1, justifyContent: 'flex-end', padding: '0.5em' }}>
            <input type="search" placeholder="Search movies" style={{ padding: '0.5em', borderRadius: 6, border: '1px solid #e5e7eb', width: 200, marginLeft: 'auto', textAlign: 'right' }} />
            <span style={{ fontSize: 20, color: '#a855f7' }}>🔎</span>
          </div>
        </div>

        {/* Settings > Radarr submenu content */}
        {selectedSection === 'Settings' && selectedSettingsSub === 'Radarr' && (
          <div style={{ background: '#fff', borderRadius: 8, boxShadow: '0 1px 4px #e5e7eb', padding: '0em', width: 400, marginBottom: '0em' }}>
            <h3 style={{ color: '#a855f7', marginTop: 0 }}>Radarr Connection</h3>
            <div style={{ marginBottom: '1em' }}>
              <label style={{ display: 'block', marginBottom: 4 }}>Radarr URL</label>
              <input type="text" value={radarrUrl} onChange={e => setRadarrUrl(e.target.value)} style={{ width: '100%', padding: '0.5em', borderRadius: 6, border: '1px solid #e5e7eb' }} placeholder="http://localhost:7878" />
            </div>
            <div style={{ marginBottom: '1em' }}>
              <label style={{ display: 'block', marginBottom: 4 }}>API Key</label>
              <input type="text" value={radarrApiKey} onChange={e => setRadarrApiKey(e.target.value)} style={{ width: '100%', padding: '0.5em', borderRadius: 6, border: '1px solid #e5e7eb' }} placeholder="Your Radarr API Key" />
            </div>
            <button
              style={{ background: '#a855f7', color: '#fff', border: 'none', borderRadius: 6, padding: '0.5em 1em', cursor: 'pointer' }}
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
        )}
  <div style={{
    background: darkMode ? '#23232a' : '#fff',
    borderRadius: 8,
    boxShadow: darkMode ? '0 1px 4px #222' : '0 1px 4px #e5e7eb',
    padding: '0em',
    width: '100%',
    maxWidth: '100%',
    flex: 1,
    overflowY: 'auto',
    overflowX: 'hidden',
    color: darkMode ? '#e5e7eb' : '#222'
  }}>
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
                    <Link
                      to={`/movies/${movie.id}`}
                      style={{ color: '#a855f7', textDecoration: 'underline', cursor: 'pointer', fontWeight: 'bold', textAlign: 'left', display: 'block' }}
                    >{movie.title}</Link>
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
      <Route path="/settings" element={
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
      } />
      <Route path="/" element={<div>Welcome to Extrazarr</div>} />
    </Routes>
  </div>
      </main>
    </div>
  );
}

export default App;
