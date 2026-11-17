import React, { useState, useEffect, useRef } from 'react';
import {
  SlidersHorizontal,
  Database,
  Bell,
  Clock,
  Users,
  Key,
  ChevronDown,
  Plus,
  Trash2,
  Copy,
  Check,
  Loader2,
  AlertCircle,
  RefreshCw,
  ShieldCheck,
  ShieldAlert,
} from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { authApi } from '../../lib/api';
import type { APIKey } from '../../lib/api';

const SECTIONS = ['General', 'Data Sources', 'Alerting', 'Retention', 'Team', 'API Keys'] as const;
type Section = typeof SECTIONS[number];

export const SettingsTab: React.FC = () => {
  const [activeSection, setActiveSection] = useState<Section>('General');

  return (
    <div className="flex flex-col lg:flex-row gap-6 text-[#f3f4f8] font-sans pb-10">
      {/* LEFT SUB-NAV SIDEBAR */}
      <div className="w-full lg:w-52 flex flex-row lg:flex-col gap-1.5 shrink-0 overflow-x-auto lg:overflow-visible pb-2 lg:pb-0">
        {SECTIONS.map((s) => {
          const isActive = activeSection === s;
          return (
            <button
              key={s}
              type="button"
              onClick={() => setActiveSection(s)}
              className={`flex items-center gap-3 px-4 py-2.5 rounded-[9999px] lg:rounded-[8px] text-[13px] font-medium transition-all text-left whitespace-nowrap focus:outline-none ${
                isActive
                  ? 'bg-[#7c6cf0] text-white shadow-md'
                  : 'text-[#9aa0b5] hover:text-[#f3f4f8] hover:bg-white/[0.04]'
              }`}
            >
              {s === 'General' && <SlidersHorizontal className="w-4 h-4 shrink-0" />}
              {s === 'Data Sources' && <Database className="w-4 h-4 shrink-0" />}
              {s === 'Alerting' && <Bell className="w-4 h-4 shrink-0" />}
              {s === 'Retention' && <Clock className="w-4 h-4 shrink-0" />}
              {s === 'Team' && <Users className="w-4 h-4 shrink-0" />}
              {s === 'API Keys' && <Key className="w-4 h-4 shrink-0" />}
              <span>{s}</span>
            </button>
          );
        })}
      </div>

      {/* RIGHT MAIN CONTENT AREA */}
      <div className="flex-1 min-w-0">
        {activeSection === 'General' && <GeneralSettings />}
        {activeSection === 'Data Sources' && <DataSourcesSettings />}
        {activeSection === 'Alerting' && <AlertingSettings />}
        {activeSection === 'Retention' && <RetentionSettings />}
        {activeSection === 'Team' && <TeamSettings />}
        {activeSection === 'API Keys' && <ApiKeysSettings />}
      </div>
    </div>
  );
};

// --- General Settings (Mockup Pixel-Exact Tile) ---
const GeneralSettings: React.FC = () => {
  const [orgName, setOrgName] = useState('Acme Corp');
  const [region, setRegion] = useState('us-east-1');
  const [theme, setTheme] = useState<'dark' | 'light'>('dark');
  const [savedNotice, setSavedNotice] = useState(false);

  const handleSave = () => {
    setSavedNotice(true);
    setTimeout(() => setSavedNotice(false), 2500);
  };

  const handleDiscard = () => {
    setOrgName('Acme Corp');
    setRegion('us-east-1');
    setTheme('dark');
  };

  return (
    <div className="relative rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] p-7 md:p-8 overflow-hidden shadow-xl">
      {/* Decorative subtle violet wave lines in header matching source mockup */}
      <div
        className="absolute right-4 top-4 w-72 md:w-96 h-28 pointer-events-none opacity-30 overflow-hidden"
        aria-hidden="true"
      >
        <svg viewBox="0 0 380 100" fill="none" className="w-full h-full text-[#7c6cf0]">
          <path
            d="M0 60 C 80 20, 160 90, 240 40 C 290 10, 330 50, 380 30"
            stroke="currentColor"
            strokeWidth="2"
            strokeLinecap="round"
            opacity="0.9"
          />
          <path
            d="M20 70 C 100 30, 180 95, 260 50 C 310 20, 350 60, 380 42"
            stroke="currentColor"
            strokeWidth="1.2"
            opacity="0.6"
          />
          <path
            d="M40 80 C 120 40, 200 100, 280 60 C 330 30, 360 70, 380 55"
            stroke="currentColor"
            strokeWidth="0.8"
            strokeDasharray="4 4"
            opacity="0.4"
          />
        </svg>
      </div>

      {/* Header Block */}
      <div className="relative z-10 mb-8">
        <h2 className="text-[22px] font-semibold text-[#f3f4f8] tracking-tight">
          General Settings
        </h2>
        <p className="text-[13px] text-[#9aa0b5] mt-1">
          Configure your organization, region, and appearance.
        </p>
      </div>

      {/* Form Fields */}
      <div className="relative z-10 flex flex-col gap-6 max-w-xl">
        {/* Field 1: Organization Name */}
        <div className="flex flex-col gap-2">
          <label className="text-[13px] font-medium text-[#9aa0b5]">
            Organization Name
          </label>
          <input
            type="text"
            value={orgName}
            onChange={(e) => setOrgName(e.target.value)}
            className="bg-[#0a0b14] border border-[rgba(255,255,255,0.12)] text-[#f3f4f8] text-[13px] rounded-[8px] px-4 py-2.5 w-full focus:outline-none focus:border-[#7c6cf0] transition-colors"
          />
        </div>

        {/* Field 2: Default Region */}
        <div className="flex flex-col gap-2">
          <label className="text-[13px] font-medium text-[#9aa0b5]">
            Default Region
          </label>
          <div className="relative">
            <select
              value={region}
              onChange={(e) => setRegion(e.target.value)}
              className="bg-[#0a0b14] border border-[rgba(255,255,255,0.12)] text-[#f3f4f8] text-[13px] rounded-[8px] px-4 py-2.5 w-full focus:outline-none focus:border-[#7c6cf0] appearance-none cursor-pointer transition-colors"
            >
              <option value="us-east-1">us-east-1</option>
              <option value="eu-west-1">eu-west-1</option>
              <option value="ap-south-1">ap-south-1</option>
            </select>
            <ChevronDown className="w-4 h-4 text-[#6b7188] absolute right-3.5 top-1/2 -translate-y-1/2 pointer-events-none" />
          </div>
        </div>

        {/* Field 3: Theme */}
        <div className="flex flex-col gap-2">
          <label className="text-[13px] font-medium text-[#9aa0b5]">
            Theme
          </label>
          <div className="flex items-center gap-6 mt-1">
            <button
              type="button"
              onClick={() => setTheme('dark')}
              className="flex items-center gap-2.5 cursor-pointer text-[13px] focus:outline-none"
            >
              <span
                className={`w-4 h-4 rounded-full border flex items-center justify-center transition-colors ${
                  theme === 'dark'
                    ? 'border-[#7c6cf0]'
                    : 'border-[rgba(255,255,255,0.25)]'
                }`}
              >
                {theme === 'dark' && <span className="w-2 h-2 rounded-full bg-[#7c6cf0]" />}
              </span>
              <span className={theme === 'dark' ? 'text-[#f3f4f8]' : 'text-[#9aa0b5]'}>
                Dark
              </span>
            </button>

            <button
              type="button"
              onClick={() => setTheme('light')}
              className="flex items-center gap-2.5 cursor-pointer text-[13px] focus:outline-none opacity-60 hover:opacity-100 transition-opacity"
            >
              <span
                className={`w-4 h-4 rounded-full border flex items-center justify-center transition-colors ${
                  theme === 'light'
                    ? 'border-[#7c6cf0]'
                    : 'border-[rgba(255,255,255,0.25)]'
                }`}
              >
                {theme === 'light' && <span className="w-2 h-2 rounded-full bg-[#7c6cf0]" />}
              </span>
              <span className={theme === 'light' ? 'text-[#f3f4f8]' : 'text-[#9aa0b5]'}>
                Light
              </span>
            </button>
          </div>
        </div>

        {/* Bottom Actions Bar */}
        <div className="flex items-center justify-end gap-3 pt-6 mt-2 border-t border-[rgba(255,255,255,0.06)]">
          {savedNotice && (
            <span className="text-[12px] text-[#34d399] flex items-center gap-1.5 mr-auto animate-in fade-in">
              <Check className="w-3.5 h-3.5" />
              Settings saved successfully
            </span>
          )}

          <button
            type="button"
            onClick={handleDiscard}
            className="px-5 py-2 rounded-[8px] text-[13px] font-medium text-[#9aa0b5] hover:text-[#f3f4f8] hover:bg-white/[0.04] transition-colors focus:outline-none"
          >
            Discard
          </button>
          <button
            type="button"
            onClick={handleSave}
            className="px-5 py-2 rounded-[8px] text-[13px] font-medium text-white bg-[#7c6cf0] hover:bg-[#6b5bd6] transition-colors shadow-sm focus:outline-none"
          >
            Save Changes
          </button>
        </div>
      </div>
    </div>
  );
};

// --- Modals Base ---
const Modal: React.FC<{ title: string; onClose: () => void; children: React.ReactNode }> = ({
  title,
  onClose,
  children,
}) => {
  const overlayRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => {
      if (e.key === 'Escape') onClose();
    };
    window.addEventListener('keydown', handleEsc);
    return () => window.removeEventListener('keydown', handleEsc);
  }, [onClose]);

  return (
    <div
      ref={overlayRef}
      onClick={(e) => {
        if (e.target === overlayRef.current) onClose();
      }}
      className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4 animate-in fade-in duration-200"
    >
      <div className="bg-[#10121f] border border-[rgba(255,255,255,0.12)] rounded-[14px] shadow-2xl w-full max-w-lg overflow-hidden flex flex-col animate-in zoom-in-95 duration-200">
        <div className="p-4 px-6 border-b border-[rgba(255,255,255,0.08)] flex justify-between items-center">
          <h3 className="font-semibold text-[15px] text-[#f3f4f8]">{title}</h3>
          <button
            onClick={onClose}
            className="text-[#9aa0b5] hover:text-[#f3f4f8] text-lg leading-none p-1"
          >
            &times;
          </button>
        </div>
        <div className="p-6 flex flex-col gap-4 text-[#f3f4f8]">{children}</div>
      </div>
    </div>
  );
};

// --- Data Sources Settings ---
const DataSourcesSettings: React.FC = () => {
  const [modalOpen, setModalOpen] = useState(false);
  const [testState, setTestState] = useState<'idle' | 'testing' | 'success'>('idle');

  return (
    <div className="rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] p-7 flex flex-col gap-6 animate-in fade-in">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-[20px] font-semibold text-[#f3f4f8]">Data Sources</h2>
          <p className="text-[13px] text-[#9aa0b5] mt-0.5">
            Ingestion endpoints for OpenTelemetry collectors and sidecars.
          </p>
        </div>
        <button
          onClick={() => setModalOpen(true)}
          className="flex items-center gap-2 bg-[#7c6cf0] hover:bg-[#6b5bd6] text-white px-4 py-2 rounded-[8px] text-[13px] font-medium transition-colors shadow-sm"
        >
          <Plus className="w-4 h-4" /> Add Source
        </button>
      </div>

      <div className="border border-[rgba(255,255,255,0.08)] rounded-[10px] overflow-hidden">
        <table className="w-full text-left text-[12px]">
          <thead className="bg-white/[0.02] border-b border-[rgba(255,255,255,0.08)] text-[#6b7188] font-mono uppercase text-[11px]">
            <tr>
              <th className="px-4 py-3 font-medium">Name</th>
              <th className="px-4 py-3 font-medium">Type</th>
              <th className="px-4 py-3 font-medium">Status</th>
              <th className="px-4 py-3 font-medium text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-[rgba(255,255,255,0.04)] font-mono">
            <tr>
              <td className="px-4 py-3.5 font-medium text-[#f3f4f8]">Prod OTLP Collector</td>
              <td className="px-4 py-3.5 text-[#9aa0b5]">OTLP / HTTP (Protobuf + Gzip)</td>
              <td className="px-4 py-3.5">
                <span className="text-[#34d399] bg-[rgba(52,211,153,0.12)] border border-[rgba(52,211,153,0.2)] px-2 py-0.5 rounded-[9999px] text-[11px]">
                  Connected
                </span>
              </td>
              <td className="px-4 py-3.5 text-right">
                <button className="text-[#f87171] hover:text-red-400 p-1">
                  <Trash2 className="w-4 h-4 inline" />
                </button>
              </td>
            </tr>
          </tbody>
        </table>
      </div>

      {modalOpen && (
        <Modal title="Add Data Source" onClose={() => setModalOpen(false)}>
          <div className="flex flex-col gap-4 text-[13px]">
            <input
              type="text"
              placeholder="Name (e.g. Prod Collector)"
              className="bg-[#0a0b14] border border-[rgba(255,255,255,0.12)] rounded-[8px] px-4 py-2.5 w-full focus:outline-none focus:border-[#7c6cf0] text-[#f3f4f8]"
            />
            <select className="bg-[#0a0b14] border border-[rgba(255,255,255,0.12)] rounded-[8px] px-4 py-2.5 w-full focus:outline-none focus:border-[#7c6cf0] text-[#f3f4f8]">
              <option>OTLP / HTTP (Protobuf)</option>
              <option>Jaeger HTTP</option>
              <option>Zipkin v2</option>
            </select>
            <input
              type="text"
              placeholder="Endpoint URL (e.g. http://localhost:8080/v1/traces)"
              className="bg-[#0a0b14] border border-[rgba(255,255,255,0.12)] rounded-[8px] px-4 py-2.5 w-full focus:outline-none focus:border-[#7c6cf0] font-mono text-[#f3f4f8]"
            />

            <div className="flex items-center justify-between mt-4 pt-4 border-t border-[rgba(255,255,255,0.08)]">
              <button
                type="button"
                onClick={() => {
                  setTestState('testing');
                  setTimeout(() => setTestState('success'), 800);
                }}
                className="text-[12px] px-4 py-2 bg-white/[0.05] hover:bg-white/[0.1] rounded-[8px] transition-colors text-[#9aa0b5] hover:text-[#f3f4f8]"
              >
                {testState === 'idle'
                  ? 'Test Connection'
                  : testState === 'testing'
                  ? 'Testing...'
                  : 'Connection Verified!'}
              </button>
              <div className="flex gap-2">
                <button
                  type="button"
                  onClick={() => setModalOpen(false)}
                  className="text-[12px] text-[#9aa0b5] hover:text-[#f3f4f8] px-3 py-2"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={() => setModalOpen(false)}
                  disabled={testState !== 'success'}
                  className="text-[12px] bg-[#7c6cf0] text-white px-4 py-2 rounded-[8px] font-medium disabled:opacity-50"
                >
                  Add Source
                </button>
              </div>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
};

// --- Alerting Settings ---
const AlertingSettings: React.FC = () => (
  <div className="rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] p-7 flex flex-col gap-6 animate-in fade-in">
    <div className="flex justify-between items-center">
      <div>
        <h2 className="text-[20px] font-semibold text-[#f3f4f8]">Alert Rules</h2>
        <p className="text-[13px] text-[#9aa0b5] mt-0.5">
          Configure real-time notifications for latency regressions and error spikes.
        </p>
      </div>
      <button className="flex items-center gap-2 bg-[#7c6cf0] hover:bg-[#6b5bd6] text-white px-4 py-2 rounded-[8px] text-[13px] font-medium transition-colors shadow-sm">
        <Plus className="w-4 h-4" /> Create Rule
      </button>
    </div>
    <div className="flex flex-col items-center justify-center py-16 border border-dashed border-[rgba(255,255,255,0.1)] rounded-[10px] text-[#9aa0b5]">
      <Bell className="w-10 h-10 mb-3 text-[#6b7188] opacity-50" />
      <p className="text-[13px] text-[#f3f4f8]">No alert rules configured yet.</p>
      <span className="text-[12px] text-[#6b7188] mt-1">
        Configure threshold-based anomaly notifications via Slack, PagerDuty, or Webhook.
      </span>
    </div>
  </div>
);

// --- Retention Settings ---
const RetentionSettings: React.FC = () => (
  <div className="rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] p-7 flex flex-col gap-6 max-w-2xl animate-in fade-in">
    <div>
      <h2 className="text-[20px] font-semibold text-[#f3f4f8]">Data Retention</h2>
      <p className="text-[13px] text-[#9aa0b5] mt-0.5">
        ClickHouse columnar MergeTree automated TTL policies.
      </p>
    </div>

    <div className="flex flex-col gap-6 bg-[rgba(255,255,255,0.02)] p-6 rounded-[10px] border border-[rgba(255,255,255,0.06)]">
      <div className="flex flex-col gap-2">
        <div className="flex justify-between text-[13px]">
          <label className="text-[#f3f4f8] font-medium">Raw Trace Retention</label>
          <span className="font-mono text-[#7c6cf0] font-semibold">14 Days (TTL Enforced)</span>
        </div>
        <input
          type="range"
          min="1"
          max="30"
          defaultValue="14"
          className="w-full accent-[#7c6cf0]"
        />
        <span className="text-[11px] text-[#6b7188]">
          Non-error spans automatically pruned after 14 days; error spans retained for 30 days.
        </span>
      </div>

      <div className="flex flex-col gap-2">
        <div className="flex justify-between text-[13px]">
          <label className="text-[#f3f4f8] font-medium">Aggregated Metrics & Anomalies</label>
          <span className="font-mono text-[#7c6cf0] font-semibold">90 Days</span>
        </div>
        <input
          type="range"
          min="30"
          max="365"
          defaultValue="90"
          className="w-full accent-[#7c6cf0]"
        />
      </div>

      <button className="bg-[#7c6cf0] hover:bg-[#6b5bd6] text-white px-5 py-2 rounded-[8px] text-[13px] font-medium self-start mt-2 transition-colors shadow-sm">
        Update Retention Policies
      </button>
    </div>
  </div>
);

// --- Team Settings ---
const TeamSettings: React.FC = () => (
  <div className="rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] p-7 flex flex-col gap-6 animate-in fade-in">
    <div className="flex justify-between items-center">
      <div>
        <h2 className="text-[20px] font-semibold text-[#f3f4f8]">Team Members</h2>
        <p className="text-[13px] text-[#9aa0b5] mt-0.5">
          Manage tenant user accounts and role-based access control.
        </p>
      </div>
      <button className="flex items-center gap-2 bg-[#7c6cf0] hover:bg-[#6b5bd6] text-white px-4 py-2 rounded-[8px] text-[13px] font-medium transition-colors shadow-sm">
        <Plus className="w-4 h-4" /> Invite Member
      </button>
    </div>

    <div className="border border-[rgba(255,255,255,0.08)] rounded-[10px] overflow-hidden">
      <table className="w-full text-left text-[12px]">
        <thead className="bg-white/[0.02] border-b border-[rgba(255,255,255,0.08)] text-[#6b7188] font-mono uppercase text-[11px]">
          <tr>
            <th className="px-4 py-3 font-medium">Name</th>
            <th className="px-4 py-3 font-medium">Email</th>
            <th className="px-4 py-3 font-medium">Role</th>
            <th className="px-4 py-3 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-[rgba(255,255,255,0.04)] font-mono">
          <tr>
            <td className="px-4 py-3.5 font-medium text-[#f3f4f8]">Admin User</td>
            <td className="px-4 py-3.5 text-[#9aa0b5]">admin@acme.com</td>
            <td className="px-4 py-3.5">
              <span className="bg-[rgba(124,108,240,0.15)] text-[#b08cff] border border-[rgba(124,108,240,0.3)] px-2 py-0.5 rounded-[9999px] text-[11px]">
                Owner
              </span>
            </td>
            <td className="px-4 py-3.5 text-right"></td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
);

// --- API Keys Settings (Full Backend Hook Integration) ---
const ApiKeysSettings: React.FC = () => {
  const queryClient = useQueryClient();
  const [modalOpen, setModalOpen] = useState(false);
  const [keyName, setKeyName] = useState('');
  const [isProd, setIsProd] = useState(true);
  const [newGeneratedKey, setNewGeneratedKey] = useState<string | null>(null);
  const [copied, setCopied] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);
  const [revokingId, setRevokingId] = useState<string | null>(null);

  const {
    data: keys = [],
    isLoading,
    isError,
    error,
    refetch,
    isFetching,
  } = useQuery<APIKey[]>({
    queryKey: ['apiKeys'],
    queryFn: () => authApi.listApiKeys(),
  });

  const createMutation = useMutation({
    mutationFn: async () => {
      setActionError(null);
      const name = keyName.trim() || 'Ingest Key';
      return authApi.createApiKey(name, isProd);
    },
    onSuccess: (data) => {
      queryClient.invalidateQueries({ queryKey: ['apiKeys'] });
      setNewGeneratedKey(data.key);
      setKeyName('');
    },
    onError: (err: Error) => {
      setActionError(err.message || 'Failed to generate API key');
    },
  });

  const revokeMutation = useMutation({
    mutationFn: async (id: string) => {
      setActionError(null);
      setRevokingId(id);
      return authApi.revokeApiKey(id);
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: ['apiKeys'] });
      setRevokingId(null);
    },
    onError: (err: Error) => {
      setActionError(err.message || 'Failed to revoke API key');
      setRevokingId(null);
    },
  });

  const handleOpenCreateModal = () => {
    setNewGeneratedKey(null);
    setKeyName('');
    setIsProd(true);
    setActionError(null);
    setCopied(false);
    setModalOpen(true);
  };

  const handleCloseModal = () => {
    setModalOpen(false);
    setNewGeneratedKey(null);
    setActionError(null);
  };

  return (
    <div className="rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] p-7 flex flex-col gap-6 animate-in fade-in">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-[20px] font-semibold text-[#f3f4f8]">API Keys</h2>
          <p className="text-[13px] text-[#9aa0b5] mt-0.5">
            Manage ingestion credentials for OpenTelemetry collectors and SDKs.
          </p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => refetch()}
            disabled={isFetching}
            className="p-2 border border-[rgba(255,255,255,0.1)] rounded-[8px] text-[#9aa0b5] hover:text-[#f3f4f8] hover:bg-white/[0.04] transition-colors"
            title="Refresh keys"
          >
            <RefreshCw className={`w-4 h-4 ${isFetching ? 'animate-spin text-[#7c6cf0]' : ''}`} />
          </button>
          <button
            onClick={handleOpenCreateModal}
            className="flex items-center gap-2 bg-[#7c6cf0] hover:bg-[#6b5bd6] text-white px-4 py-2 rounded-[8px] text-[13px] font-medium transition-colors shadow-sm"
          >
            <Plus className="w-4 h-4" /> Generate Key
          </button>
        </div>
      </div>

      {actionError && (
        <div className="p-3 bg-red-500/10 border border-red-500/20 rounded-[8px] text-red-300 text-[12px] flex items-center gap-2">
          <AlertCircle className="w-4 h-4 shrink-0 text-red-400" />
          <span>{actionError}</span>
        </div>
      )}

      {isError && (
        <div className="p-4 bg-red-500/10 border border-red-500/20 rounded-[8px] text-red-300 text-[13px] flex items-center justify-between">
          <div className="flex items-center gap-2">
            <ShieldAlert className="w-5 h-5 shrink-0 text-red-400" />
            <span>Failed to load API keys: {(error as Error)?.message || 'Unknown error'}</span>
          </div>
          <button
            onClick={() => refetch()}
            className="px-3 py-1 bg-white/[0.05] hover:bg-white/[0.1] rounded text-[12px] text-white font-mono"
          >
            Retry
          </button>
        </div>
      )}

      <div className="border border-[rgba(255,255,255,0.08)] rounded-[10px] overflow-hidden">
        {isLoading ? (
          <div className="p-8 flex items-center justify-center text-[#9aa0b5] text-[13px] gap-2">
            <Loader2 className="w-4 h-4 animate-spin text-[#7c6cf0]" />
            Loading API keys...
          </div>
        ) : keys.length === 0 ? (
          <div className="p-12 text-center text-[#6b7188] text-[13px]">
            No API keys found. Generate a key to begin ingesting OTLP traces.
          </div>
        ) : (
          <table className="w-full text-left text-[12px]">
            <thead className="bg-white/[0.02] border-b border-[rgba(255,255,255,0.08)] text-[#6b7188] font-mono uppercase text-[11px]">
              <tr>
                <th className="px-4 py-3 font-medium">Name</th>
                <th className="px-4 py-3 font-medium">Prefix</th>
                <th className="px-4 py-3 font-medium">Environment</th>
                <th className="px-4 py-3 font-medium">Created</th>
                <th className="px-4 py-3 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[rgba(255,255,255,0.04)] font-mono">
              {keys.map((k) => (
                <tr key={k.id} className="hover:bg-white/[0.02] transition-colors">
                  <td className="px-4 py-3.5 font-medium text-[#f3f4f8]">{k.name}</td>
                  <td className="px-4 py-3.5 text-[#9aa0b5]">{k.key_prefix}...</td>
                  <td className="px-4 py-3.5">
                    <span
                      className={`px-2 py-0.5 rounded-[9999px] text-[11px] ${
                        k.is_prod
                          ? 'bg-[rgba(52,211,153,0.12)] text-[#34d399] border border-[rgba(52,211,153,0.25)]'
                          : 'bg-white/[0.05] text-[#9aa0b5]'
                      }`}
                    >
                      {k.is_prod ? 'Production' : 'Development'}
                    </span>
                  </td>
                  <td className="px-4 py-3.5 text-[#6b7188]">
                    {new Date(k.created_at).toLocaleDateString()}
                  </td>
                  <td className="px-4 py-3.5 text-right">
                    <button
                      onClick={() => revokeMutation.mutate(k.id)}
                      disabled={revokingId === k.id}
                      className="text-[#f87171] hover:text-red-400 p-1 disabled:opacity-50 transition-colors"
                      title="Revoke Key"
                    >
                      {revokingId === k.id ? (
                        <Loader2 className="w-4 h-4 animate-spin inline" />
                      ) : (
                        <Trash2 className="w-4 h-4 inline" />
                      )}
                    </button>
                  </td>
                </tr>
              ))}
            </tbody>
          </table>
        )}
      </div>

      {/* Create Key Modal */}
      {modalOpen && (
        <Modal title="Generate New Ingestion Key" onClose={handleCloseModal}>
          {newGeneratedKey ? (
            <div className="flex flex-col gap-4 text-[13px]">
              <div className="p-3 bg-[rgba(52,211,153,0.1)] border border-[rgba(52,211,153,0.2)] rounded-[8px] flex items-center gap-2 text-[#34d399]">
                <ShieldCheck className="w-5 h-5 shrink-0" />
                <span>Save this key now. It will not be shown again.</span>
              </div>
              <div className="flex items-center gap-2 bg-[#0a0b14] border border-[rgba(255,255,255,0.12)] rounded-[8px] p-3 font-mono text-[12px] text-[#f3f4f8]">
                <span className="flex-1 select-all break-all">{newGeneratedKey}</span>
                <button
                  onClick={() => {
                    navigator.clipboard.writeText(newGeneratedKey);
                    setCopied(true);
                    setTimeout(() => setCopied(false), 2000);
                  }}
                  className="px-3 py-1.5 bg-[#7c6cf0] hover:bg-[#6b5bd6] text-white rounded-[6px] text-[11px] font-sans flex items-center gap-1.5 transition-colors"
                >
                  {copied ? <Check className="w-3.5 h-3.5" /> : <Copy className="w-3.5 h-3.5" />}
                  {copied ? 'Copied' : 'Copy'}
                </button>
              </div>
              <button
                onClick={handleCloseModal}
                className="w-full mt-2 py-2.5 bg-white/[0.05] hover:bg-white/[0.1] rounded-[8px] text-[13px] text-[#f3f4f8] transition-colors"
              >
                Done
              </button>
            </div>
          ) : (
            <div className="flex flex-col gap-4 text-[13px]">
              <div className="flex flex-col gap-1.5">
                <label className="text-[12px] text-[#9aa0b5]">Key Identifier</label>
                <input
                  type="text"
                  placeholder="e.g. Production OTel Ingest"
                  value={keyName}
                  onChange={(e) => setKeyName(e.target.value)}
                  className="bg-[#0a0b14] border border-[rgba(255,255,255,0.12)] rounded-[8px] px-4 py-2.5 focus:outline-none focus:border-[#7c6cf0] text-[#f3f4f8]"
                />
              </div>

              <div className="flex items-center gap-4 mt-1">
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    name="env"
                    checked={isProd}
                    onChange={() => setIsProd(true)}
                    className="accent-[#7c6cf0]"
                  />
                  <span>Production</span>
                </label>
                <label className="flex items-center gap-2 cursor-pointer">
                  <input
                    type="radio"
                    name="env"
                    checked={!isProd}
                    onChange={() => setIsProd(false)}
                    className="accent-[#7c6cf0]"
                  />
                  <span>Development</span>
                </label>
              </div>

              <div className="flex justify-end gap-2 mt-4 pt-4 border-t border-[rgba(255,255,255,0.08)]">
                <button
                  type="button"
                  onClick={handleCloseModal}
                  className="px-4 py-2 text-[#9aa0b5] hover:text-[#f3f4f8] text-[12px]"
                >
                  Cancel
                </button>
                <button
                  type="button"
                  onClick={() => createMutation.mutate()}
                  disabled={createMutation.isPending}
                  className="bg-[#7c6cf0] hover:bg-[#6b5bd6] text-white px-5 py-2 rounded-[8px] font-medium text-[12px] flex items-center gap-2 shadow-sm disabled:opacity-50"
                >
                  {createMutation.isPending && <Loader2 className="w-3.5 h-3.5 animate-spin" />}
                  Generate
                </button>
              </div>
            </div>
          )}
        </Modal>
      )}
    </div>
  );
};
