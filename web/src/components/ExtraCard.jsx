import React, { useState } from 'react';
import IconButton from './IconButton.jsx';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faTrashCan, faCheckSquare } from '@fortawesome/free-regular-svg-icons';
import { faPlay, faDownload } from '@fortawesome/free-solid-svg-icons';

export default function ExtraCard({
  extra,
  idx,
  type,
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
}) {
  const baseTitle = extra.Title || String(extra);
  const totalCount = typeExtras.filter(e => (e.Title || String(e)) === baseTitle).length;
  let displayTitle = totalCount > 1 ? `${baseTitle} (${typeExtras.slice(0, idx + 1).filter(e => (e.Title || String(e)) === baseTitle).length})` : baseTitle;
  const maxLen = 40;
  if (displayTitle.length > maxLen) {
    displayTitle = displayTitle.slice(0, maxLen - 3) + '...';
  }
  let youtubeID = '';
  if (extra.URL) {
    if (extra.URL.includes('youtube.com/watch?v=')) {
      youtubeID = extra.URL.split('v=')[1]?.split('&')[0] || '';
    } else if (extra.URL.includes('youtu.be/')) {
      youtubeID = extra.URL.split('youtu.be/')[1]?.split(/[?&]/)[0] || '';
    }
  }
  let posterUrl = `https://img.youtube.com/vi/${youtubeID}/hqdefault.jpg`;
  let titleFontSize = 16;
  if (displayTitle.length > 22) titleFontSize = 14;
  if (displayTitle.length > 32) titleFontSize = 12;
  const downloaded = extra.Status === 'downloaded';
  const rejected = extra.Status === 'rejected';
  const [errorCard, setErrorCard] = useState(null);
  const isError = errorCard === idx;
  const handleDownloadClick = async () => {
    if (downloaded) return;
    try {
      const getExtraUrl = extra => typeof extra.URL === 'string' ? extra.URL : extra.URL?.url ?? '';
      const res = await fetch(`/api/extras/download`, {
        method: 'POST',
        headers: { 'Content-Type': 'application/json' },
        body: JSON.stringify({
          mediaType: mediaType,
          mediaId: media.id,
          extraType: extra.Type,
          extraTitle: extra.Title,
          url: getExtraUrl(extra)
        })
      });
      if (res.ok) {
        setExtras(prev => prev.map((e, i) => i === idx && e.Type === type ? { ...e, Status: 'downloaded' } : e));
        setErrorCard(null);
      } else {
        const data = await res.json();
        let msg = data?.error || 'Download failed';
        if (msg.includes('UNPLAYABLE') || msg.includes('no se encuentra disponible')) {
          msg = 'This YouTube video is unavailable and cannot be downloaded.';
        }
        setModalMsg(msg);
        setShowModal(true);
        setErrorCard(idx);
      }
    } catch (e) {
      let msg = (e.message || e);
      if (msg.includes('UNPLAYABLE') || msg.includes('no se encuentra disponible')) {
        msg = 'This YouTube video is unavailable and cannot be downloaded.';
      }
      setModalMsg(msg);
      setShowModal(true);
      setErrorCard(idx);
    }
  };
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
      <div style={{ width: '100%', background: '#222', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
        <div style={{position: 'relative', width: '100%'}}>
          {extra.URL && (extra.URL.includes('youtube.com/watch?v=') || extra.URL.includes('youtu.be/')) && (
            <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', zIndex: 2 }}>
              <IconButton
                icon={<FontAwesomeIcon icon={faPlay} color="#fff" size="lg" style={{ filter: 'drop-shadow(0 2px 8px #000)' }} />}
                title="Play"
                onClick={() => {
                  let youtubeID = '';
                  if (extra.URL.includes('youtube.com/watch?v=')) {
                    youtubeID = extra.URL.split('v=')[1]?.split('&')[0] || '';
                  } else if (extra.URL.includes('youtu.be/')) {
                    youtubeID = extra.URL.split('youtu.be/')[1]?.split(/[?&]/)[0] || '';
                  }
                  if (youtubeID) setYoutubeModal({ open: true, videoId: youtubeID });
                }}
              />
            </div>
          )}
          {posterUrl ? (
            <img src={posterUrl} alt={displayTitle} style={{ width: '100%', height: 'auto', objectFit: 'contain', maxHeight: 260, background: '#222' }} />
          ) : (
            <div style={{ color: '#fff', fontSize: 18, textAlign: 'center', padding: 12 }}>No Image</div>
          )}
          {extra.URL && (extra.URL.includes('youtube.com/watch?v=') || extra.URL.includes('youtu.be/')) && !downloaded && (
            <div style={{ position: 'absolute', top: 8, right: downloaded ? 36 : 8, zIndex: 2 }}>
              <IconButton
                icon={<FontAwesomeIcon icon={faDownload} color={rejected ? '#aaa' : '#fff'} size="lg" />}
                title={rejected ? (extra.reason ? `Rejected: ${extra.reason}` : 'Rejected (cannot download)') : 'Download'}
                onClick={rejected ? () => { if (extra.reason) setModalMsg(extra.reason); } : handleDownloadClick}
                disabled={rejected}
                aria-label="Download"
                style={{ opacity: rejected ? 0.5 : 1 }}
              />
            </div>
          )}
          {downloaded && (
            <div style={{ position: 'absolute', top: 8, right: 8, zIndex: 2 }}>
              <IconButton icon={<FontAwesomeIcon icon={faCheckSquare} color="#22c55e" size="lg" />} title="Downloaded" disabled />
            </div>
          )}
          {downloaded && (
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
                    setExtras(prev => prev.map((e, i) => i === idx && e.Type === type ? { ...e, Status: 'missing' } : e));
                  } catch (e) {
                    let msg = e?.message || e;
                    if (e?.detail) msg += `\n${e.detail}`;
                    setModalMsg(msg || 'Delete failed');
                    setShowModal(true);
                  }
                }}
              />
            </div>
          )}
        </div>
      </div>
      <div style={{ width: '100%', padding: '12px 10px 0 10px', display: 'flex', flexDirection: 'column', alignItems: 'center' }}>
        <div style={{ fontWeight: 600, fontSize: titleFontSize, color: darkMode ? '#e5e7eb' : '#222', textAlign: 'center', marginBottom: 4, height: 50, display: 'flex', alignItems: 'center', justifyContent: 'center', overflow: 'hidden', width: '100%' }}>{displayTitle}</div>
        <div style={{ width: '100%', display: 'flex', justifyContent: 'flex-end', alignItems: 'center', gap: 18, position: 'absolute', bottom: 12, left: 0 }}></div>
        {youtubeModal.open && (
          <div className="youtube-modal-backdrop" style={{
            position: 'fixed', top: 0, left: 0, width: '100vw', height: '100vh', background: 'rgba(0,0,0,0.7)', zIndex: 99999, display: 'flex', alignItems: 'center', justifyContent: 'center',
          }}>
            <div style={{
              position: 'relative',
              background: '#18181b',
              borderRadius: 12,
              boxShadow: '0 2px 24px #000',
              padding: 0,
              width: '90vw',
              maxWidth: 800,
              aspectRatio: '16/9',
              display: 'flex',
              alignItems: 'center',
              justifyContent: 'center',
              overflow: 'hidden',
            }}>
              <IconButton
                icon={<span style={{ fontSize: 28, color: '#fff' }}>×</span>}
                onClick={() => {
                  // Unmount the iframe immediately by closing modal and clearing videoId
                  setYoutubeModal({ open: false, videoId: '' });
                }}
                title="Close"
                style={{ position: 'absolute', top: 8, right: 12, background: 'transparent', zIndex: 2 }}
              />
              {/* Only render YoutubeEmbed if modal is open and videoId is set */}
              {youtubeModal.open && youtubeModal.videoId && <YoutubeEmbed videoId={youtubeModal.videoId} />}
            </div>
          </div>
        )}
      </div>
    </div>
  );
}
