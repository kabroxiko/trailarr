import React, { useEffect, useState, Suspense } from 'react';
import Select from 'react-select';
import axios from 'axios';
import Container from './Container.jsx';
import SaveLane from './SaveLane.jsx';
import SectionHeader from './SectionHeader.jsx';

const ExtrasTypeMappingConfig = React.lazy(() => import('./ExtrasTypeMappingConfig.jsx'));

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

export default function ExtrasSettings({ darkMode }) {
  const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
  useEffect(() => {
    const setColors = () => {
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
  }, [darkMode]);
  const [settings, setSettings] = useState({});
  const [ytFlags, setYtFlags] = useState({});
  const [loading, setLoading] = useState(true);
  const [saving, setSaving] = useState(false);
  const [error, setError] = useState('');
  const [ytError, setYtError] = useState('');
  const [ytSaving, setYtSaving] = useState(false);
  const [tmdbTypes, setTmdbTypes] = useState([]);
  const [plexTypes, setPlexTypes] = useState([]);
  const [mapping, setMapping] = useState({});

  useEffect(() => {
    setLoading(true);
    Promise.all([
      axios.get('/api/tmdb/extratypes'),
      axios.get('/api/settings/extratypes'),
      axios.get('/api/settings/canonicalizeextratype'),
      axios.get('/api/settings/ytdlpflags'),
    ])
      .then(([tmdbRes, plexRes, mapRes, ytRes]) => {
        setTmdbTypes(tmdbRes.data.tmdbExtraTypes || []);
        setPlexTypes(Object.keys(plexRes.data));
        const initialMapping = { ...mapRes.data.mapping };
        tmdbRes.data.tmdbExtraTypes.forEach(type => {
          if (!initialMapping[type]) {
            initialMapping[type] = "Other";
          }
        });
        setMapping(initialMapping);
        setSettings(plexRes.data);
        setYtFlags(ytRes.data);
        setLoading(false);
      })
      .catch(() => {
        setError('Failed to load settings');
        setLoading(false);
      });
  }, [darkMode]);

  const handleMappingChange = (newMapping) => {
    setMapping(newMapping);
  };

  const handleChange = (key) => {
    setSettings(prev => ({ ...prev, [key]: !prev[key] }));
  };

  const handleYtFlagChange = (key, value) => {
    setYtFlags(prev => ({ ...prev, [key]: value }));
  };

  const handleSave = async () => {
    setSaving(true);
    try {
      await axios.post('/api/settings/extratypes', settings);
      await axios.post('/api/settings/canonicalizeextratype', { mapping });
      setSaving(false);
    } catch {
      setError('Failed to save settings or mapping');
      setSaving(false);
    }
  };

  const handleYtSave = () => {
    if (
      typeof ytFlags.maxSleepInterval === 'number' &&
      typeof ytFlags.sleepInterval === 'number' &&
      ytFlags.maxSleepInterval < ytFlags.sleepInterval
    ) {
      setYtError('Max Sleep Interval must not be lower than Sleep Interval.');
      return;
    }
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

  // Save lane logic
  const isChanged = EXTRA_TYPES.some(({ key }) => settings[key] !== undefined && settings[key] !== false) || Object.keys(ytFlags).length > 0;

  return (
    <Container>
      {/* Save lane */}
      <SaveLane onSave={handleSave} saving={saving} isChanged={isChanged} error={error} />
      <div style={{ marginTop: '4.5rem', background: 'var(--settings-bg, #fff)', color: 'var(--settings-text, #222)', borderRadius: 12, boxShadow: '0 1px 4px #0001', padding: '2rem' }}>
        <SectionHeader>Extra Types</SectionHeader>
        <div style={{ marginBottom: '2em' }}>
          <Select
            isMulti
            options={EXTRA_TYPES.map(({ key, label }) => ({ value: key, label }))}
            value={EXTRA_TYPES.filter(({ key }) => settings[key]).map(({ key, label }) => ({ value: key, label }))}
            onChange={selected => {
              const newSettings = { ...settings };
              EXTRA_TYPES.forEach(({ key }) => { newSettings[key] = false; });
              selected.forEach(({ value }) => { newSettings[value] = true; });
              setSettings(newSettings);
            }}
            styles={{
              control: (base, state) => ({
                ...base,
                background: isDark ? '#23232a' : '#fff',
                borderColor: state.isFocused ? '#a855f7' : '#444',
                boxShadow: state.isFocused ? '0 0 0 2px #a855f7' : 'none',
                color: isDark ? '#fff' : '#222',
                borderRadius: 8,
                minHeight: 32,
                fontSize: 13,
                padding: '0 4px',
                maxWidth: 480,
              }),
              valueContainer: base => ({
                ...base,
                padding: '2px 4px',
              }),
              indicatorsContainer: base => ({
                ...base,
                height: 32,
              }),
              multiValue: base => ({
                ...base,
                background: isDark ? '#333' : '#e5e7eb',
                color: isDark ? '#fff' : '#222',
                borderRadius: 6,
                fontSize: 13,
                height: 24,
                margin: '2px 2px',
                display: 'flex',
                alignItems: 'center',
              }),
              multiValueLabel: base => ({
                ...base,
                color: isDark ? '#fff' : '#222',
                fontWeight: 500,
                fontSize: 13,
                padding: '0 6px',
              }),
              multiValueRemove: base => ({
                ...base,
                color: isDark ? '#a855f7' : '#6d28d9',
                fontSize: 13,
                height: 24,
                ':hover': { background: isDark ? '#a855f7' : '#6d28d9', color: '#fff' },
              }),
              menu: base => ({
                ...base,
                background: isDark ? '#23232a' : '#fff',
                color: isDark ? '#fff' : '#222',
                borderRadius: 8,
                fontSize: 13,
              }),
              option: (base, state) => ({
                ...base,
                background: state.isSelected ? (isDark ? '#a855f7' : '#6d28d9') : (state.isFocused ? (isDark ? '#333' : '#eee') : (isDark ? '#23232a' : '#fff')),
                color: state.isSelected ? '#fff' : (isDark ? '#fff' : '#222'),
                fontWeight: state.isSelected ? 600 : 400,
                fontSize: 13,
                height: 32,
                display: 'flex',
                alignItems: 'center',
                lineHeight: 'normal',
              }),
            }}
            placeholder="Select extra types..."
            closeMenuOnSelect={false}
            hideSelectedOptions={false}
            menuPortalTarget={document.body}
          />
        </div>
        {/* Mapping config UI integration */}
        <Suspense fallback={<div>Loading mapping config...</div>}>
          <ExtrasTypeMappingConfig
            isDark={isDark}
            mapping={mapping}
            onMappingChange={handleMappingChange}
            tmdbTypes={tmdbTypes}
            plexTypes={plexTypes.map(key => {
              switch (key) {
                case "behindTheScenes": return "Behind The Scenes";
                case "deletedScenes": return "Deleted Scenes";
                case "featurettes": return "Featurettes";
                case "interviews": return "Interviews";
                case "scenes": return "Scenes";
                case "shorts": return "Shorts";
                case "trailers": return "Trailers";
                case "other": return "Other";
                default: return key;
              }
            })}
          />
        </Suspense>
        <hr style={{ margin: '2em 0', borderColor: isDark ? '#444' : '#eee' }} />
        <SectionHeader>yt-dlp Download Flags</SectionHeader>
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
                    style={{ marginRight: 12, accentColor: isDark ? '#2563eb' : '#6d28d9' }}
                  />
                  <label htmlFor={key} style={{ fontSize: 16 }}>{label}</label>
                </>
              ) : (
                <>
                  <label htmlFor={key} style={{ fontSize: 16, minWidth: 180, textAlign: 'left', width: 180 }}>{label}</label>
                  <input
                    type={type === 'number' ? 'number' : 'text'}
                    id={key}
                    value={ytFlags[key] ?? ''}
                    onChange={e => handleYtFlagChange(key, type === 'number' ? Number(e.target.value) : e.target.value)}
                    style={{
                      marginLeft: 12,
                      width: 120,
                      minWidth: 80,
                      maxWidth: 160,
                      padding: '0.15em 0.5em',
                      fontSize: 13,
                      border: '1px solid',
                      borderColor: isDark ? '#444' : '#ccc',
                      borderRadius: 4,
                      background: isDark ? '#23232a' : '#fff',
                      color: isDark ? '#e5e7eb' : '#222',
                    }}
                  />
                </>
              )}
            </div>
          ))}
        </form>
        {ytError && <div style={{ color: 'red', marginBottom: 12 }}>{ytError}</div>}
      </div>
    </Container>
  );
}
