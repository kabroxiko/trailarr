import React, { useEffect, useState } from 'react';
import DirectoryPicker from './DirectoryPicker';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faFolderOpen, faSave, faPlug, faCheckCircle, faTimesCircle } from '@fortawesome/free-solid-svg-icons';
import { faTrashAlt } from '@fortawesome/free-regular-svg-icons';

export default function SettingsPage({ type }) {
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState('');
  // type: 'radarr' or 'sonarr'
  const [originalSettings, setOriginalSettings] = useState(null);
  const [settings, setSettings] = useState({ url: '', apiKey: '', pathMappings: [] });
  const [rootFolders, setRootFolders] = useState([]);
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    const setColors = () => {
        const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
        document.documentElement.style.setProperty('--settings-bg', isDark ? '#222' : '#fff');
        document.documentElement.style.setProperty('--settings-text', isDark ? '#eee' : '#222');
        document.documentElement.style.setProperty('--save-lane-bg', isDark ? '#333' : '#e5e7eb');
        document.documentElement.style.setProperty('--save-lane-text', isDark ? '#eee' : '#222');
        document.documentElement.style.setProperty('--settings-input-bg', isDark ? '#333' : '#f5f5f5');
        document.documentElement.style.setProperty('--settings-input-text', isDark ? '#eee' : '#222');
  document.documentElement.style.setProperty('--settings-table-bg', isDark ? '#444' : '#f7f7f7');
  document.documentElement.style.setProperty('--settings-table-text', isDark ? '#f3f3f3' : '#222');
  document.documentElement.style.setProperty('--settings-table-header-bg', isDark ? '#555' : '#ededed');
  document.documentElement.style.setProperty('--settings-table-header-text', isDark ? '#fff' : '#222');
    };
    setColors();
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', setColors);
    return () => {
      window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', setColors);
    };
  }, []);

  useEffect(() => {
    setLoading(true);
    fetch(`/api/settings/${type}`)
      .then(res => res.json())
      .then(async data => {
        // Fetch root folders
        let folders = [];
        if (data.url && data.apiKey) {
          try {
            const res = await fetch(`/api/rootfolders?url=${encodeURIComponent(data.url)}&apiKey=${encodeURIComponent(data.apiKey)}&type=${type}`);
            folders = await res.json();
          } catch {}
        }
        setRootFolders(folders);
        // Create pathMappings for each root folder if not present
        let pathMappings = Array.isArray(data.pathMappings) ? data.pathMappings : [];
        if (folders.length > 0) {
          const folderPaths = folders.map(f => f.path || f);
          pathMappings = folderPaths.map((path, idx) => {
            const existing = pathMappings.find(m => m.from === path);
            return existing || { from: path, to: '' };
          });
        }
        const normalized = {
          ...data,
          pathMappings
        };
        setSettings(normalized);
        setOriginalSettings(normalized);
        setLoading(false);
      });
  }, [type]);

  function isSettingsChanged() {
    if (!originalSettings) return false;
    if (settings.url !== originalSettings.url) return true;
    if (settings.apiKey !== originalSettings.apiKey) return true;
    const a = settings.pathMappings || [];
    const b = originalSettings.pathMappings || [];
    if (a.length !== b.length) return true;
    for (let i = 0; i < a.length; i++) {
      if (a[i].from !== b[i].from || a[i].to !== b[i].to) return true;
    }
    return false;
  }

  const handleChange = e => {
    setSettings({ ...settings, [e.target.name]: e.target.value });
  };

  const handleMappingChange = (idx, field, value) => {
    const updated = settings.pathMappings.map((m, i) => i === idx ? { ...m, [field]: value } : m);
    setSettings({ ...settings, pathMappings: updated });
  };

  const removeMapping = idx => {
    setSettings({ ...settings, pathMappings: settings.pathMappings.filter((_, i) => i !== idx) });
  };

  const saveSettings = async () => {
    setSaving(true);
    setMessage('');
    try {
      const res = await fetch(`/api/settings/${type}`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify(settings)
      });
      if (res.ok) {
        setMessage('Settings saved successfully!');
        setOriginalSettings(settings);
      } else {
        setMessage('Error saving settings.');
      }
    } catch {
      setMessage('Error saving settings.');
    }
    setSaving(false);
  };

  const testConnection = async () => {
    setTesting(true);
    setTestResult('');
    try {
      const res = await fetch(`/api/test/${type}?url=${encodeURIComponent(settings.url)}&apiKey=${encodeURIComponent(settings.apiKey)}`);
      if (res.ok) {
        const data = await res.json();
        if (data.success) {
          setTestResult('Connection successful!');
          // Refresh path mappings after successful test
          try {
            const foldersRes = await fetch(`/api/rootfolders?url=${encodeURIComponent(settings.url)}&apiKey=${encodeURIComponent(settings.apiKey)}&type=${type}`);
            if (foldersRes.ok) {
              const folders = await foldersRes.json();
              setRootFolders(folders);
              // Update pathMappings in settings and originalSettings
              const folderPaths = folders.map(f => f.path || f);
              let pathMappings = Array.isArray(settings.pathMappings) ? settings.pathMappings : [];
              pathMappings = folderPaths.map((path) => {
                const existing = pathMappings.find(m => m.from === path);
                return existing || { from: path, to: '' };
              });
              setSettings(s => ({ ...s, pathMappings }));
              setOriginalSettings(s => ({ ...s, pathMappings }));
            }
          } catch {}
        } else {
          setTestResult(data.error || 'Connection failed.');
        }
      } else {
        setTestResult('Connection failed.');
      }
    } catch {
      setTestResult('Connection failed.');
    }
    setTesting(false);
  };

  return (
  <div style={{ width: '100%', margin: 0, height: '100%', padding: '2rem', background: 'var(--settings-bg, #fff)', borderRadius: 12, boxShadow: '0 2px 12px #0002', color: 'var(--settings-text, #222)', boxSizing: 'border-box', overflowX: 'hidden', overflowY: 'auto', position: 'relative' }}>
      {/* Save lane */}
      <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', background: 'var(--save-lane-bg, #f3f4f6)', color: 'var(--save-lane-text, #222)', padding: '0.7rem 2rem', display: 'flex', alignItems: 'center', gap: '1rem', borderTopLeftRadius: 12, borderTopRightRadius: 12, zIndex: 10, boxShadow: '0 2px 8px #0001' }}>
        <button onClick={saveSettings} disabled={saving || !isSettingsChanged()} style={{ background: 'none', color: '#222', border: 'none', borderRadius: 6, padding: '0.3rem 1rem', cursor: saving || !isSettingsChanged() ? 'not-allowed' : 'pointer', opacity: saving || !isSettingsChanged() ? 0.7 : 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '0.2rem' }}>
          <FontAwesomeIcon icon={faSave} style={{ fontSize: 22, color: 'var(--save-lane-text, #222)' }} />
          <span style={{ fontWeight: 500, fontSize: '0.85em', color: 'var(--save-lane-text, #222)', marginTop: 2, display: 'flex', flexDirection: 'column', alignItems: 'center', lineHeight: 1.1 }}>
            <span>{saving || !isSettingsChanged() ? 'No' : 'Save'}</span>
            <span>Changes</span>
          </span>
        </button>
        {message && <div style={{ marginLeft: 16, color: message.includes('success') ? '#0f0' : '#f44', fontWeight: 500 }}>{message}</div>}
      </div>
  <div style={{ marginTop: '4.5rem', background: 'var(--settings-bg, #fff)', color: 'var(--settings-text, #222)', borderRadius: 12, boxShadow: '0 1px 4px #0001', padding: '2rem' }}>
        {loading ? (
          <div style={{ textAlign: 'center', margin: '2rem' }}>Loading...</div>
        ) : (
          <>
            <div style={{ marginBottom: '1.5rem', display: 'block', width: '100%' }}>
              <div style={{ width: '100%', marginBottom: '1.2rem' }}>
                <label style={{ fontWeight: 600, fontSize: '1.15em', marginBottom: 6, display: 'block', textAlign: 'left' }}>{type === 'radarr' ? 'Radarr URL' : 'Sonarr URL'}<br />
                  <input name="url" value={settings.url} onChange={handleChange} style={{ width: '60%', minWidth: 220, maxWidth: 600, padding: '0.5rem', borderRadius: 6, border: '1px solid #bbb', background: 'var(--settings-input-bg, #f5f5f5)', color: 'var(--settings-input-text, #222)' }} />
                </label>
              </div>
              <div style={{ width: '100%', display: 'flex', flexDirection: 'column', alignItems: 'flex-start', gap: '0.7rem' }}>
                <label style={{ fontWeight: 600, fontSize: '1.15em', marginBottom: 6, display: 'block', textAlign: 'left', width: '100%' }}>API Key<br />
                  <input name="apiKey" value={settings.apiKey} onChange={handleChange} style={{ width: '60%', minWidth: 220, maxWidth: 600, padding: '0.5rem', borderRadius: 6, border: '1px solid #bbb', background: 'var(--settings-input-bg, #f5f5f5)', color: 'var(--settings-input-text, #222)' }} />
                </label>
                <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', width: '100%' }}>
                  <span
                    role="button"
                    tabIndex={0}
                    onClick={testConnection}
                    onKeyDown={e => { if ((e.key === 'Enter' || e.key === ' ') && !testing && settings.url && settings.apiKey) testConnection(); }}
                    title="Test Connection"
                    aria-label="Test Connection"
                    style={{
                      cursor: testing || !settings.url || !settings.apiKey ? 'not-allowed' : 'pointer',
                      opacity: testing || !settings.url || !settings.apiKey ? 0.6 : 1,
                      display: 'inline-flex',
                      alignItems: 'center',
                      justifyContent: 'center',
                      background: 'none',
                      border: 'none',
                      padding: 0,
                      margin: 0,
                      outline: 'none'
                    }}
                  >
                    <span style={{ position: 'relative', display: 'inline-block', width: 22, height: 22 }}>
                      <FontAwesomeIcon
                        icon={faPlug}
                        style={{
                          fontSize: 22,
                          color: 'var(--settings-text, #222)',
                          transition: 'color 0.2s',
                          position: 'absolute',
                          left: 0,
                          top: 0
                        }}
                      />
                      {testResult && testResult.includes('success') && (
                        <FontAwesomeIcon
                          icon={faCheckCircle}
                          style={{
                            fontSize: 13,
                            color: '#0a0',
                            position: 'absolute',
                            right: -8,
                            bottom: -8,
                            pointerEvents: 'none',
                            background: 'var(--settings-bg, #fff)',
                            borderRadius: '50%'
                          }}
                        />
                      )}
                      {testResult && !testResult.includes('success') && (
                        <FontAwesomeIcon
                          icon={faTimesCircle}
                          style={{
                            fontSize: 13,
                            color: '#c00',
                            position: 'absolute',
                            right: -8,
                            bottom: -8,
                            pointerEvents: 'none',
                            background: 'var(--settings-bg, #fff)',
                            borderRadius: '50%'
                          }}
                        />
                      )}
                    </span>
                  </span>
                  {testResult && (
                    <span style={{ marginLeft: 12, color: testResult.includes('success') ? '#0a0' : '#c00', fontWeight: 500 }}>{testResult}</span>
                  )}
                </div>
              </div>
            </div>
            <h3 style={{ margin: '2rem 0 1rem', textAlign: 'left' }}>Path Mappings</h3>
            <table style={{ width: '100%', minWidth: 300, maxWidth: 620, marginLeft: 0, marginRight: 'auto', borderCollapse: 'collapse', background: 'var(--settings-table-bg, #f5f5f5)', borderRadius: 8, overflow: 'hidden', marginTop: '1rem', color: 'var(--settings-table-text, #222)' }}>
              <thead>
                  <tr style={{ background: 'var(--settings-table-header-bg, #eaeaea)', color: 'var(--settings-table-header-text, #222)' }}>
                  <th style={{ padding: '0.5rem', textAlign: 'left' }}>{type === 'radarr' ? 'Radarr Path' : 'Sonarr Path'}</th>
                  <th style={{ padding: '0.5rem', textAlign: 'left' }}>Trailarr Path</th>
                  <th style={{ padding: '0.5rem' }}></th>
                </tr>
              </thead>
              <tbody>
                {(Array.isArray(settings.pathMappings) ? settings.pathMappings : []).map((m, idx) => (
                  <tr key={m.from + '-' + idx}>
                    <td style={{ textAlign: 'left' }}>
                      <input value={m.from} onChange={e => handleMappingChange(idx, 'from', e.target.value)} placeholder={type === 'radarr' ? 'Radarr path' : 'Sonarr path'} style={{ width: '90%', minWidth: 210, maxWidth: 500, padding: '0.4rem', borderRadius: 4, border: '1px solid #bbb', background: 'var(--settings-input-bg, #f5f5f5)', color: 'var(--settings-input-text, #222)' }} />
                    </td>
                    <td style={{ textAlign: 'left' }}>
                      <div style={{ display: 'flex', alignItems: 'center', gap: '0.5rem', height: '100%' }}>
                        <DirectoryPicker
                          value={m.to}
                          onChange={path => handleMappingChange(idx, 'to', path)}
                          label={null}
                          disabled={saving || loading}
                          icon={<FontAwesomeIcon icon={faFolderOpen} style={{ fontSize: 20, background: 'none', padding: 0, margin: 0, border: 'none' }} />}
                        />
                      </div>
                    </td>
                    <td style={{ textAlign: 'left' }}>
                        <span
                          role="button"
                          tabIndex={0}
                          onClick={() => removeMapping(idx)}
                          onKeyDown={e => { if (e.key === 'Enter' || e.key === ' ') removeMapping(idx); }}
                          title="Remove"
                          style={{ cursor: 'pointer', display: 'inline-flex', alignItems: 'center', height: '100%', justifyContent: 'center', verticalAlign: 'middle' }}
                          aria-label="Remove path mapping"
                        >
                          <FontAwesomeIcon icon={faTrashAlt} style={{ fontSize: 20, color: 'var(--settings-text, #222)', filter: 'drop-shadow(0 1px 2px #0002)', alignSelf: 'center' }} />
                        </span>
                    </td>
                  </tr>
                ))}
              </tbody>
            </table>
          </>
        )}
      </div>
    </div>
  );
}
