
import { useState, useEffect } from 'react';
import './App.css';
import { searchExtras, downloadExtra, fetchPlexItems, getRadarrSettings, getRadarrMovies } from './api';

function App() {
  const [selectedMovie, setSelectedMovie] = useState(null);
  const [plexItems, setPlexItems] = useState([]);
  const [plexError, setPlexError] = useState('');
  const [radarrMovies, setRadarrMovies] = useState([]);
  const [radarrMoviesError, setRadarrMoviesError] = useState('');
  const [selectedSection, setSelectedSection] = useState('Movies');
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
    if (selectedSection === 'Movies') {
      getRadarrMovies()
        .then(res => setRadarrMovies(res.movies || []))
        .catch(e => setRadarrMoviesError(e.message));
    }
  }, [selectedSection]);

  return (
    <div style={{ display: 'flex', width: '100vw', height: '100vh', fontFamily: 'sans-serif', background: '#f7f8fa', overflowX: 'hidden', overflowY: 'hidden', position: 'fixed', left: 0, top: 0 }}>
      {/* Sidebar */}
  <aside style={{ width: 220, background: '#fff', borderRight: '1px solid #e5e7eb', padding: '1em 0', height: '100vh', boxSizing: 'border-box' }}>
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
                <button
                  style={{
                    background: selectedSection === name ? '#f3e8ff' : 'none',
                    border: 'none',
                    color: selectedSection === name ? '#a855f7' : '#333',
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
                </button>
                {/* Settings submenu */}
                {name === 'Settings' && selectedSection === 'Settings' && (
                  <ul style={{
                    listStyle: 'none',
                    padding: 0,
                    margin: '8px 0 0 0',
                    background: '#f3f4f6',
                    borderRadius: 6,
                  }}>
                    {['General', 'Languages', 'Providers', 'Subtitles', 'Sonarr', 'Radarr', 'Plex', 'Notifications', 'Scheduler', 'UI'].map((submenu, idx) => (
                      <li key={submenu} style={{
                        padding: '0.5em 1em',
                        borderLeft: selectedSettingsSub === submenu ? '3px solid #d6b4f7' : '3px solid transparent',
                        background: selectedSettingsSub === submenu ? '#fff' : 'none',
                        color: selectedSettingsSub === submenu ? '#a855f7' : '#333',
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
  <main style={{ flex: 1, padding: '2em', height: '100vh', boxSizing: 'border-box', overflowY: 'auto', overflowX: 'hidden', display: 'flex', flexDirection: 'column', alignItems: 'flex-start', justifyContent: 'stretch', maxWidth: 'calc(100vw - 220px)' }}>
        <div style={{ marginBottom: '2em', display: 'flex', justifyContent: 'space-between', alignItems: 'center' }}>
          <h2 style={{ color: '#a855f7', margin: 0 }}>{selectedSection}</h2>
          <input type="search" placeholder="Search" style={{ padding: '0.5em', borderRadius: 6, border: '1px solid #e5e7eb', width: 200 }} />
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
  <div style={{ background: '#fff', borderRadius: 8, boxShadow: '0 1px 4px #e5e7eb', padding: '1em', width: '100%', maxWidth: '100%', flex: 1, overflowY: 'auto', overflowX: 'hidden' }}>
    {selectedSection === 'Movies' ? (
      <>
        <h3 style={{ color: '#a855f7', marginTop: 0 }}>Radarr Movies</h3>
        {!selectedMovie ? (
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
                  <td style={{ padding: '0.5em' }}>
                    <a
                      href="#"
                      style={{ color: '#a855f7', textDecoration: 'underline', cursor: 'pointer', fontWeight: 'bold' }}
                      onClick={e => {
                        e.preventDefault();
                        setSelectedMovie(movie);
                      }}
                    >{movie.title}</a>
                  </td>
                  <td style={{ padding: '0.5em' }}>{movie.year}</td>
                  <td style={{ padding: '0.5em' }}>{movie.path}</td>
                </tr>
              ))}
            </tbody>
          </table>
        ) : (
          <div style={{ display: 'flex', gap: 32 }}>
            <div style={{ minWidth: 300 }}>
              {/* Movie poster from Radarr cache */}
              <img
                src={`/mediacover/${selectedMovie.id}/poster-500.jpg`}
                alt={selectedMovie.title}
                style={{ width: 300, height: 450, objectFit: 'cover', borderRadius: 12, marginBottom: 16, background: '#222' }}
                onError={e => { e.target.onerror = null; e.target.src = 'https://via.placeholder.com/300x450?text=No+Poster'; }}
              />
              <button style={{ marginTop: 8, background: '#eee', border: 'none', borderRadius: 6, padding: '0.5em 1em', cursor: 'pointer' }} onClick={() => setSelectedMovie(null)}>Back to list</button>
            </div>
            <div style={{ flex: 1 }}>
              <h2 style={{ color: '#a855f7', margin: 0 }}>{selectedMovie.title}</h2>
              <div style={{ marginBottom: 8, color: '#888' }}>{selectedMovie.year} &bull; {selectedMovie.path}</div>
              <div style={{ marginBottom: 16, color: '#333' }}>Movie extras would be listed here.</div>
              {/* TODO: Integrate actual extras data here */}
            </div>
          </div>
        )}
        {radarrMoviesError && <div style={{ color: 'red', marginTop: '1em' }}>{radarrMoviesError}</div>}
      </>
    ) : (
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
    )}
  </div>
      </main>
    </div>
  );
}

export default App;
