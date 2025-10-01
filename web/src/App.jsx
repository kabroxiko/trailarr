import React, { useState, useEffect } from 'react';
import MediaList from './components/MediaList';
import MediaDetails from './components/MediaDetails';
import Header from './components/Header';
import Sidebar from './components/Sidebar';
import GeneralSettings from './components/GeneralSettings';
import Tasks from './components/Tasks';
import HistoryPage from './components/HistoryPage';
import Wanted from './components/Wanted';
import { Routes, Route } from 'react-router-dom';
import { Navigate } from 'react-router-dom';
import './App.css';
import { getRadarrSettings, getMovies, getSeries } from './api';

function App() {
  const [search, setSearch] = useState('');
  const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
  const [darkMode, setDarkMode] = useState(prefersDark);
  useEffect(() => {
    const listener = e => setDarkMode(e.matches);
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', listener);
    return () => window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', listener);
  }, []);
  const [selectedSection, setSelectedSection] = useState('Movies');

  // Reset search when changing main section (Movies/Series)
  useEffect(() => {
    setSearch('');
  }, [selectedSection]);
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
        setSelectedSettingsSub(sub.charAt(0).toUpperCase() + sub.slice(1));
      }
    } else if (path.startsWith('/settings')) {
      setSelectedSection('Settings');
      setSelectedSettingsSub('General');
    } else if (path.startsWith('/movies')) {
      setSelectedSection('Movies');
    } else if (path.startsWith('/series')) {
      setSelectedSection('Series');
    } else if (path.startsWith('/history')) {
      setSelectedSection('History');
    }
  }, []);

  // Sonarr series fetch from backend
  useEffect(() => {
    setSonarrSeriesLoading(true);
    getSeries()
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
    getMovies()
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

  // Separate search results into title and overview matches
  const getSearchSections = (items) => {
    if (!search.trim()) return { titleMatches: items, overviewMatches: [] };
    const q = search.trim().toLowerCase();
    const titleMatches = items.filter(item => item.title && item.title.toLowerCase().includes(q));
    const overviewMatches = items.filter(item =>
      !titleMatches.includes(item) && item.overview && item.overview.toLowerCase().includes(q)
    );
    return { titleMatches, overviewMatches };
  };

  return (
    <div className="app-container">
      <Header darkMode={darkMode} search={search} setSearch={setSearch} />
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
                (() => {
                  const { titleMatches, overviewMatches } = getSearchSections(sonarrSeries);
                  return (
                    <>
                      {search.trim() ? (
                        <>
                          <MediaList items={titleMatches} darkMode={darkMode} type="series" />
                          <div style={{ margin: '1.5em 0 0.5em 1em', fontWeight: 700, fontSize: 26, textAlign: 'left', width: '100%', letterSpacing: 0.5 }}>Other Results</div>
                          <MediaList items={overviewMatches} darkMode={darkMode} type="series" />
                        </>
                      ) : (
                        <MediaList items={sonarrSeries} darkMode={darkMode} type="series" />
                      )}
                      {sonarrSeriesError && <div style={{ color: 'red', marginTop: '1em' }}>{sonarrSeriesError}</div>}
                    </>
                  );
                })()
              } />
              <Route path="/movies" element={
                (() => {
                  const { titleMatches, overviewMatches } = getSearchSections(radarrMovies);
                  return (
                    <>
                      {search.trim() ? (
                        <>
                          <MediaList items={titleMatches} darkMode={darkMode} type="movie" />
                          <div style={{ margin: '1.5em 0 0.5em 1em', fontWeight: 700, fontSize: 26, textAlign: 'left', width: '100%', letterSpacing: 0.5 }}>Other Results</div>
                          <MediaList items={overviewMatches} darkMode={darkMode} type="movie" />
                        </>
                      ) : (
                        <MediaList items={radarrMovies} darkMode={darkMode} type="movie" />
                      )}
                      {radarrMoviesError && <div style={{ color: 'red', marginTop: '1em' }}>{radarrMoviesError}</div>}
                    </>
                  );
                })()
              } />
              <Route path="/movies/:id" element={<MediaDetails mediaItems={radarrMovies} loading={radarrMoviesLoading} mediaType="movie" />} />
              <Route path="/series/:id" element={<MediaDetails mediaItems={sonarrSeries} loading={sonarrSeriesLoading} mediaType="tv" />} />
              <Route path="/history" element={<HistoryPage />} />
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
              <Route path="/settings/general" element={<GeneralSettings />} />
              <Route path="/" element={<Navigate to="/movies" replace />} />
              <Route path="/system/tasks" element={<Tasks />} />
              <Route path="/wanted" element={<Wanted darkMode={darkMode} />} />
            </Routes>
          </div>
        </main>
      </div>
    </div>
  );
}

export default App;
