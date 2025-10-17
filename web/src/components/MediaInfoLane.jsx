import React, { useState, useEffect } from 'react';
import PropTypes from 'prop-types';
import IconButton from './IconButton.jsx';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faBookmark } from '@fortawesome/free-regular-svg-icons';
import { faLanguage } from '@fortawesome/free-solid-svg-icons';

export default function MediaInfoLane({ media, mediaType, darkMode = false, error: _error = '' }) {
	// Reference unused prop to satisfy ESLint's no-unused-vars rule without changing behavior
	void _error;
	const [cast, setCast] = useState([]);
	const [castLoading, setCastLoading] = useState(false);
	const [castError, setCastError] = useState('');
	const [showAlt, setShowAlt] = useState(false);

	useEffect(() => {
		if (!media || !media.id || !mediaType) {
			setCast([]);
			setCastError('');
			return;
		}
		setCastLoading(true);
		setCastError('');
		let url = '';
		if (mediaType === 'movie') {
			url = `/api/movies/${media.id}/cast`;
		} else if (mediaType === 'series' || mediaType === 'tv') {
			url = `/api/series/${media.id}/cast`;
		} else {
			setCast([]);
			setCastError('Unknown media type');
			setCastLoading(false);
			return;
		}
		fetch(url)
			.then(res => {
				if (!res.ok) throw new Error('Failed to fetch cast');
				return res.json();
			})
			.then(data => {
				setCast(Array.isArray(data.cast) ? data.cast : []);
				setCastLoading(false);
			})
			.catch(() => {
				setCast([]);
				setCastError('Failed to load cast');
				setCastLoading(false);
			});
	}, [media, mediaType]);

	if (!media) return null;

	let background;
	if (mediaType === 'tv') {
		// Position background slightly below the top (around 30%) to show upper-to-middle of the fanart
		background = `url(/mediacover/Series/${media.id}/fanart-1280.jpg) center 30%/cover no-repeat`;
	} else {
		// Position background slightly below the top (around 30%) to show upper-to-middle of the fanart
		background = `url(/mediacover/Movies/${media.id}/fanart-1280.jpg) center 30%/cover no-repeat`;
	}

	return (
		<div style={{
			width: '100%',
			position: 'relative',
			background,
			minHeight: 420,
			display: 'flex',
			flexDirection: 'row',
			alignItems: 'flex-start',
			boxSizing: 'border-box',
			padding: 0,
		}}>
			<div style={{
				position: 'absolute',
				top: 0,
				left: 0,
				width: '100%',
				height: '100%',
				background: 'rgba(0,0,0,0.55)',
				zIndex: 1,
			}} />
			<div style={{ minWidth: 150, zIndex: 2, display: 'flex', justifyContent: 'flex-start', alignItems: 'flex-start', height: '100%', padding: '32px 32px' }}>
				<img
					src={mediaType === 'tv'
						? `/mediacover/Series/${media.id}/poster-500.jpg`
						: `/mediacover/Movies/${media.id}/poster-500.jpg`}
					style={{ height: 370, objectFit: 'cover', borderRadius: 4, background: '#222', boxShadow: '0 2px 8px rgba(0,0,0,0.22)' }}
					onError={e => { e.target.onerror = null; e.target.src = '/logo.svg'; }}
				/>
			</div>
			<div style={{ flex: 1, zIndex: 2, display: 'flex', flexDirection: 'column', justifyContent: 'flex-start', height: '100%', marginLeft: 32, marginTop: 32 }}>
				<h2 style={{ color: '#fff', margin: 0, fontSize: 32, fontWeight: 600, textShadow: '0 1px 2px #000', letterSpacing: 0.2, textAlign: 'left', display: 'flex', alignItems: 'center', gap: 8 }}>
					<IconButton icon={<FontAwesomeIcon icon={faBookmark} color="#eee" style={{ marginLeft: -10 }} />} disabled style={{ background: 'none', border: 'none', padding: 0, margin: 0 }} />
					<span style={{ display: 'inline-flex', alignItems: 'center', gap: 8 }}>
						<span>{media.title}</span>
						{(() => {
							const raw = media.alternateTitles || [];
							const altArr = raw.map(item => (typeof item === 'string' ? item : (item.title || item.name || item.Title || JSON.stringify(item))));
							const original = media.original_title || media.originalTitle || media.OriginalTitle || '';
							const norm = s => (s || '').toString().trim();
							const displayed = norm(media.title || '');
							const hasAlts = Array.isArray(media.alternateTitles) && media.alternateTitles.length > 0;
							const seen = new Set();
							const filteredAlt = altArr.map(a => norm(a)).filter(a => {
								if (!a) return false;
								if (a === displayed) return false;
								if (seen.has(a)) return false;
								seen.add(a);
								return true;
							});
							const origNorm = norm(original);
							const showOriginal = origNorm && origNorm !== displayed && !seen.has(origNorm);
							const showIcon = hasAlts || showOriginal;
							if (!showIcon) return null;
							return (
								<span style={{ position: 'relative', display: 'inline-flex', alignItems: 'center' }}>
									<button
										aria-label={`${altArr.length} alternate titles`}
										onMouseEnter={() => setShowAlt(true)}
										onMouseLeave={() => setShowAlt(false)}
										onFocus={() => setShowAlt(true)}
										onBlur={() => setShowAlt(false)}
										style={{ background: 'transparent', border: 'none', color: '#fff', cursor: 'default', padding: 6, marginLeft: 2 }}
									>
										<FontAwesomeIcon icon={faLanguage} style={{ fontSize: 18, color: '#eee' }} />
									</button>
									{showAlt && (
										<div
											role="tooltip"
											style={{
												position: 'absolute',
												top: '110%',
												left: 0,
												zIndex: 60,
												background: darkMode ? '#111' : '#fff',
												color: darkMode ? '#e5e7eb' : '#111',
												border: darkMode ? '1px solid #333' : '1px solid #ddd',
												boxShadow: '0 6px 18px rgba(0,0,0,0.12)',
												padding: 8,
												borderRadius: 8,
												minWidth: 200,
												maxWidth: 420,
												maxHeight: 220,
												overflow: 'auto',
												fontSize: 13,
											}}
										>
											{showOriginal && (
												<div style={{ marginBottom: 8 }}>
													<div style={{ fontWeight: 600, marginBottom: 6 }}>Original title</div>
													<ul style={{ margin: 0, paddingLeft: 16, marginBottom: 8 }}>
														<li style={{ marginBottom: 6, lineHeight: 1.2 }}>{original}</li>
													</ul>
												</div>
											)}
											{filteredAlt.length > 0 && (
												<div>
													<div style={{ fontWeight: 600, marginBottom: 6 }}>Alternate titles</div>
													<ul style={{ margin: 0, paddingLeft: 16 }}>
														{filteredAlt.map((t, i) => (
															<li key={i} style={{ marginBottom: 6, lineHeight: 1.2 }}>{t}</li>
														))}
													</ul>
												</div>
											)}
										</div>
									)}
								</span>
							);
						})()}
					</span>
				</h2>
				{media.overview && (
					<div style={{ color: '#e5e7eb', fontSize: 15, margin: '10px 0 6px 0', textShadow: '0 1px 2px #000', textAlign: 'left', lineHeight: 1.5, maxWidth: 700 }}>
						{media.overview}
					</div>
				)}
				<div style={{ marginBottom: 6, color: '#e5e7eb', textAlign: 'left', fontSize: 13, textShadow: '0 1px 2px #000' }}>{media.year} &bull; {media.path}</div>
				<div style={{ marginBottom: 10, width: '100%' }}>
					<div style={{ height: 20, marginBottom: 4 }} />
					{castLoading && <div style={{ fontSize: '0.95em', color: '#bbb' }}>Loading cast...</div>}
					{castError && <div style={{ color: 'red', fontSize: '0.95em' }}>{castError}</div>}
					{!castLoading && !castError && cast && cast.length > 0 && (
						<div style={{ display: 'flex', flexWrap: 'wrap', gap: '1.2em 1.5em' }}>
							{cast.slice(0, 10).map(actor => (
								<div key={actor.id} style={{ display: 'flex', flexDirection: 'column', alignItems: 'center', gap: 3, minWidth: 0 }}>
									{actor.profile_path && actor.profile_path !== '' ? (
										<img
											src={`https://image.tmdb.org/t/p/w185${actor.profile_path}`}
											alt={actor.name}
											style={{ width: 56, height: 80, objectFit: 'cover', borderRadius: 4, background: '#2222', marginBottom: 2 }}
											onError={e => { e.target.onerror = null; e.target.src = '/logo.svg'; }}
										/>
									) : (
										<div style={{ width: 56, height: 80, background: '#4444', borderRadius: 4, marginBottom: 2, display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
											<svg width="32" height="32" viewBox="0 0 32 32" fill="none" xmlns="http://www.w3.org/2000/svg">
												<circle cx="16" cy="12" r="7" fill="#888" />
												<ellipse cx="16" cy="25" rx="11" ry="7" fill="#888" />
											</svg>
										</div>
									)}
									<span style={{ fontWeight: 500, fontSize: '0.68em', color: '#fff', whiteSpace: 'nowrap', textOverflow: 'ellipsis', overflow: 'hidden', maxWidth: 80, textAlign: 'center' }}>{actor.name}</span>
									<span style={{ fontSize: '0.60em', color: '#fff', whiteSpace: 'nowrap', textOverflow: 'ellipsis', overflow: 'hidden', maxWidth: 80, textAlign: 'center' }}>{actor.character}</span>
								</div>
							))}
						</div>
					)}
					{!castLoading && !castError && (!cast || cast.length === 0) && (
						<div style={{ fontSize: '0.95em', color: '#bbb' }}>No cast information available.</div>
					)}
				</div>
			</div>
		</div>
	);
}

MediaInfoLane.propTypes = {
	media: PropTypes.object,
	mediaType: PropTypes.oneOf(['movie', 'series', 'tv']),
	darkMode: PropTypes.bool,
	error: PropTypes.string,
};
