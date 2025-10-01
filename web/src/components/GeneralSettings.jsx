import React, { useState, useEffect } from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faSave } from '@fortawesome/free-solid-svg-icons';

export default function GeneralSettings() {
  const [tmdbKey, setTmdbKey] = useState('');
  const [originalKey, setOriginalKey] = useState('');
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');
  useEffect(() => {
    const setColors = () => {
      const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      document.documentElement.style.setProperty('--settings-bg', isDark ? '#222' : '#fff');
      document.documentElement.style.setProperty('--settings-text', isDark ? '#eee' : '#222');
      document.documentElement.style.setProperty('--save-lane-bg', isDark ? '#333' : '#e5e7eb');
      document.documentElement.style.setProperty('--save-lane-text', isDark ? '#eee' : '#222');
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
      });
  }, []);
  const isChanged = tmdbKey !== originalKey;
  const handleSave = async () => {
    setSaving(true);
    setMessage('');
    try {
      const res = await fetch('/api/settings/general', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tmdbKey })
      });
      if (res.ok) {
        setMessage('Settings saved successfully!');
        setOriginalKey(tmdbKey);
      } else {
        setMessage('Error saving settings.');
      }
    } catch {
      setMessage('Error saving settings.');
    }
    setSaving(false);
  };
  return (
    <div style={{ width: '100%', height: '100%', padding: '2rem', background: 'var(--settings-bg, #fff)', borderRadius: 12, boxShadow: '0 2px 12px #0002', color: 'var(--settings-text, #222)', boxSizing: 'border-box', overflow: 'auto', position: 'relative' }}>
      {/* Save lane */}
      <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', background: 'var(--save-lane-bg, #f3f4f6)', color: 'var(--save-lane-text, #222)', padding: '0.7rem 2rem', display: 'flex', alignItems: 'center', gap: '1rem', borderTopLeftRadius: 12, borderTopRightRadius: 12, zIndex: 10, boxShadow: '0 2px 8px #0001' }}>
        <button onClick={handleSave} disabled={saving || !isChanged} style={{ background: 'none', color: '#222', border: 'none', borderRadius: 6, padding: '0.3rem 1rem', cursor: saving || !isChanged ? 'not-allowed' : 'pointer', opacity: saving || !isChanged ? 0.7 : 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '0.2rem' }}>
          <FontAwesomeIcon icon={faSave} style={{ fontSize: 22, color: '#222' }} />
          <span style={{ fontWeight: 500, fontSize: '0.85em', color: '#222', marginTop: 2, display: 'flex', flexDirection: 'column', alignItems: 'center', lineHeight: 1.1 }}>
            <span>{saving || !isChanged ? 'No' : 'Save'}</span>
            <span>Changes</span>
          </span>
        </button>
        {message && <div style={{ marginLeft: 16, color: message.includes('success') ? '#0f0' : '#f44', fontWeight: 500 }}>{message}</div>}
      </div>
      <div style={{ marginTop: '4.5rem', background: 'var(--settings-bg, #fff)', color: 'var(--settings-text, #222)', borderRadius: 12, boxShadow: '0 1px 4px #0001', padding: '2rem' }}>
        <div style={{ marginBottom: '1.5rem', display: 'block', width: '100%' }}>
          <label style={{ fontWeight: 600, fontSize: '1.15em', marginBottom: 6, display: 'block', textAlign: 'left' }}>TMDB API Key<br />
            <input
              type="text"
              value={tmdbKey}
              onChange={e => setTmdbKey(e.target.value)}
              style={{ width: '60%', minWidth: 220, maxWidth: 600, padding: '0.5rem', borderRadius: 6, border: '1px solid #bbb', background: '#f5f5f5', color: '#222' }}
            />
          </label>
        </div>
      </div>
    </div>
  );
}
