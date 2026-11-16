import React from 'react';

export interface CardProps extends React.HTMLAttributes<HTMLDivElement> {
  title?: React.ReactNode;
  subtitle?: React.ReactNode;
  action?: React.ReactNode;
  hoverable?: boolean;
  padding?: 'none' | 'sm' | 'md' | 'lg';
}

export const Card: React.FC<CardProps> = ({
  title,
  subtitle,
  action,
  hoverable = false,
  padding = 'md',
  className = '',
  children,
  ...props
}) => {
  const paddingClasses = {
    none: 'p-0',
    sm: 'p-3.5',
    md: 'p-5',
    lg: 'p-6',
  }[padding];

  return (
    <div
      className={`rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] text-[#f3f4f8] shadow-sm transition-all duration-150 ${
        hoverable ? 'hover:border-[rgba(255,255,255,0.14)] hover:bg-[#141727]' : ''
      } ${className}`}
      {...props}
    >
      {(title || subtitle || action) && (
        <div className="flex items-start justify-between gap-4 pb-4 border-b border-[rgba(255,255,255,0.05)] px-5 pt-5">
          <div>
            {title && (
              <h3 className="text-[14px] font-semibold text-[#f3f4f8] tracking-tight">{title}</h3>
            )}
            {subtitle && (
              <p className="text-[12px] text-[#9aa0b5] mt-0.5 leading-relaxed">{subtitle}</p>
            )}
          </div>
          {action && <div className="flex items-center gap-2">{action}</div>}
        </div>
      )}
      <div className={paddingClasses}>{children}</div>
    </div>
  );
};

