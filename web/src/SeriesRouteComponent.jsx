import React from "react";
import PropTypes from "prop-types";
import MediaList from "./components/MediaList";

function SeriesRouteComponent({ series, search, darkMode, seriesError, getSearchSections }) {
  const { titleMatches, overviewMatches } = getSearchSections(series);
  return (
    <>
      {search.trim() ? (
        <>
          <MediaList items={titleMatches} darkMode={darkMode} type="series" />
          <div style={{ margin: '1.5em 0 0.5em 1em', fontWeight: 700, fontSize: 26, textAlign: 'left', width: '100%', letterSpacing: 0.5 }}>Other Results</div>
          <MediaList items={overviewMatches} darkMode={darkMode} type="series" />
        </>
      ) : (
        <MediaList items={series} darkMode={darkMode} type="series" />
      )}
      {seriesError && <div style={{ color: 'red', marginTop: '1em' }}>{seriesError}</div>}
    </>
  );
}

export default SeriesRouteComponent;

SeriesRouteComponent.propTypes = {
  series: PropTypes.array.isRequired,
  search: PropTypes.string.isRequired,
  darkMode: PropTypes.bool.isRequired,
  seriesError: PropTypes.string,
  getSearchSections: PropTypes.func.isRequired,
};
