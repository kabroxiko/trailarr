import React from 'react';

export default function SectionHeader({ children, style = {}, ...props }) {
  const defaultStyle = {
    fontWeight: 600,
    fontSize: '1.35em',
    margin: '0 0 18px 8px',
    textAlign: 'left',
    textTransform: 'capitalize',
    color: 'var(--settings-section-header, #222)',
    ...style,
  };
  return (
    <h3 style={defaultStyle} {...props}>
      {children}
    </h3>
  );
}
