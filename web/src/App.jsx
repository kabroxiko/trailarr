
import { useState, useEffect } from 'react';
import './App.css';
import { searchExtras, downloadExtra, fetchPlexItems } from './api';

function App() {
  const [plexItems, setPlexItems] = useState([]);
  const [plexError, setPlexError] = useState('');
  const [selectedSection, setSelectedSection] = useState('Movies');

  useEffect(() => {
    fetchPlexItems()
      .then(res => setPlexItems(res.items || []))
      .catch(e => setPlexError(e.message));
  }, []);

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
                        borderLeft: idx === 1 ? '3px solid #d6b4f7' : '3px solid transparent',
                        background: idx === 1 ? '#fff' : 'none',
                        color: idx === 1 ? '#a855f7' : '#333',
                        fontWeight: idx === 1 ? 'bold' : 'normal',
                        cursor: 'pointer',
                      }}>{submenu}</li>
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
  <div style={{ background: '#fff', borderRadius: 8, boxShadow: '0 1px 4px #e5e7eb', padding: '1em', width: '100%', maxWidth: '100%', flex: 1, overflowY: 'auto', overflowX: 'hidden' }}>
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
        </div>
      </main>
    </div>
  );
}

export default App;
