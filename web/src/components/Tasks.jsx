import React, { useEffect, useState, useRef } from 'react';
import { FaArrowsRotate, FaClock } from 'react-icons/fa6';
import './Tasks.css';

// Inline style to remove focus outline from the force icon
const iconNoOutline = {
  outline: 'none',
  boxShadow: 'none',
};

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
  const [loading, setLoading] = useState(true);
  const [status, setStatus] = useState(null);
  const [queues, setQueues] = useState([]);
  const [queueLoading, setQueueLoading] = useState(true);
  const [iconRotation, setIconRotation] = useState({});
  const [iconPrevRotation, setIconPrevRotation] = useState({});
  const rotationIntervals = useRef({});
  const [darkMode, setDarkMode] = useState(false);

  // Fetch status from API for polling fallback and force execute
  async function fetchStatus() {
    setLoading(true);
    try {
      const res = await fetch('/api/tasks/status');
      const data = await res.json();
      setStatus(data);
    } catch (e) {
      setStatus(null);
    }
    setLoading(false);
  }
  // Fetch queue from new endpoint
  async function fetchQueue() {
    setQueueLoading(true);
    try {
      const res = await fetch('/api/tasks/queue');
      const data = await res.json();
      if (data && Array.isArray(data.queues)) {
        setQueues(data.queues);
      } else {
        setQueues([]);
      }
    } catch (e) {
      setQueues([]);
    }
    setQueueLoading(false);
  }

  // Converts a time value in milliseconds to human-readable text, showing only the largest non-zero unit
  // durationToText: ms to human text, with rounding option
  // roundType: 'ceil' (default), 'cut', 'round'
  function durationToText(ms, suffix = '', roundType = 'ceil') {
    if (typeof ms !== 'number' || isNaN(ms) || ms < 0) return `0 seconds${suffix}`;
    let totalSeconds;
    if (roundType === 'cut') {
      totalSeconds = Math.floor(ms / 1000);
    } else if (roundType === 'round') {
      totalSeconds = Math.round(ms / 1000);
    } else {
      totalSeconds = Math.ceil(ms / 1000);
    }
    if (totalSeconds >= 86400) {
      let days;
      if (roundType === 'cut') days = Math.floor(totalSeconds / 86400);
      else if (roundType === 'round') days = Math.round(totalSeconds / 86400);
      else days = Math.ceil(totalSeconds / 86400);
      return `${days} day${days > 1 ? 's' : ''}${suffix}`;
    }
    if (totalSeconds >= 3600) {
      let hours;
      if (roundType === 'cut') hours = Math.floor(totalSeconds / 3600);
      else if (roundType === 'round') hours = Math.round(totalSeconds / 3600);
      else hours = Math.ceil(totalSeconds / 3600);
      return `${hours} hour${hours > 1 ? 's' : ''}${suffix}`;
    }
    if (totalSeconds >= 60) {
      let minutes;
      if (roundType === 'cut') minutes = Math.floor(totalSeconds / 60);
      else if (roundType === 'round') minutes = Math.round(totalSeconds / 60);
      else minutes = Math.ceil(totalSeconds / 60);
      return `${minutes} minute${minutes > 1 ? 's' : ''}${suffix}`;
    }
    return `${totalSeconds} second${totalSeconds !== 1 ? 's' : ''}${suffix}`;
  }

  useEffect(() => {
    // Detect dark mode
    const mq = window.matchMedia('(prefers-color-scheme: dark)');
    setDarkMode(mq.matches);
    const handler = (e) => setDarkMode(e.matches);
    mq.addEventListener('change', handler);
    return () => mq.removeEventListener('change', handler);
  }, []);

  useEffect(() => {
    let ws;
    let pollingInterval;
    let queueInterval;
    let wsConnected = false;
    function startPolling() {
      fetchStatus();
      pollingInterval = setInterval(fetchStatus, 500);
      fetchQueue();
      queueInterval = setInterval(fetchQueue, 1000);
    }
    function stopPolling() {
      if (pollingInterval) clearInterval(pollingInterval);
      if (queueInterval) clearInterval(queueInterval);
    }
    // Try to connect to WebSocket
    try {
      ws = new window.WebSocket((window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host + '/ws/tasks');
      ws.onopen = () => {
        wsConnected = true;
        stopPolling();
        fetchQueue();
        queueInterval = setInterval(fetchQueue, 1000);
      };
      ws.onmessage = (event) => {
        try {
          const data = JSON.parse(event.data);
          setStatus(data);
          setLoading(false);
        } catch (e) {}
      };
      ws.onerror = () => {
        wsConnected = false;
        startPolling();
      };
      ws.onclose = () => {
        wsConnected = false;
        startPolling();
      };
    } catch (e) {
      startPolling();
    }
    // Fallback to polling if WebSocket fails
    if (!wsConnected) startPolling();
    return () => {
      stopPolling();
      if (ws) ws.close();
    };
  }, []);

  // Icon rotation effect for running tasks
  useEffect(() => {
    if (!status || !status.schedules) return;
    status.schedules.forEach(sch => {
      const key = sch.taskId;
      if (sch.status === 'running') {
        if (!rotationIntervals.current[key]) {
          rotationIntervals.current[key] = setInterval(() => {
            setIconRotation(rot => {
              const prev = rot[key] || 0;
              const next = prev + 18;
              setIconPrevRotation(prevRot => ({ ...prevRot, [key]: prev }));
              return { ...rot, [key]: next };
            });
          }, 50);
        }
      } else {
        if (rotationIntervals.current[key]) {
          clearInterval(rotationIntervals.current[key]);
          rotationIntervals.current[key] = null;
        }
        setIconRotation(rot => ({ ...rot, [key]: 0 }));
      }
    });
    // Cleanup intervals on unmount only
    return () => {
      Object.values(rotationIntervals.current).forEach(interval => interval && clearInterval(interval));
      rotationIntervals.current = {};
      setIconRotation({});
    };
  }, [status]);

  async function forceExecute(taskId) {
    try {
      await fetch(`/api/tasks/force`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({ taskId }),
      });
      // Immediately update status after force execute
      fetchStatus();
    } catch (e) {}
  }

  // Helper to format interval values for scheduled tasks
  // Unified formatter for intervals and time differences
  function formatTimeDiff({from, to, suffix = '', roundType = 'ceil'}) {
    if (!from || !to) return '-';
    let diff = Math.max(0, to - from);
    return durationToText(diff, suffix, roundType);
  }

  // For interval values (minutes, hours, days)
  function formatInterval(interval) {
    if (interval == null || interval === '') return '-';
    if (typeof interval === 'number') {
      return durationToText(interval * 60 * 1000);
    }
    if (typeof interval !== 'string') interval = String(interval);
    // Parse patterns like '2h30m', '1d2h', '90m', '1h', '1d', etc.
    const regex = /(?:(\d+)d)?(?:(\d+)h)?(?:(\d+)m)?/;
    const match = interval.match(regex);
    if (!match) return interval;
    const days = parseInt(match[1] || '0', 10);
    const hours = parseInt(match[2] || '0', 10);
    const minutes = parseInt(match[3] || '0', 10);
    if (days > 0 || hours > 0 || minutes > 0) {
      return durationToText((days * 86400 + hours * 3600 + minutes * 60) * 1000);
    }
    // fallback: try to parse as a number of minutes
    const min = parseInt(interval, 10);
    if (!isNaN(min)) {
      return durationToText(min * 60 * 1000);
    }
    return interval;
  }

  function formatDuration(duration) {
    if (!duration || duration === '-') return '-';
    // Accepts either ms (number) or string like '1m23.456s' or '267.00858ms'
    if (typeof duration === 'number') {
      if (duration < 1000) {
        return `${duration.toFixed(2)} ms`;
      }
      return durationToText(duration);
    }
    // Handle ms string like '267.00858ms'
    if (duration.endsWith('ms')) {
      const ms = parseFloat(duration.replace('ms', ''));
      if (ms < 1000) {
        return `${ms.toFixed(2)} ms`;
      }
      return durationToText(ms);
    }
    // Parse string like '1h2m3.456s', '2m3.456s', or '3.456s'
    const match = duration.match(/(?:(\d+)h)?(?:(\d+)m)?([\d.]+)s/);
    if (!match) return duration;
    const hours = parseInt(match[1] || '0', 10);
    const minutes = parseInt(match[2] || '0', 10);
    const secondsFloat = parseFloat(match[3] || '0');
    if (secondsFloat < 1 && hours === 0 && minutes === 0) {
      return `${(secondsFloat * 1000).toFixed(2)} ms`;
    }
    return durationToText((hours * 3600 + minutes * 60 + Math.floor(secondsFloat)) * 1000);
  }

  const styles = getStyles(darkMode);

  // Debounced loading indicator
  const [showLoading, setShowLoading] = useState(false);
  useEffect(() => {
    let timer;
    if (loading) {
      timer = setTimeout(() => setShowLoading(true), 500);
    } else {
      setShowLoading(false);
    }
    return () => timer && clearTimeout(timer);
  }, [loading]);

  if (showLoading) return <div style={styles.container}>Loading...</div>;
  if (!status) return <div style={styles.container}>Error loading task status.</div>;

  const schedules = status.schedules || [];

  return (
    <div style={styles.container}>
      <div style={styles.header}>Scheduled</div>
      <table style={styles.table}>
        <thead>
          <tr>
            <th style={styles.th}>Name</th>
            <th style={{...styles.th, textAlign: 'center'}}>Status</th>
            <th style={{...styles.th, textAlign: 'center'}}>Interval</th>
            <th style={{...styles.th, textAlign: 'center'}}>Last Execution</th>
            <th style={{...styles.th, textAlign: 'center'}}>Last Duration</th>
            <th style={{...styles.th, textAlign: 'center'}}>Next Execution</th>
            <th style={{...styles.th, textAlign: 'center'}}></th>
          </tr>
        </thead>
        <tbody>
          {schedules.length === 0 ? (
            <tr><td colSpan={7} style={styles.td}>No scheduled tasks</td></tr>
          ) : schedules.map((scheduled, idx) => (
            <tr key={idx}>
              <td style={styles.td}>{scheduled.name}</td>
              <td style={{...styles.td, textAlign: 'center'}}>{(() => {
                if (scheduled.interval === 0) {
                  return <span style={{color: darkMode ? '#888' : '#bbb', fontStyle: 'italic'}}>Disabled</span>;
                }
                const status = scheduled.status;
                if (!status) return <span>-</span>;
                if (status === 'running') return <span style={{color: darkMode ? '#66aaff' : '#007bff'}}>Running</span>;
                if (status === 'success') return <span style={{color: darkMode ? '#4fdc7b' : '#28a745'}}>Success</span>;
                if (status === 'failed') return <span style={{color: darkMode ? '#ff6b6b' : '#dc3545'}}>Failed</span>;
                return <span>{status}</span>;
              })()}</td>
              <td style={{...styles.td, textAlign: 'center'}}>{scheduled.interval === 0
                ? <span style={{color: darkMode ? '#888' : '#bbb', fontStyle: 'italic'}}>Disabled</span>
                : formatInterval(scheduled.interval)
              }</td>
              <td style={{...styles.td, textAlign: 'center'}}>{scheduled.lastExecution ? formatTimeDiff({from: new Date(scheduled.lastExecution), to: new Date(), suffix: ' ago', roundType: 'cut'}) : '-'}</td>
              <td style={{...styles.td, textAlign: 'center'}}>{scheduled.lastDuration ? formatDuration(scheduled.lastDuration) : '-'}</td>
              <td style={{...styles.td, textAlign: 'center'}}>{scheduled.interval === 0
                ? <span style={{color: darkMode ? '#888' : '#bbb', fontStyle: 'italic'}}>Disabled</span>
                : (scheduled.nextExecution ? formatTimeDiff({from: new Date(), to: new Date(scheduled.nextExecution)}) : '-')
              }</td>
              <td style={{...styles.td, textAlign: 'center'}}>
                <span
                  style={{
                    display: 'inline-block',
                    marginLeft: '0.5em',
                    verticalAlign: 'middle',
                  }}
                >
                  <FaArrowsRotate
                    onClick={scheduled.status === 'running' ? undefined : () => forceExecute(scheduled.taskId)}
                    className={scheduled.status === 'running' ? 'spin-icon' : ''}
                    style={{
                      cursor: scheduled.status === 'running' ? 'not-allowed' : 'pointer',
                      opacity: scheduled.status === 'running' ? 0.5 : 1,
                      color: (scheduled.status === 'running')
                        ? (darkMode ? '#66aaff' : '#007bff')
                        : (darkMode ? '#aaa' : '#888'),
                      ...iconNoOutline,
                    }}
                    size={20}
                    title={scheduled.status === 'running' ? 'Task is running' : 'Force Execute'}
                    tabIndex={scheduled.status === 'running' ? -1 : 0}
                    aria-disabled={scheduled.status === 'running'}
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
            <th style={{...styles.th, textAlign: 'center'}}></th>
            <th style={styles.th}>Task Name</th>
            <th style={{...styles.th, textAlign: 'center'}}>Queued</th>
            <th style={{...styles.th, textAlign: 'center'}}>Started</th>
            <th style={{...styles.th, textAlign: 'center'}}>Ended</th>
            <th style={{...styles.th, textAlign: 'center'}}>Duration</th>
          </tr>
        </thead>
        <tbody>
          {(!queueLoading && (!queues || queues.length === 0)) ? (
            <tr><td colSpan={6} style={{...styles.td, textAlign: 'center'}}>No queue items</td></tr>
          ) : (Array.isArray(queues) ? queues : []).map((item, idx) => {
            // Try to get the task name from schedules (by taskId)
            let taskId = item.TaskId;
            if (schedules && item.TaskId) {
              const sch = schedules.find(s => s.taskId === item.TaskId);
              if (sch && sch.name) taskId = sch.name;
            }
            return (
              <tr key={idx}>
                <td style={{...styles.td, textAlign: 'center'}}>
                  {(() => {
                    if (!item.Status) return <span title="Unknown">-</span>;
                    if (item.Status === 'success') return <span title="Success" style={{color: darkMode ? '#4fdc7b' : '#28a745'}}>&#x2714;</span>;
                    if (item.Status === 'running') return <span title="Running" style={{color: darkMode ? '#66aaff' : '#007bff'}}>&#x25D4;</span>;
                    if (item.Status === 'failed') return <span title="Failed" style={{color: darkMode ? '#ff6b6b' : '#dc3545'}}>&#x2716;</span>;
                    if (item.Status === 'queued') return <FaClock title="Queued" style={{color: darkMode ? '#ffb300' : '#e6b800', verticalAlign: 'middle'}} />;
                    return <span title={item.Status}>{item.Status}</span>;
                  })()}
                </td>
                <td style={styles.td}>{taskId || '-'}</td>
                <td style={{...styles.td, textAlign: 'center'}}>{item.Queued ? new Date(item.Queued).toLocaleString() : '—'}</td>
                <td style={{...styles.td, textAlign: 'center'}}>{item.Started ? new Date(item.Started).toLocaleString() : '—'}</td>
                <td style={{...styles.td, textAlign: 'center'}}>{item.Ended ? new Date(item.Ended).toLocaleString() : '—'}</td>
                <td style={styles.td}>{(() => {
                  if (!item.Duration || item.Duration === '') return '—';
                  let dur = item.Duration;
                  if (typeof dur === 'string') {
                    dur = Number(dur);
                  }
                  if (typeof dur === 'number' && !isNaN(dur)) {
                    // If > 1s, show seconds, else show ms
                    if (dur >= 1e9) {
                      return `${(dur / 1e9).toFixed(2)} s`;
                    } else if (dur >= 1e6) {
                      return `${Math.round(dur / 1e6)} ms`;
                    } else if (dur >= 1e3) {
                      return `${Math.round(dur / 1e3)} μs`;
                    } else {
                      return `${dur} ns`;
                    }
                  }
                  // If string and not a number, fallback to previous logic
                  return item.Duration;
                })()}</td>
              </tr>
            );
          })}
        </tbody>
      </table>
    </div>
  );
}
