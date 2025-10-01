import React from 'react';
import { Link } from 'react-router-dom';

export default function SeriesTable({ series, darkMode }) {
  return (
    <table style={{ width: '100%', borderCollapse: 'collapse' }}>
      <thead>
        <tr style={{ background: darkMode ? '#23232a' : '#f3e8ff' }}>
          <th style={{ textAlign: 'left', padding: '0.5em', color: darkMode ? '#e5e7eb' : '#6d28d9' }}>Poster</th>
          <th style={{ textAlign: 'left', padding: '0.5em', color: darkMode ? '#e5e7eb' : '#6d28d9' }}>Title</th>
          <th style={{ textAlign: 'left', padding: '0.5em', color: darkMode ? '#e5e7eb' : '#6d28d9' }}>Year</th>
          <th style={{ textAlign: 'left', padding: '0.5em', color: darkMode ? '#e5e7eb' : '#6d28d9' }}>Path</th>
        </tr>
      </thead>
      <tbody>
        {series.map((item, idx) => (
          <tr key={idx} style={{ borderBottom: '1px solid #f3e8ff' }}>
            <td style={{ padding: '0.5em', textAlign: 'left' }}>
              <img
                src={`/api/sonarr/poster/${item.id}`}
                style={{ width: 48, height: 72, objectFit: 'cover', borderRadius: 2, background: '#222', boxShadow: '0 1px 4px rgba(0,0,0,0.18)' }}
                onError={e => { e.target.onerror = null; e.target.src = 'https://via.placeholder.com/48x72?text=No+Poster'; }}
              />
            </td>
            <td style={{ padding: '0.5em', textAlign: 'left' }}>
              <Link to={`/series/${item.id}`} style={{ color: '#a855f7', textDecoration: 'underline', cursor: 'pointer', fontWeight: 'bold', textAlign: 'left', display: 'block' }}>{item.title}</Link>
            </td>
            <td style={{ padding: '0.5em', textAlign: 'left' }}>{item.year}</td>
            <td style={{ padding: '0.5em', textAlign: 'left' }}>{item.path}</td>
          </tr>
        ))}
      </tbody>
    </table>
  );
}
