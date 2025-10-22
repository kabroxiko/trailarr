import React from "react";
import PropTypes from "prop-types";
import IconButton from "./IconButton.jsx";
import { FontAwesomeIcon } from "@fortawesome/react-fontawesome";
import { Link } from "react-router-dom";
import {
  faCog,
  faFilm,
  faHistory,
  faStar,
  faBan,
  faServer,
} from "@fortawesome/free-solid-svg-icons";

export default function SidebarMobile({
  darkMode,
  open,
  onClose,
  selectedSection,
  selectedSettingsSub,
  selectedSystemSub,
  isOpen,
  handleToggle,
}) {
  function renderWantedSubmenu() {
    return (
      <ul
        style={{
          listStyle: "none",
          padding: 0,
          margin: "8px 0 0 0",
          background: darkMode ? "#23232a" : "#f3f4f6",
          borderRadius: 6,
          color: darkMode ? "#e5e7eb" : "#222",
          textAlign: "left",
        }}
      >
        {["Movies", "Series"].map((submenu) => {
          const selected = selectedSettingsSub === submenu;
          const borderLeft = selected
            ? "3px solid #a855f7"
            : "3px solid transparent";
          let color;
          if (selected) {
            color = darkMode ? "#a855f7" : "#6d28d9";
          } else {
            color = darkMode ? "#e5e7eb" : "#333";
          }
          const fontWeight = selected ? "bold" : "normal";
          const styleLink = {
            color,
            textDecoration: "none",
            display: "block",
            width: "100%",
            textAlign: "left",
            background: "none",
            border: "none",
            fontWeight,
            cursor: "pointer",
          };
          return (
            <li
              key={submenu}
              style={{
                padding: "0.5em 1em",
                borderLeft,
                background: "none",
                color,
                fontWeight,
                cursor: "pointer",
                textAlign: "left",
              }}
            >
              <Link
                to={`/wanted/${submenu.toLowerCase()}`}
                style={styleLink}
                onClick={onClose}
              >
                {submenu}
              </Link>
            </li>
          );
        })}
      </ul>
    );
  }
  function renderSettingsSubmenu() {
    return (
      <ul
        style={{
          listStyle: "none",
          padding: 0,
          margin: "8px 0 0 0",
          background: darkMode ? "#23232a" : "#f3f4f6",
          borderRadius: 6,
          color: darkMode ? "#e5e7eb" : "#222",
          textAlign: "left",
        }}
      >
        {["General", "Radarr", "Sonarr", "Extras"].map((submenu) => {
          const selected = selectedSettingsSub === submenu;
          const borderLeft = selected
            ? "3px solid #a855f7"
            : "3px solid transparent";
          let color;
          if (selected) {
            color = darkMode ? "#a855f7" : "#6d28d9";
          } else {
            color = darkMode ? "#e5e7eb" : "#333";
          }
          const fontWeight = selected ? "bold" : "normal";
          const styleLink = {
            color,
            textDecoration: "none",
            display: "block",
            width: "100%",
            textAlign: "left",
            background: "none",
            border: "none",
            fontWeight,
            cursor: "pointer",
          };
          return (
            <li
              key={submenu}
              style={{
                padding: "0.5em 1em",
                borderLeft,
                background: "none",
                color,
                fontWeight,
                cursor: "pointer",
                textAlign: "left",
              }}
            >
              <Link
                to={`/settings/${submenu.toLowerCase()}`}
                style={styleLink}
                onClick={onClose}
              >
                {submenu}
              </Link>
            </li>
          );
        })}
      </ul>
    );
  }
  function renderSystemSubmenu() {
    return (
      <ul
        style={{
          listStyle: "none",
          padding: 0,
          margin: "8px 0 0 0",
          background: darkMode ? "#23232a" : "#f3f4f6",
          borderRadius: 6,
          color: darkMode ? "#e5e7eb" : "#222",
          textAlign: "left",
        }}
      >
        {["Tasks", "Logs"].map((submenu) => {
          const selected = selectedSystemSub === submenu;
          const borderLeft = selected
            ? "3px solid #a855f7"
            : "3px solid transparent";
          let color;
          if (selected) {
            color = darkMode ? "#a855f7" : "#6d28d9";
          } else {
            color = darkMode ? "#e5e7eb" : "#333";
          }
          const fontWeight = selected ? "bold" : "normal";
          const styleLink = {
            color,
            textDecoration: "none",
            display: "block",
            width: "100%",
            textAlign: "left",
            background: "none",
            border: "none",
            fontWeight,
            cursor: "pointer",
          };
          return (
            <li
              key={submenu}
              style={{
                padding: "0.5em 1em",
                borderLeft,
                background: "none",
                color,
                fontWeight,
                cursor: "pointer",
                textAlign: "left",
              }}
            >
              <Link
                to={submenu === "Tasks" ? "/system/tasks" : "/system/logs"}
                style={styleLink}
                onClick={onClose}
              >
                {submenu}
              </Link>
            </li>
          );
        })}
      </ul>
    );
  }
  const menuItems = [
    { name: "Movies", icon: faFilm, route: "/" },
    { name: "Series", icon: faCog, route: "/series" },
    { name: "History", icon: faHistory, route: "/history" },
    { name: "Wanted", icon: faStar },
    { name: "Blacklist", icon: faBan, route: "/blacklist" },
    { name: "Settings", icon: faCog },
    { name: "System", icon: faServer },
  ];
  return (
    <>
      {open && (
        <button
          type="button"
          className="sidebar-mobile__backdrop"
          onClick={onClose}
          aria-label="Close sidebar"
          style={{
            background: "transparent",
            border: "none",
            padding: 0,
            margin: 0,
            position: "fixed",
            top: 0,
            left: 0,
            width: "100vw",
            height: "100vh",
            zIndex: 1000,
          }}
        />
      )}
      <div
        className={`sidebar-mobile${open ? " open" : ""}`}
        style={{
          "--sidebar-bg": darkMode ? "#23232a" : "#fff",
          position: "fixed",
          zIndex: 1001,
        }}
      >
        <nav>
          <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
            {menuItems.map(({ name, icon, route }) => {
              let background, color, fontWeight;
              if (selectedSection === name) {
                background = darkMode ? "#333" : "#f3f4f6";
                color = darkMode ? "#a855f7" : "#6d28d9";
                fontWeight = "bold";
              } else {
                background = "none";
                color = darkMode ? "#e5e7eb" : "#333";
                fontWeight = "normal";
              }
              const styleCommon = {
                textDecoration: "none",
                background,
                border: "none",
                color,
                fontWeight,
                width: "100%",
                textAlign: "left",
                padding: "0.5em 1em",
                borderRadius: 6,
                cursor: "pointer",
                display: "flex",
                alignItems: "center",
                gap: "0.75em",
              };
              if (route) {
                return (
                  <li key={name} style={{ marginBottom: 16 }}>
                    <Link to={route} style={styleCommon} onClick={onClose}>
                      <IconButton
                        icon={
                          <FontAwesomeIcon
                            icon={icon}
                            color={darkMode ? "#e5e7eb" : "#333"}
                          />
                        }
                        style={{
                          background: "none",
                          padding: 0,
                          margin: 0,
                          border: "none",
                        }}
                      />
                      {name}
                    </Link>
                  </li>
                );
              }
              // Render menu toggle and submenus
              return (
                <li key={name} style={{ marginBottom: 16 }}>
                  <button
                    type="button"
                    style={styleCommon}
                    onClick={() => handleToggle(name)}
                  >
                    <IconButton
                      icon={
                        <FontAwesomeIcon
                          icon={icon}
                          color={darkMode ? "#e5e7eb" : "#333"}
                        />
                      }
                      style={{
                        background: "none",
                        padding: 0,
                        margin: 0,
                        border: "none",
                      }}
                    />
                    {name}
                  </button>
                  {name === "Wanted" &&
                    isOpen("Wanted") &&
                    renderWantedSubmenu()}
                  {name === "Settings" &&
                    isOpen("Settings") &&
                    renderSettingsSubmenu()}
                  {name === "System" &&
                    isOpen("System") &&
                    renderSystemSubmenu()}
                </li>
              );
            })}
          </ul>
        </nav>
      </div>
    </>
  );
}

SidebarMobile.propTypes = {
  darkMode: PropTypes.bool.isRequired,
  open: PropTypes.bool.isRequired,
  onClose: PropTypes.func.isRequired,
  selectedSection: PropTypes.string.isRequired,
  selectedSettingsSub: PropTypes.string,
  selectedSystemSub: PropTypes.string,
  isOpen: PropTypes.func.isRequired,
  handleToggle: PropTypes.func.isRequired,
};
