import React, { useState, useEffect, useRef } from 'react';
import { Search } from 'lucide-react';
import { useNavigate } from 'react-router-dom';

export interface GlobalSearchProps {
  className?: string;
  initialValue?: string;
}

export const GlobalSearch: React.FC<GlobalSearchProps> = ({ className = '', initialValue = '' }) => {
  const [query, setQuery] = useState(initialValue);
  const inputRef = useRef<HTMLInputElement>(null);
  const navigate = useNavigate();

  useEffect(() => {
    setQuery(initialValue);
  }, [initialValue]);

  useEffect(() => {
    const handleKeyDown = (e: KeyboardEvent) => {
      if ((e.ctrlKey || e.metaKey) && e.key.toLowerCase() === 'k') {
        e.preventDefault();
        inputRef.current?.focus();
        inputRef.current?.select();
      }
    };
    window.addEventListener('keydown', handleKeyDown);
    return () => window.removeEventListener('keydown', handleKeyDown);
  }, []);

  const handleSubmit = (e: React.FormEvent) => {
    e.preventDefault();
    const trimmed = query.trim();
    if (!trimmed) return;

    // Check if 16-hex or 32-hex trace ID
    const isHexId = /^[a-fA-F0-9]{16,32}$/.test(trimmed);
    if (isHexId) {
      navigate(`/traces?trace=${trimmed.toLowerCase()}`);
    } else {
      navigate(`/traces?q=${encodeURIComponent(trimmed)}`);
    }
  };

  return (
    <form onSubmit={handleSubmit} className={`relative w-full ${className}`}>
      <div className="relative flex items-center w-full">
        <Search
          className="absolute left-3.5 w-4 h-4 text-[#6b7188] pointer-events-none"
          aria-hidden="true"
        />
        <input
          ref={inputRef}
          type="text"
          value={query}
          onChange={(e) => setQuery(e.target.value)}
          placeholder="Search by trace ID, service, or operation..."
          className="w-full h-10 pl-10 pr-16 bg-[#10121f] text-[#f3f4f8] placeholder-[#6b7188] text-[13px] rounded-[8px] border border-[rgba(255,255,255,0.07)] focus:outline-none focus:border-[#7c6cf0] focus:ring-1 focus:ring-[#7c6cf0] transition-colors"
          aria-label="Search traces, services, or operations"
        />
        <div className="absolute right-3 flex items-center gap-1 pointer-events-none">
          <kbd className="px-1.5 py-0.5 text-[10px] font-mono text-[#6b7188] bg-[#0a0b14] border border-[rgba(255,255,255,0.09)] rounded shadow-sm">
            Ctrl
          </kbd>
          <kbd className="px-1.5 py-0.5 text-[10px] font-mono text-[#6b7188] bg-[#0a0b14] border border-[rgba(255,255,255,0.09)] rounded shadow-sm">
            K
          </kbd>
        </div>
      </div>
    </form>
  );
};

