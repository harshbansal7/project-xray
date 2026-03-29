# Architecture & Design Decisions

## 2026-03-29: Event Input/Output Visualization

### Problem
The dashboard displays events with annotations, but lacks visualization for the actual input and output data that was recorded using `SetInput()` and `SetOutput()`. Users can see the event exists in the flow diagram, but cannot inspect the actual data samples that were processed.

### Current State
- Backend already captures and returns `input_sample` and `output_sample` arrays
- TypeScript interface in `api.ts` already includes `input_sample` and `output_sample` fields
- Current trace detail page only shows annotations in the expanded event view

### Root Cause Analysis
During implementation, discovered **two critical issues**:

1. **API Bug**: The batch endpoint (`/api/v1/events/batch`) was not copying `input_sample` and `output_sample` fields from the request to the database model, causing SDK-generated events to have empty sample arrays.

2. **SDK Limitation**: The `sampleItems()` function only handled arrays/slices. When users called `SetInput()` with a single object (map, struct, etc.), it would return `nil` instead of capturing the data. This affected use cases like:
   ```go
   event.SetInput(map[string]interface{}{
       "current_speaker": speaker,
       "current_text":    text,
   })
   ```

### Solution
Create a high-quality, reusable `EventDataView` component that:
1. Displays both input and output samples in an organized, readable format
2. Uses collapsible sections to handle large data sets
3. Provides JSON syntax highlighting for complex objects
4. Shows data type indicators (array, object, primitive)
5. Includes sample count badges
6. Matches existing UI design patterns with proper color scheme

**SDK Fix**: Updated `sampleItems()` in `sdk/go/event.go` to wrap single objects (maps, structs, primitives) in an array, making them visible in the dashboard.

**API Fix**: Updated batch ingestion handler to properly copy `input_sample` and `output_sample` fields.

### 2026-03-29 Follow-up: Wrapping + Sampling Clarity

User reported two UX issues:
1. JSON required horizontal scrolling in event payload view.
2. Confusion when counts (e.g. `2`) exceeded visible samples (e.g. `1`).

Decisions:
- Enable wrapped JSON rendering (`whitespace-pre-wrap`, `break-words`, no horizontal overflow) to fit container width.
- Always render Input/Output sections when counts exist, even if payload sample is empty.
- Show explicit `shown / total` badge and `sampled` indicator when sample length is less than count.
- Display an explicit empty-state message when counts exist but no sample payload was captured.

### 2026-03-29 Follow-up: Advanced JSON Reader UX

User requested higher readability and exploration for large payloads.
Implemented in dashboard EventDataView:
- Syntax-colored JSON (keys, strings, numbers, booleans, null).
- Large modal viewer for complex payloads.
- Expand/collapse tree for objects and arrays.
- Search across keys/values with highlighting.
- Expand-all / collapse-all controls.

This keeps event inspection usable for deeply nested payloads while preserving existing compact view.

### 2026-03-29 Follow-up: 2K Responsiveness

User reported poor visual balance on large (2K) displays where content remained too narrow and dense.

Decisions:
- Remove tight max-width constraints on trace detail page and allow full-width content area with responsive paddings.
- Delay two-column Input/Output split to extra-wide breakpoints (`2xl`) so medium/large screens keep readable single-column sections.
- Improve row/header wrapping behavior in payload cards to reduce cramped alignment and clipped labels.
- Increase modal effective width on large displays for better deep JSON inspection.

### 2026-03-29 Follow-up: Global Large-Screen Polish

User requested the same large-monitor quality improvements across all dashboard pages.

Decisions:
- Standardize page container strategy to full-width responsive paddings (`px-4 sm:px-6 lg:px-8 2xl:px-10`) instead of narrow fixed max-width wrappers.
- Increase information density progressively at larger breakpoints (e.g. more columns on pipeline cards).
- Keep dense tables usable with horizontal overflow wrappers and minimum table widths where necessary.
- Delay multi-column dashboard splits to wider breakpoints (`xl`/`2xl`) for improved readability on laptop and standard desktop widths.

### Design Decisions
- **Component Structure**: Create `EventDataView` component in `/dashboard/src/components/trace/EventDataView.tsx`
- **Layout**: Side-by-side or stacked layout depending on available data
- **Data Rendering**: 
  - Primitives: inline display
  - Objects/Arrays: Pretty-printed JSON with syntax highlighting
  - Empty data: Clear "No data" indicators
- **Interaction**: Expandable/collapsible for each sample item
- **Styling**: Match existing card/section patterns from DecisionStream and trace detail page

### Integration Points
- Update `traces/[id]/page.tsx` to display EventDataView when event is expanded
- Place it between annotations and decisions sections for logical flow
- Ensure proper styling hierarchy with existing components
