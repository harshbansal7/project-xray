'use client';

import React, { Fragment, useMemo, useState } from 'react';
import { 
  ChevronDown, 
  ChevronRight, 
  Database, 
  ArrowRight,
  FileJson,
  List,
  Info,
  Maximize2,
  Minimize2,
  Search,
  X
} from 'lucide-react';

interface EventDataViewProps {
  inputSample?: unknown[];
  outputSample?: unknown[];
  inputCount?: number;
  outputCount?: number;
  className?: string;
}

export default function EventDataView({
  inputSample,
  outputSample,
  inputCount,
  outputCount,
  className = ''
}: EventDataViewProps) {
  const hasInput = !!(inputSample && inputSample.length > 0);
  const hasOutput = !!(outputSample && outputSample.length > 0);
  const hasCounts = inputCount !== undefined || outputCount !== undefined;
  
  if (!hasInput && !hasOutput && !hasCounts) {
    return null;
  }

  return (
    <div className={`space-y-3 ${className}`}>
      <div className="flex items-center gap-2 text-[10px] uppercase tracking-wider text-[var(--text-tertiary)] font-semibold">
        <Database className="w-3.5 h-3.5" />
        Input/Output Data
      </div>

      <div className="grid gap-3 grid-cols-1 2xl:grid-cols-2">
        {hasInput && (
          <DataSection
            title="Input"
            sample={inputSample}
            count={inputCount}
            icon={<List className="w-3 h-3" />}
            accentColor="var(--accent)"
          />
        )}
        {!hasInput && inputCount !== undefined && (
          <DataSection
            title="Input"
            sample={[]}
            count={inputCount}
            icon={<List className="w-3 h-3" />}
            accentColor="var(--accent)"
          />
        )}
        
        {hasOutput && (
          <DataSection
            title="Output"
            sample={outputSample}
            count={outputCount}
            icon={<ArrowRight className="w-3 h-3" />}
            accentColor="var(--success)"
          />
        )}
        {!hasOutput && outputCount !== undefined && (
          <DataSection
            title="Output"
            sample={[]}
            count={outputCount}
            icon={<ArrowRight className="w-3 h-3" />}
            accentColor="var(--success)"
          />
        )}
      </div>
    </div>
  );
}

interface DataSectionProps {
  title: string;
  sample: unknown[];
  count?: number;
  icon: React.ReactNode;
  accentColor: string;
}

function DataSection({ title, sample, count, icon, accentColor }: DataSectionProps) {
  const [expandedIndices, setExpandedIndices] = useState<Set<number>>(new Set([0])); // First item expanded by default
  const sampleSize = sample.length;
  const hasSubset = count !== undefined && sampleSize < count;

  const toggleItem = (index: number) => {
    const newExpanded = new Set(expandedIndices);
    if (newExpanded.has(index)) {
      newExpanded.delete(index);
    } else {
      newExpanded.add(index);
    }
    setExpandedIndices(newExpanded);
  };

  return (
    <div className="bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-lg p-2.5 sm:p-3">
      {/* Section Header */}
      <div className="flex items-start justify-between gap-2 mb-2">
        <div className="flex items-center gap-1.5">
          <div style={{ color: accentColor }}>{icon}</div>
          <span className="text-xs font-semibold text-[var(--text-primary)]">{title}</span>
        </div>
        <div className="flex items-center gap-1.5 sm:gap-2 shrink-0">
          <span 
            className="text-[10px] font-mono px-1.5 py-0.5 rounded whitespace-nowrap"
            style={{ 
              backgroundColor: `${accentColor}15`,
              color: accentColor
            }}
          >
            {sampleSize}
            {count !== undefined ? ` / ${count}` : ''}
          </span>
          {hasSubset && (
            <span className="text-[10px] text-[var(--warning)] inline-flex items-center gap-1 whitespace-nowrap">
              <Info className="w-3 h-3" />
              sampled
            </span>
          )}
        </div>
      </div>

      {/* Sample Items */}
      <div className="space-y-1.5 min-w-0">
        {sampleSize === 0 && (
          <div className="text-xs text-[var(--text-tertiary)] bg-[var(--bg-tertiary)] border border-[var(--border-primary)] rounded p-2.5">
            {count && count > 0
              ? 'No sample payload captured for this event.'
              : 'No data available.'}
          </div>
        )}
        {sample.map((item, index) => (
          <DataItem
            key={index}
            item={item}
            index={index}
            expanded={expandedIndices.has(index)}
            onToggle={() => toggleItem(index)}
            accentColor={accentColor}
          />
        ))}
      </div>
    </div>
  );
}

interface DataItemProps {
  item: unknown;
  index: number;
  expanded: boolean;
  onToggle: () => void;
  accentColor: string;
}

function DataItem({ item, index, expanded, onToggle, accentColor }: DataItemProps) {
  const itemType = getDataType(item);
  const isComplex = itemType === 'object' || itemType === 'array';
  const previewText = getPreviewText(item, itemType);
  const [showModal, setShowModal] = useState(false);

  return (
    <div 
      className="bg-[var(--bg-tertiary)] rounded border border-[var(--border-primary)] overflow-hidden hover:border-[var(--border-secondary)] transition-colors"
    >
      {/* Item Header */}
      <button
        onClick={onToggle}
        className="w-full flex items-center justify-between gap-2 p-2 text-left hover:bg-[var(--bg-secondary)]/50 transition-colors"
      >
        <div className="flex items-center gap-1.5 sm:gap-2 flex-1 min-w-0 overflow-hidden">
          <div className="flex-shrink-0">
            {isComplex ? (
              expanded ? (
                <ChevronDown className="w-3 h-3 text-[var(--text-tertiary)]" />
              ) : (
                <ChevronRight className="w-3 h-3 text-[var(--text-tertiary)]" />
              )
            ) : (
              <FileJson className="w-3 h-3 text-[var(--text-tertiary)]" />
            )}
          </div>
          
          <span className="text-[10px] font-mono text-[var(--text-tertiary)]">
            [{index}]
          </span>
          
          <span 
            className="text-[9px] uppercase font-semibold px-1.5 py-0.5 rounded shrink-0"
            style={{ 
              backgroundColor: `${accentColor}10`,
              color: accentColor
            }}
          >
            {itemType}
          </span>

          {!expanded && previewText && (
            <span className="text-xs text-[var(--text-secondary)] truncate min-w-0">
              {previewText}
            </span>
          )}
        </div>
        {isComplex && (
          <button
            onClick={(e) => {
              e.stopPropagation();
              setShowModal(true);
            }}
            className="p-1 rounded text-[var(--text-tertiary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-secondary)] shrink-0"
            title="Open in large view"
          >
            <Maximize2 className="w-3.5 h-3.5" />
          </button>
        )}
      </button>

      {/* Expanded Content */}
      {expanded && isComplex && (
        <div className="border-t border-[var(--border-primary)] bg-[var(--bg-primary)]/30">
          <JsonExplorer value={item} maxHeightClass="max-h-60" />
        </div>
      )}

      {expanded && !isComplex && (
        <div className="border-t border-[var(--border-primary)] bg-[var(--bg-primary)]/30 p-2.5">
          <span className="text-xs text-[var(--text-secondary)] font-mono break-all">
            {String(item)}
          </span>
        </div>
      )}

      {showModal && isComplex && (
        <JsonModal
          title={`Sample [${index}]`}
          value={item}
          onClose={() => setShowModal(false)}
        />
      )}
    </div>
  );
}

// Helper functions

function JsonModal({ title, value, onClose }: { title: string; value: unknown; onClose: () => void }) {
  return (
    <div
      className="fixed inset-0 z-50 bg-black/60 flex items-center justify-center p-2 sm:p-4"
      onClick={onClose}
    >
      <div
        className="w-full max-w-[96vw] xl:max-w-[92vw] 2xl:max-w-[88vw] max-h-[94vh] bg-[var(--bg-secondary)] border border-[var(--border-primary)] rounded-lg sm:rounded-xl shadow-2xl overflow-hidden"
        onClick={(e) => e.stopPropagation()}
      >
        <div className="px-4 py-3 border-b border-[var(--border-primary)] flex items-center justify-between">
          <div className="flex items-center gap-2">
            <Minimize2 className="w-4 h-4 text-[var(--text-tertiary)]" />
            <span className="text-sm font-medium text-[var(--text-primary)]">{title}</span>
          </div>
          <button
            onClick={onClose}
            className="p-1 rounded text-[var(--text-tertiary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]"
            title="Close"
          >
            <X className="w-4 h-4" />
          </button>
        </div>
        <JsonExplorer value={value} maxHeightClass="max-h-[calc(94vh-64px)]" />
      </div>
    </div>
  );
}

function JsonExplorer({ value, maxHeightClass }: { value: unknown; maxHeightClass: string }) {
  const [search, setSearch] = useState('');
  const [expandedPaths, setExpandedPaths] = useState<Set<string>>(() => initializeExpanded(value));
  const normalizedSearch = search.trim().toLowerCase();

  const togglePath = (path: string) => {
    setExpandedPaths(prev => {
      const next = new Set(prev);
      if (next.has(path)) next.delete(path);
      else next.add(path);
      return next;
    });
  };

  const controls = useMemo(() => (
    <div className="px-3 py-2 border-b border-[var(--border-primary)] bg-[var(--bg-secondary)]/80">
      <div className="flex flex-wrap items-center gap-2">
        <div className="relative flex-1 min-w-[180px]">
          <Search className="w-3.5 h-3.5 absolute left-2.5 top-1/2 -translate-y-1/2 text-[var(--text-tertiary)]" />
          <input
            type="text"
            value={search}
            onChange={(e) => setSearch(e.target.value)}
            placeholder="Search keys/values..."
            className="w-full h-8 pl-8 pr-3 text-xs rounded border border-[var(--border-primary)] bg-[var(--bg-tertiary)] text-[var(--text-primary)] placeholder:text-[var(--text-tertiary)] focus:outline-none focus:border-[var(--accent)]"
          />
        </div>
        <button
          onClick={() => setExpandedPaths(expandAllPaths(value))}
          className="h-8 px-2.5 text-xs rounded border border-[var(--border-primary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]"
        >
          Expand all
        </button>
        <button
          onClick={() => setExpandedPaths(new Set(['$']))}
          className="h-8 px-2.5 text-xs rounded border border-[var(--border-primary)] text-[var(--text-secondary)] hover:text-[var(--text-primary)] hover:bg-[var(--bg-tertiary)]"
        >
          Collapse all
        </button>
      </div>
    </div>
  ), [search, value]);

  return (
    <div className="w-full">
      {controls}
      <div className={`p-2 sm:p-2.5 overflow-y-auto overflow-x-hidden ${maxHeightClass}`}>
        <div className="font-mono text-[11px] leading-5 text-[var(--text-secondary)] whitespace-pre-wrap break-words">
          <JsonNode
            nodeKey="$"
            value={value}
            depth={0}
            path="$"
            expandedPaths={expandedPaths}
            onToggle={togglePath}
            search={normalizedSearch}
            parentIsArray={false}
          />
        </div>
      </div>
    </div>
  );
}

interface JsonNodeProps {
  nodeKey: string;
  value: unknown;
  depth: number;
  path: string;
  expandedPaths: Set<string>;
  onToggle: (path: string) => void;
  search: string;
  parentIsArray: boolean;
}

function JsonNode({
  nodeKey,
  value,
  depth,
  path,
  expandedPaths,
  onToggle,
  search,
  parentIsArray,
}: JsonNodeProps) {
  if (search && !nodeMatchesSearch(nodeKey, value, search)) {
    return null;
  }

  const isObj = isPlainObject(value);
  const isArr = Array.isArray(value);
  const isContainer = isObj || isArr;
  const isExpanded = expandedPaths.has(path);
  const indent = { paddingLeft: `${depth * 12}px` };

  if (!isContainer) {
    return (
      <div style={indent}>
        {!parentIsArray && <JsonKeyText text={nodeKey} search={search} />}
        {!parentIsArray && <span>: </span>}
        <JsonPrimitive value={value} search={search} />
      </div>
    );
  }

  const entries = isArr
    ? (value as unknown[]).map((v, i) => [String(i), v] as const)
    : Object.entries(value as Record<string, unknown>);
  const openChar = isArr ? '[' : '{';
  const closeChar = isArr ? ']' : '}';

  return (
    <div>
      <div style={indent} className="flex items-center gap-1">
        <button
          onClick={() => onToggle(path)}
          className="p-0.5 rounded text-[var(--text-tertiary)] hover:text-[var(--text-primary)]"
        >
          {isExpanded ? <ChevronDown className="w-3 h-3" /> : <ChevronRight className="w-3 h-3" />}
        </button>
        {!parentIsArray && <JsonKeyText text={nodeKey} search={search} />}
        {!parentIsArray && <span>: </span>}
        <span className="text-[var(--text-primary)]">{openChar}</span>
        {!isExpanded && (
          <span className="text-[var(--text-tertiary)] ml-1">
            {entries.length} {isArr ? 'items' : 'keys'}
          </span>
        )}
        {!isExpanded && <span className="text-[var(--text-primary)] ml-1">{closeChar}</span>}
      </div>

      {isExpanded && (
        <>
          {entries.map(([k, v]) => (
            <JsonNode
              key={`${path}.${k}`}
              nodeKey={k}
              value={v}
              depth={depth + 1}
              path={`${path}.${k}`}
              expandedPaths={expandedPaths}
              onToggle={onToggle}
              search={search}
              parentIsArray={isArr}
            />
          ))}
          <div style={indent} className="text-[var(--text-primary)]">{closeChar}</div>
        </>
      )}
    </div>
  );
}

function JsonPrimitive({ value, search }: { value: unknown; search: string }) {
  if (value === null) return <span className="text-slate-400">null</span>;

  switch (typeof value) {
    case 'string':
      return <span className="text-emerald-400">"<HighlightText text={value} search={search} />"</span>;
    case 'number':
      return <span className="text-amber-400">{String(value)}</span>;
    case 'boolean':
      return <span className="text-violet-400">{String(value)}</span>;
    default:
      return <span className="text-slate-300">{String(value)}</span>;
  }
}

function JsonKeyText({ text, search }: { text: string; search: string }) {
  return (
    <span className="text-sky-400">
      <HighlightText text={text} search={search} />
    </span>
  );
}

function HighlightText({ text, search }: { text: string; search: string }) {
  if (!search) return <>{text}</>;

  const lower = text.toLowerCase();
  const idx = lower.indexOf(search);
  if (idx === -1) return <>{text}</>;

  return (
    <Fragment>
      {text.slice(0, idx)}
      <mark className="bg-yellow-400/30 text-yellow-100 rounded px-0.5">
        {text.slice(idx, idx + search.length)}
      </mark>
      {text.slice(idx + search.length)}
    </Fragment>
  );
}

function getDataType(item: unknown): string {
  if (item === null) return 'null';
  if (item === undefined) return 'undefined';
  if (Array.isArray(item)) return 'array';
  return typeof item;
}

function getPreviewText(item: unknown, type: string): string {
  switch (type) {
    case 'string':
      return `"${String(item).substring(0, 40)}${String(item).length > 40 ? '...' : ''}"`;
    case 'number':
    case 'boolean':
      return String(item);
    case 'object':
      if (item === null) return 'null';
      const keys = Object.keys(item as object);
      return `{ ${keys.slice(0, 3).join(', ')}${keys.length > 3 ? '...' : ''} }`;
    case 'array':
      return `[${(item as unknown[]).length} items]`;
    default:
      return '';
  }
}

function isPlainObject(value: unknown): value is Record<string, unknown> {
  return typeof value === 'object' && value !== null && !Array.isArray(value);
}

function nodeMatchesSearch(key: string, value: unknown, search: string): boolean {
  if (!search) return true;
  if (key.toLowerCase().includes(search)) return true;

  if (value === null || value === undefined) return false;

  if (Array.isArray(value)) {
    return value.some(v => nodeMatchesSearch('', v, search));
  }

  if (isPlainObject(value)) {
    return Object.entries(value).some(([k, v]) => nodeMatchesSearch(k, v, search));
  }

  return String(value).toLowerCase().includes(search);
}

function initializeExpanded(value: unknown): Set<string> {
  const paths = new Set<string>(['$']);
  if (Array.isArray(value)) {
    value.forEach((_, idx) => paths.add(`$.${idx}`));
  } else if (isPlainObject(value)) {
    Object.keys(value).forEach(k => paths.add(`$.${k}`));
  }
  return paths;
}

function expandAllPaths(value: unknown): Set<string> {
  const paths = new Set<string>();

  const walk = (v: unknown, path: string) => {
    paths.add(path);
    if (Array.isArray(v)) {
      v.forEach((item, idx) => walk(item, `${path}.${idx}`));
      return;
    }
    if (isPlainObject(v)) {
      Object.entries(v).forEach(([k, item]) => walk(item, `${path}.${k}`));
    }
  };

  walk(value, '$');
  return paths;
}
