import React from 'react';
import PropTypes from 'prop-types';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faSave } from '@fortawesome/free-solid-svg-icons';

function SaveLane({ onSave, saving, isChanged, error }) {
  return (
    <div style={{ position: 'absolute', top: 0, left: 0, width: '100%', background: 'var(--save-lane-bg, #f3f4f6)', color: 'var(--save-lane-text, #222)', padding: '0.7rem 2rem', display: 'flex', alignItems: 'center', gap: '1rem', zIndex: 10, boxShadow: '0 2px 8px #0001' }}>
      <button onClick={onSave} disabled={saving || !isChanged} style={{ background: 'none', color: '#222', border: 'none', padding: '0.3rem 1rem', cursor: saving || !isChanged ? 'not-allowed' : 'pointer', opacity: saving || !isChanged ? 0.7 : 1, display: 'flex', flexDirection: 'column', alignItems: 'center', gap: '0.2rem' }}>
        <FontAwesomeIcon icon={faSave} style={{ fontSize: 22, color: 'var(--save-lane-text, #222)' }} />
        <span style={{ fontWeight: 500, fontSize: '0.85em', color: 'var(--save-lane-text, #222)', marginTop: 2, display: 'flex', flexDirection: 'column', alignItems: 'center', lineHeight: 1.1 }}>
          <span>{saving || !isChanged ? 'No' : 'Save'}</span>
          <span>Changes</span>
        </span>
      </button>
      {error && <div style={{ marginLeft: 16, color: '#f44', fontWeight: 500 }}>{error}</div>}
    </div>
  );
}

SaveLane.propTypes = {
  onSave: PropTypes.func.isRequired,
  saving: PropTypes.bool.isRequired,
  isChanged: PropTypes.bool.isRequired,
  error: PropTypes.string,
};

export default SaveLane;
