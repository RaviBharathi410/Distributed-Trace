import React from 'react';
import { Link } from 'react-router-dom';

export interface BrandLogoProps {
  to?: string;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
  showText?: boolean;
}

export const BrandLogoMark: React.FC<{ size?: number; className?: string }> = ({
  size = 20,
  className = '',
}) => {
  return (
    <svg
      viewBox="0 0 256 256"
      width={size}
      height={size}
      className={`shrink-0 fill-[#ffffff] ${className}`}
      aria-hidden="true"
    >
      <path d="M128 192L128 256L64.5 256L32 223L0 192L0 128L64 128ZM256 192L256 256L192.5 256L160 223L128 192L128 128L192 128ZM128 64L128 128L64.5 128L32 95L0 64L0 0L64 0ZM256 64L256 128L192.5 128L160 95L128 64L128 0L192 0Z" />
    </svg>
  );
};

export const BrandLogo: React.FC<BrandLogoProps> = ({
  to = '/',
  size = 'md',
  className = '',
  showText = true,
}) => {
  const iconSize = size === 'sm' ? 16 : size === 'lg' ? 24 : 20;
  const textSize = size === 'sm' ? 'text-xs' : size === 'lg' ? 'text-base' : 'text-sm';

  const content = (
    <div className={`flex items-center gap-2 select-none ${className}`}>
      <BrandLogoMark size={iconSize} />
      {showText && (
        <span className={`text-white font-normal tracking-tight ${textSize}`}>
          distributedtrace
        </span>
      )}
    </div>
  );

  if (to) {
    return (
      <Link
        to={to}
        className="flex items-center focus:outline-none focus:ring-2 focus:ring-[#7c6cf0] rounded-md px-1 py-0.5 group"
        aria-label="distributedtrace"
      >
        {content}
      </Link>
    );
  }

  return content;
};
