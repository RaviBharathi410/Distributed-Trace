import React from 'react';
import { Link } from 'react-router-dom';

export interface BrandLogoProps {
  to?: string;
  size?: 'sm' | 'md' | 'lg';
  className?: string;
  showText?: boolean;
}

export const BrandLogoMark: React.FC<{ size?: number; className?: string }> = ({
  size = 22,
  className = '',
}) => {
  return (
    <svg
      width={size}
      height={size}
      viewBox="0 0 24 24"
      fill="none"
      xmlns="http://www.w3.org/2000/svg"
      className={`shrink-0 ${className}`}
      aria-hidden="true"
    >
      {/* 4 geometric rounded quadrant petals forming the iconic pinwheel */}
      <rect
        x="10.5"
        y="1.5"
        width="3"
        height="8"
        rx="1.5"
        fill="#f3f4f8"
      />
      <rect
        x="10.5"
        y="14.5"
        width="3"
        height="8"
        rx="1.5"
        fill="#f3f4f8"
      />
      <rect
        x="1.5"
        y="10.5"
        width="8"
        height="3"
        rx="1.5"
        fill="#f3f4f8"
      />
      <rect
        x="14.5"
        y="10.5"
        width="8"
        height="3"
        rx="1.5"
        fill="#f3f4f8"
      />
      {/* 4 corner accent petals */}
      <rect
        x="4.2"
        y="4.2"
        width="3"
        height="6"
        rx="1.5"
        transform="rotate(-45 4.2 4.2)"
        fill="#7c6cf0"
        opacity="0.9"
      />
      <rect
        x="18.2"
        y="4.2"
        width="3"
        height="6"
        rx="1.5"
        transform="rotate(45 18.2 4.2)"
        fill="#7c6cf0"
        opacity="0.9"
      />
      <rect
        x="4.2"
        y="18.2"
        width="3"
        height="6"
        rx="1.5"
        transform="rotate(45 4.2 18.2)"
        fill="#7c6cf0"
        opacity="0.9"
      />
      <rect
        x="18.2"
        y="18.2"
        width="3"
        height="6"
        rx="1.5"
        transform="rotate(-45 18.2 18.2)"
        fill="#7c6cf0"
        opacity="0.9"
      />
    </svg>
  );
};

export const BrandLogo: React.FC<BrandLogoProps> = ({
  to = '/',
  size = 'md',
  className = '',
  showText = true,
}) => {
  const iconSize = size === 'sm' ? 18 : size === 'lg' ? 26 : 22;
  const textSize = size === 'sm' ? 'text-[14px]' : size === 'lg' ? 'text-[18px]' : 'text-[15px]';

  const content = (
    <div className={`flex items-center gap-2.5 select-none ${className}`}>
      <BrandLogoMark size={iconSize} />
      {showText && (
        <span className={`font-semibold tracking-tight text-[#f3f4f8] ${textSize}`}>
          distributed<span className="text-[#f3f4f8] font-semibold">trace</span>
        </span>
      )}
    </div>
  );

  if (to) {
    return (
      <Link
        to={to}
        className="flex items-center focus:outline-none focus:ring-2 focus:ring-[#7c6cf0] rounded-md px-1 py-0.5 group"
        aria-label="DistributedTrace Home"
      >
        {content}
      </Link>
    );
  }

  return content;
};

