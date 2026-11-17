import React, { useState } from 'react';
import { NavLink, Outlet, useLocation } from 'react-router-dom';
import {
  LayoutGrid,
  Activity,
  Share2,
  TriangleAlert,
  Layers,
  Settings,
  Sparkles,
  Home,
} from 'lucide-react';
import { BrandLogo } from '../common/BrandLogo';

export const AppShell: React.FC = () => {
  const location = useLocation();
  const isDemoMode = import.meta.env.VITE_DEMO_MODE === 'true';
  const [selectedRegion, setSelectedRegion] = useState<'us-east-1' | 'eu-west-1'>('us-east-1');

  const navItems = [
    { name: 'Overview', path: '/overview', icon: LayoutGrid },
    { name: 'Traces', path: '/traces', icon: Activity },
    { name: 'Service Map', path: '/service-map', icon: Share2 },
    { name: 'Anomalies', path: '/anomalies', icon: TriangleAlert },
    { name: 'Costs', path: '/costs', icon: Layers },
    { name: 'Settings', path: '/settings', icon: Settings },
  ];

  return (
    <div className="min-h-screen flex flex-col bg-black text-[#f3f4f8] selection:bg-[#7c6cf0]/30 selection:text-white">
      {/* Top Navigation Bar */}
      <header className="sticky top-0 z-40 h-14 border-b border-white/[0.08] bg-black/95 backdrop-blur-md px-6 flex items-center justify-between">
        {/* Brand / Logo */}
        <div className="flex items-center gap-6">
          <BrandLogo to="/" size="md" />
        </div>

        {/* Center Nav Pills */}
        <nav className="flex items-center gap-1" aria-label="Main Navigation">
          {navItems.map((item) => {
            const Icon = item.icon;
            const isActive = location.pathname.startsWith(item.path);
            return (
              <NavLink
                key={item.path}
                to={item.path}
                className={`h-8 px-3.5 rounded-full text-[13px] font-medium inline-flex items-center gap-2 transition-all duration-150 ${
                  isActive
                    ? 'bg-[#7c6cf0]/20 border border-[#7c6cf0]/40 text-white shadow-sm'
                    : 'text-[#9aa0b5] hover:text-[#f3f4f8] hover:bg-white/[0.04] border border-transparent'
                }`}
              >
                <Icon className={`w-3.5 h-3.5 ${isActive ? 'text-[#7c6cf0]' : 'text-[#6b7188]'}`} />
                <span>{item.name}</span>
              </NavLink>
            );
          })}
        </nav>

        {/* Right Actions & Utilities (Region Pills matching reference image) */}
        <div className="flex items-center gap-2.5">
          {/* Region selector pills matching mockup */}
          <div className="hidden sm:flex items-center gap-1.5 bg-[#0a0a0d] p-0.5 rounded-full border border-white/[0.08]">
            <button
              type="button"
              onClick={() => setSelectedRegion('us-east-1')}
              className={`px-2.5 py-0.5 rounded-full text-[11px] font-mono transition-all ${
                selectedRegion === 'us-east-1'
                  ? 'bg-[#7c6cf0]/20 text-[#f3f4f8] border border-[#7c6cf0]/40 font-semibold'
                  : 'text-[#6b7188] hover:text-[#9aa0b5] border border-transparent'
              }`}
            >
              us-east-1
            </button>
            <button
              type="button"
              onClick={() => setSelectedRegion('eu-west-1')}
              className={`px-2.5 py-0.5 rounded-full text-[11px] font-mono transition-all ${
                selectedRegion === 'eu-west-1'
                  ? 'bg-[#7c6cf0]/20 text-[#f3f4f8] border border-[#7c6cf0]/40 font-semibold'
                  : 'text-[#6b7188] hover:text-[#9aa0b5] border border-transparent'
              }`}
            >
              eu-west-1
            </button>
          </div>

          {/* Quick link to Landing Page */}
          <NavLink
            to="/"
            className="text-[12px] text-[#6b7188] hover:text-[#9aa0b5] px-2 py-1 rounded transition-colors inline-flex items-center gap-1 border border-transparent hover:border-white/10"
            title="Return to Landing Page"
          >
            <Home className="w-3.5 h-3.5" />
            <span className="hidden lg:inline text-[11px]">Landing</span>
          </NavLink>

          {isDemoMode && (
            <span
              className="inline-flex items-center gap-1.5 px-2 py-0.5 rounded-full text-[10px] font-medium bg-blue-500/10 text-blue-300 border border-blue-500/30"
              title="Running with mock fixture demo dataset"
            >
              <span className="w-1.5 h-1.5 rounded-full bg-blue-400 animate-pulse" />
              Demo
            </span>
          )}

          <NavLink
            to="/_kit"
            className="text-[12px] text-[#6b7188] hover:text-[#9aa0b5] px-2 py-1 rounded transition-colors inline-flex items-center gap-1 border border-transparent hover:border-white/10"
            title="Design System Component Gallery"
          >
            <Sparkles className="w-3 h-3 text-[#7c6cf0]" />
            <span className="hidden md:inline text-[11px]">Kit</span>
          </NavLink>
        </div>
      </header>

      {/* Main Page Area */}
      <main className="flex-1 flex flex-col w-full max-w-[1600px] mx-auto p-6 md:p-8">
        <Outlet />
      </main>
    </div>
  );
};

