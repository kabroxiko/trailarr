import React from "react";
import {
  Typography,
  Box,
  CircularProgress,
  Paper,
} from "@mui/material";
import { DndProvider, useDrag, useDrop } from 'react-dnd';
import { HTML5Backend } from 'react-dnd-html5-backend';
import SectionHeader from "./SectionHeader";

const ItemTypes = { CHIP: 'chip' };

function TmdbChip({ tmdbType, plexType, onMove, isDark }) {
  const [{ isDragging }, drag] = useDrag({
    type: ItemTypes.CHIP,
    item: { tmdbType, plexType },
    collect: (monitor) => ({ isDragging: monitor.isDragging() }),
  });
  return (
    <Box
      ref={drag}
      sx={{
        background: isDark ? '#222' : '#e5e7eb',
        color: isDark ? '#e5e7eb' : '#222',
        borderRadius: 2,
        fontSize: 13,
        height: 24,
        lineHeight: '24px',
        margin: '2px 6px 2px 0',
        padding: '0 10px',
        fontWeight: 500,
        display: 'flex',
        alignItems: 'center',
        opacity: isDragging ? 0.5 : 1,
        cursor: 'grab',
      }}
      title="Drag to another Plex type"
    >
      {tmdbType}
      <Box
        component="span"
        sx={{
          cursor: 'pointer',
          marginLeft: 6,
          color: isDark ? '#c084fc' : '#a855f7',
          fontWeight: 700,
          fontSize: 15,
        }}
        onClick={e => {
          e.stopPropagation();
          onMove(tmdbType, "Other");
        }}
        title="Remove assignment"
      >
        ×
      </Box>
    </Box>
  );
}

function PlexTypeBox({ plexType, onDropChip, children, isDark }) {
  const [{ isOver, canDrop }, drop] = useDrop({
    accept: ItemTypes.CHIP,
    drop: (item) => onDropChip(item.tmdbType, plexType),
    canDrop: (item) => item.plexType !== plexType,
    collect: (monitor) => ({
      isOver: monitor.isOver(),
      canDrop: monitor.canDrop(),
    }),
  });
  return (
    <Box
      ref={drop}
      sx={{
        display: 'flex',
        flexWrap: 'wrap',
        alignItems: 'center',
        minHeight: 36,
        background: isOver && canDrop
          ? (isDark ? '#047857' : '#d1fae5')
          : (isDark ? '#333' : '#f5f5f5'),
        border: isDark ? '1px solid #e5e7eb' : '1px solid #000',
        borderRadius: 2,
        padding: '4px 8px',
        transition: 'background 0.2s',
      }}
    >
      {children}
    </Box>
  );
}

// Removed unused fetchTMDBTypes, fetchPlexTypes, fetchMapping, saveMapping

export default function ExtrasTypeMappingConfig({ mapping, onMappingChange, tmdbTypes, plexTypes, isDark = false }) {
  // Removed unused error and loading state

  // Only local error/loading, all mapping state is controlled by parent

  // Removed unused usedPlexTypes

  const handleMoveChip = (tmdbType, newPlexType) => {
    if (onMappingChange) {
      onMappingChange({ ...mapping, [tmdbType]: newPlexType });
    }
  };

  // Removed undefined loading check

  return (
    <DndProvider backend={HTML5Backend}>
      <Box>
        <SectionHeader>
          TMDB to Plex Extra Type Mapping
        </SectionHeader>
        {/* Removed undefined error display */}
        <Paper sx={{
          mt: 2,
          p: 1,
          maxWidth: 470,
          ml: 0,
          boxShadow: 'none',
          border: 'none',
          background: 'transparent',
          color: isDark ? '#e5e7eb' : '#222',
        }}>
          {plexTypes.map((plexType) => {
            const assignedTmdbTypes = tmdbTypes.filter(
              (tmdbType) => mapping[tmdbType] === plexType
            );
            return (
              <Box key={plexType} display="flex" alignItems="center" mb={1}>
                <Box minWidth={120} fontWeight={500} fontSize={14} textAlign="left" sx={{ color: isDark ? '#e5e7eb' : '#222' }}>
                  {plexType}
                </Box>
                <Box flex={1} ml={1}>
                  <PlexTypeBox plexType={plexType} assignedTmdbTypes={assignedTmdbTypes} onDropChip={handleMoveChip} isDark={isDark}>
                    {assignedTmdbTypes.map((tmdbType) => (
                      <TmdbChip key={tmdbType} tmdbType={tmdbType} plexType={plexType} onMove={handleMoveChip} isDark={isDark} />
                    ))}
                  </PlexTypeBox>
                </Box>
              </Box>
            );
          })}
        </Paper>
      </Box>
    </DndProvider>
  );
}
