import React, { useState, useEffect } from 'react';
// Helper functions to avoid deep nesting
function filterAndSortMedia(items) {
  return (items || [])
    .filter(item => item?.title)
    .sort((a, b) => a.title.localeCompare(b.title));
}
import BlacklistPage from './components/BlacklistPage';
import SeriesRouteComponent from './SeriesRouteComponent';
import MoviesRouteComponent from './MoviesRouteComponent';
import Toast from './components/Toast';
import { Routes, Route, useLocation } from 'react-router-dom';

// Helper to load a component dynamically, but only once
function loadComponent(importFn, ref) {
  if (!ref.current) {
    ref.current = React.lazy(importFn);
  }
  return ref.current;
}

const MediaListRef = { current: null };
const MediaDetailsRef = { current: null };
const HeaderRef = { current: null };
const SidebarRef = { current: null };
const GeneralSettingsRef = { current: null };
const TasksRef = { current: null };
const HistoryPageRef = { current: null };
const WantedRef = { current: null };
const SettingsPageRef = { current: null };
const ExtrasSettingsRef = { current: null };
const LogsPageRef = { current: null };

function App() {
  const location = useLocation();
  const [search, setSearch] = useState('');
  const prefersDark = globalThis.matchMedia?.('(prefers-color-scheme: dark)').matches;
  const [darkMode, setDarkMode] = useState(prefersDark);
  useEffect(() => {
    const listener = e => setDarkMode(e.matches);
    globalThis.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', listener);
    return () => globalThis.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', listener);
  }, []);
  const [selectedSection, setSelectedSection] = useState('Movies');
  const [selectedSystemSub, setSelectedSystemSub] = useState('Tasks');

  // Toast state
  const [toastMessage, setToastMessage] = useState('');

  // Reset search when changing main section (Movies/Series)
  useEffect(() => {
    setSearch('');
  }, [selectedSection]);
  const [selectedSettingsSub, setSelectedSettingsSub] = useState('General');

  // Sonarr series state
  const [series, setSeries] = useState([]);
  const [seriesError, setSeriesError] = useState('');
  const [seriesLoading, setSeriesLoading] = useState(true);

  // Sync sidebar state and page title with route on every navigation
  useEffect(() => {
    const path = location.pathname;
    if (path.startsWith('/settings/')) {
      setSelectedSection('Settings');
      const sub = path.split('/')[2];
      if (sub) {
        setSelectedSettingsSub(sub.charAt(0).toUpperCase() + sub.slice(1));
      }
    } else if (path.startsWith('/settings')) {
      setSelectedSection('Settings');
      setSelectedSettingsSub('General');
    } else if (path.startsWith('/wanted/movies')) {
      setSelectedSection('Wanted');
      setSelectedSettingsSub('Movies');
    } else if (path.startsWith('/wanted/series')) {
      setSelectedSection('Wanted');
      setSelectedSettingsSub('Series');
    } else if (path.startsWith('/blacklist')) {
      setSelectedSection('Blacklist');
    } else if (path === '/' || /^\/[0-9a-zA-Z_-]+$/.exec(path)) {
      setSelectedSection('Movies');
    } else if (path.startsWith('/series')) {
      setSelectedSection('Series');
    } else if (path.startsWith('/history')) {
      setSelectedSection('History');
    } else if (path.startsWith('/system/tasks')) {
      setSelectedSection('System');
      setSelectedSystemSub('Tasks');
    } else if (path.startsWith('/system/logs')) {
      setSelectedSection('System');
      setSelectedSystemSub('Logs');
    }
  }, [location.pathname]);

  // Sonarr series fetch from backend
  useEffect(() => {
    setSeriesLoading(true);
    import('./api').then(({ getSeries }) => {
      getSeries()
        .then(data => {
          setSeries(filterAndSortMedia(data.series));
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
          setMovies(filterAndSortMedia(res.movies));
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
    pageTitle = `${selectedSettingsSub || ''} Settings`;
  } else if (selectedSection === 'Wanted') {
    pageTitle = `Wanted${selectedSettingsSub ? ' ' + selectedSettingsSub : ''}`;
  } else if (selectedSection === 'System') {
    pageTitle = `System${selectedSystemSub ? ' ' + selectedSystemSub : ''}`;
  }

  // Update document title dynamically
  useEffect(() => {
    if (globalThis.setTrailarrTitle) {
      globalThis.setTrailarrTitle(pageTitle);
    }
  }, [pageTitle]);

  // Dynamically load components
  // Removed unused MediaList variable assignment per SonarLint
  const MediaDetails = loadComponent(() => import('./components/MediaDetails'), MediaDetailsRef);
  const Header = loadComponent(() => import('./components/Header'), HeaderRef);
  const Sidebar = loadComponent(() => import('./components/Sidebar'), SidebarRef);
  const GeneralSettings = loadComponent(() => import('./components/GeneralSettings'), GeneralSettingsRef);
  const Tasks = loadComponent(() => import('./components/Tasks'), TasksRef);
  const HistoryPage = loadComponent(() => import('./components/HistoryPage'), HistoryPageRef);
  const Wanted = loadComponent(() => import('./components/Wanted'), WantedRef);
  const SettingsPage = loadComponent(() => import('./components/SettingsPage'), SettingsPageRef);
  const ExtrasSettings = loadComponent(() => import('./components/ExtrasSettings'), ExtrasSettingsRef);
  const LogsPage = loadComponent(() => import('./components/LogsPage'), LogsPageRef);

  // Mobile detection
  const [isMobile, setIsMobile] = useState(window.innerWidth < 900);
  useEffect(() => {
    const handleResize = () => setIsMobile(window.innerWidth < 900);
    window.addEventListener('resize', handleResize);
    return () => window.removeEventListener('resize', handleResize);
  }, []);

  // Sidebar open state for mobile
  const [sidebarOpen, setSidebarOpen] = useState(false);
  const handleSidebarToggle = () => setSidebarOpen((v) => !v);
  const handleSidebarClose = () => setSidebarOpen(false);

  return (
  <div className="app-container" style={{ width: '100vw', minHeight: '100vh', overflowX: 'hidden' }}>
      <Header
        darkMode={darkMode}
        search={search}
        setSearch={setSearch}
        pageTitle={pageTitle}
        mobile={isMobile}
        sidebarOpen={sidebarOpen}
        onSidebarToggle={handleSidebarToggle}
      />
  <div style={{ display: 'flex', width: '100vw', height: 'calc(100vh - 64px)', position: 'relative' }}>
        <Sidebar
          selectedSection={selectedSection}
          setSelectedSection={setSelectedSection}
          selectedSettingsSub={selectedSettingsSub}
          setSelectedSettingsSub={setSelectedSettingsSub}
          darkMode={darkMode}
          selectedSystemSub={selectedSystemSub}
          setSelectedSystemSub={setSelectedSystemSub}
          mobile={isMobile}
          open={sidebarOpen}
          onClose={handleSidebarClose}
          onToggle={handleSidebarToggle}
        />
        <main style={{ flex: 1, padding: '0em', height: '100%', boxSizing: 'border-box', overflowY: 'auto', overflowX: 'hidden', display: 'flex', flexDirection: 'column', alignItems: 'flex-start', justifyContent: 'stretch', maxWidth: '100vw', background: darkMode ? '#18181b' : '#fff', color: darkMode ? '#e5e7eb' : '#222' }}>
          <div style={{ background: darkMode ? '#23232a' : '#fff', boxShadow: darkMode ? '0 1px 4px #222' : '0 1px 4px #e5e7eb', padding: '0em', width: '100%', maxWidth: '100%', flex: 1, overflowY: 'auto', overflowX: 'hidden', color: darkMode ? '#e5e7eb' : '#222' }}>
            <React.Suspense fallback={null}>
              <Routes>
                <Route path="/series" element={<SeriesRouteComponent
                  series={series}
                  search={search}
                  darkMode={darkMode}
                  seriesError={seriesError}
                  getSearchSections={getSearchSections}
                />} />
                <Route path="/" element={<MoviesRouteComponent
                  movies={movies}
                  search={search}
                  darkMode={darkMode}
                  moviesError={moviesError}
                  getSearchSections={getSearchSections}
                />} />
                <Route path="/movies/:id" element={<MediaDetails mediaItems={movies} loading={moviesLoading} mediaType="movie" />} />
                <Route path="/series/:id" element={<MediaDetails mediaItems={series} loading={seriesLoading} mediaType="tv" />} />
                <Route path="/wanted/movies/:id" element={<MediaDetails mediaItems={movies} loading={moviesLoading} mediaType="movie" />} />
                <Route path="/wanted/series/:id" element={<MediaDetails mediaItems={series} loading={seriesLoading} mediaType="tv" />} />
                <Route path="/history/movies/:id" element={<MediaDetails mediaItems={movies} loading={moviesLoading} mediaType="movie" />} />
                <Route path="/history/series/:id" element={<MediaDetails mediaItems={series} loading={seriesLoading} mediaType="tv" />} />
                <Route path="/history" element={<HistoryPage />} />
                <Route path="/wanted/movies" element={<Wanted darkMode={darkMode} type="movie" />} />
                <Route path="/wanted/series" element={<Wanted darkMode={darkMode} type="series" />} />
                <Route path="/settings/radarr" element={<SettingsPage type="radarr"/>} />
                <Route path="/settings/sonarr" element={<SettingsPage type="sonarr"/>} />
                <Route path="/settings/general" element={<GeneralSettings />} />
                <Route path="/settings/extras" element={<ExtrasSettings darkMode={darkMode} />} />
                <Route path="/system/tasks" element={<Tasks />} />
                <Route path="/system/logs" element={<LogsPage />} />
                <Route path="/blacklist" element={<BlacklistPage />} />
              </Routes>
            </React.Suspense>
          </div>
        </main>
      </div>

      {/* Toast Modal */}
      <Toast
        message={toastMessage}
        onClose={() => setToastMessage('')}
        darkMode={darkMode}
      />
    </div>
  );
}

export default App;
