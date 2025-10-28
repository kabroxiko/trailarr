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
import "./SidebarDesktop.css";
import { isDark } from "../utils/isDark";

export default function SidebarDesktop({
  selectedSection,
  selectedSettingsSub,
  selectedSystemSub,
  isOpen,
  handleToggle,
  healthCount = 0,
}) {
  function renderWantedSubmenu() {
    return (
      <ul
        style={{
          listStyle: "none",
          padding: 0,
          margin: "8px 0 0 0",
          background: isDark ? "#23232a" : "#f3f4f6",
          borderRadius: 6,
          color: isDark ? "#e5e7eb" : "#222",
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
            color = isDark ? "#a855f7" : "#6d28d9";
          } else {
            color = isDark ? "#e5e7eb" : "#333";
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
              <Link to={`/wanted/${submenu.toLowerCase()}`} style={styleLink}>
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
          background: isDark ? "#23232a" : "#f3f4f6",
          borderRadius: 6,
          color: isDark ? "#e5e7eb" : "#222",
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
            color = isDark ? "#a855f7" : "#6d28d9";
          } else {
            color = isDark ? "#e5e7eb" : "#333";
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
              <Link to={`/settings/${submenu.toLowerCase()}`} style={styleLink}>
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
          background: isDark ? "#23232a" : "#f3f4f6",
          borderRadius: 6,
          color: isDark ? "#e5e7eb" : "#222",
          textAlign: "left",
        }}
      >
        {["Status", "Tasks", "Logs"].map((submenu) => {
          const selected = selectedSystemSub === submenu;
          const borderLeft = selected
            ? "3px solid #a855f7"
            : "3px solid transparent";
          let color;
          if (selected) {
            color = isDark ? "#a855f7" : "#6d28d9";
          } else {
            color = isDark ? "#e5e7eb" : "#333";
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
          const toRoute = (() => {
            if (submenu === "Status") return "/system/status";
            if (submenu === "Tasks") return "/system/tasks";
            return "/system/logs";
          })();
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
                display: "flex",
                alignItems: "center",
                justifyContent: "space-between",
              }}
            >
              <Link
                to={toRoute}
                style={{ ...styleLink, flex: 1 }}
              >
                {submenu}
              </Link>
              {submenu === "Status" && healthCount > 0 && (
                (() => {
                  const display = healthCount > 9 ? "9+" : String(healthCount);
                  return (
                    <span
                      style={{
                        background: "#ef4444",
                        color: "#fff",
                        borderRadius: 6,
                        width: 20,
                        height: 20,
                        display: "inline-flex",
                        alignItems: "center",
                        justifyContent: "center",
                        fontSize: "0.75em",
                        lineHeight: 1,
                        marginLeft: 8,
                        textAlign: "center",
                        boxSizing: "border-box",
                      }}
                      aria-label={`${healthCount} health issues`}
                    >
                      {display}
                    </span>
                  );
                })()
              )}
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
  const firstSubmenuRoute = {
    Wanted: "/wanted/movies",
    Settings: "/settings/general",
    System: "/system/status",
  };
  const handleMenuClick = (name) => {
    if (firstSubmenuRoute[name]) {
      globalThis.location.href = firstSubmenuRoute[name];
    } else {
      handleToggle(name);
    }
  };
  return (
    <aside
      className="sidebar-desktop"
      style={{
        width: 220,
        background: isDark ? "#23232a" : "#fff",
        borderRight: isDark ? "1px solid #333" : "1px solid #e5e7eb",
        padding: "0em 0",
        height: "calc(100vh - 64px)",
        boxSizing: "border-box",
        position: "fixed",
        top: 64,
        left: 0,
        zIndex: 105,
      }}
    >
      <nav>
        <ul style={{ listStyle: "none", padding: 0, margin: 0 }}>
          {menuItems.map(({ name, icon, route }) => {
            let background, color, fontWeight;
            if (selectedSection === name) {
              background = isDark ? "#333" : "#f3f4f6";
              color = isDark ? "#a855f7" : "#6d28d9";
              fontWeight = "bold";
            } else {
              background = "none";
              color = isDark ? "#e5e7eb" : "#333";
              fontWeight = "normal";
            }
            const styleCommon = {
              textDecoration: "none",
              background,
              border: "none",
              color,
              fontWeight,
              textAlign: "left",
              padding: "0.5em 1em",
              borderRadius: 6,
              cursor: "pointer",
              display: "flex",
              alignItems: "center",
              gap: "0.75em",
              outline: "none",
              boxShadow: "none",
              WebkitTapHighlightColor: "transparent",
              transition: "box-shadow 0.1s",
            };
            if (route) {
              return (
                <li key={name} style={{ marginBottom: 16 }}>
                  <Link
                    to={route}
                    style={styleCommon}
                    className="sidebar-menu-link"
                  >
                    <IconButton
                      icon={
                        <FontAwesomeIcon
                          icon={icon}
                          color={isDark ? "#e5e7eb" : "#333"}
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
                  className="sidebar-menu-btn"
                  onClick={() => handleMenuClick(name)}
                >
                  <IconButton
                    icon={
                      <FontAwesomeIcon
                        icon={icon}
                        color={isDark ? "#e5e7eb" : "#333"}
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
                {name === "Wanted" && isOpen("Wanted") && renderWantedSubmenu()}
                {name === "Settings" &&
                  isOpen("Settings") &&
                  renderSettingsSubmenu()}
                {name === "System" && isOpen("System") && renderSystemSubmenu()}
              </li>
            );
          })}
        </ul>
      </nav>
    </aside>
  );
}

SidebarDesktop.propTypes = {
  selectedSection: PropTypes.string.isRequired,
  selectedSettingsSub: PropTypes.string,
  selectedSystemSub: PropTypes.string,
  isOpen: PropTypes.func.isRequired,
  handleToggle: PropTypes.func.isRequired,
  healthCount: PropTypes.number,
};
