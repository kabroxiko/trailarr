import React, { useEffect } from 'react';
import PropTypes from 'prop-types';

function ToastModal({
  message,
  onClose,
  darkMode,
  borderColor,
  iconBg,
  closeBtnColor,
  closeBtnHoverBg,
  closeBtnHoverColor,
  toastBg,
  toastColor,
  toastBoxShadow,
  success
}) {
  const handleMouseOverOrFocus = e => {
    e.target.style.backgroundColor = closeBtnHoverBg;
    e.target.style.color = closeBtnHoverColor;
  };
  const handleMouseOutOrBlur = e => {
    e.target.style.backgroundColor = 'transparent';
    e.target.style.color = closeBtnColor;
  };
  const backdropStyle = {
    position: 'fixed',
    top: 0,
    left: 0,
    right: 0,
    bottom: 0,
    backgroundColor: 'rgba(0, 0, 0, 0.1)',
    zIndex: 99998,
    pointerEvents: 'none'
  };
  const toastStyle = {
    position: 'fixed',
    left: 20,
    bottom: 20,
    zIndex: 99999,
    background: toastBg,
    color: toastColor,
    border: `2px solid ${borderColor}`,
    borderRadius: 12,
    padding: '20px 24px',
    minWidth: 300,
    maxWidth: 400,
    boxShadow: toastBoxShadow,
    fontSize: 15,
    fontWeight: 500,
    display: 'flex',
    alignItems: 'flex-start',
    gap: 16,
    animation: 'toastSlideIn 0.3s ease-out',
    backdropFilter: 'blur(8px)',
    WebkitBackdropFilter: 'blur(8px)',
  };
  const iconStyle = {
    width: 20,
    height: 20,
    borderRadius: '50%',
    backgroundColor: iconBg,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    color: 'white',
    fontSize: 12,
    fontWeight: 'bold',
    flexShrink: 0,
    marginTop: 2
  };
  const messageStyle = {
    flex: 1,
    lineHeight: 1.4,
    wordWrap: 'break-word'
  };
  const closeBtnStyle = {
    background: 'none',
    border: 'none',
    color: closeBtnColor,
    fontSize: 18,
    cursor: 'pointer',
    padding: 4,
    borderRadius: 4,
    display: 'flex',
    alignItems: 'center',
    justifyContent: 'center',
    width: 24,
    height: 24,
    flexShrink: 0,
    transition: 'all 0.2s ease'
  };
  return (
    <>
      <div style={backdropStyle} />
      <div style={toastStyle}>
        <div style={iconStyle}>{success ? '✓' : '!'}</div>
        <div style={messageStyle}>{message}</div>
        <button
          onClick={onClose}
          style={closeBtnStyle}
          title="Close"
          onMouseOver={handleMouseOverOrFocus}
          onFocus={handleMouseOverOrFocus}
          onMouseOut={handleMouseOutOrBlur}
          onBlur={handleMouseOutOrBlur}
        >
          ×
        </button>
      </div>
      <style>{`
        @keyframes toastSlideIn {
          from {
            transform: translateX(-100%);
            opacity: 0;
          }
          to {
            transform: translateX(0);
            opacity: 1;
          }
        }
      `}</style>
    </>
  );
}

function Toast({ message, onClose, darkMode, autoClose = true, duration = 4000, success = false }) {
  useEffect(() => {
    if (message && autoClose) {
      const timer = setTimeout(() => {
        onClose();
      }, duration);
      return () => clearTimeout(timer);
    }
  }, [message, autoClose, duration, onClose]);

  if (!message) return null;

  let borderColor;
  if (success) {
    borderColor = darkMode ? '#22c55e' : '#16a34a';
  } else {
    borderColor = darkMode ? '#ef4444' : '#dc2626';
  }

  let iconBg;
  if (success) {
    iconBg = darkMode ? '#22c55e' : '#16a34a';
  } else {
    iconBg = '#ef4444';
  }
  const closeBtnColor = darkMode ? '#9ca3af' : '#6b7280';
  const closeBtnHoverBg = darkMode ? '#374151' : '#f3f4f6';
  const closeBtnHoverColor = darkMode ? '#e5e7eb' : '#1f2937';
  const toastBg = darkMode ? '#1f1f23' : '#ffffff';
  const toastColor = darkMode ? '#e5e7eb' : '#1f2937';
  const toastBoxShadow = darkMode
    ? '0 10px 25px rgba(0, 0, 0, 0.5), 0 4px 10px rgba(0, 0, 0, 0.3)'
    : '0 10px 25px rgba(0, 0, 0, 0.15), 0 4px 10px rgba(0, 0, 0, 0.1)';

  return (
    <ToastModal
      message={message}
      onClose={onClose}
      darkMode={darkMode}
      borderColor={borderColor}
      iconBg={iconBg}
      closeBtnColor={closeBtnColor}
      closeBtnHoverBg={closeBtnHoverBg}
      closeBtnHoverColor={closeBtnHoverColor}
      toastBg={toastBg}
      toastColor={toastColor}
      toastBoxShadow={toastBoxShadow}
      success={success}
    />
  );
}

Toast.propTypes = {
  message: PropTypes.string,
  onClose: PropTypes.func.isRequired,
  darkMode: PropTypes.bool,
  autoClose: PropTypes.bool,
  duration: PropTypes.number,
  success: PropTypes.bool,
};

export default Toast;
