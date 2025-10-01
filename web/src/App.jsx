import React, { useState, useEffect } from 'react';
import MovieTable from './components/MovieTable';
import SeriesTable from './components/SeriesTable';
import MovieDetails from './components/MovieDetails';
import Header from './components/Header';
import Sidebar from './components/Sidebar';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faFilm, faHistory, faStar, faBan, faCog, faServer, faBookmark } from '@fortawesome/free-solid-svg-icons';
import { Routes, Route, Link, useParams } from 'react-router-dom';
import { Navigate } from 'react-router-dom';
import './App.css';
import { fetchPlexItems, getRadarrSettings, getRadarrMovies } from './api';
// ...existing code...

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

  // Sonarr series state
  const [sonarrSeries, setSonarrSeries] = useState([]);
  const [sonarrSeriesError, setSonarrSeriesError] = useState('');
  const [sonarrSeriesLoading, setSonarrSeriesLoading] = useState(true);

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
    } else if (path.startsWith('/series')) {
      setSelectedSection('Series');
    }
  }, []);

  // Sonarr series fetch from backend
  useEffect(() => {
    setSonarrSeriesLoading(true);
    fetch('/api/sonarr/series')
      .then(r => {
        if (!r.ok) throw new Error('Failed to fetch Sonarr series');
        return r.json();
      })
      .then(data => {
        const sorted = (data.series || []).slice().sort((a, b) => {
          if (!a.title) return 1;
          if (!b.title) return -1;
          return a.title.localeCompare(b.title);
        });
        setSonarrSeries(sorted);
        setSonarrSeriesLoading(false);
        setSonarrSeriesError('');
      })
      .catch(e => {
        setSonarrSeries([]);
        setSonarrSeriesLoading(false);
        setSonarrSeriesError(e.message || 'Sonarr series API not available');
      });
  }, []);

  const [plexItems, setPlexItems] = useState([]);
  const [plexError, setPlexError] = useState('');
  const [radarrMovies, setRadarrMovies] = useState([]);
  const [radarrMoviesError, setRadarrMoviesError] = useState('');
  const [radarrMoviesLoading, setRadarrMoviesLoading] = useState(true);
  const [radarrUrl, setRadarrUrl] = useState('');
  const [radarrApiKey, setRadarrApiKey] = useState('');
  const [radarrStatus, setRadarrStatus] = useState('');
  const [sonarrUrl, setSonarrUrl] = useState('');
  const [sonarrApiKey, setSonarrApiKey] = useState('');
  const [sonarrStatus, setSonarrStatus] = useState('');

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
    // Sonarr settings fetch fallback
    async function getSonarrSettings() {
      try {
        const res = await fetch('/api/settings/sonarr');
        if (!res.ok) throw new Error('Failed to fetch Sonarr settings');
        return await res.json();
      } catch {
        return { url: '', apiKey: '' };
      }
    }
    getSonarrSettings()
      .then(res => {
        setSonarrUrl(res.url || '');
        setSonarrApiKey(res.apiKey || '');
        // Record Sonarr settings in localStorage
        localStorage.setItem('sonarrUrl', res.url || '');
        localStorage.setItem('sonarrApiKey', res.apiKey || '');
      })
      .catch(() => {
        setSonarrUrl('');
        setSonarrApiKey('');
        localStorage.setItem('sonarrUrl', '');
        localStorage.setItem('sonarrApiKey', '');
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
      <Header darkMode={darkMode} />
      <div style={{ display: 'flex', width: '100%', height: 'calc(100vh - 64px)' }}>
        <Sidebar
          selectedSection={selectedSection}
          setSelectedSection={setSelectedSection}
          selectedSettingsSub={selectedSettingsSub}
          setSelectedSettingsSub={setSelectedSettingsSub}
          darkMode={darkMode}
        />
        <main style={{ flex: 1, padding: '0em', height: '100%', boxSizing: 'border-box', overflowY: 'auto', overflowX: 'hidden', display: 'flex', flexDirection: 'column', alignItems: 'flex-start', justifyContent: 'stretch', maxWidth: 'calc(100vw - 220px)', background: darkMode ? '#18181b' : '#fff', color: darkMode ? '#e5e7eb' : '#222' }}>
          {/* Removed content title (Movies, Settings, etc) */}
          {/* Radarr Connection block is now rendered via a dedicated route below */}
          <div style={{ background: darkMode ? '#23232a' : '#fff', borderRadius: 8, boxShadow: darkMode ? '0 1px 4px #222' : '0 1px 4px #e5e7eb', padding: '0em', width: '100%', maxWidth: '100%', flex: 1, overflowY: 'auto', overflowX: 'hidden', color: darkMode ? '#e5e7eb' : '#222' }}>
            <Routes>
              <Route path="/series" element={
                <>
                  <SeriesTable series={sonarrSeries} darkMode={darkMode} />
                  {sonarrSeriesError && <div style={{ color: 'red', marginTop: '1em' }}>{sonarrSeriesError}</div>}
                </>
              } />
              <Route path="/movies" element={
                <>
                  <MovieTable movies={radarrMovies} darkMode={darkMode} />
                  {radarrMoviesError && <div style={{ color: 'red', marginTop: '1em' }}>{radarrMoviesError}</div>}
                </>
              } />
              <Route path="/movies/:id" element={<MovieDetails movies={radarrMovies} loading={radarrMoviesLoading} />} />
              <Route path="/series/:id" element={<MovieDetails movies={sonarrSeries} loading={sonarrSeriesLoading} />} />
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
              <Route path="/settings/sonarr" element={
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
                  <h3 style={{ color: '#e5e7eb', marginTop: 0 }}>Sonarr Connection</h3>
                  <div style={{ marginBottom: '1em' }}>
                    <label style={{ display: 'block', marginBottom: 4, color: darkMode ? '#e5e7eb' : '#222' }}>Sonarr URL</label>
                    <input
                      type="text"
                      value={sonarrUrl}
                      onChange={e => setSonarrUrl(e.target.value)}
                      style={{
                        width: '100%',
                        padding: '0.5em',
                        borderRadius: 6,
                        border: darkMode ? '1px solid #333' : '1px solid #e5e7eb',
                        background: darkMode ? '#18181b' : '#fff',
                        color: darkMode ? '#e5e7eb' : '#222',
                      }}
                      placeholder="http://localhost:8989"
                    />
                  </div>
                  <div style={{ marginBottom: '1em' }}>
                    <label style={{ display: 'block', marginBottom: 4, color: darkMode ? '#e5e7eb' : '#222' }}>API Key</label>
                    <input
                      type="text"
                      value={sonarrApiKey}
                      onChange={e => setSonarrApiKey(e.target.value)}
                      style={{
                        width: '100%',
                        padding: '0.5em',
                        borderRadius: 6,
                        border: darkMode ? '1px solid #333' : '1px solid #e5e7eb',
                        background: darkMode ? '#18181b' : '#fff',
                        color: darkMode ? '#e5e7eb' : '#222',
                      }}
                      placeholder="Your Sonarr API Key"
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
                      setSonarrStatus('');
                      try {
                        const res = await fetch('/api/settings/sonarr', {
                          method: 'POST',
                          headers: { 'Content-Type': 'application/json' },
                          body: JSON.stringify({ url: sonarrUrl, apiKey: sonarrApiKey })
                        });
                        if (!res.ok) throw new Error('Failed to save');
                        setSonarrStatus('Saved!');
                      } catch {
                        setSonarrStatus('Error saving');
                      }
                    }}
                  >Save</button>
                  {sonarrStatus && <div style={{ marginTop: '1em', color: '#22c55e' }}>{sonarrStatus}</div>}
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
              <Route path="/" element={<Navigate to="/movies" replace />} />
            </Routes>
          </div>
        </main>
      </div>
    </div>
  );
}

export default App;
