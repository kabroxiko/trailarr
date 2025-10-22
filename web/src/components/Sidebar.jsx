import React from "react";
import PropTypes from "prop-types";
import { useLocation } from "react-router-dom";
import SidebarDesktop from "./SidebarDesktop.jsx";
import SidebarMobile from "./SidebarMobile.jsx";
import "./Sidebar.mobile.css";

function getSelectedSection(path) {
  if (path === "/" || path.startsWith("/movies")) return "Movies";
  if (path.startsWith("/series")) return "Series";
  if (path.startsWith("/history")) return "History";
  if (path.startsWith("/wanted")) return "Wanted";
  if (path.startsWith("/blacklist")) return "Blacklist";
  if (path.startsWith("/settings")) return "Settings";
  if (path.startsWith("/system")) return "System";
  return "";
}

function getSelectedSettingsSub(path) {
  if (path.startsWith("/wanted/")) {
    if (path.startsWith("/wanted/movies")) return "Movies";
    if (path.startsWith("/wanted/series")) return "Series";
    return "Movies";
  }
  if (path.startsWith("/settings/")) {
    if (path.startsWith("/settings/general")) return "General";
    if (path.startsWith("/settings/radarr")) return "Radarr";
    if (path.startsWith("/settings/sonarr")) return "Sonarr";
    if (path.startsWith("/settings/extras")) return "Extras";
    return "General";
  }
  return "";
}

function getSelectedSystemSub(path) {
  if (path.startsWith("/system/")) {
    if (path.startsWith("/system/tasks")) return "Tasks";
    if (path.startsWith("/system/logs")) return "Logs";
  }
  return "";
}

export default function Sidebar({ darkMode, mobile, open, onClose, onToggle }) {
  const location = useLocation();
  const path = location.pathname;
  const selectedSection = getSelectedSection(path);
  const selectedSettingsSub = getSelectedSettingsSub(path);
  const selectedSystemSub = getSelectedSystemSub(path);
  // onToggle may be unused depending on consumer; reference to avoid linter errors
  void onToggle;

  // Local state for submenu expansion
  const [openMenus, setOpenMenus] = React.useState({});
  React.useEffect(() => {
    let menuToOpen = null;
    if (selectedSection === "Wanted") menuToOpen = "Wanted";
    else if (selectedSection === "Settings") menuToOpen = "Settings";
    else if (selectedSection === "System") menuToOpen = "System";
    if (menuToOpen) {
      setOpenMenus({ [menuToOpen]: true });
    } else {
      setOpenMenus({});
    }
  }, [selectedSection]);

  const isOpen = (menu) => !!openMenus[menu];
  const handleToggle = (menu) => {
    setOpenMenus((prev) => {
      const isOpening = !prev[menu];
      const newState = {};
      if (isOpening) {
        newState[menu] = true;
      }
      return newState;
    });
  };

  if (mobile) {
    return (
      <SidebarMobile
        darkMode={darkMode}
        open={open}
        onClose={onClose}
        selectedSection={selectedSection}
        selectedSettingsSub={selectedSettingsSub}
        selectedSystemSub={selectedSystemSub}
        isOpen={isOpen}
        handleToggle={handleToggle}
      />
    );
  }
  return (
    <SidebarDesktop
      darkMode={darkMode}
      selectedSection={selectedSection}
      selectedSettingsSub={selectedSettingsSub}
      selectedSystemSub={selectedSystemSub}
      isOpen={isOpen}
      handleToggle={handleToggle}
    />
  );
}

Sidebar.propTypes = {
  darkMode: PropTypes.bool.isRequired,
  mobile: PropTypes.bool,
  open: PropTypes.bool,
  onClose: PropTypes.func,
  onToggle: PropTypes.func,
};
