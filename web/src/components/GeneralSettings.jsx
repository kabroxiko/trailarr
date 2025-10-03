import React, { useState, useEffect } from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faSave, faPlug, faCheckCircle, faTimesCircle } from '@fortawesome/free-solid-svg-icons';

export default function GeneralSettings() {
  const [testing, setTesting] = useState(false);
  const [testResult, setTestResult] = useState('');
  const [tmdbKey, setTmdbKey] = useState('');
  const [autoDownloadExtras, setAutoDownloadExtras] = useState(true);
  const [originalKey, setOriginalKey] = useState('');
  const [originalAutoDownload, setOriginalAutoDownload] = useState(true);
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
    };
    setColors();
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', setColors);
    return () => {
      window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', setColors);
    };
  }, []);
  useEffect(() => {
    fetch('/api/settings/general')
      .then(r => r.json())
      .then(data => {
        setTmdbKey(data.tmdbKey || '');
        setOriginalKey(data.tmdbKey || '');
        setAutoDownloadExtras(data.autoDownloadExtras !== false); // default true
        setOriginalAutoDownload(data.autoDownloadExtras !== false);
      });
  }, []);
  const isChanged = tmdbKey !== originalKey || autoDownloadExtras !== originalAutoDownload;

  const testTmdbKey = async () => {
    setTesting(true);
    setTestResult('');
    try {
      const res = await fetch(`/api/test/tmdb?apiKey=${encodeURIComponent(tmdbKey)}`);
      if (res.ok) {
        const data = await res.json();
        if (data.success) {
          setTestResult('Connection successful!');
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

  const handleSave = async () => {
    setSaving(true);
    setMessage('');
    try {
      const res = await fetch('/api/settings/general', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tmdbKey, autoDownloadExtras })
      });
      if (res.ok) {
        setMessage('Settings saved successfully!');
        setOriginalKey(tmdbKey);
        setOriginalAutoDownload(autoDownloadExtras);
      } else {
        setMessage('Error saving settings.');
      }
    } catch {
      setMessage('Error saving settings.');
    }
    setSaving(false);
  };
  return (
  <div style={{ width: '100%', margin: 0, height: '100%', padding: '2rem', background: 'var(--settings-bg, #fff)', borderRadius: 12, boxShadow: '0 2px 12px #0002', color: 'var(--settings-text, #222)', boxSizing: 'border-box', overflowX: 'hidden', overflowY: 'auto', position: 'relative' }}>
      {/* Save lane */}
      <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', background: 'var(--save-lane-bg, #f3f4f6)', color: 'var(--save-lane-text, #222)', padding: '0.7rem 2rem', display: 'flex', alignItems: 'center', gap: '1rem', borderTopLeftRadius: 12, borderTopRightRadius: 12, zIndex: 10, boxShadow: '0 2px 8px #0001' }}>
        <button onClick={handleSave} disabled={saving || !isChanged} style={{ background: 'none', color: '#222', border: 'none', borderRadius: 6, padding: '0.3rem 1rem', cursor: saving || !isChanged ? 'not-allowed' : 'pointer', opacity: saving || !isChanged ? 0.7 : 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '0.2rem' }}>
          <FontAwesomeIcon icon={faSave} style={{ fontSize: 22, color: 'var(--save-lane-text, #222)' }} />
          <span style={{ fontWeight: 500, fontSize: '0.85em', color: 'var(--save-lane-text, #222)', marginTop: 2, display: 'flex', flexDirection: 'column', alignItems: 'center', lineHeight: 1.1 }}>
            <span>{saving || !isChanged ? 'No' : 'Save'}</span>
            <span>Changes</span>
          </span>
        </button>
        {message && <div style={{ marginLeft: 16, color: message.includes('success') ? '#0f0' : '#f44', fontWeight: 500 }}>{message}</div>}
      </div>
      <div style={{ marginTop: '4.5rem', background: 'var(--settings-bg, #fff)', color: 'var(--settings-text, #222)', borderRadius: 12, boxShadow: '0 1px 4px #0001', padding: '2rem' }}>
        <div style={{ marginBottom: '1.5rem', display: 'block', width: '100%' }}>
          <label style={{ fontWeight: 600, fontSize: '1.15em', marginBottom: 6, display: 'block', textAlign: 'left' }}>TMDB API Key<br />
            <div style={{ width: '100%' }}>
              <input
                type="text"
                value={tmdbKey}
                onChange={e => setTmdbKey(e.target.value)}
                style={{ width: '60%', minWidth: 220, maxWidth: 600, padding: '0.5rem', borderRadius: 6, border: '1px solid #bbb', background: 'var(--settings-input-bg, #f5f5f5)', color: 'var(--settings-input-text, #222)' }}
              />
              <div style={{ marginTop: '0.7rem', display: 'flex', flexDirection: 'column', alignItems: 'flex-start', gap: '0.5rem', width: '60%' }}>
                <div style={{ display: 'flex', alignItems: 'center', gap: '1rem', width: '100%' }}>
                  <span
                    role="button"
                    tabIndex={0}
                    onClick={testTmdbKey}
                    onKeyDown={e => { if ((e.key === 'Enter' || e.key === ' ') && !testing && tmdbKey) testTmdbKey(); }}
                    title="Test TMDB Key"
                    aria-label="Test TMDB Key"
                    style={{
                      cursor: testing || !tmdbKey ? 'not-allowed' : 'pointer',
                      opacity: testing || !tmdbKey ? 0.6 : 1,
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
                    <span style={{ color: testResult.includes('success') ? '#0a0' : '#c00', fontWeight: 500 }}>{testResult}</span>
                  )}
                </div>
              </div>
            </div>
          </label>
        </div>
        <div style={{ marginBottom: '1.5rem', display: 'block', width: '100%' }}>
          <label style={{ fontWeight: 600, fontSize: '1.15em', marginBottom: 6, display: 'block', textAlign: 'left' }}>
            <input
              type="checkbox"
              checked={autoDownloadExtras}
              onChange={e => setAutoDownloadExtras(e.target.checked)}
              style={{ marginRight: 8 }}
            />
            Enable automatic download of extras
          </label>
        </div>
      </div>
    </div>
  );
}
