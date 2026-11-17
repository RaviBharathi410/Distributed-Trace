import React, { useState, useRef, useEffect } from 'react';
import { useNavigate } from 'react-router-dom';

// --- CountUp Utility ---
const CountUp: React.FC<{ end: number; format?: string; duration?: number }> = ({
  end,
  format = '{n}',
  duration = 1500,
}) => {
  const [val, setVal] = useState(0);
  useEffect(() => {
    let start = 0;
    const step = (ts: number) => {
      if (!start) start = ts;
      const p = Math.min((ts - start) / duration, 1);
      setVal(Math.floor(p * end));
      if (p < 1) requestAnimationFrame(step);
    };
    requestAnimationFrame(step);
  }, [end, duration]);
  return <>{format.replace('{n}', String(val))}</>;
};

// --- Scroll Reveal Hook ---
const useScrollReveal = () => {
  const ref = useRef<HTMLElement>(null);
  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const obs = new IntersectionObserver(
      ([entry]) => {
        if (entry.isIntersecting) el.classList.add('visible');
      },
      { threshold: 0.15 }
    );
    obs.observe(el);
    return () => obs.disconnect();
  }, []);
  return ref;
};

// --- Platform Section ---
const PlatformSection: React.FC = () => {
  const ref = useScrollReveal();
  return (
    <section
      id="platform"
      ref={ref as React.RefObject<HTMLElement>}
      className="scroll-reveal py-32 px-6 md:px-20 border-t border-white/[0.06]"
    >
      <div className="max-w-6xl mx-auto">
        <h2 className="text-5xl md:text-7xl font-medium tracking-tight mb-6">platform</h2>
        <p className="text-white/60 max-w-2xl text-lg mb-16">
          A unified observability layer built for distributed systems. Ingest, correlate, and analyze traces across your entire microservices architecture in real-time.
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {[
            {
              title: 'Distributed Tracing',
              desc: 'End-to-end trace correlation across services with sub-millisecond precision. OpenTelemetry native.',
            },
            {
              title: 'Anomaly Detection',
              desc: 'Deterministic Welford baseline tracking and trace critical-path graph analysis implemented in native Go.',
            },
            {
              title: 'Real-time Pipeline',
              desc: 'Go + Kafka ingestion pipeline with ClickHouse storage. 50K spans/sec sustained throughput.',
            },
          ].map((item, i) => (
            <div
              key={i}
              className="p-8 border border-white/[0.08] rounded-lg hover:border-white/20 transition-colors group"
            >
              <div className="w-2 h-2 rounded-full bg-white mb-6 group-hover:scale-150 transition-transform" />
              <h3 className="text-xl font-medium mb-3">{item.title}</h3>
              <p className="text-white/50 text-sm leading-relaxed">{item.desc}</p>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

// --- Proof Section ---
const ProofSection: React.FC = () => {
  const ref = useScrollReveal();
  return (
    <section
      ref={ref as React.RefObject<HTMLElement>}
      className="scroll-reveal py-32 px-6 md:px-20 border-t border-white/[0.06]"
    >
      <div className="max-w-6xl mx-auto">
        <h2 className="text-5xl md:text-7xl font-medium tracking-tight mb-6">proof</h2>
        <p className="text-white/60 max-w-2xl text-lg mb-16">
          Production-grade architecture designed for predictable cost, deterministic root-cause attribution, and high-throughput OTLP ingest.
        </p>
        <div className="grid grid-cols-2 md:grid-cols-4 gap-8">
          {[
            { num: '99.99%', label: 'uptime SLA' },
            { num: '0.14ms', label: 'causal traversal per 100-span trace' },
            { num: '50K', label: 'spans/sec ingest' },
            { num: '3+', label: 'geographic regions' },
          ].map((item, i) => (
            <div key={i} className="proof-mockup p-8 border border-white/[0.08] rounded-lg text-center">
              <div className="text-3xl md:text-4xl font-medium mb-2">{item.num}</div>
              <div className="text-white/50 text-sm">{item.label}</div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

// --- How It Works ---
const HowItWorksSection: React.FC = () => {
  const ref = useScrollReveal();
  return (
    <section
      ref={ref as React.RefObject<HTMLElement>}
      className="scroll-reveal py-32 px-6 md:px-20 border-t border-white/[0.06]"
    >
      <div className="max-w-6xl mx-auto">
        <h2 className="text-5xl md:text-7xl font-medium tracking-tight mb-16">how it works</h2>
        <div className="flex flex-col gap-12">
          {[
            {
              step: '01',
              title: 'Instrument',
              desc: 'Add OpenTelemetry SDK to your services. Auto-instrumentation available for Go, Python, Java, and Node.js.',
            },
            {
              step: '02',
              title: 'Ingest',
              desc: 'Spans flow through our Go + Kafka pipeline, deduplicated and enriched with service topology metadata.',
            },
            {
              step: '03',
              title: 'Analyze',
              desc: 'ClickHouse stores and indexes traces. The Go ingestion engine evaluates Welford baselines and critical-path bottleneck traversal post-ingest.',
            },
            {
              step: '04',
              title: 'Alert',
              desc: 'Get notified via Slack, PagerDuty, or webhooks when anomalies are detected. Full root cause analysis included.',
            },
          ].map((item, i) => (
            <div key={i} className="flex gap-8 items-start group">
              <span className="text-6xl font-medium text-white/10 group-hover:text-white/30 transition-colors font-mono shrink-0 w-24">
                {item.step}
              </span>
              <div className="border-l border-white/[0.08] pl-8 group-hover:border-white/20 transition-colors">
                <h3 className="text-2xl font-medium mb-2">{item.title}</h3>
                <p className="text-white/50 max-w-lg leading-relaxed">{item.desc}</p>
              </div>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

// --- Pricing Section ---
const PricingSection: React.FC<{ onGetStarted: () => void }> = ({ onGetStarted }) => {
  const ref = useScrollReveal();
  return (
    <section
      id="pricing"
      ref={ref as React.RefObject<HTMLElement>}
      className="scroll-reveal py-32 px-6 md:px-20 border-t border-white/[0.06]"
    >
      <div className="max-w-6xl mx-auto">
        <h2 className="text-5xl md:text-7xl font-medium tracking-tight mb-6">pricing</h2>
        <p className="text-white/60 max-w-2xl text-lg mb-16">
          Simple, transparent pricing. No per-seat charges. Pay for what you ingest.
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {[
            {
              name: 'Starter',
              price: '$0',
              desc: 'For hobby projects',
              features: ['1M spans/month', '24h retention', '1 data source', 'Community support'],
            },
            {
              name: 'Pro',
              price: '$99',
              desc: 'For growing teams',
              features: ['100M spans/month', '30-day retention', 'Unlimited sources', 'Slack alerts', 'API access'],
              popular: true,
            },
            {
              name: 'Enterprise',
              price: 'Custom',
              desc: 'For large organizations',
              features: ['Unlimited spans', '1-year retention', 'SSO & RBAC', 'Dedicated support', 'On-prem option'],
            },
          ].map((plan, i) => (
            <div
              key={i}
              className={`p-8 border rounded-lg flex flex-col ${
                plan.popular ? 'border-white/40 bg-white/[0.03]' : 'border-white/[0.08]'
              }`}
            >
              {plan.popular && (
                <span className="text-xs uppercase tracking-widest text-white/60 mb-4">Most Popular</span>
              )}
              <h3 className="text-2xl font-medium">{plan.name}</h3>
              <div className="text-4xl font-medium mt-2 mb-1">
                {plan.price}
                <span className="text-lg text-white/40">{plan.price !== 'Custom' ? '/mo' : ''}</span>
              </div>
              <p className="text-white/50 text-sm mb-8">{plan.desc}</p>
              <ul className="flex flex-col gap-3 mb-8 flex-1">
                {plan.features.map((f, j) => (
                  <li key={j} className="flex items-center gap-2 text-sm text-white/70">
                    <div className="w-1 h-1 rounded-full bg-white/40" />
                    {f}
                  </li>
                ))}
              </ul>
              <button
                onClick={onGetStarted}
                className={`w-full py-3 rounded-full text-sm font-medium transition-colors ${
                  plan.popular ? 'bg-white text-black hover:bg-neutral-200' : 'bg-white/[0.06] text-white hover:bg-white/10'
                }`}
              >
                {plan.price === 'Custom' ? 'Contact Sales' : 'Get Started'}
              </button>
            </div>
          ))}
        </div>
      </div>
    </section>
  );
};

// --- Company Section ---
const CompanySection: React.FC = () => {
  const ref = useScrollReveal();
  return (
    <section
      id="company"
      ref={ref as React.RefObject<HTMLElement>}
      className="scroll-reveal py-32 px-6 md:px-20 border-t border-white/[0.06]"
    >
      <div className="max-w-6xl mx-auto">
        <h2 className="text-5xl md:text-7xl font-medium tracking-tight mb-6">company</h2>
        <p className="text-white/60 max-w-2xl text-lg mb-16">
          Built by engineers who lived through production outages and wished for better tooling.
        </p>
        <div className="grid grid-cols-1 md:grid-cols-2 gap-16">
          <div className="flex flex-col gap-6">
            <h3 className="text-2xl font-medium">Our Mission</h3>
            <p className="text-white/60 leading-relaxed">
              We believe observability should be accessible, fast, and intelligent. Every engineering team deserves
              the tooling that was previously only available to hyperscalers.
            </p>
          </div>
          <div className="flex flex-col gap-6">
            <h3 className="text-2xl font-medium">Open Source First</h3>
            <p className="text-white/60 leading-relaxed">
              Our core tracing pipeline is open source. We build on OpenTelemetry standards and contribute back
              to the community. No vendor lock-in, ever.
            </p>
          </div>
        </div>
      </div>
    </section>
  );
};

// --- Support Section ---
const SupportSection: React.FC = () => {
  const ref = useScrollReveal();
  return (
    <section
      id="support"
      ref={ref as React.RefObject<HTMLElement>}
      className="scroll-reveal py-32 px-6 md:px-20 border-t border-white/[0.06]"
    >
      <div className="max-w-6xl mx-auto text-center">
        <h2 className="text-5xl md:text-7xl font-medium tracking-tight mb-6">support</h2>
        <p className="text-white/60 max-w-2xl mx-auto text-lg mb-16">
          We're here to help you get the most out of your observability stack. Multiple channels, one goal: your success.
        </p>
        <div className="grid grid-cols-1 md:grid-cols-3 gap-8">
          {[
            {
              title: 'Documentation',
              desc: 'Comprehensive guides, API references, and tutorials to get you started quickly.',
            },
            {
              title: 'Community',
              desc: 'Join our Discord server with 5,000+ engineers sharing best practices and helping each other.',
            },
            {
              title: 'Enterprise Support',
              desc: 'Dedicated account manager, 24/7 on-call support, and custom SLAs for mission-critical workloads.',
            },
          ].map((item, i) => (
            <div
              key={i}
              className="p-8 border border-white/[0.08] rounded-lg hover:border-white/20 transition-colors text-left"
            >
              <h3 className="text-xl font-medium mb-3">{item.title}</h3>
              <p className="text-white/50 text-sm leading-relaxed">{item.desc}</p>
            </div>
          ))}
        </div>
      </div>

      {/* FOOTER */}
      <div className="mt-32 border-t border-white/[0.06] pt-8 flex flex-col md:flex-row items-center justify-between text-white/30 text-sm max-w-6xl mx-auto">
        <span>© 2026 distributedtrace. All rights reserved.</span>
        <div className="flex gap-6 mt-4 md:mt-0">
          <a href="#" className="hover:text-white/60 transition-colors">
            Privacy
          </a>
          <a href="#" className="hover:text-white/60 transition-colors">
            Terms
          </a>
          <a href="#" className="hover:text-white/60 transition-colors">
            Status
          </a>
        </div>
      </div>
    </section>
  );
};

// --- LandingPage Main View ---
export const LandingPage: React.FC = () => {
  const navigate = useNavigate();
  const heroRef = useRef<HTMLElement>(null);
  const [heroMouse, setHeroMouse] = useState({ x: 0, y: 0 });

  const handleGetStarted = () => {
    navigate('/overview');
  };

  useEffect(() => {
    const handleMouseMove = (e: MouseEvent) => {
      if (!heroRef.current) return;
      const { innerWidth, innerHeight } = window;
      const x = (e.clientX / innerWidth) * 2 - 1;
      const y = (e.clientY / innerHeight) * 2 - 1;
      setHeroMouse({ x, y });
    };
    window.addEventListener('mousemove', handleMouseMove);
    return () => window.removeEventListener('mousemove', handleMouseMove);
  }, []);

  return (
    <div className="flex flex-col bg-black text-white w-full">
      <style
        dangerouslySetInnerHTML={{
          __html: `
        @keyframes dash-entry { 0% { opacity: 0; } 100% { opacity: 1; } }
        @keyframes pulse-dot { 0%, 100% { opacity: 1; } 50% { opacity: 0.4; } }
        @keyframes scroll-reveal-up {
          0% { opacity: 0; transform: translateY(48px); }
          100% { opacity: 1; transform: translateY(0); }
        }
        .scroll-reveal {
          opacity: 0;
          transform: translateY(48px);
          transition: opacity 0.7s cubic-bezier(0.16, 1, 0.3, 1), transform 0.7s cubic-bezier(0.16, 1, 0.3, 1);
        }
        .scroll-reveal.visible {
          opacity: 1;
          transform: translateY(0);
        }
        .proof-mockup {
          background-image: repeating-linear-gradient(
            -45deg,
            rgba(255,255,255,0.06),
            rgba(255,255,255,0.06) 1px,
            transparent 1px,
            transparent 6px
          );
        }
        .trace-row:nth-child(odd) {
          background: rgba(255,255,255,0.02);
        }
        .pricing-card {
          background-image: repeating-linear-gradient(
            45deg,
            rgba(255,255,255,0.2),
            rgba(255,255,255,0.2) 4px,
            transparent 4px,
            transparent 8px
          );
        }
        @keyframes fadeSlideLeft {
          0% { transform: translateX(-40px); opacity: 0; }
          100% { transform: translateX(0); opacity: 1; }
        }
        @keyframes fadeSlideRight {
          0% { transform: translateX(40px); opacity: 0; }
          100% { transform: translateX(0); opacity: 1; }
        }
        @keyframes fadeIn {
          0% { opacity: 0; }
          100% { opacity: 1; }
        }
        @media (prefers-reduced-motion: reduce) {
          .hero-video-wrapper, .proof-mockup, .pricing-card, .anomaly-card {
            transform: none !important;
          }
          .hero-word {
            transform: none !important;
            transition: opacity 100ms !important;
          }
          .scroll-reveal {
            transform: none !important;
            transition: opacity 100ms !important;
          }
          .trace-row {
            transform: none !important;
          }
        }
      `,
        }}
      />

      {/* SECTION 1: HERO */}
      <section
        ref={heroRef}
        className="relative h-screen w-full overflow-hidden bg-black flex-shrink-0"
        style={{ '--mx': heroMouse.x, '--my': heroMouse.y } as React.CSSProperties}
      >
        <div
          className="absolute inset-0 w-full h-full hero-video-wrapper overflow-hidden flex items-center justify-center"
          style={{
            transform: 'translate(calc(var(--mx) * 5px), calc(var(--my) * 3px))',
            transition: 'transform 0.1s linear',
          }}
        >
          <img
            src="/bg.png"
            alt=""
            className="w-full h-full object-contain scale-[0.90]"
            style={{ transform: 'translateX(120px) translateY(30px)' }}
          />
        </div>

        {/* Soft edge vignette — keeps background crisp in center, fades only at borders */}
        <div
          className="absolute inset-0 z-[1] pointer-events-none"
          style={{
            background: `
            linear-gradient(to right, black 0%, transparent 12%, transparent 88%, black 100%),
            linear-gradient(to bottom, black 0%, transparent 15%, transparent 80%, black 100%)
          `,
          }}
        />

        {/* NAVBAR */}
        <div className="z-20 absolute top-0 left-0 right-0 px-6 md:px-10 pt-6 flex flex-row items-center justify-between gap-4">
          <div className="bg-neutral-900/90 backdrop-blur rounded-full pl-4 pr-6 py-3 flex items-center gap-2">
            <svg viewBox="0 0 256 256" className="h-5 w-5 fill-[#ffffff]">
              <path d="M128 192L128 256L64.5 256L32 223L0 192L0 128L64 128ZM256 192L256 256L192.5 256L160 223L128 192L128 128L192 128ZM128 64L128 128L64.5 128L32 95L0 64L0 0L64 0ZM256 64L256 128L192.5 128L160 95L128 64L128 0L192 0Z" />
            </svg>
            <span className="text-white text-sm font-normal tracking-tight">distributedtrace</span>
          </div>

          <div className="hidden md:flex bg-neutral-900/90 backdrop-blur rounded-full px-3 py-2 items-center">
            {['platform', 'solutions', 'company', 'support'].map((link) => (
              <a
                key={link}
                href={`#${link}`}
                onClick={(e) => {
                  e.preventDefault();
                  document.getElementById(link)?.scrollIntoView({ behavior: 'smooth' });
                }}
                className="text-neutral-300 hover:text-white transition-colors text-sm px-5 py-2 rounded-full focus:outline-none focus:ring-2 focus:ring-white focus:ring-offset-2 focus:ring-offset-black"
              >
                {link}
              </a>
            ))}
          </div>

          <button
            onClick={handleGetStarted}
            className="bg-white text-black text-sm font-normal rounded-full px-6 py-3 hover:bg-neutral-200 transition-colors active:scale-97 focus:outline-none focus:ring-2 focus:ring-white focus:ring-offset-2 focus:ring-offset-black"
          >
            get started
          </button>
        </div>

        {/* HERO CONTENT */}
        <div className="relative h-full w-full z-10 pointer-events-none">
          <div
            className="absolute left-[4vw] top-[45vh] -translate-y-1/2 flex flex-col select-none"
            style={{
              textShadow: '0 4px 40px rgba(0,0,0,0.8)',
              transform: 'translate(calc(var(--mx) * 5px), calc(var(--my) * 3px))',
              transition: 'transform 0.1s linear',
            }}
          >
            <h1 aria-label="trace every request" className="flex flex-col gap-[20px] leading-[0.86] tracking-[-0.06em]">
              <span
                aria-hidden="true"
                className="hero-word block text-white/80 font-bold"
                style={{
                  fontSize: 'clamp(90px, 10vw, 110px)',
                  animation: 'fadeSlideLeft 700ms cubic-bezier(0.16, 1, 0.3, 1) forwards',
                  opacity: 0,
                }}
              >
                trace
              </span>

              <span
                aria-hidden="true"
                className="hero-word block text-white/80 font-bold"
                style={{
                  fontSize: 'clamp(90px, 10vw, 110px)',
                  animation: 'fadeSlideRight 700ms cubic-bezier(0.16, 1, 0.3, 1) 160ms forwards',
                  opacity: 0,
                }}
              >
                every
              </span>

              <span
                aria-hidden="true"
                className="hero-word block font-bold text-white/80"
                style={{
                  fontSize: 'clamp(90px, 10vw, 110px)',
                  animation: 'fadeSlideLeft 700ms cubic-bezier(0.16, 1, 0.3, 1) 320ms forwards',
                  opacity: 0,
                }}
              >
                request
              </span>
            </h1>
          </div>

          {/* 50K stat — top right */}
          <div
            className="absolute flex items-center gap-4 pointer-events-auto select-none"
            style={{
              right: '6vw',
              top: '12vh',
              textShadow: '0 2px 20px rgba(0,0,0,1)',
              animation: 'fadeIn 800ms ease 400ms both',
              transform: 'translate(calc(var(--mx) * 5px), calc(var(--my) * 3px))',
              transition: 'transform 0.1s linear',
            }}
          >
            <div className="h-[1px] w-16 bg-white/30 rotate-[-15deg] mt-6" />
            <div className="flex flex-col">
              <span
                className="text-white font-medium"
                style={{ fontSize: 'clamp(28px, 3vw, 42px)', lineHeight: 1, letterSpacing: '-0.02em' }}
              >
                <CountUp end={50} format="{n}K" duration={1500} />
              </span>
              <span className="text-white/60 text-[10px] mt-1">spans/sec ingest rate</span>
            </div>
          </div>

          {/* Bottom gradient fade */}
          <div className="pointer-events-none absolute bottom-0 left-0 right-0 h-40 bg-gradient-to-b from-transparent to-black" />

          {/* 10TB stat — bottom left */}
          <div
            className="absolute flex items-center gap-4 pointer-events-auto select-none"
            style={{
              left: '6vw',
              bottom: '12vh',
              textShadow: '0 2px 20px rgba(0,0,0,1)',
              animation: 'fadeIn 800ms ease 700ms both',
              transform: 'translate(calc(var(--mx) * 5px), calc(var(--my) * 3px))',
              transition: 'transform 0.1s linear',
            }}
          >
            <div className="flex flex-col">
              <span
                className="text-white font-medium"
                style={{ fontSize: 'clamp(28px, 3vw, 42px)', lineHeight: 1, letterSpacing: '-0.02em' }}
              >
                <CountUp end={10} format="{n}TB" duration={1800} />
              </span>
              <span className="text-white/60 text-[10px] mt-1">compressed trace storage</span>
            </div>
            <div className="h-[1px] w-16 bg-white/30 rotate-[-25deg] mb-8" />
          </div>
        </div>
      </section>

      {/* SECTION 2: PLATFORM */}
      <PlatformSection />

      {/* SECTION 3: SOLUTIONS */}
      <div id="solutions">
        <ProofSection />
        <HowItWorksSection />
      </div>

      {/* SECTION 4: PRICING */}
      <PricingSection onGetStarted={handleGetStarted} />

      {/* SECTION 5: COMPANY */}
      <CompanySection />

      {/* SECTION 6: SUPPORT */}
      <SupportSection />
    </div>
  );
};
