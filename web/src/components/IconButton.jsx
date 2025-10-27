import React from "react";
import PropTypes from "prop-types";

export default function IconButton({
  icon,
  onClick,
  title,
  disabled = false,
  style = {},
  ...props
}) {
  return (
    <button
      onClick={onClick}
      title={title}
      disabled={disabled}
      style={{
        background: "none",
        border: "none",
        outline: "none",
        boxShadow: "none",
        padding: 0,
        margin: 0,
        cursor: disabled ? "not-allowed" : "pointer",
        opacity: disabled ? 0.6 : 1,
        display: "inline-flex",
        alignItems: "center",
        justifyContent: "center",
        ...style,
      }}
      onFocus={(e) => {
        e.target.style.outline = "none";
        e.target.style.boxShadow = "none";
      }}
      onMouseDown={(e) => {
        e.target.style.outline = "none";
        e.target.style.boxShadow = "none";
      }}
      {...props}
    >
      {icon}
    </button>
  );
}

IconButton.propTypes = {
  icon: PropTypes.node.isRequired,
  onClick: PropTypes.func.isRequired,
  title: PropTypes.string,
  disabled: PropTypes.bool,
  style: PropTypes.object,
};
