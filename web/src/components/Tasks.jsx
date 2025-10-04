import React, { useEffect, useState } from 'react';
import { FaArrowsRotate } from 'react-icons/fa6';

const getStyles = (darkMode) => ({
  table: {
    width: '100%',
    marginBottom: '2em',
    borderCollapse: 'collapse',
    background: darkMode ? '#23272f' : '#f6f7f9',
    color: darkMode ? '#eee' : '#222',
    fontSize: '15px',
  },
  th: {
    textAlign: 'left',
    padding: '0.75em 0.5em',
    fontWeight: 500,
    background: darkMode ? '#23272f' : '#f6f7f9',
    borderBottom: darkMode ? '1px solid #444' : '1px solid #e5e7eb',
    color: darkMode ? '#eee' : '#222',
  },
  td: {
    padding: '0.75em 0.5em',
    borderBottom: darkMode ? '1px solid #444' : '1px solid #e5e7eb',
    background: darkMode ? '#181a20' : '#fff',
    textAlign: 'left',
    color: darkMode ? '#eee' : '#222',
  },
  header: {
    fontSize: '1.4em',
    fontWeight: 600,
    margin: '0 0 1em 0',
    color: darkMode ? '#eee' : '#222',
  },
  container: {
    padding: '2em',
    background: darkMode ? '#181a20' : '#f6f7f9',
    minHeight: '100vh',
    color: darkMode ? '#eee' : '#222',
  },
});

export default function Tasks() {
  const [status, setStatus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [spinning, setSpinning] = useState({});
  const [rotation, setRotation] = useState({});
  const [darkMode, setDarkMode] = useState(false);

  useEffect(() => {
    // Detect dark mode
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    setDarkMode(mq.matches);
    const handler = (e) => setDarkMode(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);

  useEffect(() => {
    async function fetchStatus() {
      setLoading(true);
      try {
        const res = await fetch('/api/tasks/status');
        const data = await res.json();
        console.log('Task API response:', data); // Debug log
        setStatus(data);
      } catch (e) {
        console.error('Error fetching task status:', e); // Debug log
        setStatus(null);
      }
      setLoading(false);
    }
    fetchStatus();
    const interval = setInterval(fetchStatus, 10000);
    return () => clearInterval(interval);
  }, []);

  async function forceExecute(name) {
    setSpinning(s => ({ ...s, [name]: true }));
    setRotation(r => ({ ...r, [name]: (r[name] || 0) + 1080 }));
    try {
      await fetch(`/api/tasks/force`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ name }),
      });
    } catch (e) {}
    setTimeout(() => setSpinning(s => ({ ...s, [name]: false })), 1500);
  }

  function formatNextExecution(nextExecution) {
    if (!nextExecution) return '-';
    const now = new Date();
    const next = new Date(nextExecution);
    const diff = Math.max(0, next - now);
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    if (hours > 0) return `${hours} hour${hours > 1 ? 's' : ''}`;
    if (minutes > 0) return `${minutes} minute${minutes > 1 ? 's' : ''}`;
    return `${seconds} second${seconds !== 1 ? 's' : ''}`;
  }

  function formatLastExecution(lastExecution) {
    if (!lastExecution) return '-';
    const now = new Date();
    const last = new Date(lastExecution);
    const diff = Math.max(0, now - last);
    const seconds = Math.floor(diff / 1000);
    const minutes = Math.floor(seconds / 60);
    const hours = Math.floor(minutes / 60);
    if (hours > 0) return `${hours} hour${hours > 1 ? 's' : ''} ago`;
    if (minutes > 0) return `${minutes} minute${minutes > 1 ? 's' : ''} ago`;
    return `${seconds} second${seconds !== 1 ? 's' : ''} ago`;
  }

  function formatDuration(duration) {
    if (!duration || duration === '-') return '-';
    // Accepts either ms (number) or string like '1m23.456s' or '267.00858ms'
    if (typeof duration === 'number') {
      let totalSeconds = Math.floor(duration / 1000);
      const hours = Math.floor(totalSeconds / 3600);
      totalSeconds %= 3600;
      const minutes = Math.floor(totalSeconds / 60);
      const seconds = totalSeconds % 60;
      return `${hours.toString().padStart(2, '0')}:${minutes
        .toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }
    // Handle ms string like '267.00858ms'
    if (duration.endsWith('ms')) {
      const ms = parseFloat(duration.replace('ms', ''));
      const totalSeconds = Math.floor(ms / 1000);
      const hours = Math.floor(totalSeconds / 3600);
      const minutes = Math.floor((totalSeconds % 3600) / 60);
      const seconds = totalSeconds % 60;
      return `${hours.toString().padStart(2, '0')}:${minutes
        .toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
    }
    // Parse string like '1h2m3.456s', '2m3.456s', or '3.456s'
    const match = duration.match(/(?:(\d+)h)?(?:(\d+)m)?([\d.]+)s/);
    if (!match) return duration;
    const hours = parseInt(match[1] || '0', 10);
    const minutes = parseInt(match[2] || '0', 10);
    const seconds = Math.floor(parseFloat(match[3] || '0'));
    return `${hours.toString().padStart(2, '0')}:${minutes
      .toString().padStart(2, '0')}:${seconds.toString().padStart(2, '0')}`;
  }

  const styles = getStyles(darkMode);

  if (loading) return <div style={styles.container}>Loading...</div>;
  if (!status) return <div style={styles.container}>Error loading task status.</div>;

  const schedules = status.schedules || [];
  const queues = status.queues || [];

  return (
    <div style={styles.container}>
      <div style={styles.header}>Scheduled</div>
      <table style={styles.table}>
        <thead>
          <tr>
            <th style={styles.th}>Name</th>
            <th style={styles.th}>Interval</th>
            <th style={styles.th}>Last Execution</th>
            <th style={styles.th}>Last Duration</th>
            <th style={styles.th}>Next Execution</th>
            <th style={styles.th}></th>
          </tr>
        </thead>
        <tbody>
          {schedules.length === 0 ? (
            <tr><td colSpan={6} style={styles.td}>No scheduled tasks</td></tr>
          ) : schedules.map((scheduled, idx) => (
            <tr key={idx}>
              <td style={styles.td}>{scheduled.name}</td>
              <td style={styles.td}>{scheduled.interval}</td>
              <td style={styles.td}>{scheduled.lastExecution ? formatLastExecution(scheduled.lastExecution) : '-'}</td>
              <td style={styles.td}>{scheduled.lastDuration ? formatDuration(scheduled.lastDuration) : '-'}</td>
              <td style={styles.td}>{scheduled.nextExecution ? formatNextExecution(scheduled.nextExecution) : '-'}</td>
              <td style={styles.td}>
                <span
                  style={{
                    display: 'inline-block',
                    marginLeft: '0.5em',
                    verticalAlign: 'middle',
                  }}
                >
                  <FaArrowsRotate
                    onClick={() => forceExecute(scheduled.name)}
                    style={{
                      cursor: 'pointer',
                      color: spinning[scheduled.name]
                        ? (darkMode ? '#66aaff' : '#007bff')
                        : (darkMode ? '#aaa' : '#888'),
                      transition: spinning[scheduled.name]
                        ? 'transform 5s cubic-bezier(0.4, 0.2, 0.2, 1)'
                        : 'none',
                      transform: `rotate(${rotation[scheduled.name] || 0}deg)`,
                    }}
                    size={20}
                    title="Force Execute"
                  />
                </span>
              </td>
            </tr>
          ))}
        </tbody>
      </table>
      <div style={styles.header}>Queue</div>
      <table style={styles.table}>
        <thead>
          <tr>
            <th style={styles.th}></th>
            <th style={styles.th}>Name</th>
            <th style={styles.th}>Queued</th>
            <th style={styles.th}>Started</th>
            <th style={styles.th}>Ended</th>
            <th style={styles.th}>Duration</th>
          </tr>
        </thead>
        <tbody>
          {queues.length === 0 ? (
            <tr><td colSpan={6} style={styles.td}>No queue items</td></tr>
          ) : queues.map((item, idx) => (
            <tr key={idx}>
              <td style={{...styles.td, textAlign: 'center'}}>
                {(() => {
                  if (!item.Status) return <span title="Unknown">-</span>;
                  if (item.Status === 'success') return <span title="Success" style={{color: darkMode ? '#4fdc7b' : '#28a745'}}>&#x2714;</span>;
                  if (item.Status === 'running') return <span title="Running" style={{color: darkMode ? '#66aaff' : '#007bff'}}>&#x25D4;</span>;
                  if (item.Status === 'failed') return <span title="Failed" style={{color: darkMode ? '#ff6b6b' : '#dc3545'}}>&#x2716;</span>;
                  return <span title={item.Status}>{item.Status}</span>;
                })()}
              </td>
              <td style={styles.td}>{item.type}</td>
              <td style={styles.td}>{item.Queued ? new Date(item.Queued).toLocaleString() : '—'}</td>
              <td style={styles.td}>{item.Started ? new Date(item.Started).toLocaleString() : '—'}</td>
              <td style={styles.td}>{item.Ended ? new Date(item.Ended).toLocaleString() : '—'}</td>
              <td style={styles.td}>{(() => {
                if (!item.Duration || item.Duration === '') return '—';
                if (typeof item.Duration === 'number') {
                  // If > 1s, show seconds, else show ms
                  if (item.Duration >= 1e9) {
                    return `${(item.Duration / 1e9).toFixed(2)} s`;
                  } else {
                    return `${Math.round(item.Duration / 1e6)} ms`;
                  }
                }
                // If string, fallback to previous logic
                return item.Duration;
              })()}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
