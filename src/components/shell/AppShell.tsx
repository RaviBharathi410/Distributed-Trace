import React from 'react';
import { NavLink, Outlet, useLocation } from 'react-router-dom';
import {
  LayoutGrid,
  Activity,
  Share2,
  TriangleAlert,
  Layers,
  Settings,
  Sparkles,
} from 'lucide-react';

export const AppShell: React.FC = () => {
  const location = useLocation();
  const isDemoMode = import.meta.env.VITE_DEMO_MODE === 'true';

  const navItems = [
    { name: 'Overview', path: '/overview', icon: LayoutGrid },
    { name: 'Traces', path: '/traces', icon: Activity },
    { name: 'Service Map', path: '/service-map', icon: Share2 },
    { name: 'Anomalies', path: '/anomalies', icon: TriangleAlert },
    { name: 'Costs', path: '/costs', icon: Layers },
    { name: 'Settings', path: '/settings', icon: Settings },
  ];

  return (
    <div className="min-h-screen flex flex-col bg-[#0a0b14] text-[#f3f4f8] selection:bg-[#7c6cf0]/30 selection:text-white">
      {/* Top Navigation Bar */}
      <header className="sticky top-0 z-40 h-14 border-b border-[rgba(255,255,255,0.07)] bg-[#0a0b14]/90 backdrop-blur-md px-6 flex items-center justify-between">
        {/* Brand / Logo */}
        <div className="flex items-center gap-6">
          <NavLink
            to="/overview"
            className="flex items-center gap-2.5 group focus:outline-none focus:ring-2 focus:ring-[#7c6cf0] rounded-md px-1 py-0.5"
            aria-label="DistributedTrace Home"
          >
            {/* Custom SVG logo mark */}
            <div className="w-7 h-7 rounded-lg bg-gradient-to-br from-[#7c6cf0]/30 to-[#9b5cf6]/10 border border-[#7c6cf0]/40 flex items-center justify-center shadow-sm group-hover:border-[#7c6cf0]/80 transition-colors">
              <svg
                width="18"
                height="18"
                viewBox="0 0 24 24"
                fill="none"
                xmlns="http://www.w3.org/2000/svg"
                className="text-[#7c6cf0]"
              >
                <circle cx="6" cy="6" r="2.5" fill="#7c6cf0" />
                <circle cx="18" cy="8" r="2.5" fill="#7c6cf0" />
                <circle cx="12" cy="18" r="2.5" fill="#34d399" />
                <path
                  d="M7.8 7.2L16.2 7.8M7.2 8L10.8 16M16.8 9.8L13.2 16.2"
                  stroke="currentColor"
                  strokeWidth="1.75"
                  strokeLinecap="round"
                />
              </svg>
            </div>
            <span className="font-semibold text-[15px] tracking-tight text-[#f3f4f8] flex items-center">
              <span>distributed</span>
              <span className="text-[#9aa0b5] font-normal">trace</span>
            </span>
          </NavLink>
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

        {/* Right Actions & Utilities */}
        <div className="flex items-center gap-3">
          {isDemoMode && (
            <span
              className="inline-flex items-center gap-1.5 px-2.5 py-1 rounded-full text-[11px] font-medium bg-amber-500/10 text-amber-300 border border-amber-500/30"
              title="Running with mock fixture demo dataset"
            >
              <span className="w-1.5 h-1.5 rounded-full bg-amber-400 animate-pulse" />
              Demo data
            </span>
          )}

          <NavLink
            to="/_kit"
            className="text-[12px] text-[#6b7188] hover:text-[#9aa0b5] px-2 py-1 rounded transition-colors inline-flex items-center gap-1 border border-transparent hover:border-white/10"
            title="Design System Component Gallery"
          >
            <Sparkles className="w-3 h-3 text-[#7c6cf0]" />
            <span className="hidden md:inline">Kit</span>
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

