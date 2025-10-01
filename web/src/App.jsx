
import { useState, useEffect } from 'react';
import { Routes, Route, Link, useNavigate, useParams, useLocation } from 'react-router-dom';
import './App.css';
import { searchExtras, downloadExtra, fetchPlexItems, getRadarrSettings, getRadarrMovies } from './api';

function MovieDetails({ movies }) {
  const { id } = useParams();
  const movie = movies.find(m => String(m.id) === id);
  const navigate = useNavigate();
  const [extras, setExtras] = useState([]);
  const [loading, setLoading] = useState(false);
  const [error, setError] = useState('');
  if (!movie) return <div>Movie not found</div>;
  const handleSearchExtras = async () => {
    setLoading(true);
    setError('');
    try {
      const res = await searchExtras(movie.title);
      setExtras(res.extras || []);
    } catch (e) {
      setError('Failed to search extras');
    } finally {
      setLoading(false);
    }
  };
  return (
    <div style={{ display: 'flex', flexDirection: 'column', gap: 24 }}>
      <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'space-between', marginBottom: 16, border: '2px solid #22c55e', padding: '0.5em' }}>
        <div
          style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer', fontWeight: 'bold', color: '#a855f7', fontSize: 18 }}
          onClick={handleSearchExtras}
        >
          <span style={{ fontSize: 20, display: 'inline-block' }}>🔎</span>
          <span>{loading ? 'Searching...' : 'Search'}</span>
        </div>
        <button style={{ background: '#eee', border: 'none', borderRadius: 6, padding: '0.5em 1em', cursor: 'pointer', fontWeight: 'bold' }} onClick={() => navigate('/movies')}>Back to list</button>
      </div>
  <div style={{ display: 'flex', gap: 32, border: '2px dotted #f59e42', padding: '0.5em' }}>
        <div style={{ minWidth: 300 }}>
          <img
            src={`/mediacover/${movie.id}/poster-500.jpg`}
            alt={movie.title}
            style={{ width: 300, height: 450, objectFit: 'cover', borderRadius: 12, marginBottom: 16, background: '#222' }}
            onError={e => { e.target.onerror = null; e.target.src = 'https://via.placeholder.com/300x450?text=No+Poster'; }}
          />
        </div>
        <div style={{ flex: 1 }}>
          <h2 style={{ color: '#a855f7', margin: 0 }}>{movie.title}</h2>
          <div style={{ marginBottom: 8, color: '#888', textAlign: 'left' }}>{movie.year} &bull; {movie.path}</div>
          <div style={{ marginBottom: 16, color: '#333' }}>Movie extras would be listed here.</div>
          {error && <div style={{ color: 'red', marginBottom: 8 }}>{error}</div>}
          {extras.length > 0 && (
            <ul style={{ marginTop: 16, textAlign: 'left', paddingLeft: 0 }}>
              {extras.map((extra, idx) => (
                <li key={idx}>
                  {typeof extra === 'object' && extra.type ? (
                    <span style={{ fontWeight: 'bold', marginRight: 8 }}>{extra.type}:</span>
                  ) : null}
                  {typeof extra === 'object' && extra.title ? extra.title : String(extra)}
                </li>
              ))}
            </ul>
          )}
        </div>
      </div>
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
    getRadarrMovies()
      .then(res => {
        const sorted = (res.movies || []).slice().sort((a, b) => {
          if (!a.title) return 1;
          if (!b.title) return -1;
          return a.title.localeCompare(b.title);
        });
        setRadarrMovies(sorted);
      })
      .catch(e => setRadarrMoviesError(e.message));
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
    padding: '1em 0',
    height: '100vh',
    boxSizing: 'border-box'
  }}>
        <div style={{ textAlign: 'center', marginBottom: '2em' }}>
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
    padding: '2em',
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
  <div style={{ marginBottom: '2em', display: 'flex', width: '100%', justifyContent: 'space-between', alignItems: 'center' }}>
          <h2 style={{ color: '#a855f7', margin: 0 }}>{selectedSection}</h2>
          <div style={{ display: 'flex', alignItems: 'center', gap: 8, flex: 1, justifyContent: 'flex-end', padding: '0.5em' }}>
            <input type="search" placeholder="Search movies" style={{ padding: '0.5em', borderRadius: 6, border: '1px solid #e5e7eb', width: 200, marginLeft: 'auto', textAlign: 'right' }} />
            <span style={{ fontSize: 20, color: '#a855f7' }}>🔎</span>
          </div>
        </div>

        {/* Settings > Radarr submenu content */}
        {selectedSection === 'Settings' && selectedSettingsSub === 'Radarr' && (
          <div style={{ background: '#fff', borderRadius: 8, boxShadow: '0 1px 4px #e5e7eb', padding: '2em', width: 400, marginBottom: '2em' }}>
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
    padding: '1em',
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
          <h3 style={{ color: '#a855f7', marginTop: 0 }}>Radarr Movies</h3>
          <table style={{ width: '100%', borderCollapse: 'collapse' }}>
            <thead>
              <tr style={{ background: '#f3e8ff' }}>
                <th style={{ textAlign: 'left', padding: '0.5em' }}>Title</th>
                <th style={{ textAlign: 'left', padding: '0.5em' }}>Year</th>
                <th style={{ textAlign: 'left', padding: '0.5em' }}>Path</th>
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
      <Route path="/movies/:id" element={<MovieDetails movies={radarrMovies} />} />
      <Route path="/settings" element={
      <>
        <table style={{ width: '100%', borderCollapse: 'collapse' }}>
          <thead>
            <tr style={{ background: '#f3e8ff' }}>
              <th style={{ textAlign: 'left', padding: '0.5em' }}>Name</th>
              <th style={{ textAlign: 'left', padding: '0.5em' }}>Language</th>
              <th style={{ textAlign: 'left', padding: '0.5em' }}>Extras</th>
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
