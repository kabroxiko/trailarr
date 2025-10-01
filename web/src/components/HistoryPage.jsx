import React, { useEffect, useState } from 'react';
import { getHistory } from '../api';
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
    getHistory()
      .then(data => {
        setHistory(data);
        setLoading(false);
      })
      .catch(err => {
        setError(err.message || 'Failed to load history');
        setLoading(false);
      });
  }, []);

  let content;
  if (loading) {
    content = <div>Loading...</div>;
  } else if (error) {
    content = <div style={{ color: 'red' }}>{error}</div>;
  } else {
    content = (
      <div style={{ overflowX: 'auto', boxShadow: '0 2px 12px rgba(0,0,0,0.08)', borderRadius: 12, background: '#fff', padding: 0 }}>
        <table className="history-table" style={{ width: '100%', borderCollapse: 'separate', borderSpacing: 0, fontSize: 16 }}>
          <thead>
            <tr style={{ background: '#f3e8ff', color: '#7c3aed', fontWeight: 600 }}>
              <th style={{ padding: '14px 10px', textAlign: 'center', borderBottom: '2px solid #e5e7eb' }}></th>
              <th style={{ padding: '14px 10px', textAlign: 'center', borderBottom: '2px solid #e5e7eb' }}>Media Type</th>
              <th style={{ padding: '14px 10px', textAlign: 'left', borderBottom: '2px solid #e5e7eb' }}>Title</th>
              <th style={{ padding: '14px 10px', textAlign: 'left', borderBottom: '2px solid #e5e7eb' }}>Extra Type</th>
              <th style={{ padding: '14px 10px', textAlign: 'left', borderBottom: '2px solid #e5e7eb' }}>Extra Title</th>
              <th style={{ padding: '14px 10px', textAlign: 'left', borderBottom: '2px solid #e5e7eb' }}>Date</th>
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
                <tr key={key} style={{ background: idx % 2 === 0 ? '#fafafc' : '#f3e8ff', transition: 'background 0.2s' }}>
                  <td style={{ textAlign: 'center', padding: '10px 0' }}>{icon}</td>
                  <td style={{ padding: '10px 10px', textAlign: 'center', textTransform: 'capitalize', color: '#7c3aed', fontWeight: 500 }}>{item.mediaType}</td>
                  <td style={{ padding: '10px 10px', textAlign: 'left', fontWeight: 500, color: '#222' }}>{item.title}</td>
                  <td style={{ padding: '10px 10px', textAlign: 'left', color: '#6d28d9', fontWeight: 500 }}>{item.extraType}</td>
                  <td style={{ padding: '10px 10px', textAlign: 'left', color: '#444' }}>{item.extraTitle}</td>
                  <td style={{ padding: '10px 10px', textAlign: 'left', color: '#888', fontSize: 15 }}>{formatDate(item.date)}</td>
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
    const setIconColor = () => {
      const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      document.documentElement.style.setProperty('--history-icon-color', isDark ? '#fff' : '#111');
    };
    setIconColor();
    window.matchMedia('(prefers-color-scheme: dark)').addEventListener('change', setIconColor);
    return () => {
      window.matchMedia('(prefers-color-scheme: dark)').removeEventListener('change', setIconColor);
    };
  }, []);
  return (
    <div className="history-page" style={{ padding: '32px', width: '100vw', margin: '0', boxSizing: 'border-box' }}>
      <h2 style={{ fontSize: 32, fontWeight: 700, marginBottom: 24, color: '#7c3aed', letterSpacing: 0.5 }}>History</h2>
      {content}
    </div>
  );
};

export default HistoryPage;
