import React, { useEffect, useState } from 'react';
import { Link } from 'react-router-dom';
import Container from './Container';
import { FaDownload, FaTrash } from 'react-icons/fa';

function formatDate(date) {
  if (!date) return '';
  const d = typeof date === 'string' ? new Date(date) : date;
  const now = new Date();
  const diff = Math.floor((now - d) / 86400000);
  if (diff === 0) {
    // Show as hh:mm in 24h format
    const hours = d.getHours().toString().padStart(2, '0');
    const minutes = d.getMinutes().toString().padStart(2, '0');
    return `${hours}:${minutes}`;
  }
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
  const [page, setPage] = useState(1);
  const pageSize = 20;
  const totalPages = Math.ceil(history.length / pageSize);
  const paginatedHistory = history.slice((page - 1) * pageSize, page * pageSize);

  // Helper to get link for a history item
  function getMediaLink(item) {
    // Only use mediaId for links
    if (item.mediaId) {
      if (item.mediaType === 'movie') {
        return `/movies/${item.mediaId}`;
      } else if (item.mediaType === 'tv') {
        return `/series/${item.mediaId}`;
      }
    }
    return null;
  }

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
    padding: '8px 6px',
    textAlign: 'left',
    borderBottom: '2px solid var(--history-table-border, #e5e7eb)',
    background: 'var(--history-table-header-bg, #f3e8ff)',
    color: 'var(--history-table-header-text, #7c3aed)',
    fontWeight: 600,
    width: undefined // will be set per column
  };
  const trStyles = idx => ({
    background: idx % 2 === 0 ? 'var(--history-table-row-bg1, #fafafc)' : 'var(--history-table-row-bg2, #f3e8ff)',
    transition: 'background 0.2s'
  });
  const tdStyles = {
    padding: '6px 6px',
    textAlign: 'left',
    color: 'var(--history-table-cell-text, #222)'
  };
  if (loading) {
    content = <div>Loading...</div>;
  } else if (error) {
    content = <div style={{ color: 'red' }}>{error}</div>;
  } else {
    content = (
      <div style={{ overflowX: 'auto', boxShadow: '0 2px 12px rgba(0,0,0,0.08)', background: 'var(--history-table-bg, #fff)' }}>
        <table className="history-table" style={{ ...tableStyles }}>
          <colgroup>
            <col style={{ width: '20px' }} />
            <col style={{ width: '20px' }} />
            <col style={{ width: '220px' }} />
            <col style={{ width: '140px' }} />
            <col style={{ width: '180px' }} />
            <col style={{ width: '120px' }} />
          </colgroup>
          <thead>
            <tr>
              <th style={{ ...thStyles, textAlign: 'center', width: '20px' }}></th>
              <th style={{ ...thStyles, textAlign: 'center', width: '20px' }}>Media Type</th>
              <th style={{ ...thStyles, width: '220px' }}>Title</th>
              <th style={{ ...thStyles, width: '140px' }}>Extra Type</th>
              <th style={{ ...thStyles, width: '180px' }}>Extra Title</th>
              <th style={{ ...thStyles, width: '120px' }}>Date</th>
            </tr>
          </thead>
          <tbody>
            {paginatedHistory.map((item, idx) => {
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
                  <td style={{ ...tdStyles, textAlign: 'center', textTransform: 'capitalize', color: 'var(--history-table-media-type, #7c3aed)', fontWeight: 'normal' }}>{item.mediaType}</td>
                  <td style={{ ...tdStyles, fontWeight: 500 }}>
                    {getMediaLink(item) ? (
                      <Link
                        to={getMediaLink(item)}
                        style={{ color: '#6d28d9', textDecoration: 'none', fontWeight: 100 }}
                      >
                        {item.title}
                      </Link>
                    ) : (
                      item.title
                    )}
                  </td>
                  <td style={{ ...tdStyles, color: 'var(--history-table-extra-type, #6d28d9)', fontWeight: 'normal' }}>{item.extraType}</td>
                  <td style={{ ...tdStyles, color: 'var(--history-table-extra-title, #444)' }}>{item.extraTitle}</td>
                  <td style={{ ...tdStyles, color: 'var(--history-table-date, #888)', fontSize: 15 }}>{formatDate(item.date)}</td>
                </tr>
              );
            })}
          </tbody>
        </table>
        {/* Pagination Controls */}
        <div style={{ display: 'flex', justifyContent: 'center', alignItems: 'center', gap: 12, margin: '18px 0' }}>
          <button onClick={() => setPage(p => Math.max(1, p - 1))} disabled={page === 1} style={{ padding: '6px 16px', borderRadius: 6, border: '1px solid #ccc', background: page === 1 ? '#eee' : '#fff', color: '#222', cursor: page === 1 ? 'not-allowed' : 'pointer', fontWeight: 500 }}>Prev</button>
          <span style={{ fontWeight: 600, fontSize: 16 }}>Page {page} of {totalPages}</span>
          <button onClick={() => setPage(p => Math.min(totalPages, p + 1))} disabled={page === totalPages} style={{ padding: '6px 16px', borderRadius: 6, border: '1px solid #ccc', background: page === totalPages ? '#eee' : '#fff', color: '#222', cursor: page === totalPages ? 'not-allowed' : 'pointer', fontWeight: 500 }}>Next</button>
        </div>
      </div>
    );
  }
  // Set icon color variable for dark/light mode
  useEffect(() => {
    const setTableColors = () => {
      const isDark = window.matchMedia('(prefers-color-scheme: dark)').matches;
      document.documentElement.style.setProperty('--history-table-bg', isDark ? '#18181b' : '#fff');
      document.documentElement.style.setProperty('--history-table-text', isDark ? '#e5e7eb' : '#222');
      document.documentElement.style.setProperty('--history-table-header-bg', isDark ? '#27272a' : '#fff');
      document.documentElement.style.setProperty('--history-table-header-text', isDark ? '#fff' : '#222');
      document.documentElement.style.setProperty('--history-table-border', isDark ? '#444' : '#e5e7eb');
      document.documentElement.style.setProperty('--history-table-row-bg1', isDark ? '#232326' : '#f3f3f3');
      document.documentElement.style.setProperty('--history-table-row-bg2', isDark ? '#18181b' : '#fff');
      document.documentElement.style.setProperty('--history-table-cell-text', isDark ? '#e5e7eb' : '#222');
      document.documentElement.style.setProperty('--history-table-media-type', isDark ? '#fff' : '#000');
      document.documentElement.style.setProperty('--history-table-extra-type', isDark ? '#fff' : '#000');
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
    <Container>
      {content}
    </Container>
  );
};

export default HistoryPage;
