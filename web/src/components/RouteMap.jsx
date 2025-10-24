import React from "react";
import MediaList from "./MediaList";
import MediaDetails from "./MediaDetails";
import GeneralSettings from "./GeneralSettings";
import ExtrasSettings from "./ExtrasSettings";
import HistoryPage from "./HistoryPage";
import LogsPage from "./LogsPage";
import SettingsPage from "./SettingsPage";
import Tasks from "./Tasks";
import Wanted from "./Wanted";

export const RouteMap = [
  { pattern: /^\/$/, section: "Movies" },
  { pattern: /^\/movies/, section: "Movies" },
  { pattern: /^\/series/, section: "Series" },
  { pattern: /^\/history/, section: "History" },
  { pattern: /^\/blacklist/, section: "Blacklist" },
  { pattern: /^\/wanted\/movies/, section: "Wanted", submenu: "Movies" },
  { pattern: /^\/wanted\/series/, section: "Wanted", submenu: "Series" },
  { pattern: /^\/wanted/, section: "Wanted", submenu: "Movies" },
  {
    pattern: /^\/settings\/(radarr|sonarr|general|extras)/,
    section: "Settings",
  },
  { pattern: /^\/settings\/radarr/, section: "Settings", submenu: "Radarr" },
  { pattern: /^\/settings\/sonarr/, section: "Settings", submenu: "Sonarr" },
  { pattern: /^\/settings\/general/, section: "Settings", submenu: "General" },
  { pattern: /^\/settings\/extras/, section: "Settings", submenu: "Extras" },
  { pattern: /^\/settings/, section: "Settings", submenu: "General" },
  { pattern: /^\/system\/tasks/, section: "System", systemSub: "Tasks" },
  { pattern: /^\/system\/logs/, section: "System", systemSub: "Logs" },
  { pattern: /^\/system/, section: "System", systemSub: "Tasks" },
];

export const appRoutes = [
  // Dynamic routes (functions)
  {
    path: "/series",
    dynamic: true,
    render: (props) => {
      const { series, search, darkMode, getSearchSections, seriesError } =
        props;
      const { titleMatches, overviewMatches } = getSearchSections(series);
      return (
        <>
          {search.trim() ? (
            <>
              {(() =>
                React.createElement(MediaList, {
                  items: titleMatches,
                  darkMode,
                  type: "series",
                }))()}
              <div
                style={{
                  margin: "1.5em 0 0.5em 1em",
                  fontWeight: 700,
                  fontSize: 26,
                  textAlign: "left",
                  width: "100%",
                  letterSpacing: 0.5,
                }}
              >
                Other Results
              </div>
                {(() =>
                React.createElement(MediaList, {
                  items: overviewMatches,
                  darkMode,
                  type: "series",
                }))()}
            </>
          ) : (
            (() => React.createElement(MediaList, { items: series, darkMode, type: "series" }))()
          )}
          {seriesError && (
            <div style={{ color: "red", marginTop: "1em" }}>{seriesError}</div>
          )}
        </>
      );
    },
  },
  {
    path: "/",
    dynamic: true,
    render: (props) => {
      const { movies, search, darkMode, getSearchSections, moviesError } =
        props;
      const { titleMatches, overviewMatches } = getSearchSections(movies);
      return (
        <>
          {search.trim() ? (
            <>
              {(() =>
    React.createElement(MediaList, { items: titleMatches, darkMode, type: "movie" }))()}
              <div
                style={{
                  margin: "1.5em 0 0.5em 1em",
                  fontWeight: 700,
                  fontSize: 26,
                  textAlign: "left",
                  width: "100%",
                  letterSpacing: 0.5,
                }}
              >
                Other Results
              </div>
              {(() => React.createElement(MediaList, { items: overviewMatches, darkMode, type: "movie" }))()}
            </>
          ) : (
            (() => React.createElement(MediaList, { items: movies, darkMode, type: "movie" }))()
          )}
          {moviesError && (
            <div style={{ color: "red", marginTop: "1em" }}>{moviesError}</div>
          )}
        </>
      );
    },
  },
  {
    path: "/movies/:id",
    dynamic: true,
    render: (props) =>
      React.createElement(MediaDetails, {
          mediaItems: props.movies,
          loading: props.moviesLoading,
          mediaType: "movie",
        }),
  },
  {
    path: "/series/:id",
    dynamic: true,
    render: (props) =>
      React.createElement(MediaDetails, {
        mediaItems: props.series,
        loading: props.seriesLoading,
        mediaType: "tv",
      }),
  },
  // Static routes
  {
    path: "/history",
    element: React.createElement(HistoryPage),
  },
  {
    path: "/wanted/movies",
    dynamic: true,
    render: (props) =>
      React.createElement(Wanted, {
        type: "movie",
        darkMode: props.darkMode,
        items: props.movies,
      }),
  },
  {
    path: "/wanted/series",
    dynamic: true,
    render: (props) =>
      React.createElement(Wanted, {
        type: "series",
        darkMode: props.darkMode,
        items: props.series,
      }),
  },
  {
    path: "/settings/radarr",
    element: React.createElement(SettingsPage, { type: "radarr" }),
  },
  {
    path: "/settings/sonarr",
    element: React.createElement(SettingsPage, { type: "sonarr" }),
  },
  {
    path: "/settings/general",
    element: React.createElement(GeneralSettings),
  },
  {
    path: "/settings/extras",
    element: React.createElement(ExtrasSettings),
  },
  {
    path: "/system/tasks",
    element: React.createElement(Tasks),
  },
  {
    path: "/system/logs",
    element: React.createElement(LogsPage),
  },
];
