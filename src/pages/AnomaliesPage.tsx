import React from 'react';
import { AnomaliesTab } from '../components/dashboard/AnomaliesTab';

export const AnomaliesPage: React.FC = () => {
  return (
    <div className="flex flex-col gap-6 w-full animate-in fade-in duration-200">
      <AnomaliesTab />
    </div>
  );
};
