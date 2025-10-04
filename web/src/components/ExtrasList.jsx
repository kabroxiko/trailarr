import ExtraCard from './ExtraCard.jsx';
import SectionHeader from './SectionHeader.jsx';
import React, { useState } from 'react';
import Toast from './Toast';

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
            youtubeModal={youtubeModal}
            setYoutubeModal={setYoutubeModal}
            YoutubeEmbed={YoutubeEmbed}
          />
        ))}
      </div>
    </div>
  );

  // Render 'Trailers' first, then others except 'Others', then 'Others' last
  return (
    <>
      <Toast message={toastMsg} onClose={() => setToastMsg('')} darkMode={darkMode} />
      {extrasByType['Trailers'] && renderExtrasGroup('Trailers', extrasByType['Trailers'])}
      {Object.entries(extrasByType)
        .filter(([type]) => type !== 'Trailers' && type !== 'Others')
        .map(([type, typeExtras]) => renderExtrasGroup(type, typeExtras))}
      {extrasByType['Others'] && renderExtrasGroup('Others', extrasByType['Others'])}
    </>
  );
}

export default ExtrasList;
