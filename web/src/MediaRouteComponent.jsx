import React from "react";
import PropTypes from "prop-types";
import MediaList from "./components/MediaList";

function MediaRouteComponent({ items, search, darkMode, error, getSearchSections, type }) {
  const { titleMatches, overviewMatches } = getSearchSections(items);
  return (
    <>
      {search.trim() ? (
        <>
          <MediaList items={titleMatches} darkMode={darkMode} type={type} />
          <div style={{ margin: '1.5em 0 0.5em 1em', fontWeight: 700, fontSize: 26, textAlign: 'left', width: '100%', letterSpacing: 0.5 }}>Other Results</div>
          <MediaList items={overviewMatches} darkMode={darkMode} type={type} />
        </>
      ) : (
        <MediaList items={items} darkMode={darkMode} type={type} />
      )}
      {error && <div style={{ color: 'red', marginTop: '1em' }}>{error}</div>}
    </>
  );
}

export default MediaRouteComponent;

MediaRouteComponent.propTypes = {
  items: PropTypes.array.isRequired,
  search: PropTypes.string.isRequired,
  darkMode: PropTypes.bool.isRequired,
  error: PropTypes.string,
  getSearchSections: PropTypes.func.isRequired,
  type: PropTypes.string.isRequired,
};
