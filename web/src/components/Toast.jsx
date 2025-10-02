import React from 'react';

function Toast({ message, onClose, darkMode }) {
  if (!message) return null;
  return (
    <div style={{
      position: 'fixed',
      left: 24,
      bottom: 24,
      zIndex: 99999,
      background: darkMode ? '#222' : '#fff',
      color: darkMode ? '#fff' : '#222',
      border: '2px solid #ef4444',
      borderRadius: 8,
      padding: '16px 24px',
      minWidth: 240,
      boxShadow: '0 2px 16px rgba(0,0,0,0.18)',
      fontSize: 16,
      fontWeight: 500,
      display: 'flex',
      alignItems: 'center',
      gap: 16,
      animation: 'fadein 0.2s',
    }}>
      <span style={{ flex: 1 }}>{message}</span>
      <button onClick={onClose} style={{ background: 'none', border: 'none', color: darkMode ? '#fff' : '#222', fontSize: 22, cursor: 'pointer', marginLeft: 8 }} title="Close">×</button>
    </div>
  );
}

export default Toast;
