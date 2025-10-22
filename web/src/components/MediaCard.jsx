import React from "react";
import PropTypes from "prop-types";

// Compact MediaCard for use in MediaList (tiles)
function MediaCard({ media, mediaType, darkMode = false }) {
  if (!media) return null;
  const poster =
    mediaType === "series"
      ? `/mediacover/Series/${media.id}/poster-500.jpg`
      : `/mediacover/Movies/${media.id}/poster-500.jpg`;
  return (
    <div
      style={{
        width: "100%",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
      }}
    >
      {/* Mobile: just poster, rounded borders, no frame/box/title/year */}
      <style>{`
        @media (max-width: 900px) {
          .media-card-poster {
            border-radius: 12px !important;
            box-shadow: none !important;
            border: none !important;
            width: 100vw !important;
            width: calc(100vw / 3 - 1rem) !important;
            height: calc((100vw / 3 - 1rem) * 1.5) !important;
            aspect-ratio: 2/3 !important;
            margin: 0 !important;
            background: none !important;
            display: block !important;
          }
        }
      `}</style>
      <img
        className="media-card-poster"
        src={poster}
        loading="lazy"
        style={{
          width: "100%",
          height: "auto",
          objectFit: "cover",
          borderRadius: 8,
          display: "block",
          aspectRatio: "2/3",
          maxWidth: "220px",
        }}
        onError={(e) => {
          e.target.onerror = null;
          e.target.src = "/logo.svg";
        }}
        alt={media.title}
      />
      {/* Desktop: show title/year/frame/box as before */}
      <div
        className="media-card-details"
        style={{
          marginTop: 8,
          textAlign: "center",
          width: "100%",
          maxWidth: "220px",
          display: "none",
        }}
        title={media.title}
      >
        <div
          style={{
            color: darkMode ? "#fff" : "#222",
            fontWeight: 600,
            fontSize: 14,
          }}
        >
          {media.title}
        </div>
        <div style={{ color: darkMode ? "#ddd" : "#666", fontSize: 12 }}>
          {media.year || media.airDate || ""}
        </div>
      </div>
      <style>{`
        @media (min-width: 901px) {
          .media-card-details {
            display: block !important;
          }
        }
      `}</style>
    </div>
  );
}

MediaCard.propTypes = {
  media: PropTypes.shape({
    id: PropTypes.oneOfType([PropTypes.string, PropTypes.number]).isRequired,
    title: PropTypes.string,
    year: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
    airDate: PropTypes.oneOfType([PropTypes.string, PropTypes.number]),
  }).isRequired,
  mediaType: PropTypes.string.isRequired,
  darkMode: PropTypes.bool,
};

export default MediaCard;
