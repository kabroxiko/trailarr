import IconButton from './IconButton.jsx';
import React from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faCog, faFilm, faHistory, faStar, faBan, faServer } from '@fortawesome/free-solid-svg-icons';
import { Link, useLocation, useNavigate } from 'react-router-dom';

export default function Sidebar({ darkMode }) {
  const location = useLocation();
  const navigate = useNavigate();
  const path = location.pathname;
  // Determine selected section and submenus from path
  let selectedSection = '';
  let selectedSettingsSub = '';
  let selectedSystemSub = '';
  if (path === '/' || path.startsWith('/movies')) selectedSection = 'Movies';
  else if (path.startsWith('/series')) selectedSection = 'Series';
  else if (path.startsWith('/history')) selectedSection = 'History';
  else if (path.startsWith('/wanted')) selectedSection = 'Wanted';
  else if (path.startsWith('/blacklist')) selectedSection = 'Blacklist';
  else if (path.startsWith('/settings')) selectedSection = 'Settings';
  else if (path.startsWith('/system')) selectedSection = 'System';

  // Submenus
  if (path.startsWith('/wanted/')) {
    if (path.startsWith('/wanted/movies')) selectedSettingsSub = 'Movies';
    else if (path.startsWith('/wanted/series')) selectedSettingsSub = 'Series';
    else selectedSettingsSub = 'Movies';
  }
  if (path.startsWith('/settings/')) {
    if (path.startsWith('/settings/general')) selectedSettingsSub = 'General';
    else if (path.startsWith('/settings/radarr')) selectedSettingsSub = 'Radarr';
    else if (path.startsWith('/settings/sonarr')) selectedSettingsSub = 'Sonarr';
    else if (path.startsWith('/settings/extras')) selectedSettingsSub = 'Extras';
    else selectedSettingsSub = 'General';
  }
  if (path.startsWith('/system/')) {
    if (path.startsWith('/system/tasks')) selectedSystemSub = 'Tasks';
    else if (path.startsWith('/system/logs')) selectedSystemSub = 'Logs';
  }

  // Local state for submenu expansion
  const [openMenus, setOpenMenus] = React.useState({});
  // Submenus stay open if path matches a submenu section
  React.useEffect(() => {
    // Determine which menu should be open based on path
    let menuToOpen = null;
    if (selectedSection === 'Wanted') menuToOpen = 'Wanted';
    else if (selectedSection === 'Settings') menuToOpen = 'Settings';
    else if (selectedSection === 'System') menuToOpen = 'System';
    if (menuToOpen) {
      setOpenMenus({ [menuToOpen]: true });
    } else {
      setOpenMenus({});
    }
  }, [selectedSection]);

  const isOpen = (menu) => !!openMenus[menu];
  const handleToggle = (menu) => {
    setOpenMenus((prev) => {
      const isOpening = !prev[menu];
      const newState = {};
      if (isOpening) {
        newState[menu] = true;
        // Navigate to first submenu item if opening
        if (menu === 'Wanted') navigate('/wanted/movies');
        if (menu === 'Settings') navigate('/settings/general');
        if (menu === 'System') navigate('/system/tasks');
      }
      return newState;
    });
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
            { name: 'Blacklist', icon: faBan, route: '/blacklist' },
            { name: 'Settings', icon: faCog },
            { name: 'System', icon: faServer }
          ].map(({ name, icon, route }) => (
            <li key={name} style={{ marginBottom: 16 }}>
              {route ? (
                <Link
                  to={route}
                  style={{ textDecoration: 'none', background: selectedSection === name ? (darkMode ? '#333' : '#f3f4f6') : 'none', border: 'none', color: selectedSection === name ? (darkMode ? '#a855f7' : '#6d28d9') : (darkMode ? '#e5e7eb' : '#333'), fontWeight: selectedSection === name ? 'bold' : 'normal', width: '100%', textAlign: 'left', padding: '0.5em 1em', borderRadius: 6, cursor: 'pointer', display: 'flex', alignItems: 'center', gap: '0.75em' }}
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
                  onClick={() => handleToggle(name)}
                >
                  <IconButton icon={<FontAwesomeIcon icon={icon} color={darkMode ? '#e5e7eb' : '#333'} />} style={{ background: 'none', padding: 0, margin: 0, border: 'none' }} />
                  {name}
                </div>
              )}
              {name === 'Wanted' && isOpen('Wanted') && (
                <ul style={{ listStyle: 'none', padding: 0, margin: '8px 0 0 0', background: darkMode ? '#23232a' : '#f3f4f6', borderRadius: 6, color: darkMode ? '#e5e7eb' : '#222', textAlign: 'left' }}>
                  {['Movies', 'Series'].map((submenu) => (
                    <li key={submenu} style={{ padding: '0.5em 1em', borderLeft: selectedSettingsSub === submenu ? '3px solid #a855f7' : '3px solid transparent', background: 'none', color: selectedSettingsSub === submenu ? (darkMode ? '#a855f7' : '#6d28d9') : (darkMode ? '#e5e7eb' : '#333'), fontWeight: selectedSettingsSub === submenu ? 'bold' : 'normal', cursor: 'pointer', textAlign: 'left' }}>
                      <Link
                        to={`/wanted/${submenu.toLowerCase()}`}
                        style={{ color: 'inherit', textDecoration: 'none', display: 'block', width: '100%', textAlign: 'left' }}
                      >{submenu}</Link>
                    </li>
                  ))}
                </ul>
              )}
              {name === 'Settings' && isOpen('Settings') && (
                <ul style={{ listStyle: 'none', padding: 0, margin: '8px 0 0 0', background: darkMode ? '#23232a' : '#f3f4f6', borderRadius: 6, color: darkMode ? '#e5e7eb' : '#222', textAlign: 'left' }}>
                  {['General', 'Radarr', 'Sonarr', 'Extras'].map((submenu) => (
                    <li key={submenu} style={{ padding: '0.5em 1em', borderLeft: selectedSettingsSub === submenu ? '3px solid #a855f7' : '3px solid transparent', background: 'none', color: selectedSettingsSub === submenu ? (darkMode ? '#a855f7' : '#6d28d9') : (darkMode ? '#e5e7eb' : '#333'), fontWeight: selectedSettingsSub === submenu ? 'bold' : 'normal', cursor: 'pointer', textAlign: 'left' }}>
                      <Link
                        to={`/settings/${submenu.toLowerCase()}`}
                        style={{ color: 'inherit', textDecoration: 'none', display: 'block', width: '100%', textAlign: 'left' }}
                      >{submenu}</Link>
                    </li>
                  ))}
                </ul>
              )}
              {name === 'System' && isOpen('System') && (
                <ul style={{ listStyle: 'none', padding: 0, margin: '8px 0 0 0', background: darkMode ? '#23232a' : '#f3f4f6', borderRadius: 6, color: darkMode ? '#e5e7eb' : '#222', textAlign: 'left' }}>
                  {['Tasks', 'Logs'].map((submenu) => (
                    <li key={submenu} style={{ padding: '0.5em 1em', borderLeft: selectedSystemSub === submenu ? '3px solid #a855f7' : '3px solid transparent', background: 'none', color: selectedSystemSub === submenu ? (darkMode ? '#a855f7' : '#6d28d9') : (darkMode ? '#e5e7eb' : '#333'), fontWeight: selectedSystemSub === submenu ? 'bold' : 'normal', cursor: 'pointer', textAlign: 'left' }}>
                      {(submenu === 'Tasks' || submenu === 'Logs') ? (
                        <Link
                          to={submenu === 'Tasks' ? "/system/tasks" : "/system/logs"}
                          style={{ color: 'inherit', textDecoration: 'none', display: 'block', width: '100%', textAlign: 'left' }}
                        >{submenu}</Link>
                      ) : (
                        <span
                          style={{ color: 'inherit', textDecoration: 'none', display: 'block', width: '100%', textAlign: 'left' }}
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
