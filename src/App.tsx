import React from 'react';
import { Routes, Route, Navigate } from 'react-router-dom';
import { AppShell } from './components/shell/AppShell';
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
      <Route element={<AppShell />}>
        <Route path="/" element={<Navigate to="/overview" replace />} />
        <Route path="/overview" element={<OverviewPage />} />
        <Route path="/traces" element={<TracesPage />} />
        <Route path="/service-map" element={<ServiceMapPage />} />
        <Route path="/anomalies" element={<AnomaliesPage />} />
        <Route path="/costs" element={<CostsPage />} />
        <Route path="/settings" element={<SettingsPage />} />
        <Route path="/_kit" element={<ComponentKitPage />} />
        <Route path="*" element={<Navigate to="/overview" replace />} />
      </Route>
    </Routes>
  );
}

export default App;
