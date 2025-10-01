import React, { useEffect, useState } from 'react';

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
          </tr>
        </thead>
        <tbody>
          {schedules.length === 0 ? (
            <tr><td colSpan={5} style={tdStyle}>No scheduled tasks</td></tr>
          ) : schedules.map((scheduled, idx) => (
            <tr key={idx}>
              <td style={tdStyle}>{scheduled.name}</td>
              <td style={tdStyle}>{scheduled.interval}</td>
              <td style={tdStyle}>{scheduled.lastExecution ? new Date(scheduled.lastExecution).toLocaleString() : '-'}</td>
              <td style={tdStyle}>{scheduled.lastDuration}</td>
              <td style={tdStyle}>{scheduled.nextExecution ? new Date(scheduled.nextExecution).toLocaleString() : '-'}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <div style={headerStyle}>Queue</div>
      <table style={tableStyle}>
        <thead>
          <tr>
            <th style={thStyle}>Name</th>
            <th style={thStyle}>Queued</th>
            <th style={thStyle}>Started</th>
            <th style={thStyle}>Ended</th>
            <th style={thStyle}>Duration</th>
          </tr>
        </thead>
        <tbody>
          {queues.length === 0 ? (
            <tr><td colSpan={5} style={tdStyle}>No queue items</td></tr>
          ) : queues.map((item, idx) => (
            <tr key={idx}>
              <td style={tdStyle}>{item.type}</td>
              <td style={tdStyle}>{item.Queued ? new Date(item.Queued).toLocaleString() : '-'}</td>
              <td style={tdStyle}>{item.Started ? new Date(item.Started).toLocaleString() : '-'}</td>
              <td style={tdStyle}>{item.Ended ? new Date(item.Ended).toLocaleString() : '-'}</td>
              <td style={tdStyle}>{item.Duration ? (typeof item.Duration === 'string' ? item.Duration : `${item.Duration} ms`) : '-'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
