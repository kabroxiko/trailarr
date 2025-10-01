import React, { useEffect, useState } from 'react';
// ...existing code...
import { FaDownload, FaTrash } from 'react-icons/fa';

function formatDate(date) {
  if (!date) return '';
  const d = typeof date === 'string' ? new Date(date) : date;
  const now = new Date();
  const diff = Math.floor((now - d) / 86400000);
  if (diff === 0) return 'Today';
  if (diff === 1) return 'Yesterday';
  return `${diff} days ago`;
}

function getActionIcon(action) {
  // Deprecated, not used anymore
  return null;
}

const HistoryPage = () => {
  const [history, setHistory] = useState([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState(null);

  useEffect(() => {
    setLoading(true);
    import('../api').then(({ getHistory }) => {
      getHistory()
        .then(data => {
          setHistory(data);
          setLoading(false);
        })
        .catch(err => {
          setError(err.message || 'Failed to load history');
          setLoading(false);
        });
    });
  }, []);

  let content;
  // Dark mode friendly colors
  const tableStyles = {
    width: '100%',
    borderCollapse: 'separate',
    borderSpacing: 0,
    fontSize: 16,
    background: 'var(--history-table-bg, #fff)',
    color: 'var(--history-table-text, #222)'
  };
  const thStyles = {
    padding: '14px 10px',
    textAlign: 'left',
    borderBottom: '2px solid var(--history-table-border, #e5e7eb)',
    background: 'var(--history-table-header-bg, #f3e8ff)',
    color: 'var(--history-table-header-text, #7c3aed)',
    fontWeight: 600
  };
  const trStyles = idx => ({
    background: idx % 2 === 0 ? 'var(--history-table-row-bg1, #fafafc)' : 'var(--history-table-row-bg2, #f3e8ff)',
    transition: 'background 0.2s'
  });
  const tdStyles = {
    padding: '10px 10px',
    textAlign: 'left',
    color: 'var(--history-table-cell-text, #222)'
  };
  if (loading) {
    content = <div>Loading...</div>;
  } else if (error) {
    content = <div style={{ color: 'red' }}>{error}</div>;
  } else {
    content = (
      <div style={{ overflowX: 'auto', boxShadow: '0 2px 12px rgba(0,0,0,0.08)', borderRadius: 12, background: 'var(--history-table-bg, #fff)', padding: 0 }}>
        <table className="history-table" style={tableStyles}>
          <thead>
            <tr>
              <th style={{ ...thStyles, textAlign: 'center' }}></th>
              <th style={{ ...thStyles, textAlign: 'center' }}>Media Type</th>
              <th style={thStyles}>Title</th>
              <th style={thStyles}>Extra Type</th>
              <th style={thStyles}>Extra Title</th>
              <th style={thStyles}>Date</th>
            </tr>
          </thead>
          <tbody>
            {history.map((item, idx) => {
              const key = item.date + '-' + item.title + '-' + item.extraTitle + '-' + item.action;
              let icon = null;
              if (item.action === 'download') {
                icon = <FaDownload title="Downloaded" style={{ fontSize: 20, color: 'var(--history-icon-color, #111)' }} />;
              } else if (item.action === 'delete') {
                icon = <FaTrash title="Deleted" style={{ fontSize: 20, color: 'var(--history-icon-color, #111)' }} />;
              }
              return (
                <tr key={key} style={trStyles(idx)}>
                  <td style={{ ...tdStyles, textAlign: 'center' }}>{icon}</td>
                  <td style={{ ...tdStyles, textAlign: 'center', textTransform: 'capitalize', color: 'var(--history-table-media-type, #7c3aed)', fontWeight: 500 }}>{item.mediaType}</td>
                  <td style={{ ...tdStyles, fontWeight: 500 }}>{item.title}</td>
                  <td style={{ ...tdStyles, color: 'var(--history-table-extra-type, #6d28d9)', fontWeight: 500 }}>{item.extraType}</td>
                  <td style={{ ...tdStyles, color: 'var(--history-table-extra-title, #444)' }}>{item.extraTitle}</td>
                  <td style={{ ...tdStyles, color: 'var(--history-table-date, #888)', fontSize: 15 }}>{formatDate(item.date)}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
      </div>
    );
  }
  // Set icon color variable for dark/light mode
  useEffect(() => {
    const setTableColors = () => {
      const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      document.documentElement.style.setProperty('--history-table-bg', isDark ? '#18181b' : '#fff');
      document.documentElement.style.setProperty('--history-table-text', isDark ? '#e5e7eb' : '#222');
      document.documentElement.style.setProperty('--history-table-header-bg', isDark ? '#27272a' : '#f3e8ff');
      document.documentElement.style.setProperty('--history-table-header-text', isDark ? '#c7d2fe' : '#7c3aed');
      document.documentElement.style.setProperty('--history-table-border', isDark ? '#444' : '#e5e7eb');
      document.documentElement.style.setProperty('--history-table-row-bg1', isDark ? '#232326' : '#fafafc');
      document.documentElement.style.setProperty('--history-table-row-bg2', isDark ? '#18181b' : '#f3e8ff');
      document.documentElement.style.setProperty('--history-table-cell-text', isDark ? '#e5e7eb' : '#222');
      document.documentElement.style.setProperty('--history-table-media-type', isDark ? '#a5b4fc' : '#7c3aed');
      document.documentElement.style.setProperty('--history-table-extra-type', isDark ? '#c4b5fd' : '#6d28d9');
      document.documentElement.style.setProperty('--history-table-extra-title', isDark ? '#d1d5db' : '#444');
      document.documentElement.style.setProperty('--history-table-date', isDark ? '#a1a1aa' : '#888');
      document.documentElement.style.setProperty('--history-icon-color', isDark ? '#fff' : '#111');
    };
    setTableColors();
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', setTableColors);
    return () => {
      window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', setTableColors);
    };
  }, []);
  return (
    <div className="history-page" style={{ padding: '32px', width: '100vw', margin: '0', boxSizing: 'border-box' }}>
      {content}
    </div>
  );
};

export default HistoryPage;
