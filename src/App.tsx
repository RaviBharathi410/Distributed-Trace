import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { AppShell } from './components/shell/AppShell';
import { LandingPage } from './pages/LandingPage';
import { OverviewPage } from './pages/OverviewPage';
import { TracesPage } from './pages/TracesPage';
import { ServiceMapPage } from './pages/ServiceMapPage';
import { AnomaliesPage } from './pages/AnomaliesPage';
import { CostsPage } from './pages/CostsPage';
import { SettingsPage } from './pages/SettingsPage';
import { ComponentKitPage } from './pages/ComponentKitPage';

function App() {
  return (
    <Routes>
      {/* Pitch-Black Public Landing Page */}
      <Route path="/" element={<LandingPage />} />
      <Route path="/landing" element={<LandingPage />} />

      {/* Authenticated Observability App Shell */}
      <Route element={<AppShell />}>
        <Route path="/overview" element={<OverviewPage />} />
        <Route path="/traces" element={<TracesPage />} />
        <Route path="/service-map" element={<ServiceMapPage />} />
        <Route path="/anomalies" element={<AnomaliesPage />} />
        <Route path="/costs" element={<CostsPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/_kit" element={<ComponentKitPage />} />
      </Route>

      {/* Fallback */}
      <Route path="*" element={<Navigate to="/" replace />} />
    </Routes>
  );
}

export default App;
