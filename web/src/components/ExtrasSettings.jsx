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

const YTDLP_FLAGS = [
  { key: 'quiet', label: 'Quiet (no output)', type: 'boolean' },
  { key: 'noprogress', label: 'No Progress Bar', type: 'boolean' },
  { key: 'writesubs', label: 'Write Subs', type: 'boolean' },
  { key: 'writeautosubs', label: 'Write Auto Subs', type: 'boolean' },
  { key: 'embedsubs', label: 'Embed Subs', type: 'boolean' },
  { key: 'remuxvideo', label: 'Remux Video', type: 'string' },
  { key: 'subformat', label: 'Subtitle Format', type: 'string' },
  { key: 'sublangs', label: 'Subtitle Languages', type: 'string' },
  { key: 'requestedformats', label: 'Requested Formats', type: 'string' },
  { key: 'timeout', label: 'Timeout (s)', type: 'number' },
  { key: 'sleepInterval', label: 'Sleep Interval (s)', type: 'number' },
  { key: 'maxDownloads', label: 'Max Downloads', type: 'number' },
  { key: 'limitRate', label: 'Limit Rate', type: 'string' },
  { key: 'sleepRequests', label: 'Sleep Requests', type: 'number' },
  { key: 'maxSleepInterval', label: 'Max Sleep Interval (s)', type: 'number' },
];

export default function ExtrasSettings() {
  const [settings, setSettings] = useState({});
  const [ytFlags, setYtFlags] = useState({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [ytError, setYtError] = useState('');
  const [ytSaving, setYtSaving] = useState(false);

  useEffect(() => {
    setLoading(true);
    Promise.all([
      axios.get('/api/settings/extratypes'),
      axios.get('/api/settings/ytdlpflags'),
    ])
      .then(([extrasRes, ytRes]) => {
        setSettings(extrasRes.data);
        setYtFlags(ytRes.data);
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

  const handleYtFlagChange = (key, value) => {
    setYtFlags(prev => ({ ...prev, [key]: value }));
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

  const handleYtSave = () => {
    setYtSaving(true);
    axios.post('/api/settings/ytdlpflags', ytFlags)
      .then(() => {
        setYtSaving(false);
      })
      .catch(() => {
        setYtError('Failed to save yt-dlp flags');
        setYtSaving(false);
      });
  };

  if (loading) return <div>Loading...</div>;

  return (
    <div style={{ maxWidth: 600, margin: '2em auto', padding: '2em', background: '#fff', borderRadius: 8, boxShadow: '0 2px 8px #0001' }}>
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
      <hr style={{ margin: '2em 0' }} />
      <h3 style={{ marginBottom: '1em' }}>yt-dlp Download Flags</h3>
      <form onSubmit={e => { e.preventDefault(); handleYtSave(); }}>
        {YTDLP_FLAGS.map(({ key, label, type }) => (
          <div key={key} style={{ display: 'flex', alignItems: 'center', marginBottom: 16 }}>
            {type === 'boolean' ? (
              <>
                <input
                  type="checkbox"
                  id={key}
                  checked={!!ytFlags[key]}
                  onChange={() => handleYtFlagChange(key, !ytFlags[key])}
                  style={{ marginRight: 12 }}
                />
                <label htmlFor={key} style={{ fontSize: 16 }}>{label}</label>
              </>
            ) : (
              <>
                <label htmlFor={key} style={{ fontSize: 16, minWidth: 180 }}>{label}</label>
                <input
                  type={type === 'number' ? 'number' : 'text'}
                  id={key}
                  value={ytFlags[key] ?? ''}
                  onChange={e => handleYtFlagChange(key, type === 'number' ? Number(e.target.value) : e.target.value)}
                  style={{ marginLeft: 12, flex: 1, padding: '0.3em 0.7em', fontSize: 15, border: '1px solid #ccc', borderRadius: 4 }}
                />
              </>
            )}
          </div>
        ))}
        {ytError && <div style={{ color: 'red', marginBottom: 12 }}>{ytError}</div>}
        <button type="submit" disabled={ytSaving} style={{ padding: '0.5em 1.5em', fontSize: 16, background: '#2563eb', color: '#fff', border: 'none', borderRadius: 6, cursor: 'pointer' }}>
          {ytSaving ? 'Saving...' : 'Save yt-dlp Flags'}
        </button>
      </form>
    </div>
  );
}
