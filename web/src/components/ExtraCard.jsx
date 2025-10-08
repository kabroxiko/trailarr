import React, { useState } from 'react';
import YoutubePlayer from './YoutubePlayer.jsx';
import IconButton from './IconButton.jsx';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faTrashCan, faCheckSquare } from '@fortawesome/free-regular-svg-icons';
import { faPlay, faDownload, faBan, faCircleXmark } from '@fortawesome/free-solid-svg-icons';

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
  youtubeModal,
  setYoutubeModal,
  YoutubeEmbed,
  rejected: rejectedProp,
  onRemoveBan,
}) {
  const [imgError, setImgError] = useState(false);
  const baseTitle = extra.Title;
  const totalCount = typeExtras.filter(e => e.Title === baseTitle).length;
  let displayTitle = totalCount > 1 ? `${baseTitle} (${typeExtras.slice(0, idx + 1).filter(e => e.Title === baseTitle).length})` : baseTitle;
  const maxLen = 40;
  if (displayTitle.length > maxLen) {
    displayTitle = displayTitle.slice(0, maxLen - 3) + '...';
  }
  let posterUrl = `https://img.youtube.com/vi/${extra.YoutubeId}/hqdefault.jpg`;
  React.useEffect(() => {
    let cancelled = false;
    setImgError(false);
    if (posterUrl) {
      fetch(posterUrl, { method: 'HEAD' })
        .then(res => {
          if (!res.ok && !cancelled) {
            setImgError(true);
          }
        })
        .catch(err => {
          if (!cancelled) {
            setImgError(true);
          }
        });
    }
    return () => { cancelled = true; };
  }, [posterUrl]);
  let titleFontSize = 16;
  if (displayTitle.length > 22) titleFontSize = 14;
  if (displayTitle.length > 32) titleFontSize = 12;
  const downloaded = extra.Status === 'downloaded';
  // Use the rejected prop if provided, otherwise fallback to extra.Status
  const rejected = typeof rejectedProp === 'boolean' ? rejectedProp : extra.Status === 'rejected';
  const [errorCard, setErrorCard] = useState(null);
  const isError = errorCard === idx;
  // Helper to show modal with error message
  const showErrorModal = (msg) => {
    if (msg.includes('UNPLAYABLE') || msg.includes('no se encuentra disponible')) {
      msg = 'This YouTube video is unavailable and cannot be downloaded.';
    }
    setModalMsg(msg);
    setShowModal(true);
    setErrorCard(idx);
  };

  const handleDownloadClick = async () => {
    if (downloaded) return;
    try {
      const res = await fetch(`/api/extras/download`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          mediaType,
          mediaId: media.id,
          extraType: extra.Type,
          extraTitle: extra.Title,
          youtubeId: extra.YoutubeId
        })
      });
      if (res.ok) {
        setExtras(prev => prev.map((e, i) =>
          e.Title === extra.Title && e.Type === extra.Type ? { ...e, Status: 'downloaded' } : e
        ));
        setErrorCard(null);
      } else {
        const data = await res.json();
        showErrorModal(data?.error || 'Download failed');
      }
    } catch (e) {
      showErrorModal(e.message || e);
    }
  };
  // Poster image or fallback factory
  function PosterImage({ src, alt, onError, fallbackIcon }) {
    return (
      <div style={{ width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        {src ? (
          <img
            src={src}
            alt={alt}
            style={{
              display: 'block',
              margin: '0 auto',
              maxHeight: 135,
              maxWidth: '100%',
              objectFit: 'contain',
              background: '#222222'
            }}
            onError={onError}
          />
        ) : (
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
        )}
      </div>
    );
  }

  return (
    <div style={{
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
      border: (rejected || isError)
        ? '2.5px solid #ef4444'
        : downloaded
          ? '2px solid #22c55e'
          : '2px solid transparent',
    }}>
      <div
        style={{ width: '100%', background: '#222', display: 'flex', alignItems: 'center', justifyContent: 'center', position: 'relative' }}
      >
        <div
          style={{ position: 'relative', width: '100%', height: '100%', display: 'flex', alignItems: 'center', justifyContent: 'center', cursor: extra.YoutubeId && !imgError ? 'pointer' : 'default' }}
          onClick={() => {
            if (extra.YoutubeId && !imgError) setYoutubeModal({ open: true, videoId: extra.YoutubeId });
          }}
        >
          {/** Play button overlay */}
          {extra.YoutubeId && !imgError && (
            <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', zIndex: 2 }}>
              <IconButton
                icon={<FontAwesomeIcon icon={faPlay} color="#fff" size="lg" style={{ filter: 'drop-shadow(0 2px 8px #000)' }} />}
                title="Play"
                onClick={e => {
                  e.stopPropagation();
                  setYoutubeModal({ open: true, videoId: extra.YoutubeId });
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
              onError={e => {
                setImgError(true);
              }}
            />
          ) : (
            <PosterImage src={null} alt="Denied" fallbackIcon={faBan} />
          )}
          {/* Remove Ban Button (Unban) */}
          {rejected && typeof onRemoveBan === 'function' && (
            <div style={{ position: 'absolute', top: 8, left: 8, zIndex: 2 }}>
              <IconButton
                icon={<FontAwesomeIcon icon={faCircleXmark} color="#ef4444" size="lg" />}
                title="Remove ban"
                onClick={e => {
                  e.stopPropagation();
                  onRemoveBan();
                }}
                aria-label="Remove ban"
              />
            </div>
          )}
          {/* Download or Delete Buttons */}
          {extra.YoutubeId && !downloaded && !imgError && (
            <div style={{ position: 'absolute', top: 8, right: 8, zIndex: 2 }}>
              <IconButton
                icon={<FontAwesomeIcon icon={faDownload} color={rejected ? '#aaa' : '#fff'} size="lg" />}
                title={rejected ? (extra.reason ? `Rejected: ${extra.reason}` : 'Rejected (cannot download)') : 'Download'}
                onClick={rejected
                  ? () => { if (extra.reason) setModalMsg(extra.reason); setShowModal(true); }
                  : handleDownloadClick}
                disabled={rejected}
                aria-label="Download"
                style={{ opacity: rejected ? 0.5 : 1 }}
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
                  onClick={async () => {
                    if (!window.confirm('Delete this extra?')) return;
                    try {
                      const { deleteExtra } = await import('../api');
                      const payload = {
                        mediaType,
                        mediaId: media.id,
                        extraType: extra.Type,
                        extraTitle: extra.Title
                      };
                      await deleteExtra(payload);
                      setExtras(prev => prev.map((e, i) =>
                        e.Title === extra.Title && e.Type === extra.Type ? { ...e, Status: 'missing' } : e
                      ));
                    } catch (e) {
                      let msg = e?.message || e;
                      if (e?.detail) msg += `\n${e.detail}`;
                      showErrorModal(msg || 'Delete failed');
                    }
                  }}
                />
              </div>
            </>
          )}
        </div>
      </div>
      <div style={{ width: '100%', padding: '12px 10px 0 10px', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
        <div style={{ fontWeight: 600, fontSize: titleFontSize, color: darkMode ? '#e5e7eb' : '#222', textAlign: 'center', marginBottom: 4, height: 50, display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden', width: '100%' }}>{displayTitle}</div>
        <div style={{ width: '100%', display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: 18, position: 'absolute', bottom: 12, left: 0 }}></div>
        {(youtubeModal.open && youtubeModal.videoId) && (
          <div style={{
            position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh', background: 'rgba(0,0,0,0.7)', zIndex: 99999,
            display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}>
            <div style={{
              position: 'relative',
              background: '#18181b',
              borderRadius: 16,
              boxShadow: '0 2px 24px #000',
              padding: 0,
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              overflow: 'visible',
            }}>
              <button
                onClick={() => setYoutubeModal({ open: false, videoId: '' })}
                style={{ position: 'absolute', top: 8, right: 12, zIndex: 2, fontSize: 28, color: '#fff', background: 'transparent', border: 'none', cursor: 'pointer' }}
                aria-label="Close"
              >×</button>
              <YoutubePlayer videoId={youtubeModal.videoId} />
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
