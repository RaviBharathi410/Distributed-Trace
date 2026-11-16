import React from 'react';
import { ChevronDown } from 'lucide-react';

export interface SelectOption {
  value: string;
  label: string;
}

export interface SelectProps extends React.SelectHTMLAttributes<HTMLSelectElement> {
  options: SelectOption[];
  label?: string;
  className?: string;
}

export const Select: React.FC<SelectProps> = ({
  options,
  label,
  value,
  onChange,
  className = '',
  disabled,
  ...props
}) => {
  return (
    <div className={`relative inline-flex flex-col gap-1 ${className}`}>
      {label && <span className="text-[11px] font-medium text-[#9aa0b5]">{label}</span>}
      <div className="relative">
        <select
          value={value}
          onChange={onChange}
          disabled={disabled}
          className="appearance-none h-9 pl-3 pr-8 rounded-[8px] bg-[#10121f] text-[#f3f4f8] text-[13px] font-medium border border-[rgba(255,255,255,0.07)] hover:border-[rgba(255,255,255,0.14)] focus:outline-none focus:border-[#7c6cf0] focus:ring-1 focus:ring-[#7c6cf0] transition-colors cursor-pointer disabled:opacity-50 disabled:cursor-not-allowed w-full"
          {...props}
        >
          {options.map((opt) => (
            <option key={opt.value} value={opt.value} className="bg-[#10121f] text-[#f3f4f8]">
              {opt.label}
            </option>
          ))}
        </select>
        <ChevronDown
          className="absolute right-2.5 top-1/2 -translate-y-1/2 w-4 h-4 text-[#6b7188] pointer-events-none"
          aria-hidden="true"
        />
      </div>
    </div>
  );
};

