import React, { useState } from "react";
import IconButton from "./IconButton.jsx";
import PropTypes from "prop-types";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { faTrashCan, faCheckSquare } from "@fortawesome/free-regular-svg-icons";
import {
  faPlay,
  faDownload,
  faBan,
  faCircleXmark,
  faClock,
} from "@fortawesome/free-solid-svg-icons";

// Extracted action buttons to reduce cognitive complexity
function ExtraCardActions({
  extra,
  imgError,
  isFallback,
  downloaded,
  isDownloading,
  isQueued,
  rejected,
  onPlay,
  showToast,
  setExtras,
  baseType,
  baseTitle,
  mediaType,
  media,
  handleDownloadClick,
  handleDeleteClick,
}) {
  return (
    <>
      {/* Play button overlay (with image) */}
      {extra.YoutubeId && !imgError && !isFallback && (
        <div
          style={{
            position: "absolute",
            top: "50%",
            left: "50%",
            transform: "translate(-50%, -50%)",
            zIndex: 2,
          }}
        >
          <IconButton
            icon={
              <FontAwesomeIcon
                icon={faPlay}
                color="#fff"
                size="lg"
                style={{ filter: "drop-shadow(0 2px 8px #000)" }}
              />
            }
            title="Play"
            onClick={(e) => {
              e.stopPropagation();
              if (onPlay) onPlay(extra.YoutubeId);
            }}
          />
        </div>
      )}
      {/* Failed/Rejected Icon (always show for failed/rejected) */}
      {(extra.Status === "failed" ||
        extra.Status === "rejected" ||
        extra.Status === "unknown" ||
        extra.Status === "error") && (
        <div style={{ position: "absolute", top: 8, left: 8, zIndex: 2 }}>
          <IconButton
            icon={
              <FontAwesomeIcon icon={faCircleXmark} color="#ef4444" size="lg" />
            }
            title={
              extra.Status === "failed" ? "Remove failed status" : "Remove ban"
            }
            onClick={(event) =>
              handleRemoveBan({
                event,
                extra,
                baseType,
                baseTitle,
                mediaType,
                media,
                setExtras,
                showToast,
              })
            }
            aria-label={
              extra.Status === "failed" ? "Remove failed status" : "Remove ban"
            }
          />
        </div>
      )}
      {/* Download or Delete Buttons */}
      {extra.YoutubeId && !downloaded && !imgError && !isFallback && (
        <div style={{ position: "absolute", top: 8, right: 8, zIndex: 2 }}>
          <IconButton
            icon={
              <DownloadIcon isDownloading={isDownloading} isQueued={isQueued} />
            }
            title={getDownloadButtonTitle({
              rejected,
              extra,
              isDownloading,
              isQueued,
            })}
            onClick={
              rejected || isDownloading || isQueued
                ? undefined
                : (e) => {
                    e.stopPropagation();
                    handleDownloadClick();
                  }
            }
            disabled={rejected || isDownloading || isQueued}
            aria-label="Download"
            style={(() => {
              let opacity = 1;
              let borderRadius = 0;
              if (rejected) opacity = 0.5;
              else if (isDownloading || isQueued) {
                opacity = 0.7;
                borderRadius = 8;
              }
              return {
                opacity,
                background: "transparent",
                borderRadius,
                transition: "background 0.2s, opacity 0.2s",
              };
            })()}
          />
        </div>
      )}
      {/* Downloaded Checkmark and Delete Button */}
      {downloaded && (
        <>
          <div style={{ position: "absolute", top: 8, right: 8, zIndex: 2 }}>
            <IconButton
              icon={
                <FontAwesomeIcon
                  icon={faCheckSquare}
                  color="#22c55e"
                  size="lg"
                />
              }
              title="Downloaded"
              disabled
            />
          </div>
          <div style={{ position: "absolute", bottom: 8, right: 8, zIndex: 2 }}>
            <IconButton
              icon={
                <FontAwesomeIcon icon={faTrashCan} color="#ef4444" size="md" />
              }
              title="Delete"
              onClick={handleDeleteClick}
            />
          </div>
        </>
      )}
    </>
  );
}

// Helper for download button title (SonarLint: move out of render)
function getDownloadButtonTitle({ rejected, extra, isDownloading, isQueued }) {
  if (rejected) return extra.Reason;
  if (isDownloading) return "Downloading...";
  if (isQueued) return "Queued";
  return "Download";
}

// Helper for display title
function getDisplayTitle(typeExtras, baseTitle, idx) {
  const totalCount = typeExtras.filter(
    (e) => e.ExtraTitle === baseTitle,
  ).length;
  let title =
    totalCount > 1
      ? `${baseTitle} (${typeExtras.slice(0, idx + 1).filter((e) => e.ExtraTitle === baseTitle).length})`
      : baseTitle;
  const maxLen = 40;
  if (title.length > maxLen) {
    title = title.slice(0, maxLen - 3) + "...";
  }
  return title;
}

// Helper for border color
function getBorderColor({ rejected, failed, downloaded, exists }) {
  if (rejected || failed) return "2.5px solid #ef4444";
  if (downloaded) return "2px solid #22c55e";
  if (exists) return "2px solid #8888";
  return "2px solid transparent";
}

// Poster image or fallback factory (top-level, moved out for SonarLint)
function PosterImage({ src, alt, onError, onLoad, fallbackIcon, imgError }) {
  let content;
  if (src) {
    content = (
      <img
        src={src}
        alt={alt}
        onLoad={onLoad}
        onError={onError}
        style={{
          display: "block",
          margin: "0 0",
          maxHeight: 135,
          maxWidth: "100%",
          objectFit: "contain",
          background: "#222222",
        }}
      />
    );
  } else if (!src || imgError) {
    content = (
      <span
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          width: "100%",
          height: 135,
          background: "#222222",
        }}
      >
        <FontAwesomeIcon icon={fallbackIcon} color="#888" size="4x" />
      </span>
    );
  } else {
    content = (
      <span
        style={{
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          width: "100%",
          height: 135,
          background: "#333",
          animation: "pulse 1.2s infinite",
          color: "#444",
        }}
      >
        Loading...
      </span>
    );
  }
  return (
    <div
      style={{
        width: "100%",
        height: "100%",
        display: "flex",
        alignItems: "center",
        justifyContent: "center",
      }}
    >
      {content}
    </div>
  );
}

// Top-level error modal
function ErrorModal({ message, onClose }) {
  return (
    <div
      style={{
        position: "fixed",
        top: 24,
        left: "50%",
        transform: "translateX(-50%)",
        background: "#ef4444",
        color: "#fff",
        padding: "12px 32px",
        borderRadius: 8,
        boxShadow: "0 2px 12px rgba(0,0,0,0.18)",
        zIndex: 9999,
        fontWeight: 500,
        fontSize: 16,
        minWidth: 260,
        textAlign: "center",
      }}
    >
      {message}
      <button
        onClick={onClose}
        style={{
          marginLeft: 16,
          background: "transparent",
          color: "#fff",
          border: "none",
          fontSize: 18,
          cursor: "pointer",
        }}
      >
        ×
      </button>
    </div>
  );
}

// Top-level spinner icon for download
function SpinnerIcon() {
  return (
    <span
      className="download-spinner"
      style={{
        display: "inline-block",
        width: 22,
        height: 22,
        background: "transparent",
      }}
    >
      <svg
        viewBox="0 0 50 50"
        style={{ width: 22, height: 22, background: "transparent" }}
      >
        <circle
          cx="25"
          cy="25"
          r="20"
          fill="none"
          stroke="#fff"
          strokeWidth="5"
          strokeDasharray="31.4 31.4"
          strokeLinecap="round"
        >
          <animateTransform
            attributeName="transform"
            type="rotate"
            from="0 25 25"
            to="360 25 25"
            dur="0.8s"
            repeatCount="indefinite"
          />
        </circle>
      </svg>
    </span>
  );
}

function DownloadIcon({ isDownloading, isQueued }) {
  if (isDownloading) return <SpinnerIcon />;
  if (isQueued)
    return <FontAwesomeIcon icon={faClock} color="#fff" size="lg" />;
  return <FontAwesomeIcon icon={faDownload} color="#fff" size="lg" />;
}

DownloadIcon.propTypes = {
  isDownloading: PropTypes.bool,
  isQueued: PropTypes.bool,
};

function handleRemoveBan({
  event,
  extra,
  baseType,
  baseTitle,
  mediaType,
  media,
  setExtras,
  showToast,
}) {
  event.stopPropagation();
  // Always call backend to remove from blacklist for both rejected and failed statuses
  fetch("/api/blacklist/extras/remove", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({
      mediaType,
      mediaId: media.id,
      extraType: baseType,
      extraTitle: baseTitle,
      youtubeId: extra.YoutubeId,
    }),
  })
    .then(() => {
      if (typeof setExtras === "function") {
        setExtras((prev) =>
          prev.map((ex) =>
            ex.YoutubeId === extra.YoutubeId &&
            ex.ExtraType === baseType &&
            ex.ExtraTitle === baseTitle
              ? { ...ex, Status: "" }
              : ex,
          ),
        );
      }
      if (typeof setExtras === "function" && media !== undefined && media.id) {
        fetch(
          `/api/${mediaType === "movie" ? "movies" : "series"}/${media.id}/extras`,
        )
          .then((res) => (res.ok ? res.json() : null))
          .then((data) => {
            if (Array.isArray(data)) {
              setExtras(data);
            } else if (data && Array.isArray(data.extras)) {
              setExtras(data.extras);
            }
          })
          .catch(() => {});
      }
    })
    .catch(() => {
      if (typeof showToast === "function") {
        showToast("Failed to remove ban.");
      }
    });
}

export default function ExtraCard({
  extra,
  idx,
  typeExtras,
  darkMode,
  media,
  mediaType,
  setExtras,
  rejected: rejectedProp,
  onPlay,
  showToast, // new prop for toast/modal
}) {
  const [imgError, setImgError] = useState(false);
  const [isFallback, setIsFallback] = useState(false);
  const baseTitle = extra.ExtraTitle || "";
  const baseType = extra.ExtraType || "";
  const displayTitle = getDisplayTitle(typeExtras, baseTitle, idx);
  let posterUrl = extra.YoutubeId
    ? `/api/proxy/youtube-image/${extra.YoutubeId}`
    : null;
  React.useEffect(() => {
    // Reset states when posterUrl changes
    setImgError(false);
    setIsFallback(false);
  }, [posterUrl]);
  let titleFontSize = 16;
  if (displayTitle.length > 22) titleFontSize = 14;
  if (displayTitle.length > 32) titleFontSize = 12;
  const downloaded = extra.Status === "downloaded";
  const isDownloading = extra.Status === "downloading";
  const isQueued = extra.Status === "queued";
  const failed =
    extra.Status === "failed" ||
    extra.Status === "rejected" ||
    extra.Status === "unknown" ||
    extra.Status === "error";
  const exists = extra.Status === "exists";
  const [downloading, setDownloading] = useState(false);
  // Use the rejected prop if provided, otherwise fallback to extra.Status
  const [unbanned] = useState(false);
  // Treat 'failed' as 'rejected' for UI
  const rejected =
    !unbanned &&
    (typeof rejectedProp === "boolean"
      ? rejectedProp
      : extra.Status === "rejected" || extra.Status === "failed");
  // Removed errorCard/modal state; error display is now handled at the page level

  // showErrorModal removed; use showToast for error display

  function revertStatus() {
    if (typeof setExtras === "function") {
      setExtras((prev) =>
        prev.map((ex) =>
          ex.YoutubeId === extra.YoutubeId &&
          ex.ExtraType === baseType &&
          ex.ExtraTitle === baseTitle
            ? { ...ex, Status: "" }
            : ex,
        ),
      );
    }
  }

  function handleError(msg) {
    if (typeof showToast === "function") {
      showToast(msg);
    }
    revertStatus();
  }

  const handleDownloadClick = async () => {
    if (downloaded || downloading) return;
    setDownloading(true);
    try {
      const res = await fetch(`/api/extras/download`, {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify({
          mediaType,
          mediaId: media.id,
          extraType: baseType,
          extraTitle: baseTitle,
          youtubeId: extra.YoutubeId,
        }),
      });
      if (res.ok) {
        // Only add a new backend-style extra if not already present (by YoutubeId, ExtraType, and ExtraTitle)
        if (
          typeof setExtras === "function" &&
          !downloaded &&
          !isQueued &&
          !isDownloading &&
          !exists
        ) {
          setExtras((prev) => {
            if (
              prev.some(
                (ex) =>
                  ex.YoutubeId === extra.YoutubeId &&
                  ex.ExtraType === baseType &&
                  ex.ExtraTitle === baseTitle,
              )
            )
              return prev;
            return [
              ...prev,
              {
                ...extra,
                Status: "queued",
                Reason: "",
                ExtraType: baseType,
                ExtraTitle: baseTitle,
                YoutubeId: extra.YoutubeId,
                // Add any other backend fields as needed
              },
            ];
          });
        }
      } else {
        const data = await res.json();
        handleError(data?.error || "Download failed");
      }
    } catch (error) {
      handleError(error.message || error);
    } finally {
      setDownloading(false);
    }
  };

  const borderColor = getBorderColor({ rejected, failed, downloaded, exists });
  return (
    <div
      title={rejected ? extra.Reason : undefined}
      style={{
        width: 180,
        height: 210,
        background: darkMode ? "#18181b" : "#fff",
        borderRadius: 12,
        boxShadow: darkMode
          ? "0 2px 12px rgba(0,0,0,0.22)"
          : "0 2px 12px rgba(0,0,0,0.10)",
        overflow: "hidden",
        display: "flex",
        flexDirection: "column",
        alignItems: "center",
        padding: "0 0 0 0",
        position: "relative",
        border: borderColor,
      }}
    >
      <div
        style={{
          width: "100%",
          height: 135,
          background: "#222",
          display: "flex",
          alignItems: "center",
          justifyContent: "center",
          position: "relative",
        }}
      >
        {/* Poster Image or Fallback */}
        {!imgError && posterUrl ? (
          <PosterImage
            key={posterUrl}
            src={posterUrl}
            alt={displayTitle}
            fallbackIcon={faBan}
            onLoad={(event) => {
              // If the image loads but is very small (our SVG fallback is 64x64), treat as fallback
              try {
                const img = event.target;
                if (img.naturalWidth <= 64 && img.naturalHeight <= 64) {
                  setIsFallback(true);
                  setImgError(true);
                }
              } catch {
                // ignore
              }
            }}
            onError={() => {
              setIsFallback(true);
              setImgError(true);
            }}
            imgError={imgError}
          />
        ) : (
          <PosterImage
            src={null}
            alt="Denied"
            fallbackIcon={faBan}
            imgError={imgError}
          />
        )}
        <ExtraCardActions
          extra={extra}
          imgError={imgError}
          isFallback={isFallback}
          downloaded={downloaded}
          isDownloading={isDownloading}
          isQueued={isQueued}
          rejected={rejected}
          onPlay={onPlay}
          showToast={showToast}
          setExtras={setExtras}
          baseType={baseType}
          baseTitle={baseTitle}
          mediaType={mediaType}
          media={media}
          handleDownloadClick={handleDownloadClick}
          handleDeleteClick={async (event) => {
            event.stopPropagation();
            if (!globalThis.confirm("Delete this extra?")) return;
            try {
              const { deleteExtra } = await import("../api");
              const payload = {
                mediaType,
                mediaId: media.id,
                youtubeId: extra.YoutubeId,
              };
              await deleteExtra(payload);
              setExtras((prev) =>
                prev.map((ex) =>
                  ex.ExtraTitle === baseTitle && ex.ExtraType === baseType
                    ? { ...ex, Status: "missing" }
                    : ex,
                ),
              );
            } catch (error) {
              let msg = error?.message || error;
              if (error?.detail) msg += `\n${error.detail}`;
              // showErrorModal is removed, use showToast
              if (typeof showToast === "function")
                showToast(msg || "Delete failed");
            }
          }}
          displayTitle={displayTitle}
        />
      </div>
      <div
        style={{
          width: "100%",
          padding: "12px 10px 0 10px",
          display: "flex",
          flexDirection: "column",
          alignItems: "center",
        }}
      >
        <div
          style={{
            fontWeight: 600,
            fontSize: titleFontSize,
            color: darkMode ? "#e5e7eb" : "#222",
            textAlign: "center",
            marginBottom: 4,
            height: 50,
            display: "flex",
            alignItems: "center",
            justifyContent: "center",
            overflow: "hidden",
            width: "100%",
          }}
        >
          {displayTitle}
        </div>
        <div
          style={{
            width: "100%",
            display: "flex",
            justifyContent: "flex-end",
            alignItems: "center",
            gap: 18,
            position: "absolute",
            bottom: 12,
            left: 0,
          }}
        ></div>
        {/* YouTube modal is now rendered at the page level */}
      </div>
    </div>
  );
}

ExtraCard.propTypes = {
  extra: PropTypes.shape({
    Reason: PropTypes.string,
    ExtraTitle: PropTypes.string,
    ExtraType: PropTypes.string,
    YoutubeId: PropTypes.string,
    Status: PropTypes.string,
  }).isRequired,
  idx: PropTypes.number,
  typeExtras: PropTypes.array,
  darkMode: PropTypes.bool,
  media: PropTypes.object,
  mediaType: PropTypes.string,
  setExtras: PropTypes.func,
  // setModalMsg and setShowModal removed (unused props)
  rejected: PropTypes.bool,
  onPlay: PropTypes.func,
  showToast: PropTypes.func,
};

PosterImage.propTypes = {
  src: PropTypes.string,
  alt: PropTypes.string.isRequired,
  onError: PropTypes.func,
  onLoad: PropTypes.func,
  fallbackIcon: PropTypes.object.isRequired,
  imgError: PropTypes.bool,
};

ErrorModal.propTypes = {
  message: PropTypes.string.isRequired,
  onClose: PropTypes.func.isRequired,
};

// PropTypes for ExtraCardActions
ExtraCardActions.propTypes = {
  extra: PropTypes.shape({
    YoutubeId: PropTypes.string,
    Status: PropTypes.string,
  }).isRequired,
  imgError: PropTypes.bool,
  isFallback: PropTypes.bool,
  downloaded: PropTypes.bool,
  isDownloading: PropTypes.bool,
  isQueued: PropTypes.bool,
  rejected: PropTypes.bool,
  onPlay: PropTypes.func,
  showToast: PropTypes.func,
  setExtras: PropTypes.func,
  baseType: PropTypes.string,
  baseTitle: PropTypes.string,
  mediaType: PropTypes.string,
  media: PropTypes.object,
  handleDownloadClick: PropTypes.func,
  handleDeleteClick: PropTypes.func,
  displayTitle: PropTypes.string,
};

// Export PosterImage at the end for SonarLint compliance
export { PosterImage };
