import React, { useEffect, useState } from 'react';

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
    <div style={{ padding: '2em' }}>
      <h2>Scheduled</h2>
      <table style={{ width: '100%', marginBottom: '2em', borderCollapse: 'collapse', background: '#222', color: '#eee' }}>
        <thead>
          <tr style={{ borderBottom: '1px solid #444' }}>
            <th style={{ textAlign: 'left', padding: '0.5em' }}>Type</th>
            <th>Name</th>
            <th>Interval</th>
            <th>Last Execution</th>
            <th>Last Duration</th>
            <th>Next Execution</th>
            <th>Error</th>
          </tr>
        </thead>
        <tbody>
          {schedules.length === 0 ? (
            <tr><td colSpan={7} style={{ textAlign: 'center', padding: '1em' }}>No scheduled tasks</td></tr>
          ) : schedules.map((scheduled, idx) => (
            <tr key={idx} style={{ borderBottom: '1px solid #333' }}>
              <td style={{ padding: '0.5em' }}>{scheduled.type}</td>
              <td>{scheduled.name}</td>
              <td>{scheduled.interval}</td>
              <td>{scheduled.lastExecution ? new Date(scheduled.lastExecution).toLocaleString() : '-'}</td>
              <td>{scheduled.lastDuration}</td>
              <td>{scheduled.nextExecution ? new Date(scheduled.nextExecution).toLocaleString() : '-'}</td>
              <td>{scheduled.lastError || '-'}</td>
            </tr>
          ))}
        </tbody>
      </table>
      <h2>Queue</h2>
      <table style={{ width: '100%', borderCollapse: 'collapse', background: '#222', color: '#eee' }}>
        <thead>
          <tr style={{ borderBottom: '1px solid #444' }}>
            <th style={{ textAlign: 'left', padding: '0.5em' }}>Type</th>
            <th>Name</th>
            <th>Queued</th>
            <th>Started</th>
            <th>Ended</th>
            <th>Duration</th>
            <th>Status</th>
            <th>Error</th>
          </tr>
        </thead>
        <tbody>
          {queues.length === 0 ? (
            <tr><td colSpan={8} style={{ textAlign: 'center', padding: '1em' }}>No queue items</td></tr>
          ) : queues.map((item, idx) => (
            <tr key={idx} style={{ borderBottom: '1px solid #333' }}>
              <td style={{ padding: '0.5em' }}>{item.type}</td>
              <td>{item.type}</td>
              <td>{item.Queued ? new Date(item.Queued).toLocaleString() : '-'}</td>
              <td>{item.Started ? new Date(item.Started).toLocaleString() : '-'}</td>
              <td>{item.Ended ? new Date(item.Ended).toLocaleString() : '-'}</td>
              <td>{item.Duration ? (typeof item.Duration === 'string' ? item.Duration : `${item.Duration} ms`) : '-'}</td>
              <td>{item.Status}</td>
              <td>{item.Error || '-'}</td>
            </tr>
          ))}
        </tbody>
      </table>
    </div>
  );
}
