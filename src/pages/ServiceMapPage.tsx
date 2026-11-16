import React from 'react';
import { ServiceMapTab } from '../components/dashboard/ServiceMapTab';

export const ServiceMapPage: React.FC = () => {
  return (
    <div className="flex flex-col gap-6 w-full animate-in fade-in duration-200">
      <ServiceMapTab />
    </div>
  );
};
