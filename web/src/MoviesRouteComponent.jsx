import React from "react";
import PropTypes from "prop-types";
import MediaList from "./components/MediaList";

function MoviesRouteComponent({ movies, search, darkMode, moviesError, getSearchSections }) {
  const { titleMatches, overviewMatches } = getSearchSections(movies);
  return (
    <>
      {search.trim() ? (
        <>
          <MediaList items={titleMatches} darkMode={darkMode} type="movie" />
          <div style={{ margin: '1.5em 0 0.5em 1em', fontWeight: 700, fontSize: 26, textAlign: 'left', width: '100%', letterSpacing: 0.5 }}>Other Results</div>
          <MediaList items={overviewMatches} darkMode={darkMode} type="movie" />
        </>
      ) : (
        <MediaList items={movies} darkMode={darkMode} type="movie" />
      )}
      {moviesError && <div style={{ color: 'red', marginTop: '1em' }}>{moviesError}</div>}
    </>
  );
}

export default MoviesRouteComponent;

MoviesRouteComponent.propTypes = {
  movies: PropTypes.array.isRequired,
  search: PropTypes.string.isRequired,
  darkMode: PropTypes.bool.isRequired,
  moviesError: PropTypes.string,
  getSearchSections: PropTypes.func.isRequired,
};
