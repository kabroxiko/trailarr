import React from 'react';

export default function Container({ children, style = {}, ...props }) {
	const defaultStyle = {
		width: '100%',
		margin: 0,
		height: '100%',
		padding: '2rem',
		background: 'var(--settings-bg, #fff)',
		borderRadius: 0,
		boxShadow: '0 2px 12px #0002',
		color: 'var(--settings-text, #222)',
		boxSizing: 'border-box',
		overflowX: 'hidden',
		overflowY: 'auto',
		position: 'relative',
		...style,
	};
	return (
		<div style={defaultStyle} {...props}>
			{children}
		</div>
	);
}
