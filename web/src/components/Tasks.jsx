import React, { useEffect, useState } from 'react';
import { FaArrowsRotate } from 'react-icons/fa6';

const tableStyle = {
  width: '100%',
  marginBottom: '2em',
  borderCollapse: 'collapse',
  background: '#f6f7f9',
  color: '#222',
  fontSize: '15px',
};
const thStyle = {
  textAlign: 'left',
  padding: '0.75em 0.5em',
  fontWeight: 500,
  background: '#f6f7f9',
  borderBottom: '1px solid #e5e7eb',
};
const tdStyle = {
  padding: '0.75em 0.5em',
  borderBottom: '1px solid #e5e7eb',
  background: '#fff',
  textAlign: 'left',
};
const headerStyle = {
  fontSize: '1.4em',
  fontWeight: 600,
  margin: '0 0 1em 0',
  color: '#222',
};

export default function Tasks() {
  const [status, setStatus] = useState(null);
  const [loading, setLoading] = useState(true);
  const [spinning, setSpinning] = useState({});
  const [rotation, setRotation] = useState({});

  useEffect(() => {
    async function fetchStatus() {
      setLoading(true);
      try {
        const res = await fetch('/api/tasks/status');
        setStatus(await res.json());
      } catch (e) {
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

  if (loading) return <div>Loading...</div>;
  if (!status) return <div>Error loading task status.</div>;

  const schedules = status.schedules || [];
  const queues = status.queues || [];

  return (
    <div style={{ padding: '2em', background: '#f6f7f9', minHeight: '100vh' }}>
      <div style={headerStyle}>Scheduled</div>
      <table style={tableStyle}>
        <thead>
          <tr>
            <th style={thStyle}>Name</th>
            <th style={thStyle}>Interval</th>
            <th style={thStyle}>Last Execution</th>
            <th style={thStyle}>Last Duration</th>
            <th style={thStyle}>Next Execution</th>
            <th style={thStyle}></th>
          </tr>
        </thead>
        <tbody>
          {schedules.length === 0 ? (
            <tr><td colSpan={6} style={tdStyle}>No scheduled tasks</td></tr>
          ) : schedules.map((scheduled, idx) => (
            <tr key={idx}>
              <td style={tdStyle}>{scheduled.name}</td>
              <td style={tdStyle}>{scheduled.interval}</td>
              <td style={tdStyle}>{scheduled.lastExecution ? formatLastExecution(scheduled.lastExecution) : '-'}</td>
              <td style={tdStyle}>{scheduled.lastDuration ? formatDuration(scheduled.lastDuration) : '-'}</td>
              <td style={tdStyle}>{scheduled.nextExecution ? formatNextExecution(scheduled.nextExecution) : '-'}</td>
              <td style={tdStyle}>
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
                      color: spinning[scheduled.name] ? '#007bff' : '#888',
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
      <div style={headerStyle}>Queue</div>
      <table style={tableStyle}>
        <thead>
          <tr>
            <th style={thStyle}></th>
            <th style={thStyle}>Name</th>
            <th style={thStyle}>Queued</th>
            <th style={thStyle}>Started</th>
            <th style={thStyle}>Ended</th>
            <th style={thStyle}>Duration</th>
          </tr>
        </thead>
        <tbody>
          {queues.length === 0 ? (
            <tr><td colSpan={6} style={tdStyle}>No queue items</td></tr>
          ) : queues.map((item, idx) => (
            <tr key={idx}>
              <td style={{...tdStyle, textAlign: 'center'}}>
                {(() => {
                  if (!item.Status) return <span title="Unknown">-</span>;
                  if (item.Status === 'success') return <span title="Success" style={{color:'#28a745'}}>&#x2714;</span>;
                  if (item.Status === 'running') return <span title="Running" style={{color:'#007bff'}}>&#x25D4;</span>;
                  if (item.Status === 'failed') return <span title="Failed" style={{color:'#dc3545'}}>&#x2716;</span>;
                  return <span title={item.Status}>{item.Status}</span>;
                })()}
              </td>
              <td style={tdStyle}>{item.type}</td>
              <td style={tdStyle}>{item.Queued ? new Date(item.Queued).toLocaleString() : '-'}</td>
              <td style={tdStyle}>{item.Started ? new Date(item.Started).toLocaleString() : '-'}</td>
              <td style={tdStyle}>{item.Ended ? new Date(item.Ended).toLocaleString() : '-'}</td>
              <td style={tdStyle}>{(() => {
                if (!item.Duration) return '-';
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
