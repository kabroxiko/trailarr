import React, { useEffect, useState } from 'react';
import axios from 'axios';

const EXTRA_TYPES = [
  { key: 'trailers', label: 'Trailers' },
  { key: 'scenes', label: 'Scenes' },
  { key: 'behindTheScenes', label: 'Behind the Scenes' },
  { key: 'interviews', label: 'Interviews' },
  { key: 'featurettes', label: 'Featurettes' },
  { key: 'deletedScenes', label: 'Deleted Scenes' },
  { key: 'other', label: 'Other' },
];

export default function ExtrasSettings() {
  const [settings, setSettings] = useState({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');

  useEffect(() => {
    axios.get('/api/settings/extratypes')
      .then(res => {
        setSettings(res.data);
        setLoading(false);
      })
      .catch(() => {
        setError('Failed to load settings');
        setLoading(false);
      });
  }, []);

  const handleChange = (key) => {
    setSettings(prev => ({ ...prev, [key]: !prev[key] }));
  };

  const handleSave = () => {
    setSaving(true);
    axios.post('/api/settings/extratypes', settings)
      .then(() => {
        setSaving(false);
      })
      .catch(() => {
        setError('Failed to save settings');
        setSaving(false);
      });
  };

  if (loading) return <div>Loading...</div>;

  return (
    <div style={{ maxWidth: 500, margin: '2em auto', padding: '2em', background: '#fff', borderRadius: 8, boxShadow: '0 2px 8px #0001' }}>
      <h2 style={{ marginBottom: '1em' }}>Extras Download Settings</h2>
      <p>Enable or disable automatic downloads for each extra type:</p>
      <form onSubmit={e => { e.preventDefault(); handleSave(); }}>
        {EXTRA_TYPES.map(({ key, label }) => (
          <div key={key} style={{ display: 'flex', alignItems: 'center', marginBottom: 16 }}>
            <input
              type="checkbox"
              id={key}
              checked={!!settings[key]}
              onChange={() => handleChange(key)}
              style={{ marginRight: 12 }}
            />
            <label htmlFor={key} style={{ fontSize: 16 }}>{label}</label>
          </div>
        ))}
        {error && <div style={{ color: 'red', marginBottom: 12 }}>{error}</div>}
        <button type="submit" disabled={saving} style={{ padding: '0.5em 1.5em', fontSize: 16, background: '#a855f7', color: '#fff', border: 'none', borderRadius: 6, cursor: 'pointer' }}>
          {saving ? 'Saving...' : 'Save'}
        </button>
      </form>
    </div>
  );
}
