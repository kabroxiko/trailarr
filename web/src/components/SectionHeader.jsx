import React from "react";

export default function SectionHeader({ children, style = {}, ...props }) {
  const isDark =
    window.matchMedia &&
    window.matchMedia("(prefers-color-scheme: dark)").matches;
  const headerColor = isDark ? "#eee" : "#222";
  const defaultStyle = {
    fontWeight: 600,
    fontSize: "1.35em",
    margin: "0 0 18px 8px",
    textAlign: "left",
    textTransform: "capitalize",
    color: headerColor,
    ...style,
  };
  return (
    <h3 style={defaultStyle} {...props}>
      {children}
    </h3>
  );
}
