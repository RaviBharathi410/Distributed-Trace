import React, { useState } from 'react';
import { NavLink, Outlet, useLocation, Link } from 'react-router-dom';

export const AppShell: React.FC = () => {
  const location = useLocation();
  const isDemoMode = import.meta.env.VITE_DEMO_MODE === 'true';
  const [selectedRegion, setSelectedRegion] = useState<'us-east-1' | 'eu-west-1'>('us-east-1');

  const navItems = [
    { name: 'overview', path: '/overview' },
    { name: 'traces', path: '/traces' },
    { name: 'service map', path: '/service-map' },
    { name: 'anomalies', path: '/anomalies' },
    { name: 'costs', path: '/costs' },
    { name: 'settings', path: '/settings' },
  ];

  return (
    <div className="min-h-screen flex flex-col bg-black text-[#f3f4f8] selection:bg-white/20 selection:text-white">
      {/* Top Floating Navbar matching Landing Page design */}
      <header className="sticky top-0 z-40 w-full px-6 md:px-10 pt-6 pb-2 flex flex-row items-center justify-between gap-4 bg-black/80 backdrop-blur-md">
        {/* Brand Logo Pill */}
        <Link
          to="/"
          className="bg-neutral-900/90 hover:bg-neutral-900 backdrop-blur rounded-full pl-4 pr-6 py-3 flex items-center gap-2 transition-colors focus:outline-none focus:ring-2 focus:ring-white focus:ring-offset-2 focus:ring-offset-black shrink-0"
          aria-label="distributedtrace"
        >
          <svg viewBox="0 0 256 256" className="h-5 w-5 fill-[#ffffff] shrink-0" aria-hidden="true">
            <path d="M128 192L128 256L64.5 256L32 223L0 192L0 128L64 128ZM256 192L256 256L192.5 256L160 223L128 192L128 128L192 128ZM128 64L128 128L64.5 128L32 95L0 64L0 0L64 0ZM256 64L256 128L192.5 128L160 95L128 64L128 0L192 0Z" />
          </svg>
          <span className="text-white text-sm font-normal tracking-tight">distributedtrace</span>
        </Link>

        {/* Center Nav Pills Capsule */}
        <nav
          className="flex bg-neutral-900/90 backdrop-blur rounded-full px-3 py-2 items-center overflow-x-auto no-scrollbar max-w-[55vw] lg:max-w-none"
          aria-label="Dashboard Navigation"
        >
          {navItems.map((item) => {
            const isActive = location.pathname.startsWith(item.path);
            return (
              <NavLink
                key={item.path}
                to={item.path}
                className={`text-sm px-4 md:px-5 py-2 rounded-full whitespace-nowrap transition-colors focus:outline-none focus:ring-2 focus:ring-white focus:ring-offset-2 focus:ring-offset-black ${
                  isActive
                    ? 'bg-white text-black font-medium shadow-sm'
                    : 'text-neutral-300 hover:text-white'
                }`}
              >
                {item.name}
              </NavLink>
            );
          })}
        </nav>

        {/* Right Actions (Region Pills Capsule & Landing Button) */}
        <div className="flex items-center gap-3 shrink-0">
          {/* Region selector pill */}
          <div className="hidden sm:flex bg-neutral-900/90 backdrop-blur rounded-full px-2 py-1.5 items-center gap-1">
            <button
              type="button"
              onClick={() => setSelectedRegion('us-east-1')}
              className={`px-3 py-1 rounded-full text-xs font-mono transition-all focus:outline-none ${
                selectedRegion === 'us-east-1'
                  ? 'bg-white text-black font-semibold'
                  : 'text-neutral-400 hover:text-white'
              }`}
            >
              us-east-1
            </button>
            <button
              type="button"
              onClick={() => setSelectedRegion('eu-west-1')}
              className={`px-3 py-1 rounded-full text-xs font-mono transition-all focus:outline-none ${
                selectedRegion === 'eu-west-1'
                  ? 'bg-white text-black font-semibold'
                  : 'text-neutral-400 hover:text-white'
              }`}
            >
              eu-west-1
            </button>
          </div>

          {isDemoMode && (
            <span
              className="hidden xl:inline-flex items-center gap-1.5 px-3 py-1.5 rounded-full text-xs font-medium bg-neutral-900/90 text-neutral-300 border border-white/10"
              title="Running with mock fixture demo dataset"
            >
              <span className="w-1.5 h-1.5 rounded-full bg-white animate-pulse" />
              demo
            </span>
          )}

          {/* Action Button styled identically to "get started" on the landing page */}
          <NavLink
            to="/"
            className="bg-white text-black text-sm px-6 py-3 rounded-full hover:bg-neutral-200 transition-colors font-medium whitespace-nowrap focus:outline-none focus:ring-2 focus:ring-white focus:ring-offset-2 focus:ring-offset-black"
          >
            landing
          </NavLink>
        </div>
      </header>

      {/* Main Page Area */}
      <main className="flex-1 flex flex-col w-full max-w-[1600px] mx-auto p-6 md:p-10 pt-4">
        <Outlet />
      </main>
    </div>
  );
};
