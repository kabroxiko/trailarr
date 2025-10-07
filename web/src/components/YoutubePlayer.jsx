import React, { useEffect, useRef } from 'react';

// Loads the YouTube IFrame API if not already loaded
function loadYouTubeAPI() {
  if (window.YT && window.YT.Player) return Promise.resolve();
  return new Promise(resolve => {
    const tag = document.createElement('script');
    tag.src = 'https://www.youtube.com/iframe_api';
    window.onYouTubeIframeAPIReady = () => resolve();
    document.body.appendChild(tag);
  });
}

export default function YoutubePlayer({ videoId, onReady }) {
  const playerRef = useRef();
  const ytPlayer = useRef();

  useEffect(() => {
    let destroyed = false;
    loadYouTubeAPI().then(() => {
      if (destroyed) return;
      ytPlayer.current = new window.YT.Player(playerRef.current, {
        videoId,
        events: {
          onReady: (event) => {
            event.target.playVideo();
            if (onReady) onReady(event);
          },
        },
        playerVars: {
          autoplay: 1,
          rel: 0,
          modestbranding: 1,
        },
      });
    });
    return () => {
      destroyed = true;
      if (ytPlayer.current) {
        ytPlayer.current.destroy();
        ytPlayer.current = null;
      }
    };
  }, [videoId, onReady]);

  return (
    <div
      ref={playerRef}
      style={{
        width: '80vw',
        height: '45vw',
        maxWidth: 900,
        maxHeight: 506,
        background: '#000',
        borderRadius: 12,
        boxShadow: '0 2px 24px #000',
        border: 'none',
        display: 'block',
      }}
    />
  );
}
