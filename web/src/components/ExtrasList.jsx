import React, { useState } from 'react';
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
            key={idx}
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
