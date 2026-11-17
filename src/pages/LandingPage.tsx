import React, { useState } from 'react';
import { useNavigate, Link } from 'react-router-dom';
import {
  ArrowRight,
  Share2,
  Zap,
  ShieldCheck,
  Sparkles,
  Terminal,
  DollarSign,
} from 'lucide-react';
import { BrandLogo } from '../components/common/BrandLogo';
import { DEMO_KPIS, DEMO_LIVE_TRACES, DEMO_WATERFALL_SPANS } from '../lib/fixtures';

export const LandingPage: React.FC = () => {
  const navigate = useNavigate();
  const [activePreviewTab, setActivePreviewTab] = useState<'overview' | 'traces' | 'map' | 'anomalies' | 'costs'>('overview');

  return (
    <div className="min-h-screen bg-[#0a0b14] text-[#f3f4f8] selection:bg-[#7c6cf0]/30 selection:text-white flex flex-col relative overflow-x-hidden">
      {/* Background ambient radial violet glows */}
      <div
        className="absolute top-0 right-1/4 w-[600px] h-[600px] bg-[#7c6cf0]/10 rounded-full blur-[140px] pointer-events-none"
        aria-hidden="true"
      />
      <div
        className="absolute top-[800px] left-1/10 w-[500px] h-[500px] bg-[#9b5cf6]/08 rounded-full blur-[160px] pointer-events-none"
        aria-hidden="true"
      />

      {/* Top Navbar */}
      <header className="sticky top-0 z-50 h-16 border-b border-[rgba(255,255,255,0.07)] bg-[#0a0b14]/85 backdrop-blur-md px-6 md:px-12 flex items-center justify-between">
        <BrandLogo to="/" size="md" />

        <nav className="hidden md:flex items-center gap-7 text-[13px] font-medium text-[#9aa0b5]">
          <a href="#features" className="hover:text-[#f3f4f8] transition-colors">
            Features
          </a>
          <a href="#architecture" className="hover:text-[#f3f4f8] transition-colors">
            Architecture
          </a>
          <a href="#benchmarks" className="hover:text-[#f3f4f8] transition-colors">
            Benchmarks
          </a>
          <a href="#pricing" className="hover:text-[#f3f4f8] transition-colors">
            Economics
          </a>
          <Link to="/_kit" className="hover:text-[#f3f4f8] transition-colors flex items-center gap-1 text-[#b4a7f5]">
            <Sparkles className="w-3.5 h-3.5 text-[#7c6cf0]" />
            Component Kit
          </Link>
        </nav>

        <div className="flex items-center gap-3">
          <Link
            to="/overview"
            className="h-9 px-4 rounded-full bg-[#7c6cf0] hover:bg-[#6c5ce7] text-white text-[13px] font-semibold inline-flex items-center gap-1.5 shadow-[0_0_20px_rgba(124,108,240,0.35)] hover:shadow-[0_0_25px_rgba(124,108,240,0.5)] transition-all"
          >
            <span>Live Dashboard</span>
            <ArrowRight className="w-4 h-4" />
          </Link>
        </div>
      </header>

      {/* Hero Section */}
      <section className="pt-20 pb-16 px-6 md:px-12 max-w-6xl mx-auto flex flex-col items-center text-center relative z-10">
        {/* Pill Badge */}
        <div className="inline-flex items-center gap-2 px-3.5 py-1.5 rounded-full text-[12px] font-medium bg-[#7c6cf0]/10 border border-[#7c6cf0]/30 text-[#b4a7f5] mb-6 shadow-sm">
          <span className="w-2 h-2 rounded-full bg-[#7c6cf0] animate-pulse" />
          <span>Sub-Millisecond Causal AI • Columnar ClickHouse Tracing</span>
        </div>

        {/* Headline */}
        <h1 className="text-4xl sm:text-6xl lg:text-7xl font-bold tracking-tight text-[#f3f4f8] leading-[1.08] max-w-4xl">
          Microservices Observability <br />
          <span className="bg-gradient-to-r from-[#9aa0b5] via-[#f3f4f8] to-[#7c6cf0] bg-clip-text text-transparent">
            Without the Enterprise Tax.
          </span>
        </h1>

        {/* Subtitle */}
        <p className="mt-6 text-lg sm:text-xl text-[#9aa0b5] max-w-2xl leading-relaxed font-normal">
          Real-time distributed tracing, deterministic statistical anomaly detection, and automated causal root-cause analysis on ClickHouse and Kafka.
        </p>

        {/* CTA Buttons */}
        <div className="mt-8 flex flex-wrap items-center justify-center gap-4">
          <button
            onClick={() => navigate('/overview')}
            className="h-11 px-6 rounded-full bg-[#7c6cf0] hover:bg-[#6c5ce7] text-white text-[14px] font-semibold flex items-center gap-2 shadow-[0_0_25px_rgba(124,108,240,0.4)] hover:shadow-[0_0_35px_rgba(124,108,240,0.6)] transition-all cursor-pointer"
          >
            <span>Launch Live Dashboard</span>
            <ArrowRight className="w-4 h-4" />
          </button>

          <a
            href="https://github.com/RaviBharathi410/Distributed-Trace"
            target="_blank"
            rel="noopener noreferrer"
            className="h-11 px-5 rounded-full bg-white/[0.04] hover:bg-white/[0.08] text-[#f3f4f8] border border-[rgba(255,255,255,0.1)] text-[14px] font-medium flex items-center gap-2 transition-colors"
          >
            <Terminal className="w-4 h-4 text-[#9aa0b5]" />
            <span>docker compose up</span>
          </a>
        </div>

        {/* Live Interactive Dashboard Preview Frame */}
        <div className="mt-14 w-full rounded-[16px] bg-[#10121f] border border-[rgba(255,255,255,0.1)] shadow-[0_20px_60px_rgba(0,0,0,0.8)] overflow-hidden text-left relative group">
          {/* Window Title Bar */}
          <div className="h-11 bg-[#0d0f1a] border-b border-[rgba(255,255,255,0.07)] px-4 flex items-center justify-between select-none">
            <div className="flex items-center gap-2">
              <span className="w-3 h-3 rounded-full bg-[#ff5f56]/80" />
              <span className="w-3 h-3 rounded-full bg-[#ffbd2e]/80" />
              <span className="w-3 h-3 rounded-full bg-[#27c93f]/80" />
              <span className="ml-3 text-[12px] font-mono text-[#6b7188]">
                https://distributedtrace.internal/app/{activePreviewTab}
              </span>
            </div>

            {/* Interactive Screen Switcher in Preview */}
            <div className="flex items-center gap-1">
              {(
                [
                  { id: 'overview', label: 'Overview' },
                  { id: 'traces', label: 'Traces' },
                  { id: 'map', label: 'Service Map' },
                  { id: 'anomalies', label: 'Anomalies' },
                  { id: 'costs', label: 'Costs' },
                ] as const
              ).map((tab) => (
                <button
                  key={tab.id}
                  onClick={() => setActivePreviewTab(tab.id)}
                  className={`px-2.5 py-1 text-[11px] font-medium rounded-full transition-colors ${
                    activePreviewTab === tab.id
                      ? 'bg-[#7c6cf0]/25 text-[#f3f4f8] border border-[#7c6cf0]/40'
                      : 'text-[#6b7188] hover:text-[#9aa0b5]'
                  }`}
                >
                  {tab.label}
                </button>
              ))}
            </div>
          </div>

          {/* Interactive Screen Content Preview */}
          <div className="p-6 bg-[#0a0b14] flex flex-col gap-5">
            {activePreviewTab === 'overview' && (
              <div className="flex flex-col gap-4 animate-in fade-in duration-150">
                {/* 4 KPIs Mini */}
                <div className="grid grid-cols-2 md:grid-cols-4 gap-3">
                  <div className="p-3.5 rounded-[12px] bg-[#10121f] border border-[rgba(255,255,255,0.07)]">
                    <span className="text-[11px] text-[#6b7188]">Total Traces</span>
                    <div className="text-xl font-bold text-[#f3f4f8] mt-1 flex items-baseline gap-2">
                      {DEMO_KPIS.totalTraces}
                      <span className="text-[11px] text-[#34d399] font-medium">+14.2%</span>
                    </div>
                  </div>
                  <div className="p-3.5 rounded-[12px] bg-[#10121f] border border-[rgba(255,255,255,0.07)]">
                    <span className="text-[11px] text-[#6b7188]">p99 Latency</span>
                    <div className="text-xl font-bold text-[#f3f4f8] mt-1 flex items-baseline gap-2">
                      {DEMO_KPIS.p99Latency}
                      <span className="text-[11px] text-[#34d399] font-medium">-12.4%</span>
                    </div>
                  </div>
                  <div className="p-3.5 rounded-[12px] bg-[#10121f] border border-[rgba(255,255,255,0.07)]">
                    <span className="text-[11px] text-[#6b7188]">Active Services</span>
                    <div className="text-xl font-bold text-[#f3f4f8] mt-1">7</div>
                  </div>
                  <div className="p-3.5 rounded-[12px] bg-[#10121f] border border-[rgba(255,255,255,0.07)]">
                    <span className="text-[11px] text-[#6b7188]">Active Anomalies</span>
                    <div className="text-xl font-bold text-[#f5b942] mt-1">3</div>
                  </div>
                </div>

                {/* Mini Row 2: Live stream + Checkout Waterfall */}
                <div className="grid grid-cols-1 md:grid-cols-2 gap-4">
                  <div className="p-4 rounded-[12px] bg-[#10121f] border border-[rgba(255,255,255,0.07)]">
                    <div className="text-[12px] font-semibold text-[#f3f4f8] mb-2 flex items-center justify-between">
                      <span>Live Trace Stream</span>
                      <span className="text-[10px] text-[#34d399] font-mono">● LIVE 5s</span>
                    </div>
                    <div className="flex flex-col divide-y divide-white/[0.04] text-[12px]">
                      {DEMO_LIVE_TRACES.slice(0, 3).map((tr) => (
                        <div key={tr.trace_id} className="py-1.5 flex items-center justify-between font-mono">
                          <span className="text-[#4d8dff]">{tr.trace_id.slice(0, 8)}...</span>
                          <span className="text-[#f3f4f8] font-sans">{tr.root_service}</span>
                          <span className="text-[#9aa0b5]">{tr.duration_ms}ms</span>
                        </div>
                      ))}
                    </div>
                  </div>

                  <div className="p-4 rounded-[12px] bg-[#10121f] border border-[rgba(255,255,255,0.07)]">
                    <div className="text-[12px] font-semibold text-[#f3f4f8] mb-2">
                      Checkout Flow Waterfall
                    </div>
                    <div className="flex flex-col gap-2 mt-1">
                      {DEMO_WATERFALL_SPANS.slice(0, 3).map((sp) => (
                        <div key={sp.id} className="flex items-center justify-between text-[11px]">
                          <span className={`${sp.isHot ? 'text-[#b08cff] font-semibold' : 'text-[#9aa0b5]'}`}>
                            {sp.service} {sp.operation}
                          </span>
                          <span className="font-mono text-[#6b7188]">{sp.durationMs}ms</span>
                        </div>
                      ))}
                    </div>
                  </div>
                </div>
              </div>
            )}

            {activePreviewTab === 'traces' && (
              <div className="p-4 rounded-[12px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex flex-col gap-3 animate-in fade-in duration-150">
                <div className="flex items-center justify-between text-[12px]">
                  <span className="font-semibold text-[#f3f4f8]">Two-Pane Traces Explorer (1,248 traces)</span>
                  <span className="text-[11px] text-[#4d8dff]">Auto-synchronized with URL ?trace=id</span>
                </div>
                <div className="grid grid-cols-1 md:grid-cols-2 gap-3 text-[12px]">
                  <div className="p-3 rounded bg-[#0d0f1a] border border-white/5 font-mono text-xs flex flex-col gap-1.5">
                    <div className="text-[#9aa0b5] pb-1 border-b border-white/5">Trace Table (Left Pane)</div>
                    <div className="text-[#4d8dff]">3b157068d49748 • payment-svc • 450ms</div>
                    <div className="text-[#4d8dff]">1e55033d208d48 • user-service • 120ms</div>
                    <div className="text-[#4d8dff]">994256bd6a1e78 • inventory-svc • 850ms</div>
                  </div>
                  <div className="p-3 rounded bg-[#0d0f1a] border border-white/5 font-mono text-xs flex flex-col gap-1.5">
                    <div className="text-[#9aa0b5] pb-1 border-b border-white/5">Trace Detail (Right Pane)</div>
                    <div className="text-[#f3f4f8] font-bold">3b157068d49748 [payment-svc]</div>
                    <div className="text-[#b08cff]">↳ payment-svc /charge_card (250ms hot)</div>
                    <div className="text-[#6b7188]">Tabs: Timeline | Spans | Attributes | JSON</div>
                  </div>
                </div>
              </div>
            )}

            {activePreviewTab === 'map' && (
              <div className="p-8 rounded-[12px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex flex-col items-center justify-center text-center gap-3 animate-in fade-in duration-150">
                <Share2 className="w-10 h-10 text-[#7c6cf0]" />
                <div className="text-sm font-semibold text-[#f3f4f8]">Interactive Service Topology Graph</div>
                <p className="text-xs text-[#9aa0b5] max-w-md">
                  Real-time dependency mapping with bezier curved paths, flowing traffic dots, and live p99/error sparkline indicators.
                </p>
              </div>
            )}

            {activePreviewTab === 'anomalies' && (
              <div className="p-4 rounded-[12px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex flex-col gap-2.5 animate-in fade-in duration-150">
                <div className="text-xs font-semibold text-[#f3f4f8]">AI & Statistical Anomaly Feed</div>
                <div className="p-2.5 rounded bg-white/[0.02] border border-white/5 flex items-center justify-between text-xs">
                  <span className="px-2 py-0.5 rounded bg-[#9b5cf6]/20 text-[#b4a7f5] font-bold">CRITICAL</span>
                  <span className="font-semibold text-[#f3f4f8]">payment-svc • DB connection pool exhaustion</span>
                  <span className="text-[#34d399] font-mono">+450 ms</span>
                </div>
              </div>
            )}

            {activePreviewTab === 'costs' && (
              <div className="p-4 rounded-[12px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex flex-col gap-2.5 animate-in fade-in duration-150">
                <div className="flex items-center justify-between text-xs">
                  <span className="font-semibold text-[#f3f4f8]">Tenant Cost Attribution</span>
                  <span className="font-mono text-[#34d399]">$1,248.60 / mo</span>
                </div>
                <div className="text-[11px] text-[#9aa0b5]">
                  Enforces $0.01 per-incident pre-flight token ceiling and transparent ClickHouse infrastructure floor.
                </div>
              </div>
            )}

            {/* Launch Banner */}
            <div className="pt-2 flex items-center justify-between border-t border-[rgba(255,255,255,0.06)]">
              <span className="text-[12px] text-[#6b7188]">
                Interactive preview of the 6-screen DistributedTrace suite.
              </span>
              <button
                onClick={() => navigate('/overview')}
                className="text-[12px] font-semibold text-[#7c6cf0] hover:text-[#9b8cff] flex items-center gap-1 transition-colors cursor-pointer"
              >
                <span>Open in Live App</span>
                <ArrowRight className="w-3.5 h-3.5" />
              </button>
            </div>
          </div>
        </div>
      </section>

      {/* Feature Pillars Grid */}
      <section id="features" className="py-20 px-6 md:px-12 max-w-6xl mx-auto border-t border-[rgba(255,255,255,0.07)]">
        <div className="text-center max-w-2xl mx-auto mb-14">
          <h2 className="text-3xl font-bold text-[#f3f4f8] tracking-tight">
            Engineered for Massive Scale & Zero Hallucinations
          </h2>
          <p className="mt-3 text-[#9aa0b5] text-sm">
            Proven at 20M+ spans with deterministic columnar economics.
          </p>
        </div>

        <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-4 gap-5">
          <div className="p-6 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex flex-col justify-between hover:border-[rgba(255,255,255,0.14)] transition-all">
            <div>
              <div className="w-10 h-10 rounded-[10px] bg-[#7c6cf0]/15 text-[#7c6cf0] flex items-center justify-center mb-4 border border-[#7c6cf0]/30">
                <Zap className="w-5 h-5" />
              </div>
              <h3 className="text-[16px] font-semibold text-[#f3f4f8]">Columnar Engine</h3>
              <p className="mt-2 text-[13px] text-[#9aa0b5] leading-relaxed">
                ZSTD(3) map compression and binary FixedString(16) trace IDs reduce disk footprint to ~91.8 B/span with 10x savings.
              </p>
            </div>
            <div className="mt-5 text-[11px] font-mono text-[#6b7188] pt-3 border-t border-white/[0.04]">
              ClickHouse MergeTree
            </div>
          </div>

          <div className="p-6 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex flex-col justify-between hover:border-[rgba(255,255,255,0.14)] transition-all">
            <div>
              <div className="w-10 h-10 rounded-[10px] bg-[#34d399]/15 text-[#34d399] flex items-center justify-center mb-4 border border-[#34d399]/30">
                <ShieldCheck className="w-5 h-5" />
              </div>
              <h3 className="text-[16px] font-semibold text-[#f3f4f8]">$0 Hot-Path AI</h3>
              <p className="mt-2 text-[13px] text-[#9aa0b5] leading-relaxed">
                Deterministic Welford statistical baselines and causal graph tree analysis execute in 0.14ms with zero LLM API costs.
              </p>
            </div>
            <div className="mt-5 text-[11px] font-mono text-[#6b7188] pt-3 border-t border-white/[0.04]">
              Deterministic Causality
            </div>
          </div>

          <div className="p-6 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex flex-col justify-between hover:border-[rgba(255,255,255,0.14)] transition-all">
            <div>
              <div className="w-10 h-10 rounded-[10px] bg-[#4d8dff]/15 text-[#4d8dff] flex items-center justify-center mb-4 border border-[#4d8dff]/30">
                <Share2 className="w-5 h-5" />
              </div>
              <h3 className="text-[16px] font-semibold text-[#f3f4f8]">Live Topology</h3>
              <p className="mt-2 text-[13px] text-[#9aa0b5] leading-relaxed">
                Real-time service dependency mapping with animated particle traffic flows and per-node p99 latency sparklines.
              </p>
            </div>
            <div className="mt-5 text-[11px] font-mono text-[#6b7188] pt-3 border-t border-white/[0.04]">
              D3 & Canvas Graphs
            </div>
          </div>

          <div className="p-6 rounded-[14px] bg-[#10121f] border border-[rgba(255,255,255,0.07)] flex flex-col justify-between hover:border-[rgba(255,255,255,0.14)] transition-all">
            <div>
              <div className="w-10 h-10 rounded-[10px] bg-[#f5b942]/15 text-[#f5b942] flex items-center justify-center mb-4 border border-[#f5b942]/30">
                <DollarSign className="w-5 h-5" />
              </div>
              <h3 className="text-[16px] font-semibold text-[#f3f4f8]">Unit Economics</h3>
              <p className="mt-2 text-[13px] text-[#9aa0b5] leading-relaxed">
                Strict per-tenant cost attribution with pre-flight token spend caps (&le; $0.01/incident) preventing runaway billing.
              </p>
            </div>
            <div className="mt-5 text-[11px] font-mono text-[#6b7188] pt-3 border-t border-white/[0.04]">
              Enforced Ceilings
            </div>
          </div>
        </div>
      </section>

      {/* Benchmarks & Economics Comparison */}
      <section id="benchmarks" className="py-20 px-6 md:px-12 max-w-6xl mx-auto border-t border-[rgba(255,255,255,0.07)]">
        <div className="text-center max-w-2xl mx-auto mb-14">
          <h2 className="text-3xl font-bold text-[#f3f4f8] tracking-tight">
            Transparent Market Pricing Comparison
          </h2>
          <p className="mt-3 text-[#9aa0b5] text-sm">
            Empirically measured wire and storage unit economics compared to commercial trace providers.
          </p>
        </div>

        <div className="overflow-x-auto rounded-[14px] border border-[rgba(255,255,255,0.07)] bg-[#10121f]">
          <table className="w-full text-left border-collapse text-[13px]">
            <thead>
              <tr className="border-b border-[rgba(255,255,255,0.08)] bg-[#0d0f1a] text-[#6b7188] font-normal">
                <th className="p-4 font-normal">Platform</th>
                <th className="p-4 font-normal">Storage / Wire Architecture</th>
                <th className="p-4 font-normal">Hot-Path Detection Fee</th>
                <th className="p-4 font-normal text-right">Cost / 1M Spans</th>
              </tr>
            </thead>
            <tbody className="divide-y divide-[rgba(255,255,255,0.04)] font-mono">
              <tr className="bg-[#7c6cf0]/10 border-l-2 border-[#7c6cf0]">
                <td className="p-4 font-sans font-bold text-[#f3f4f8] flex items-center gap-2">
                  <span>DistributedTrace</span>
                  <span className="px-1.5 py-0.5 rounded text-[10px] bg-[#7c6cf0] text-white">This Project</span>
                </td>
                <td className="p-4 text-[#34d399]">ClickHouse ZSTD(3) Columnar (~91 B/span)</td>
                <td className="p-4 text-[#34d399] font-sans font-medium">$0 (Deterministic Welford)</td>
                <td className="p-4 text-right font-bold text-[#34d399]">$0.0104 / mo</td>
              </tr>
              <tr>
                <td className="p-4 font-sans text-[#f3f4f8]">SigNoz Cloud</td>
                <td className="p-4 text-[#9aa0b5]">$0.30 per GB ingested</td>
                <td className="p-4 text-[#9aa0b5] font-sans">Threshold alerts</td>
                <td className="p-4 text-right text-[#f3f4f8]">$0.045 / mo</td>
              </tr>
              <tr>
                <td className="p-4 font-sans text-[#f3f4f8]">Grafana Cloud Traces</td>
                <td className="p-4 text-[#9aa0b5]">$0.45 per GB (Tempo backend)</td>
                <td className="p-4 text-[#9aa0b5] font-sans">PromQL / TraceQL</td>
                <td className="p-4 text-right text-[#f3f4f8]">$0.068 / mo</td>
              </tr>
              <tr>
                <td className="p-4 font-sans text-[#f3f4f8]">Honeycomb</td>
                <td className="p-4 text-[#9aa0b5]">Proprietary columnar event store</td>
                <td className="p-4 text-[#9aa0b5] font-sans">BubbleUp correlation</td>
                <td className="p-4 text-right text-[#f3f4f8]">$1.50 - $3.00 / mo</td>
              </tr>
            </tbody>
          </table>
        </div>
      </section>

      {/* CTA Footer Banner */}
      <section className="py-20 px-6 md:px-12 max-w-5xl mx-auto text-center border-t border-[rgba(255,255,255,0.07)]">
        <div className="p-10 rounded-[20px] bg-gradient-to-b from-[#10121f] to-[#0a0b14] border border-[#7c6cf0]/30 shadow-[0_0_50px_rgba(124,108,240,0.15)] flex flex-col items-center">
          <h2 className="text-3xl sm:text-4xl font-bold text-[#f3f4f8] tracking-tight">
            Ready to explore the live telemetry suite?
          </h2>
          <p className="mt-3 text-[#9aa0b5] text-sm max-w-md">
            Click below to inspect real-time traces, service graphs, and causal anomaly explanations in dark mode.
          </p>

          <button
            onClick={() => navigate('/overview')}
            className="mt-8 h-12 px-8 rounded-full bg-[#7c6cf0] hover:bg-[#6c5ce7] text-white text-[15px] font-semibold flex items-center gap-2 shadow-[0_0_30px_rgba(124,108,240,0.4)] hover:shadow-[0_0_40px_rgba(124,108,240,0.6)] transition-all cursor-pointer"
          >
            <span>Enter Live Dashboard</span>
            <ArrowRight className="w-4 h-4" />
          </button>
        </div>
      </section>

      {/* Footer */}
      <footer className="mt-auto py-8 border-t border-[rgba(255,255,255,0.07)] px-6 md:px-12 flex flex-col sm:flex-row items-center justify-between gap-4 text-xs text-[#6b7188]">
        <div className="flex items-center gap-3">
          <BrandLogo to="/" size="sm" />
          <span>•</span>
          <span>Open-source distributed observability</span>
        </div>

        <div className="flex items-center gap-6">
          <Link to="/overview" className="hover:text-[#f3f4f8] transition-colors">
            Overview
          </Link>
          <Link to="/traces" className="hover:text-[#f3f4f8] transition-colors">
            Traces
          </Link>
          <Link to="/service-map" className="hover:text-[#f3f4f8] transition-colors">
            Service Map
          </Link>
          <Link to="/anomalies" className="hover:text-[#f3f4f8] transition-colors">
            Anomalies
          </Link>
          <Link to="/costs" className="hover:text-[#f3f4f8] transition-colors">
            Costs
          </Link>
          <Link to="/settings" className="hover:text-[#f3f4f8] transition-colors">
            Settings
          </Link>
        </div>
      </footer>
    </div>
  );
};
