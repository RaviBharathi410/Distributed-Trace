import React from 'react';
import { TracesTab } from '../components/dashboard/TracesTab';

export const TracesPage: React.FC = () => {
  return (
    <div className="flex flex-col gap-6 w-full animate-in fade-in duration-200">
      <TracesTab />
    </div>
  );
};
