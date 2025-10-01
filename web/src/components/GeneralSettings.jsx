import React, { useState, useEffect } from 'react';

export default function GeneralSettings() {
  const [tmdbKey, setTmdbKey] = useState('');
  const [saving, setSaving] = useState(false);
  const [message, setMessage] = useState('');

  useEffect(() => {
    fetch('/api/settings/general')
      .then(r => r.json())
      .then(data => setTmdbKey(data.tmdbKey || ''));
  }, []);

  const handleSave = async () => {
    setSaving(true);
    setMessage('');
    try {
      const res = await fetch('/api/settings/general', {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ tmdbKey: tmdbKey })
      });
      if (res.ok) setMessage('Saved!');
      else setMessage('Failed to save');
    } catch {
      setMessage('Failed to save');
    }
    setSaving(false);
  };

  return (
    <div style={{ padding: 24 }}>
      <h2>General Settings</h2>
      <label style={{ display: 'block', marginBottom: 8 }}>
        TMDB API Key:
        <input
          type="text"
          value={tmdbKey}
          onChange={e => setTmdbKey(e.target.value)}
          style={{ marginLeft: 8, padding: 4, width: 320 }}
        />
      </label>
      <button onClick={handleSave} disabled={saving} style={{ padding: '6px 18px', fontWeight: 'bold' }}>
        {saving ? 'Saving...' : 'Save'}
      </button>
      {message && <div style={{ marginTop: 12, color: message === 'Saved!' ? 'green' : 'red' }}>{message}</div>}
    </div>
  );
}
