import React from 'react';
import { SettingsTab } from '../components/dashboard/SettingsTab';

export const SettingsPage: React.FC = () => {
  return (
    <div className="flex flex-col gap-6 w-full animate-in fade-in duration-200">
      <SettingsTab />
    </div>
  );
};
