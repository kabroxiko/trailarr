import React from "react";

export default function LoadingMediaSkeleton() {
  const dark = globalThis.window?.matchMedia?.("(prefers-color-scheme: dark)")?.matches;
  const isMobile = globalThis.window?.matchMedia?.("(max-width: 900px)")?.matches;

  const containerStyle = {
    padding: isMobile ? 16 : 88,
    minHeight: "60vh",
    boxSizing: "border-box",
  };

  const posterStyle = {
    width: 360,
    height: 360,
    background: dark ? "#111" : "#eaeaea",
    borderRadius: 12,
    flexShrink: 0,
  };

  const line = (w, h = 14, mb = 12) => (
    <div style={{ width: w, height: h, borderRadius: 6, background: dark ? "#202124" : "#e8e8e8", marginBottom: mb }} />
  );

  return (
    <div style={containerStyle}>
      {!isMobile ? (
        <div style={{ display: "flex", gap: 24, alignItems: "flex-start" }}>
          <div style={posterStyle} />
          <div style={{ flex: 1 }}>
            {line("60%", 28, 12)}
            {line("40%", 18, 18)}
            <div style={{ display: "flex", gap: 12, marginBottom: 12 }}>
              <div style={{ width: 120, height: 36, borderRadius: 8, background: dark ? "#202124" : "#e8e8e8" }} />
              <div style={{ width: 120, height: 36, borderRadius: 8, background: dark ? "#202124" : "#e8e8e8" }} />
            </div>
          </div>
        </div>
      ) : (
        // Mobile: no poster layout, simplified stacked skeleton
        <div style={{ display: "flex", flexDirection: "column", gap: 12 }}>
          {line("80%", 28, 8)}
          {line("50%", 18, 12)}
          <div style={{ display: "flex", gap: 8 }}>
            <div style={{ width: 96, height: 34, borderRadius: 8, background: dark ? "#202124" : "#e8e8e8" }} />
            <div style={{ width: 96, height: 34, borderRadius: 8, background: dark ? "#202124" : "#e8e8e8" }} />
          </div>
        </div>
      )}
    </div>
  );
}
