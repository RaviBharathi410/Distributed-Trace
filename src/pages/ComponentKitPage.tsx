import React from 'react';

export const ComponentKitPage: React.FC = () => {
  return (
    <div className="flex flex-col gap-8 w-full animate-in fade-in duration-200">
      <div>
        <h1 className="text-[28px] font-semibold text-[#f3f4f8]">Design System Component Kit</h1>
        <p className="text-[13px] text-[#9aa0b5] mt-1">
          Interactive gallery previewing all tokens, typography, and shared design primitives.
        </p>
      </div>

      <div className="p-6 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)]">
        <h2 className="text-[14px] font-semibold text-[#f3f4f8] mb-4">Design Tokens</h2>
        <div className="grid grid-cols-2 sm:grid-cols-4 md:grid-cols-6 gap-4">
          {[
            { name: '--bg', color: '#0a0b14' },
            { name: '--card', color: '#10121f' },
            { name: '--accent', color: '#7c6cf0' },
            { name: '--link', color: '#4d8dff' },
            { name: '--good', color: '#34d399' },
            { name: '--sev-critical', color: '#9b5cf6' },
            { name: '--sev-high', color: '#f5b942' },
            { name: '--sev-medium', color: '#b4a7f5' },
            { name: '--donut-healthy', color: '#22c7d6' },
          ].map((token) => (
            <div key={token.name} className="flex flex-col gap-1.5">
              <div
                className="w-full h-12 rounded-[8px] border border-[rgba(255,255,255,0.1)] shadow-inner"
                style={{ backgroundColor: token.color }}
              />
              <span className="text-[11px] font-mono text-[#9aa0b5]">{token.name}</span>
              <span className="text-[10px] font-mono text-[#6b7188]">{token.color}</span>
            </div>
          ))}
        </div>
      </div>
    </div>
  );
};

