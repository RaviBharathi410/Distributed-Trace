import React from 'react';
import { CostsTab } from '../components/dashboard/CostsTab';

export const CostsPage: React.FC = () => {
  return (
    <div className="flex flex-col gap-6 w-full animate-in fade-in duration-200">
      <CostsTab />
    </div>
  );
};
