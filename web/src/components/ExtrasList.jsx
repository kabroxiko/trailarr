import React from 'react';
import { FontAwesomeIcon } from '@fortawesome/react-fontawesome';
import { faTrashCan, faBookmark, faCheckSquare } from '@fortawesome/free-regular-svg-icons';
import { faPlay, faDownload } from '@fortawesome/free-solid-svg-icons';

function ExtrasList({
  extrasByType,
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
  // Helper for rendering a group of extras
  const renderExtrasGroup = (type, typeExtras) => (
    <div key={type} style={{ marginBottom: 32 }}>
      <h3 style={{
        color: '#111',
        fontSize: 20,
        fontWeight: 700,
        margin: '0 0 18px 8px',
        textTransform: 'capitalize',
        letterSpacing: 0.5,
        textAlign: 'left',
      }}>{type}</h3>
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 0px))',
        gap: '32px',
        justifyItems: 'start',
        alignItems: 'start',
        width: '100%',
        justifyContent: 'start',
      }}>
        {typeExtras.map((extra, idx) => {
          const baseTitle = extra.title || String(extra);
          const totalCount = typeExtras.filter(e => (e.title || String(e)) === baseTitle).length;
          let displayTitle = totalCount > 1 ? `${baseTitle} (${typeExtras.slice(0, idx + 1).filter(e => (e.title || String(e)) === baseTitle).length})` : baseTitle;
          const maxLen = 40;
          if (displayTitle.length > maxLen) {
            displayTitle = displayTitle.slice(0, maxLen - 3) + '...';
          }
          let youtubeID = '';
          if (extra.url) {
            if (extra.url.includes('youtube.com/watch?v=')) {
              youtubeID = extra.url.split('v=')[1]?.split('&')[0] || '';
            } else if (extra.url.includes('youtu.be/')) {
              youtubeID = extra.url.split('youtu.be/')[1]?.split(/[?&]/)[0] || '';
            }
          }
          let posterUrl = extra.poster;
          if (!posterUrl && youtubeID) {
            posterUrl = `https://img.youtube.com/vi/${youtubeID}/hqdefault.jpg`;
          }
          let titleFontSize = 16;
          if (displayTitle.length > 22) titleFontSize = 14;
          if (displayTitle.length > 32) titleFontSize = 12;
          const downloaded = extra.downloaded === 'true';
          const handleDownloadClick = async () => {
            if (downloaded) return;
            try {
              const getExtraUrl = extra => typeof extra.url === 'string' ? extra.url : extra.url?.url ?? '';
              const res = await fetch(`/api/extras/download`, {
                method: 'POST',
                headers: { 'Content-Type': 'application/json' },
                body: JSON.stringify({
                  mediaType: mediaType,
                  mediaId: media.id,
                  extraType: extra.type,
                  extraTitle: extra.title,
                  url: getExtraUrl(extra)
                })
              });
              if (res.ok) {
                setExtras(prev => prev.map((e, i) => i === idx && e.type === type ? { ...e, downloaded: 'true' } : e));
              } else {
                const data = await res.json();
                let msg = data?.error || 'Download failed';
                if (msg.includes('UNPLAYABLE') || msg.includes('no se encuentra disponible')) {
                  msg = 'This YouTube video is unavailable and cannot be downloaded.';
                }
                setModalMsg(msg);
                setShowModal(true);
              }
            } catch (e) {
              let msg = (e.message || e);
              if (msg.includes('UNPLAYABLE') || msg.includes('no se encuentra disponible')) {
                msg = 'This YouTube video is unavailable and cannot be downloaded.';
              }
              setModalMsg(msg);
              setShowModal(true);
            }
          };
          return (
            <div key={idx} style={{
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
              border: downloaded ? '2px solid #22c55e' : '2px solid transparent',
            }}>
              <div style={{ width: '100%', background: '#222', display: 'flex', alignItems: 'center', justifyContent: 'center' }}>
                <div style={{position: 'relative', width: '100%'}}>
                  {extra.url && (extra.url.includes('youtube.com/watch?v=') || extra.url.includes('youtu.be/')) && (
                    <div style={{ position: 'absolute', top: '50%', left: '50%', transform: 'translate(-50%, -50%)', zIndex: 2 }}>
                      <FontAwesomeIcon
                        icon={faPlay}
                        color="#fff"
                        size="lg"
                        style={{ cursor: 'pointer', filter: 'drop-shadow(0 2px 8px #000)' }}
                        title="Play"
                        onClick={() => {
                          let youtubeID = '';
                          if (extra.url.includes('youtube.com/watch?v=')) {
                            youtubeID = extra.url.split('v=')[1]?.split('&')[0] || '';
                          } else if (extra.url.includes('youtu.be/')) {
                            youtubeID = extra.url.split('youtu.be/')[1]?.split(/[?&]/)[0] || '';
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
                  {extra.url && (extra.url.includes('youtube.com/watch?v=') || extra.url.includes('youtu.be/')) && !downloaded && (
                    <div style={{ position: 'absolute', top: 8, right: downloaded ? 36 : 8, zIndex: 2 }}>
                      <FontAwesomeIcon
                        icon={faDownload}
                        color="#fff"
                        size="lg"
                        style={{ cursor: 'pointer' }}
                        title="Download"
                        onClick={handleDownloadClick}
                      />
                    </div>
                  )}
                  {downloaded && (
                    <div style={{ position: 'absolute', top: 8, right: 8, zIndex: 2 }}>
                      <FontAwesomeIcon icon={faCheckSquare} color="#22c55e" size="lg" title="Downloaded" />
                    </div>
                  )}
                  {downloaded && (
                    <div style={{ position: 'absolute', bottom: 8, right: 8, zIndex: 2 }}>
                      <FontAwesomeIcon
                        icon={faTrashCan}
                        color="#ef4444"
                        size="md"
                        style={{ cursor: 'pointer' }}
                        title="Delete"
                        onClick={async () => {
                          if (!window.confirm('Delete this extra?')) return;
                          try {
                            const { deleteExtra } = await import('../api');
                            const payload = {
                              mediaType,
                              mediaId: media.id,
                              extraType: extra.type,
                              extraTitle: extra.title
                            };
                            await deleteExtra(payload);
                            setExtras(prev => prev.map((e, i) => i === idx && e.type === type ? { ...e, downloaded: 'false' } : e));
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
                <div style={{ fontSize: 13, color: '#888', marginBottom: 2 }}>{extra.year || ''}</div>
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
                      <button onClick={() => setYoutubeModal({ open: false, videoId: '' })} style={{ position: 'absolute', top: 8, right: 12, background: 'transparent', color: '#fff', border: 'none', fontSize: 28, cursor: 'pointer', zIndex: 2 }} title="Close">×</button>
                      <YoutubeEmbed videoId={youtubeModal.videoId} />
                    </div>
                  </div>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </div>
  );

  // Render 'Trailers' first, then others except 'Others', then 'Others' last
  return (
    <>
      {extrasByType['Trailers'] && renderExtrasGroup('Trailers', extrasByType['Trailers'])}
      {Object.entries(extrasByType)
        .filter(([type]) => type !== 'Trailers' && type !== 'Others')
        .map(([type, typeExtras]) => renderExtrasGroup(type, typeExtras))}
      {extrasByType['Others'] && renderExtrasGroup('Others', extrasByType['Others'])}
    </>
  );
}

export default ExtrasList;
