import React from 'react';

export default function Header({ darkMode, search, setSearch }) {
  return (
    <header style={{ width: '100%', height: 64, background: darkMode ? '#23232a' : '#fff', display: 'flex', alignItems: 'center', justifyContent: 'space-between', boxShadow: darkMode ? '0 1px 4px #222' : '0 1px 4px #e5e7eb', padding: '0 32px', position: 'relative', zIndex: 10 }}>
      <div style={{ display: 'flex', alignItems: 'center', gap: 16 }}>
        <img src="/logo.svg" alt="Logo" style={{ width: 40, height: 40, marginRight: 12 }} />
        <span style={{ fontWeight: 'bold', fontSize: 22, color: '#e5e7eb', letterSpacing: 0.5 }}>Trailarr</span>
      </div>
      <nav style={{ display: 'flex', alignItems: 'center', gap: 24 }}>
        <input
          type="search"
          placeholder="Search movies or series"
          value={search}
          onChange={e => setSearch(e.target.value)}
          style={{ padding: '0.5em', borderRadius: 6, border: '1px solid #e5e7eb', width: 200, textAlign: 'left', color: darkMode ? '#e5e7eb' : '#222', background: darkMode ? '#23232a' : '#fff' }}
        />
        <span style={{ fontSize: 20, display: 'flex', alignItems: 'center' }}>
          <svg width="20" height="20" viewBox="0 0 20 20" fill="none" xmlns="http://www.w3.org/2000/svg">
            <circle cx="9" cy="9" r="7" stroke="#e5e7eb" strokeWidth="2" />
            <line x1="15" y1="15" x2="19" y2="19" stroke="#e5e7eb" strokeWidth="2" strokeLinecap="round" />
          </svg>
        </span>
      </nav>
    </header>
  );
}
