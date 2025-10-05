import React, { useState, useEffect } from 'react';
import { useLocation } from 'react-router-dom';
import Sidebar from './components/Sidebar';
import Header from './components/Header';
import { Routes, Route } from 'react-router-dom';
import { RouteMap, appRoutes } from './components/RouteMap';
import './App.css';
// Removed static import of api.js
// Refactored to use dynamic imports

function App() {
  const location = useLocation();
  const [search, setSearch] = useState('');
  const prefersDark = window.matchMedia && window.matchMedia('(prefers-color-scheme: dark)').matches;
  const [darkMode, setDarkMode] = useState(prefersDark);
  useEffect(() => {
    const listener = e => setDarkMode(e.matches);
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', listener);
    return () => window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', listener);
  }, []);
  const [selectedSection, setSelectedSection] = useState('Movies');
  const [selectedSystemSub, setSelectedSystemSub] = useState('Tasks');

  // Reset search when changing main section (Movies/Series)
  useEffect(() => {
    setSearch('');
  }, [selectedSection]);
  const [selectedSettingsSub, setSelectedSettingsSub] = useState('General');

  // Sonarr series state
  const [series, setSeries] = useState([]);
  const [seriesError, setSeriesError] = useState('');
  const [seriesLoading, setSeriesLoading] = useState(true);

  // Sync sidebar state with route changes
  useEffect(() => {
    const path = location.pathname;
    for (const entry of RouteMap) {
      if (entry.pattern.test(path)) {
        setSelectedSection(entry.section);
        if (entry.submenu) setSelectedSettingsSub(entry.submenu);
        if (entry.systemSub) setSelectedSystemSub(entry.systemSub);
        if (entry.section === 'Settings' && !entry.submenu && path.startsWith('/settings/')) {
          const sub = path.split('/')[2];
          if (sub) setSelectedSettingsSub(sub.charAt(0).toUpperCase() + sub.slice(1));
        }
        return;
      }
    }
  }, [location.pathname]);

  // Sonarr series fetch from backend
  useEffect(() => {
    setSeriesLoading(true);
    import('./api').then(({ getSeries }) => {
      getSeries()
        .then(data => {
          const sorted = (data.series || []).slice().sort((a, b) => {
            if (!a.title) return 1;
            if (!b.title) return -1;
            return a.title.localeCompare(b.title);
          });
          setSeries(sorted);
          setSeriesLoading(false);
          setSeriesError('');
        })
        .catch(e => {
          setSeries([]);
          setSeriesLoading(false);
          setSeriesError(e.message || 'Sonarr series API not available');
        });
    });
  }, []);

  const [movies, setMovies] = useState([]);
  const [moviesError, setMoviesError] = useState('');
  const [moviesLoading, setMoviesLoading] = useState(true);

  useEffect(() => {
    import('./api').then(({ getRadarrSettings }) => {
      getRadarrSettings()
        .then(res => {
          localStorage.setItem('radarrUrl', res.url || '');
          localStorage.setItem('radarrApiKey', res.apiKey || '');
        })
        .catch(() => {
          localStorage.setItem('radarrUrl', '');
          localStorage.setItem('radarrApiKey', '');
        });
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
        localStorage.setItem('sonarrUrl', res.url || '');
        localStorage.setItem('sonarrApiKey', res.apiKey || '');
      })
      .catch(() => {
        localStorage.setItem('sonarrUrl', '');
        localStorage.setItem('sonarrApiKey', '');
      });
  }, []);

  useEffect(() => {
    setMoviesLoading(true);
    import('./api').then(({ getMovies }) => {
      getMovies()
        .then(res => {
          const sorted = (res.movies || []).slice().sort((a, b) => {
            if (!a.title) return 1;
            if (!b.title) return -1;
            return a.title.localeCompare(b.title);
          });
          setMovies(sorted);
          setMoviesLoading(false);
        })
        .catch(e => {
          setMoviesError(e.message);
          setMoviesLoading(false);
        });
    });
  }, []);

  // Separate search results into title and overview matches
  const getSearchSections = (items) => {
    if (!search.trim()) return { titleMatches: items, overviewMatches: [] };
    const q = search.trim().toLowerCase();
    const titleMatches = items.filter(item => item.title?.toLowerCase().includes(q));
    const overviewMatches = items.filter(item =>
      !titleMatches.includes(item) && item.overview?.toLowerCase().includes(q)
    );
    return { titleMatches, overviewMatches };
  };

  // Compute dynamic page title
  let pageTitle = selectedSection;
  if (selectedSection === 'Settings') {
    pageTitle = `${selectedSettingsSub ? selectedSettingsSub : ''} Settings`;
  } else if (selectedSection === 'Wanted') {
    pageTitle = `Wanted${selectedSettingsSub ? ' ' + selectedSettingsSub : ''}`;
  } else if (selectedSection === 'System') {
    pageTitle = `${selectedSystemSub ? selectedSystemSub : ''}`;
  }

  // Update document title dynamically
  useEffect(() => {
    if (window.setTrailarrTitle) {
      window.setTrailarrTitle(pageTitle);
    }
  }, [pageTitle]);

  return (
    <div className="app-container">
      <Header darkMode={darkMode} search={search} setSearch={setSearch} pageTitle={pageTitle} />
      <div style={{ display: 'flex', width: '100%', height: 'calc(100vh - 64px)' }}>
        <Sidebar
          selectedSection={selectedSection}
          setSelectedSection={setSelectedSection}
          selectedSettingsSub={selectedSettingsSub}
          setSelectedSettingsSub={setSelectedSettingsSub}
          darkMode={darkMode}
          selectedSystemSub={selectedSystemSub}
          setSelectedSystemSub={setSelectedSystemSub}
        />
        <main style={{ flex: 1, padding: '0em', height: '100%', boxSizing: 'border-box', overflowY: 'auto', overflowX: 'hidden', display: 'flex', flexDirection: 'column', alignItems: 'flex-start', justifyContent: 'stretch', maxWidth: 'calc(100vw - 220px)', background: darkMode ? '#18181b' : '#fff', color: darkMode ? '#e5e7eb' : '#222' }}>
          {/* Removed content title (Movies, Settings, etc) */}
          {/* Radarr Connection block is now rendered via a dedicated route below */}
          <div style={{ background: darkMode ? '#23232a' : '#fff', boxShadow: darkMode ? '0 1px 4px #222' : '0 1px 4px #e5e7eb', padding: '0em', width: '100%', maxWidth: '100%', flex: 1, overflowY: 'auto', overflowX: 'hidden', color: darkMode ? '#e5e7eb' : '#222' }}>
            <Routes>
              {appRoutes.map((route, idx) => {
                if (route.dynamic) {
                  // Pass all needed props for dynamic routes
                  return (
                    <Route
                      key={route.path}
                      path={route.path}
                      element={route.render({
                        movies,
                        series,
                        search,
                        darkMode,
                        getSearchSections,
                        moviesError,
                        seriesError,
                        moviesLoading,
                        seriesLoading,
                      })}
                    />
                  );
                } else {
                  return (
                    <Route key={route.path} path={route.path} element={route.element} />
                  );
                }
              })}
            </Routes>
          </div>
        </main>
      </div>
    </div>
  );
}

export default App;
