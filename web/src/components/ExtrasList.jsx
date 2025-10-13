import React, { useState, useMemo, useEffect, useRef } from 'react';
import ExtraCard from './ExtraCard.jsx';
import SectionHeader from './SectionHeader.jsx';
import Toast from './Toast';

function ExtrasList({
  extrasByType,
  darkMode,
  media,
  mediaType,
  setExtras,
  setModalMsg,
  setShowModal,
  setYoutubeModal,
  YoutubeEmbed,
}) {
  const [toastMsg, setToastMsg] = useState('');
  const wsRef = useRef(null);

  // WebSocket: Listen for download queue updates
  useEffect(() => {
    const wsUrl = (window.location.protocol === 'https:' ? 'wss://' : 'ws://') + window.location.host + '/ws/download-queue';
    const ws = new window.WebSocket(wsUrl);
    wsRef.current = ws;
    ws.onopen = () => {
      console.debug('[WebSocket] Connected to download queue');
    };
    ws.onmessage = (event) => {
      try {
        const msg = JSON.parse(event.data);
        if (msg.type === 'download_queue_update' && Array.isArray(msg.queue)) {
          console.debug('[WebSocket] Received queue update', msg.queue);
          // Update extras state if possible
          if (typeof setExtras === 'function') {
            setExtras(prev => {
              // Map over all extras and update their Status if found in queue
              const queueMap = Object.fromEntries(msg.queue.map(e => [e.youtubeId, e.status]));
              return prev.map(ex => {
                if (queueMap[ex.YoutubeId]) {
                  return { ...ex, Status: queueMap[ex.YoutubeId] };
                }
                return ex;
              });
            });
          }
        }
      } catch (err) {
        console.debug('[WebSocket] Error parsing message', err);
      }
    };
    ws.onerror = (e) => {
      console.debug('[WebSocket] Error', e);
    };
    ws.onclose = () => {
      console.debug('[WebSocket] Closed');
    };
    return () => {
      ws.close();
    };
  }, [setExtras]);

  // Flatten all extras for polling
  const allExtras = useMemo(() => Object.values(extrasByType).flat(), [extrasByType]);

  // Helper for rendering a group of extras
  const renderExtrasGroup = (type, typeExtras) => (
    <div key={type} style={{ marginBottom: 32 }}>
      <SectionHeader>{type}</SectionHeader>
      <div style={{
        display: 'grid',
        gridTemplateColumns: 'repeat(auto-fit, minmax(180px, 0px))',
        gap: '32px',
        justifyItems: 'start',
        alignItems: 'start',
        width: '100%',
        justifyContent: 'start',
      }}>
        {typeExtras.map((extra, idx) => (
          <ExtraCard
            key={extra.YoutubeId || idx}
            extra={extra}
            idx={idx}
            type={type}
            typeExtras={typeExtras}
            darkMode={darkMode}
            media={media}
            mediaType={mediaType}
            setExtras={setExtras}
            setModalMsg={setModalMsg}
            setShowModal={setShowModal}
            YoutubeEmbed={YoutubeEmbed}
            onPlay={videoId => setYoutubeModal({ open: true, videoId })}
            showToast={setToastMsg}
          />
        ))}
      </div>
    </div>
  );

  // Render 'Trailers' first, then others except 'Other', then 'Other' last
  return (
    <>
      <Toast message={toastMsg} onClose={() => setToastMsg('')} darkMode={darkMode} />
      {extrasByType['Trailers'] && renderExtrasGroup('Trailers', extrasByType['Trailers'])}
      {Object.entries(extrasByType)
        .filter(([type]) => type !== 'Trailers' && type !== 'Other')
        .map(([type, typeExtras]) => renderExtrasGroup(type, typeExtras))}
      {extrasByType['Other'] && renderExtrasGroup('Other', extrasByType['Other'])}
    </>
  );
}

export default ExtrasList;
