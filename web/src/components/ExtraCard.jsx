import React, { useState } from 'react';
import IconButton from './IconButton.jsx';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faTrashCan, faCheckSquare } from '@fortawesome/free-regular-svg-icons';
import { faPlay, faDownload, faBan, faCircleXmark, faClock } from '@fortawesome/free-solid-svg-icons';

export default function ExtraCard({
  extra,
  idx,
  typeExtras,
  darkMode,
  media,
  mediaType,
  setExtras,
  setModalMsg,
  setShowModal,
  // YoutubeEmbed, // removed unused
  rejected: rejectedProp,
  onPlay,
  onDownloaded,
  showToast, // new prop for toast/modal
}) {
  const [imgError, setImgError] = useState(false);
  const [imgLoaded, setImgLoaded] = useState(false);
  const [isFallback, setIsFallback] = useState(false);
  const baseTitle = extra.ExtraTitle || '';
  const baseType = extra.ExtraType || '';
  const totalCount = typeExtras.filter(e => e.ExtraTitle === baseTitle).length;
  let displayTitle = totalCount > 1 ? `${baseTitle} (${typeExtras.slice(0, idx + 1).filter(e => e.ExtraTitle === baseTitle).length})` : baseTitle;
  const maxLen = 40;
  if (displayTitle.length > maxLen) {
    displayTitle = displayTitle.slice(0, maxLen - 3) + '...';
  }
  let posterUrl = extra.YoutubeId ? `/api/proxy/youtube-image/${extra.YoutubeId}` : null;
  React.useEffect(() => {
    // Reset states when posterUrl changes
    setImgError(false);
    setImgLoaded(false);
    setIsFallback(false);
  }, [posterUrl]);
  let titleFontSize = 16;
  if (displayTitle.length > 22) titleFontSize = 14;
  if (displayTitle.length > 32) titleFontSize = 12;
  const downloaded = extra.Status === 'downloaded';
  const isDownloading = extra.Status === 'downloading';
  const isQueued = extra.Status === 'queued';
  const failed = extra.Status === 'failed' || extra.Status === 'rejected' || extra.Status === 'unknown' || extra.Status === 'error';
  const exists = extra.Status === 'exists';
  const [downloading, setDownloading] = useState(false);
  // Use the rejected prop if provided, otherwise fallback to extra.Status
  const [unbanned, setUnbanned] = useState(false);
  // Treat 'failed' as 'rejected' for UI
  const rejected = !unbanned && (typeof rejectedProp === 'boolean' ? rejectedProp : extra.Status === 'rejected' || extra.Status === 'failed');
  const [errorCard, setErrorCard] = useState(null);
  const isError = errorCard === idx;
  // Helper to show error as toast/modal
  const showErrorModal = (msg) => {
    if (msg.includes('UNPLAYABLE') || msg.includes('no se encuentra disponible')) {
      msg = 'This YouTube video is unavailable and cannot be downloaded.';
    }
    if (typeof showToast === 'function') {
      showToast(msg);
    } else {
      setModalMsg(msg);
      setShowModal(true);
    }
    setErrorCard(idx);
  };

  const handleDownloadClick = async () => {
    if (downloaded || downloading) return;
    setDownloading(true);
    try {
      const res = await fetch(`/api/extras/download`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          mediaType,
          mediaId: media.id,
          extraType: baseType,
          extraTitle: baseTitle,
          youtubeId: extra.YoutubeId
        })
      });
      if (res.ok) {
        setErrorCard(null);
        // If this card is a search-only card (not in backend), add a backend-style extra to extras state
        if (typeof setExtras === 'function' && !downloaded && !isQueued && !isDownloading && !exists) {
          setExtras(prev => {
            // If already present as a backend extra, do nothing
            if (prev.some(ex => ex.YoutubeId === extra.YoutubeId && ex.Status && ex.Status !== '')) return prev;
            // Add a new backend-style extra with Status 'queued' (optimistic)
            return [
              ...prev,
              {
                ...extra,
                Status: 'queued',
                reason: '',
                Reason: '',
                ExtraType: baseType,
                ExtraTitle: baseTitle,
                YoutubeId: extra.YoutubeId,
                // Add any other backend fields as needed
              }
            ];
          });
        }
      } else {
        const data = await res.json();
        if (typeof showToast === 'function') {
          showToast(data?.error || 'Download failed');
        } else {
          showErrorModal(data?.error || 'Download failed');
        }
        // If failed, revert status
        if (typeof setExtras === 'function') {
          setExtras(prev => prev.map((ex) =>
            ex.YoutubeId === extra.YoutubeId && ex.ExtraType === baseType && ex.ExtraTitle === baseTitle
              ? { ...ex, Status: '' }
              : ex
          ));
        }
      }
    } catch (error) {
      if (typeof showToast === 'function') {
        showToast(error.message || error);
      } else {
        showErrorModal(error.message || error);
      }
      // If failed, revert status
      if (typeof setExtras === 'function') {
        setExtras(prev => prev.map((ex) =>
          ex.YoutubeId === extra.YoutubeId && ex.ExtraType === baseType && ex.ExtraTitle === baseTitle
            ? { ...ex, Status: '' }
            : ex
        ));
      }
    } finally {
      setDownloading(false);
    }
  };

  // Poster image or fallback factory
  function PosterImage({ src, alt, onError, onLoad, fallbackIcon, loaded }) {
    return (
      <div style={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        {src ? (
          <img
            src={src}
            alt={alt}
            onLoad={onLoad}
            onError={onError}
            style={{
              display: 'block',
              margin: '0 auto',
              maxHeight: 135,
              maxWidth: '100%',
              objectFit: 'contain',
              background: '#222222'
            }}
          />
        ) : !src || imgError ? (
          <span style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: '100%',
            height: 135,
            background: '#222222'
          }}>
            <FontAwesomeIcon icon={fallbackIcon} color="#888" size="4x" />
          </span>
        ) : (
          // Skeleton placeholder while loading
          <span style={{
            display: 'flex',
            alignItems: 'center',
            justifyContent: 'center',
            width: '100%',
            height: 135,
            background: '#333',
            animation: 'pulse 1.2s infinite',
            color: '#444'
          }}>Loading...</span>
        )}
      </div>
    );
  }

  // Determine border color based on status
  let borderColor = '2px solid transparent';
  if (rejected || isError || failed) {
    borderColor = '2.5px solid #ef4444';
  } else if (downloaded) {
    borderColor = '2px solid #22c55e';
  } else if (exists) {
    borderColor = '2px solid #8888';
  }
  return (
    <div
      title={rejected ? (extra.reason ? `Rejected: ${extra.reason}` : 'Rejected (cannot download)') : undefined}
      style={{
        width: 180,
        height: 210,
        background: darkMode ? '#18181b' : '#fff',
        borderRadius: 12,
        boxShadow: darkMode ? '0 2px 12px rgba(0,0,0,0.22)' : '0 2px 12px rgba(0,0,0,0.10)',
        overflow: 'hidden',
        display: 'flex',
        flexDirection: 'column',
        alignItems: 'center',
        padding: '0 0 0 0',
        position: 'relative',
        border: borderColor,
      }}
    >
      {/* Image or poster rendering restored */}
      <div style={{ width: '100%', height: 135, background: '#222', display: 'flex', alignItems: 'center', justifyContent: 'center', position: 'relative' }}>
        {/* Play button overlay (with image) */}
        {extra.YoutubeId && !imgError && !isFallback && (
          <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', zIndex: 2 }}>
            <IconButton
              icon={<FontAwesomeIcon icon={faPlay} color="#fff" size="lg" style={{ filter: 'drop-shadow(0 2px 8px #000)' }} />}
              title="Play"
              onClick={e => {
                e.stopPropagation();
                if (onPlay) onPlay(extra.YoutubeId);
              }}
            />
          </div>
        )}
        {/* Poster Image or Fallback */}
        {!imgError && posterUrl ? (
          <PosterImage
            key={posterUrl}
            src={posterUrl}
            alt={displayTitle}
            fallbackIcon={faBan}
            loaded={imgLoaded}
            onLoad={(e) => {
              // If the image loads but is very small (our SVG fallback is 64x64), treat as fallback
              try {
                const img = e.target;
                if (img.naturalWidth <= 64 && img.naturalHeight <= 64) {
                  setIsFallback(true);
                  setImgError(true);
                } else {
                  setImgLoaded(true);
                }
              } catch (err) {
                setImgLoaded(true);
              }
            }}
            onError={() => {
              setIsFallback(true);
              setImgError(true);
            }}
          />
        ) : (
          <PosterImage src={null} alt="Denied" fallbackIcon={faBan} loaded={false} />
        )}
        {/* Failed/Rejected Icon (always show for failed/rejected) */}
        {(extra.Status === 'failed' || extra.Status === 'rejected' || extra.Status === 'unknown' || extra.Status === 'error') && (
          <div style={{ position: 'absolute', top: 8, left: 8, zIndex: 2 }}>
            <IconButton
              icon={<FontAwesomeIcon icon={faCircleXmark} color="#ef4444" size="lg" />}
              title={extra.Status === 'failed' ? 'Remove failed status' : 'Remove ban'}
              onClick={async event => {
                event.stopPropagation();
                // Always call backend to remove from blacklist for both rejected and failed statuses
                try {
                  await fetch('/api/blacklist/extras/remove', {
                    method: 'POST',
                    headers: { 'Content-Type': 'application/json' },
                    body: JSON.stringify({
                      mediaType,
                      mediaId: media.id,
                      extraType: baseType,
                      extraTitle: baseTitle,
                      youtubeId: extra.YoutubeId
                    })
                  });
                  // Optimistically update UI immediately
                  if (typeof setExtras === 'function') {
                    setExtras(prev => prev.map((ex) =>
                      ex.YoutubeId === extra.YoutubeId && ex.ExtraType === baseType && ex.ExtraTitle === baseTitle
                        ? { ...ex, Status: '' }
                        : ex
                    ));
                  }
                  // Then refresh from backend
                  if (typeof setExtras === 'function' && typeof media !== 'undefined' && media.id) {
                    try {
                      const res = await fetch(`/api/${mediaType === 'movie' ? 'movies' : 'series'}/${media.id}/extras`);
                      if (res.ok) {
                        const data = await res.json();
                        if (Array.isArray(data)) {
                          setExtras(data);
                        } else if (data && Array.isArray(data.extras)) {
                          setExtras(data.extras);
                        }
                      }
                    } catch (e) { /* ignore */ }
                  }
                } catch {
                  setModalMsg('Failed to remove ban.');
                  setShowModal(true);
                }
              }}
              aria-label={extra.Status === 'failed' ? 'Remove failed status' : 'Remove ban'}
            />
          </div>
        )}
        {/* Download or Delete Buttons */}
        {extra.YoutubeId && !downloaded && !imgError && !isFallback && (
          <div style={{ position: 'absolute', top: 8, right: 8, zIndex: 2 }}>
            <IconButton
              icon={
                isDownloading ? (
                  <span className="download-spinner" style={{ display: 'inline-block', width: 22, height: 22, background: 'transparent' }}>
                    <svg viewBox="0 0 50 50" style={{ width: 22, height: 22, background: 'transparent' }}>
                      <circle cx="25" cy="25" r="20" fill="none" stroke="#fff" strokeWidth="5" strokeDasharray="31.4 31.4" strokeLinecap="round">
                        <animateTransform attributeName="transform" type="rotate" from="0 25 25" to="360 25 25" dur="0.8s" repeatCount="indefinite" />
                      </circle>
                    </svg>
                  </span>
                ) : isQueued ? (
                  <FontAwesomeIcon icon={faClock} color="#fff" size="lg" />
                ) : (
                  <FontAwesomeIcon icon={faDownload} color="#fff" size="lg" />
                )
              }
              title={rejected ? (extra.reason ? `Rejected: ${extra.reason}` : 'Rejected (cannot download)') : isDownloading ? 'Downloading...' : isQueued ? 'Queued' : 'Download'}
              onClick={rejected || isDownloading || isQueued
                ? undefined
                : (e => {
                    e.stopPropagation();
                    handleDownloadClick();
                  })}
              disabled={rejected || isDownloading || isQueued}
              aria-label="Download"
              style={{ opacity: rejected ? 0.5 : (isDownloading || isQueued ? 0.7 : 1), background: 'transparent', borderRadius: (isDownloading || isQueued) ? 8 : 0, transition: 'background 0.2s, opacity 0.2s' }}
            />
          </div>
        )}
        {/* Downloaded Checkmark and Delete Button */}
        {downloaded && (
          <>
            <div style={{ position: 'absolute', top: 8, right: 8, zIndex: 2 }}>
              <IconButton icon={<FontAwesomeIcon icon={faCheckSquare} color="#22c55e" size="lg" />} title="Downloaded" disabled />
            </div>
            <div style={{ position: 'absolute', bottom: 8, right: 8, zIndex: 2 }}>
              <IconButton
                icon={<FontAwesomeIcon icon={faTrashCan} color="#ef4444" size="md" />}
                title="Delete"
                onClick={async (event) => {
                  event.stopPropagation();
                  if (!window.confirm('Delete this extra?')) return;
                  try {
                    const { deleteExtra } = await import('../api');
                    const payload = {
                      mediaType,
                      mediaId: media.id,
                      youtubeId: extra.YoutubeId
                    };
                    await deleteExtra(payload);
                    setExtras(prev => prev.map((ex) =>
                      ex.ExtraTitle === baseTitle && ex.ExtraType === baseType ? { ...ex, Status: 'missing' } : ex
                    ));
                  } catch (error) {
                    let msg = error?.message || error;
                    if (error?.detail) msg += `\n${error.detail}`;
                    showErrorModal(msg || 'Delete failed');
                  }
                }}
              />
            </div>
          </>
        )}
      </div>
      <div style={{ width: '100%', padding: '12px 10px 0 10px', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
        <div style={{ fontWeight: 600, fontSize: titleFontSize, color: darkMode ? '#e5e7eb' : '#222', textAlign: 'center', marginBottom: 4, height: 50, display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden', width: '100%' }}>{displayTitle}</div>
        <div style={{ width: '100%', display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: 18, position: 'absolute', bottom: 12, left: 0 }}></div>
        {/* YouTube modal is now rendered at the page level */}
      </div>
    </div>
  );
}
