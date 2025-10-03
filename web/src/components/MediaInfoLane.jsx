import React from 'react';

export default function MediaInfoLane({ searchLoading, handleSearchExtras }) {
  return (
    <div style={{ display: 'flex', alignItems: 'center', justifyContent: 'flex-start', margin: '0px 0 0 0', padding: 0, width: '100%' }}>
      <div
        style={{ display: 'flex', alignItems: 'center', gap: 8, cursor: 'pointer', fontWeight: 'bold', color: '#e5e7eb', fontSize: 18 }}
        onClick={handleSearchExtras}
      >
        <span style={{ fontSize: 20, display: 'flex', alignItems: 'center' }}>
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
            <circle cx="9" cy="9" r="7" stroke="#e5e7eb" strokeWidth="2" />
            <line x1="15" y1="15" x2="19" y2="19" stroke="#e5e7eb" strokeWidth="2" strokeLinecap="round" />
          </svg>
        </span>
        <span>{searchLoading ? 'Searching...' : 'Search'}</span>
      </div>
    </div>
  );
}
