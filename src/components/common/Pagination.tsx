import React from 'react';
import { ChevronLeft, ChevronRight } from 'lucide-react';

export interface PaginationProps {
  page: number;
  pageSize: number;
  totalItems: number;
  onPageChange: (page: number) => void;
  className?: string;
}

export const Pagination: React.FC<PaginationProps> = ({
  page,
  pageSize,
  totalItems,
  onPageChange,
  className = '',
}) => {
  const totalPages = Math.max(1, Math.ceil(totalItems / pageSize));
  const startItem = totalItems === 0 ? 0 : (page - 1) * pageSize + 1;
  const endItem = Math.min(totalItems, page * pageSize);

  const getPages = () => {
    const pages: (number | string)[] = [];
    if (totalPages <= 5) {
      for (let i = 1; i <= totalPages; i++) pages.push(i);
    } else {
      if (page <= 3) {
        pages.push(1, 2, 3, 4, '...', totalPages);
      } else if (page >= totalPages - 2) {
        pages.push(1, '...', totalPages - 3, totalPages - 2, totalPages - 1, totalPages);
      } else {
        pages.push(1, '...', page - 1, page, page + 1, '...', totalPages);
      }
    }
    return pages;
  };

  return (
    <div className={`flex flex-col sm:flex-row items-center justify-between gap-4 py-3 px-1 text-[12px] text-[#9aa0b5] ${className}`}>
      <div>
        Showing <span className="font-medium text-[#f3f4f8]">{startItem}</span> to{' '}
        <span className="font-medium text-[#f3f4f8]">{endItem}</span> of{' '}
        <span className="font-medium text-[#f3f4f8]">{totalItems}</span>
      </div>

      <div className="flex items-center gap-1.5">
        <button
          type="button"
          onClick={() => onPageChange(page - 1)}
          disabled={page <= 1}
          className="h-8 w-8 rounded-[6px] border border-[rgba(255,255,255,0.07)] flex items-center justify-center text-[#9aa0b5] hover:text-[#f3f4f8] hover:bg-white/5 disabled:opacity-40 disabled:pointer-events-none transition-colors"
          aria-label="Previous page"
        >
          <ChevronLeft className="w-4 h-4" />
        </button>

        {getPages().map((p, idx) => {
          if (p === '...') {
            return (
              <span key={`ellipsis-${idx}`} className="px-2 py-1 text-[#6b7188]">
                ...
              </span>
            );
          }

          const pageNum = Number(p);
          const isCurrent = pageNum === page;

          return (
            <button
              key={pageNum}
              type="button"
              onClick={() => onPageChange(pageNum)}
              className={`h-8 min-w-[32px] px-2 rounded-[6px] text-[12px] font-medium transition-all ${
                isCurrent
                  ? 'bg-[#7c6cf0] text-white shadow-sm'
                  : 'border border-[rgba(255,255,255,0.07)] text-[#9aa0b5] hover:text-[#f3f4f8] hover:bg-white/5'
              }`}
            >
              {pageNum}
            </button>
          );
        })}

        <button
          type="button"
          onClick={() => onPageChange(page + 1)}
          disabled={page >= totalPages}
          className="h-8 w-8 rounded-[6px] border border-[rgba(255,255,255,0.07)] flex items-center justify-center text-[#9aa0b5] hover:text-[#f3f4f8] hover:bg-white/5 disabled:opacity-40 disabled:pointer-events-none transition-colors"
          aria-label="Next page"
        >
          <ChevronRight className="w-4 h-4" />
        </button>
      </div>
    </div>
  );
};

