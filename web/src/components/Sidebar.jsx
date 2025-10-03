import IconButton from './IconButton.jsx';
import React from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faCog, faFilm, faHistory, faStar, faBan, faServer } from '@fortawesome/free-solid-svg-icons';
import { Link, useLocation } from 'react-router-dom';

export default function Sidebar({ selectedSection, setSelectedSection, selectedSettingsSub, setSelectedSettingsSub, darkMode, selectedSystemSub, setSelectedSystemSub }) {
  const location = useLocation();
  // Helper to toggle expandable menus
  const handleSectionClick = (name) => {
    if (selectedSection === name) {
      setSelectedSection(''); // Collapse if already selected
    } else {
      setSelectedSection(name);
    }
  };

  return (
    <aside style={{ width: 220, background: darkMode ? '#23232a' : '#fff', borderRight: darkMode ? '1px solid #333' : '1px solid #e5e7eb', padding: '0em 0', height: '100%', boxSizing: 'border-box' }}>
      <nav>
        <ul style={{ listStyle: 'none', padding: 0, margin: 0 }}>
          {[
            { name: 'Movies', icon: faFilm, route: '/' },
            { name: 'Series', icon: faCog, route: '/series' },
            { name: 'History', icon: faHistory, route: '/history' },
            { name: 'Wanted', icon: faStar },
            { name: 'Blacklist', icon: faBan },
            { name: 'Settings', icon: faCog },
            { name: 'System', icon: faServer }
          ].map(({ name, icon, route }) => (
            <li key={name} style={{ marginBottom: 16 }}>
              {route ? (
                <Link
                  to={route}
                  style={{ textDecoration: 'none', background: selectedSection === name ? (darkMode ? '#333' : '#f3f4f6') : 'none', border: 'none', color: selectedSection === name ? (darkMode ? '#a855f7' : '#6d28d9') : (darkMode ? '#e5e7eb' : '#333'), fontWeight: selectedSection === name ? 'bold' : 'normal', width: '100%', textAlign: 'left', padding: '0.5em 1em', borderRadius: 6, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '0.75em' }}
                  onClick={() => setSelectedSection(name)}
                >
                  <IconButton icon={<FontAwesomeIcon icon={icon} color={darkMode ? '#e5e7eb' : '#333'} />} style={{ background: 'none', padding: 0, margin: 0, border: 'none' }} />
                  {name}
                </Link>
              ) : (
                <div
                  style={{
                    textDecoration: 'none',
                    background: selectedSection === name ? (darkMode ? '#333' : '#f3f4f6') : 'none',
                    border: 'none',
                    color: selectedSection === name ? (darkMode ? '#a855f7' : '#6d28d9') : (darkMode ? '#e5e7eb' : '#333'),
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
                  onClick={() => handleSectionClick(name)}
                >
                  <IconButton icon={<FontAwesomeIcon icon={icon} color={darkMode ? '#e5e7eb' : '#333'} />} style={{ background: 'none', padding: 0, margin: 0, border: 'none' }} />
                  {name}
                </div>
              )}
              {name === 'Wanted' && selectedSection === 'Wanted' && (
                <ul style={{ listStyle: 'none', padding: 0, margin: '8px 0 0 0', background: darkMode ? '#23232a' : '#f3f4f6', borderRadius: 6, color: darkMode ? '#e5e7eb' : '#222', textAlign: 'left' }}>
                  {['Movies', 'Series'].map((submenu) => (
                    <li key={submenu} style={{ padding: '0.5em 1em', borderLeft: selectedSettingsSub === submenu ? '3px solid #a855f7' : '3px solid transparent', background: 'none', color: selectedSettingsSub === submenu ? (darkMode ? '#a855f7' : '#6d28d9') : (darkMode ? '#e5e7eb' : '#333'), fontWeight: selectedSettingsSub === submenu ? 'bold' : 'normal', cursor: 'pointer', textAlign: 'left' }}>
                      <Link
                        to={`/wanted/${submenu.toLowerCase()}`}
                        style={{ color: 'inherit', textDecoration: 'none', display: 'block', width: '100%', textAlign: 'left' }}
                        onClick={() => setSelectedSettingsSub(submenu)}
                      >{submenu}</Link>
                    </li>
                  ))}
                </ul>
              )}
              {name === 'Settings' && selectedSection === 'Settings' && (
                <ul style={{ listStyle: 'none', padding: 0, margin: '8px 0 0 0', background: darkMode ? '#23232a' : '#f3f4f6', borderRadius: 6, color: darkMode ? '#e5e7eb' : '#222', textAlign: 'left' }}>
                  {['General', 'Radarr', 'Sonarr', 'Extras'].map((submenu, idx) => (
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
              {name === 'System' && selectedSection === 'System' && (
                <ul style={{ listStyle: 'none', padding: 0, margin: '8px 0 0 0', background: darkMode ? '#23232a' : '#f3f4f6', borderRadius: 6, color: darkMode ? '#e5e7eb' : '#222', textAlign: 'left' }}>
                  {['Tasks', 'Logs', 'Providers', 'Backups', 'Status', 'Releases'].map((submenu) => (
                    <li key={submenu} style={{ padding: '0.5em 1em', borderLeft: selectedSystemSub === submenu ? '3px solid #a855f7' : '3px solid transparent', background: 'none', color: selectedSystemSub === submenu ? (darkMode ? '#a855f7' : '#6d28d9') : (darkMode ? '#e5e7eb' : '#333'), fontWeight: selectedSystemSub === submenu ? 'bold' : 'normal', cursor: 'pointer', textAlign: 'left' }}>
                      {submenu === 'Tasks' ? (
                        <Link
                          to="/system/tasks"
                          style={{ color: 'inherit', textDecoration: 'none', display: 'block', width: '100%', textAlign: 'left' }}
                          onClick={() => setSelectedSystemSub(submenu)}
                        >{submenu}</Link>
                      ) : (
                        <span
                          style={{ color: 'inherit', textDecoration: 'none', display: 'block', width: '100%', textAlign: 'left' }}
                          onClick={() => setSelectedSystemSub(submenu)}
                        >{submenu}</span>
                      )}
                    </li>
                  ))}
                </ul>
              )}
            </li>
          ))}
        </ul>
      </nav>
    </aside>
  );
}
