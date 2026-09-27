import React, { useState, useEffect, useRef } from 'react';
import { Settings, Database, Bell, HardDrive, Users, Key, Plus, Trash2, Copy, Check, Loader2, AlertCircle, RefreshCw, ShieldCheck, ShieldAlert } from 'lucide-react';
import { useQuery, useMutation, useQueryClient } from '@tanstack/react-query';
import { authApi } from '../../lib/api';
import type { APIKey } from '../../lib/api';

const SECTIONS = ['General', 'Data Sources', 'Alerting', 'Retention', 'Team', 'API Keys'] as const;
type Section = typeof SECTIONS[number];

export const SettingsTab: React.FC = () => {
  const [activeSection, setActiveSection] = useState<Section>('General');

  return (
    <div className="flex h-full bg-black font-dm-sans rounded-lg border border-white/[0.08] overflow-hidden">
      {/* SIDEBAR */}
      <div className="w-64 bg-black border-r border-white/[0.08] flex flex-col p-4 gap-1 shrink-0">
        {SECTIONS.map(s => (
          <button
            key={s}
            onClick={() => setActiveSection(s)}
            className={`flex items-center gap-3 px-4 py-3 rounded-md text-sm transition-colors text-left focus:outline-none focus:ring-2 focus:ring-white/20 ${activeSection === s ? 'bg-white/10 text-white font-medium' : 'text-white/60 hover:text-white hover:bg-white/[0.05]'}`}
          >
            {s === 'General' && <Settings className="w-4 h-4" />}
            {s === 'Data Sources' && <Database className="w-4 h-4" />}
            {s === 'Alerting' && <Bell className="w-4 h-4" />}
            {s === 'Retention' && <HardDrive className="w-4 h-4" />}
            {s === 'Team' && <Users className="w-4 h-4" />}
            {s === 'API Keys' && <Key className="w-4 h-4" />}
            {s}
          </button>
        ))}
      </div>

      {/* CONTENT AREA */}
      <div className="flex-1 overflow-y-auto p-8 relative bg-white/[0.01]">
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

// --- Modals Base ---
const Modal: React.FC<{ title: string, onClose: () => void, children: React.ReactNode }> = ({ title, onClose, children }) => {
  const overlayRef = useRef<HTMLDivElement>(null);

  useEffect(() => {
    const handleEsc = (e: KeyboardEvent) => { if (e.key === 'Escape') onClose(); };
    window.addEventListener('keydown', handleEsc);
    return () => window.removeEventListener('keydown', handleEsc);
  }, [onClose]);

  return (
    <div
      ref={overlayRef}
      onClick={(e) => { if (e.target === overlayRef.current) onClose(); }}
      className="fixed inset-0 bg-black/80 backdrop-blur-sm z-50 flex items-center justify-center p-4 animate-in fade-in duration-200"
    >
      <div className="bg-[#111] border border-white/10 rounded-lg shadow-2xl w-full max-w-lg overflow-hidden flex flex-col animate-in zoom-in-95 duration-200">
        <div className="p-4 border-b border-white/10 flex justify-between items-center">
          <h3 className="font-semibold text-lg">{title}</h3>
          <button onClick={onClose} className="text-white/50 hover:text-white">&times;</button>
        </div>
        <div className="p-6 flex flex-col gap-4">
          {children}
        </div>
      </div>
    </div>
  );
};

// --- Section Components ---

const GeneralSettings = () => (
  <div className="max-w-2xl flex flex-col gap-8 animate-in fade-in">
    <div>
      <h2 className="text-2xl font-bold mb-6">General Settings</h2>
      <div className="flex flex-col gap-6">
        <div className="flex flex-col gap-2">
          <label className="text-sm text-white/70">Organization Name</label>
          <input type="text" defaultValue="Acme Corp" className="bg-black border border-white/20 rounded px-4 py-2 w-full max-w-md focus:border-blue-500 focus:outline-none" />
        </div>
        <div className="flex flex-col gap-2">
          <label className="text-sm text-white/70">Default Region</label>
          <select className="bg-black border border-white/20 rounded px-4 py-2 w-full max-w-md focus:border-blue-500 focus:outline-none appearance-none">
            <option>us-east-1</option>
            <option>eu-west-1</option>
            <option>ap-south-1</option>
          </select>
        </div>
        <div className="flex flex-col gap-2">
          <label className="text-sm text-white/70">Theme</label>
          <div className="flex gap-4">
            <label className="flex items-center gap-2 cursor-pointer">
              <input type="radio" name="theme" defaultChecked className="accent-blue-500" />
              <span>Dark</span>
            </label>
            <label className="flex items-center gap-2 opacity-50 cursor-not-allowed" title="Coming soon">
              <input type="radio" name="theme" disabled className="accent-blue-500" />
              <span>Light</span>
            </label>
          </div>
        </div>
        <button className="bg-white text-black px-6 py-2 rounded font-medium self-start hover:bg-white/80 transition-colors">Save Changes</button>
      </div>
    </div>
  </div>
);

const DataSourcesSettings = () => {
  const [modalOpen, setModalOpen] = useState(false);
  const [testState, setTestState] = useState<'idle' | 'testing' | 'success'>('idle');

  return (
    <div className="max-w-4xl flex flex-col gap-6 animate-in fade-in">
      <div className="flex justify-between items-center">
        <h2 className="text-2xl font-bold">Data Sources</h2>
        <button onClick={() => setModalOpen(true)} className="flex items-center gap-2 bg-white text-black px-4 py-2 rounded text-sm hover:bg-white/80 transition-colors">
          <Plus className="w-4 h-4" /> Add Source
        </button>
      </div>

      <div className="border border-white/10 rounded-lg overflow-hidden">
        <table className="w-full text-left text-sm">
          <thead className="bg-white/[0.02] border-b border-white/10">
            <tr>
              <th className="px-4 py-3 text-white/50 font-medium">Name</th>
              <th className="px-4 py-3 text-white/50 font-medium">Type</th>
              <th className="px-4 py-3 text-white/50 font-medium">Status</th>
              <th className="px-4 py-3 text-white/50 font-medium text-right">Actions</th>
            </tr>
          </thead>
          <tbody className="divide-y divide-white/5">
            <tr>
              <td className="px-4 py-4 font-medium">Prod OTLP Collector</td>
              <td className="px-4 py-4">OTLP/gRPC</td>
              <td className="px-4 py-4"><span className="text-green-400 bg-green-500/10 px-2 py-1 rounded text-xs">Connected</span></td>
              <td className="px-4 py-4 text-right"><button className="text-red-400 hover:text-red-300"><Trash2 className="w-4 h-4 inline" /></button></td>
            </tr>
          </tbody>
        </table>
      </div>

      {modalOpen && (
        <Modal title="Add Data Source" onClose={() => setModalOpen(false)}>
          <div className="flex flex-col gap-4">
            <input type="text" placeholder="Name (e.g. Prod Collector)" className="bg-black border border-white/20 rounded px-4 py-2 w-full focus:border-blue-500 focus:outline-none text-sm" />
            <select className="bg-black border border-white/20 rounded px-4 py-2 w-full focus:border-blue-500 focus:outline-none text-sm appearance-none">
              <option>OTLP (gRPC)</option>
              <option>Jaeger</option>
              <option>Zipkin</option>
            </select>
            <input type="text" placeholder="Endpoint URL (e.g. grpc://...)" className="bg-black border border-white/20 rounded px-4 py-2 w-full focus:border-blue-500 focus:outline-none font-dm-mono text-sm" />

            <div className="flex items-center justify-between mt-4">
              <button
                onClick={() => { setTestState('testing'); setTimeout(() => setTestState('success'), 1000); }}
                className="text-sm px-4 py-2 bg-white/10 rounded hover:bg-white/20 transition-colors"
              >
                {testState === 'idle' ? 'Test Connection' : testState === 'testing' ? 'Testing...' : 'Success!'}
              </button>
              <div className="flex gap-3">
                <button onClick={() => setModalOpen(false)} className="text-sm text-white/50 hover:text-white">Cancel</button>
                <button onClick={() => setModalOpen(false)} disabled={testState !== 'success'} className="text-sm bg-white text-black px-4 py-2 rounded disabled:opacity-50">Add Source</button>
              </div>
            </div>
          </div>
        </Modal>
      )}
    </div>
  );
};

const AlertingSettings = () => (
  <div className="max-w-4xl flex flex-col gap-6 animate-in fade-in">
    <div className="flex justify-between items-center">
      <h2 className="text-2xl font-bold">Alert Rules</h2>
      <button className="flex items-center gap-2 bg-white text-black px-4 py-2 rounded text-sm hover:bg-white/80 transition-colors">
        <Plus className="w-4 h-4" /> Create Rule
      </button>
    </div>
    <div className="flex flex-col items-center justify-center py-20 border border-white/10 border-dashed rounded-lg text-white/40">
      <Bell className="w-12 h-12 mb-4 opacity-50" />
      <p>No alert rules configured yet.</p>
      <button className="mt-4 text-blue-400 hover:underline">Read the docs</button>
    </div>
  </div>
);

const RetentionSettings = () => (
  <div className="max-w-2xl flex flex-col gap-8 animate-in fade-in">
    <h2 className="text-2xl font-bold">Data Retention</h2>
    <div className="flex flex-col gap-8 bg-white/[0.02] p-6 rounded-lg border border-white/10">
      <div className="flex flex-col gap-2">
        <div className="flex justify-between text-sm">
          <label className="text-white/70">Raw Trace Retention</label>
          <span className="font-dm-mono text-blue-400">7 Days</span>
        </div>
        <input type="range" min="1" max="30" defaultValue="7" className="w-full accent-blue-500" />
        <span className="text-xs text-white/40">Estimated storage: ~1.2 TB / month</span>
      </div>

      <div className="flex flex-col gap-2">
        <div className="flex justify-between text-sm">
          <label className="text-white/70">Aggregated Metrics</label>
          <span className="font-dm-mono text-blue-400">90 Days</span>
        </div>
        <input type="range" min="30" max="365" defaultValue="90" className="w-full accent-blue-500" />
      </div>

      <button className="bg-white text-black px-6 py-2 rounded font-medium self-start mt-4">Update Retention Policies</button>
    </div>
  </div>
);

const TeamSettings = () => (
  <div className="max-w-4xl flex flex-col gap-6 animate-in fade-in">
    <div className="flex justify-between items-center">
      <h2 className="text-2xl font-bold">Team Members</h2>
      <button className="flex items-center gap-2 bg-white text-black px-4 py-2 rounded text-sm hover:bg-white/80 transition-colors">
        <Plus className="w-4 h-4" /> Invite Member
      </button>
    </div>

    <div className="border border-white/10 rounded-lg overflow-hidden">
      <table className="w-full text-left text-sm">
        <thead className="bg-white/[0.02] border-b border-white/10">
          <tr>
            <th className="px-4 py-3 text-white/50 font-medium">Name</th>
            <th className="px-4 py-3 text-white/50 font-medium">Email</th>
            <th className="px-4 py-3 text-white/50 font-medium">Role</th>
            <th className="px-4 py-3 text-white/50 font-medium text-right">Actions</th>
          </tr>
        </thead>
        <tbody className="divide-y divide-white/5">
          <tr>
            <td className="px-4 py-4 font-medium">Alice Engineer</td>
            <td className="px-4 py-4 text-white/70">alice@example.com</td>
            <td className="px-4 py-4"><span className="bg-blue-500/10 text-blue-400 px-2 py-1 rounded text-xs">Admin</span></td>
            <td className="px-4 py-4 text-right"></td>
          </tr>
          <tr>
            <td className="px-4 py-4 font-medium">Bob Developer</td>
            <td className="px-4 py-4 text-white/70">bob@example.com</td>
            <td className="px-4 py-4"><span className="bg-white/10 text-white/70 px-2 py-1 rounded text-xs">Editor</span></td>
            <td className="px-4 py-4 text-right"><button className="text-red-400 hover:text-red-300"><Trash2 className="w-4 h-4 inline" /></button></td>
          </tr>
        </tbody>
      </table>
    </div>
  </div>
);

const ApiKeysSettings = () => {
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
    <div className="max-w-4xl flex flex-col gap-6 animate-in fade-in">
      <div className="flex justify-between items-center">
        <div>
          <h2 className="text-2xl font-bold">API Keys</h2>
          <p className="text-sm text-white/50 mt-1">Manage ingestion credentials for OpenTelemetry collectors and SDKs.</p>
        </div>
        <div className="flex items-center gap-3">
          <button
            onClick={() => refetch()}
            disabled={isFetching}
            className="p-2 border border-white/10 rounded text-white/60 hover:text-white hover:bg-white/[0.05] transition-colors"
            title="Refresh keys"
          >
            <RefreshCw className={`w-4 h-4 ${isFetching ? 'animate-spin' : ''}`} />
          </button>
          <button
            onClick={handleOpenCreateModal}
            className="flex items-center gap-2 bg-white text-black px-4 py-2 rounded text-sm font-medium hover:bg-white/80 transition-colors shadow-sm"
          >
            <Plus className="w-4 h-4" /> Generate Key
          </button>
        </div>
      </div>

      {actionError && (
        <div className="bg-red-500/10 border border-red-500/20 text-red-400 p-3 rounded text-sm flex items-center gap-2">
          <AlertCircle className="w-4 h-4 shrink-0" />
          <span>{actionError}</span>
        </div>
      )}

      {isLoading ? (
        <div className="border border-white/10 rounded-lg p-12 flex flex-col items-center justify-center gap-3 text-white/40">
          <Loader2 className="w-6 h-6 animate-spin text-blue-400" />
          <span className="text-sm font-dm-mono">Loading API keys from PostgreSQL...</span>
        </div>
      ) : isError ? (
        <div className="border border-red-500/20 bg-red-500/5 rounded-lg p-8 flex flex-col items-center justify-center gap-3 text-red-400">
          <AlertCircle className="w-8 h-8" />
          <p className="text-sm font-medium">Failed to load API keys</p>
          <p className="text-xs text-white/50">{(error as Error)?.message || 'Database connection error'}</p>
          <button
            onClick={() => refetch()}
            className="mt-2 px-4 py-1.5 bg-white/10 hover:bg-white/20 rounded text-xs text-white transition-colors"
          >
            Try Again
          </button>
        </div>
      ) : keys.length === 0 ? (
        <div className="border border-white/10 border-dashed rounded-lg p-12 flex flex-col items-center justify-center gap-3 text-center">
          <Key className="w-10 h-10 text-white/20" />
          <p className="text-white/80 font-medium">No API keys created</p>
          <p className="text-xs text-white/40 max-w-sm">
            Generate an API key with the <code className="text-blue-400">dt_live_</code> prefix to authenticate telemetry sent to <code className="text-blue-400">/api/v1/spans</code>.
          </p>
          <button
            onClick={handleOpenCreateModal}
            className="mt-3 flex items-center gap-2 bg-white text-black px-4 py-2 rounded text-xs font-medium hover:bg-white/80 transition-colors"
          >
            <Plus className="w-3.5 h-3.5" /> Generate First Key
          </button>
        </div>
      ) : (
        <div className="border border-white/10 rounded-lg overflow-hidden">
          <table className="w-full text-left text-sm">
            <thead className="bg-white/[0.02] border-b border-white/10">
              <tr>
                <th className="px-4 py-3 text-white/50 font-medium">Key Name & Prefix</th>
                <th className="px-4 py-3 text-white/50 font-medium">Environment</th>
                <th className="px-4 py-3 text-white/50 font-medium">Status</th>
                <th className="px-4 py-3 text-white/50 font-medium">Created</th>
                <th className="px-4 py-3 text-white/50 font-medium text-right">Actions</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-white/5">
              {keys.map((k) => {
                const isRevoked = !!k.revoked_at;
                const isProdKey = k.prefix.startsWith('dt_live_');
                return (
                  <tr key={k.id} className={isRevoked ? 'opacity-50 bg-white/[0.005]' : 'hover:bg-white/[0.01]'}>
                    <td className="px-4 py-4">
                      <div className="flex flex-col">
                        <span className="font-medium text-white/90">{k.name}</span>
                        <span className="font-dm-mono text-xs text-blue-400/80">{k.prefix}••••••••</span>
                      </div>
                    </td>
                    <td className="px-4 py-4">
                      {isProdKey ? (
                        <span className="inline-flex items-center gap-1 bg-cyan-500/10 text-cyan-400 border border-cyan-500/20 px-2 py-0.5 rounded text-[11px] font-medium">
                          <ShieldCheck className="w-3 h-3" /> Production
                        </span>
                      ) : (
                        <span className="inline-flex items-center gap-1 bg-amber-500/10 text-amber-400 border border-amber-500/20 px-2 py-0.5 rounded text-[11px] font-medium">
                          <ShieldAlert className="w-3 h-3" /> Test
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-4">
                      {isRevoked ? (
                        <span className="bg-red-500/10 text-red-400 border border-red-500/20 px-2 py-0.5 rounded text-[11px]">
                          Revoked
                        </span>
                      ) : (
                        <span className="bg-emerald-500/10 text-emerald-400 border border-emerald-500/20 px-2 py-0.5 rounded text-[11px]">
                          Active
                        </span>
                      )}
                    </td>
                    <td className="px-4 py-4 text-white/60 text-xs font-dm-mono">
                      {new Date(k.created_at).toLocaleDateString(undefined, {
                        year: 'numeric',
                        month: 'short',
                        day: 'numeric',
                      })}
                    </td>
                    <td className="px-4 py-4 text-right">
                      {isRevoked ? (
                        <span className="text-xs text-white/30 italic">Revoked</span>
                      ) : (
                        <button
                          onClick={() => {
                            if (window.confirm(`Revoke API key "${k.name}"? Ingest clients using this key will immediately be rejected.`)) {
                              revokeMutation.mutate(k.id);
                            }
                          }}
                          disabled={revokingId === k.id}
                          className="text-red-400 hover:text-red-300 hover:bg-red-500/10 disabled:opacity-50 text-xs px-3 py-1 border border-red-500/20 rounded transition-colors inline-flex items-center gap-1"
                        >
                          {revokingId === k.id ? (
                            <>
                              <Loader2 className="w-3 h-3 animate-spin" /> Revoking...
                            </>
                          ) : (
                            'Revoke'
                          )}
                        </button>
                      )}
                    </td>
                  </tr>
                );
              })}
            </tbody>
          </table>
        </div>
      )}

      {modalOpen && (
        <Modal
          title={newGeneratedKey ? 'API Key Generated' : 'Create New API Key'}
          onClose={handleCloseModal}
        >
          {newGeneratedKey ? (
            <div className="flex flex-col gap-4">
              <div className="p-3 bg-amber-500/10 border border-amber-500/20 rounded text-amber-400 text-xs flex flex-col gap-1">
                <span className="font-semibold uppercase tracking-wider">⚠️ Important Secret Warning</span>
                <span>
                  Please copy this key now. For security reasons, the full secret key hash is stored irreversibly in PostgreSQL and cannot be retrieved again.
                </span>
              </div>

              <div className="flex flex-col gap-1.5">
                <label className="text-xs text-white/50 uppercase tracking-wider">Generated Ingestion Key</label>
                <div className="flex items-center gap-2">
                  <input
                    type="text"
                    readOnly
                    value={newGeneratedKey}
                    className="flex-1 bg-black border border-white/20 rounded px-4 py-2 font-dm-mono text-sm text-blue-400 focus:outline-none select-all"
                  />
                  <button
                    onClick={() => {
                      navigator.clipboard.writeText(newGeneratedKey);
                      setCopied(true);
                      setTimeout(() => setCopied(false), 2500);
                    }}
                    className="p-2 bg-white/10 rounded hover:bg-white/20 transition-colors text-white"
                    title="Copy to clipboard"
                  >
                    {copied ? <Check className="w-5 h-5 text-emerald-400" /> : <Copy className="w-5 h-5" />}
                  </button>
                </div>
              </div>

              <div className="p-3 bg-white/[0.02] border border-white/10 rounded text-xs font-dm-mono text-white/60">
                <p className="text-white/40 mb-1">// Usage with cURL:</p>
                <p className="text-emerald-400/90 break-all">
                  curl -X POST http://localhost:8080/api/v1/spans \<br />
                  &nbsp;&nbsp;-H &quot;X-API-Key: {newGeneratedKey}&quot;
                </p>
              </div>

              <button
                onClick={handleCloseModal}
                className="w-full mt-2 bg-white text-black py-2 rounded text-sm font-medium hover:bg-white/90 transition-colors"
              >
                I have copied the key
              </button>
            </div>
          ) : (
            <form
              onSubmit={(e) => {
                e.preventDefault();
                createMutation.mutate();
              }}
              className="flex flex-col gap-4"
            >
              <div className="flex flex-col gap-1.5">
                <label className="text-xs text-white/70">Key Name / Description</label>
                <input
                  type="text"
                  placeholder="e.g. Production OTel Gateway"
                  value={keyName}
                  onChange={(e) => setKeyName(e.target.value)}
                  className="bg-black border border-white/20 rounded px-4 py-2 text-sm text-white focus:border-blue-500 focus:outline-none"
                  autoFocus
                />
              </div>

              <div className="flex flex-col gap-1.5">
                <label className="text-xs text-white/70">Key Scope / Environment</label>
                <div className="grid grid-cols-2 gap-3">
                  <button
                    type="button"
                    onClick={() => setIsProd(true)}
                    className={`p-3 rounded border text-left flex flex-col gap-1 transition-all ${
                      isProd
                        ? 'border-blue-500 bg-blue-500/10 text-white'
                        : 'border-white/10 bg-white/[0.02] text-white/60 hover:border-white/20'
                    }`}
                  >
                    <span className="text-sm font-medium">Production</span>
                    <span className="text-[11px] font-dm-mono text-blue-400">dt_live_••••</span>
                  </button>
                  <button
                    type="button"
                    onClick={() => setIsProd(false)}
                    className={`p-3 rounded border text-left flex flex-col gap-1 transition-all ${
                      !isProd
                        ? 'border-blue-500 bg-blue-500/10 text-white'
                        : 'border-white/10 bg-white/[0.02] text-white/60 hover:border-white/20'
                    }`}
                  >
                    <span className="text-sm font-medium">Test / Staging</span>
                    <span className="text-[11px] font-dm-mono text-amber-400">dt_test_••••</span>
                  </button>
                </div>
              </div>

              {actionError && (
                <div className="bg-red-500/10 border border-red-500/20 text-red-400 p-2.5 rounded text-xs flex items-center gap-2">
                  <AlertCircle className="w-4 h-4 shrink-0" />
                  <span>{actionError}</span>
                </div>
              )}

              <div className="flex justify-end gap-3 mt-4">
                <button
                  type="button"
                  onClick={handleCloseModal}
                  className="px-4 py-2 rounded text-sm text-white/60 hover:text-white transition-colors"
                >
                  Cancel
                </button>
                <button
                  type="submit"
                  disabled={createMutation.isPending}
                  className="bg-white text-black px-5 py-2 rounded text-sm font-medium hover:bg-white/90 disabled:opacity-50 transition-colors inline-flex items-center gap-2"
                >
                  {createMutation.isPending ? (
                    <>
                      <Loader2 className="w-4 h-4 animate-spin" /> Generating...
                    </>
                  ) : (
                    'Generate Key'
                  )}
                </button>
              </div>
            </form>
          )}
        </Modal>
      )}
    </div>
  );
};
