# Progress Tracker

## Dashboard Features

### ✅ Completed
- [x] Event Input/Output Visualization (2026-03-29)
  - Created EventDataView component for displaying input/output samples
  - Integrated into trace detail page
  - Supports expandable/collapsible data items
  - Shows data type indicators and preview text
  - Proper styling matching existing UI standards
  - Fixed backend bug in batch event ingestion handler
  - **Fixed SDK to support single objects** (maps, structs) in addition to arrays
  - JSON payload view now wraps to container width (no horizontal scroll)
  - Added explicit sampled-state UX (`shown / total` + `sampled` badge)
  - Added empty-state messaging when counts exist but no sample payload was captured
  - Added syntax-colored JSON viewer with searchable, collapsible tree
  - Added large modal view for complex payload inspection
  - Improved responsiveness for large (2K) screens by widening trace layout and rebalancing data panel breakpoints
  - Applied large-screen polish across all major pages (home, traces, pipelines, pipeline detail)

## Backend Bug Fixes

### ✅ Completed
- [x] Batch Event Ingestion - Input/Output Samples (2026-03-29)
  - Fixed `/api/v1/events/batch` endpoint to properly copy `input_sample` and `output_sample` fields
  - Samples are now correctly stored and retrieved from database
  - SDK can now properly send and display input/output data

## SDK Enhancements

### ✅ Completed
- [x] Single Object Support in SetInput/SetOutput (2026-03-29)
  - Updated `sampleItems()` function to wrap single objects (maps, structs, primitives) in arrays
  - Previously only arrays/slices were captured
  - Now supports common use cases like `SetInput(map[string]interface{}{...})`
  - Maintains backward compatibility with existing array-based usage

### 🚧 In Progress
- None

### 📋 Planned
- TBD
