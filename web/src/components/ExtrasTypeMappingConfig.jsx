import React, { useState } from "react";
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

function TmdbChip({ tmdbType, plexType, onMove }) {
  const [{ isDragging }, drag] = useDrag({
    type: ItemTypes.CHIP,
    item: { tmdbType, plexType },
    collect: (monitor) => ({ isDragging: monitor.isDragging() }),
  });
  return (
    <Box
      ref={drag}
      sx={{
        background: '#e5e7eb',
        color: '#222',
        borderRadius: 2, // slightly more rounded
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
          color: '#a855f7',
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

function PlexTypeBox({ plexType, onDropChip, children }) {
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
        background: isOver && canDrop ? '#d1fae5' : '#f5f5f5',
        border: '1px solid #000',
        borderRadius: 2, // slightly more rounded
        padding: '4px 8px',
        transition: 'background 0.2s',
      }}
    >
      {children}
    </Box>
  );
}

const fetchTMDBTypes = async () => {
  const res = await fetch("/api/tmdb/extratypes");
  const data = await res.json();
  return data.tmdbExtraTypes || [];
};

const fetchPlexTypes = async () => {
  const res = await fetch("/api/settings/extratypes");
  const data = await res.json();
  // Use all keys as Plex types, even if disabled
  return Object.keys(data);
};

const fetchMapping = async () => {
  const res = await fetch("/api/settings/canonicalizeextratype");
  const data = await res.json();
  return data.mapping || {};
};

const saveMapping = async (mapping) => {
  await fetch("/api/settings/canonicalizeextratype", {
    method: "POST",
    headers: { "Content-Type": "application/json" },
    body: JSON.stringify({ mapping }),
  });
};

export default function ExtrasTypeMappingConfig({ mapping, onMappingChange, tmdbTypes, plexTypes }) {
  const [error, setError] = useState("");
  const [loading, setLoading] = useState(false);
  const [movingChip, setMovingChip] = useState(null);

  // Only local error/loading, all mapping state is controlled by parent

  // Only allow each Plex type to be assigned once
  const usedPlexTypes = Object.values(mapping);

  const handleMoveChip = (tmdbType, newPlexType) => {
    if (onMappingChange) {
      onMappingChange({ ...mapping, [tmdbType]: newPlexType });
    }
  };

  if (loading) {
    return (
      <Box display="flex" justifyContent="center" alignItems="center" minHeight={200}>
        <CircularProgress />
      </Box>
    );
  }

  return (
    <DndProvider backend={HTML5Backend}>
      <Box>
        <SectionHeader>
          TMDB to Plex Extra Type Mapping
        </SectionHeader>
        {error && (
          <Typography color="error" gutterBottom>
            {error}
          </Typography>
        )}
        <Paper sx={{ mt: 2, p: 1, maxWidth: 470, ml: 0, boxShadow: 'none', border: 'none' }}>
          {plexTypes.map((plexType) => {
            const assignedTmdbTypes = tmdbTypes.filter(
              (tmdbType) => mapping[tmdbType] === plexType
            );
            return (
              <Box key={plexType} display="flex" alignItems="center" mb={1}>
                <Box minWidth={120} fontWeight={500} fontSize={14} textAlign="left">
                  {plexType}
                </Box>
                <Box flex={1} ml={1}>
                  <PlexTypeBox plexType={plexType} assignedTmdbTypes={assignedTmdbTypes} onDropChip={handleMoveChip}>
                    {assignedTmdbTypes.map((tmdbType) => (
                      <TmdbChip key={tmdbType} tmdbType={tmdbType} plexType={plexType} onMove={handleMoveChip} />
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
