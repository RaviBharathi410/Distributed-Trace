export default function LandingPage() {
  return (
    <div className="bg-neutral-900 text-white min-h-screen">
      <nav className="p-4 flex justify-between items-center border-b border-neutral-800">
        <div className="font-bold text-xl tracking-tight">Securify</div>
        <div className="space-x-6 text-sm text-neutral-400">
          <a href="#" className="hover:text-white">Platform</a>
          <a href="#" className="hover:text-white">Solutions</a>
          <a href="#" className="hover:text-white">Company</a>
        </div>
        <a href="/dashboard" className="bg-white text-black px-4 py-2 rounded-full text-sm font-medium hover:bg-neutral-200">
          Get Started
        </a>
      </nav>

      <main className="flex flex-col items-center justify-center pt-32 px-4 text-center">
        <h1 className="text-6xl md:text-8xl font-black tracking-tighter leading-[0.95] mb-6">
          <span className="block text-neutral-500">protect</span>
          <span className="block text-neutral-300">your</span>
          <span className="block text-white">data</span>
        </h1>
        <p className="text-neutral-400 max-w-lg mt-4 text-lg">
          The most advanced observability platform. Stop guessing and start knowing what's happening inside your microservices.
        </p>

        <div className="mt-16 grid grid-cols-1 md:grid-cols-3 gap-12 border-t border-neutral-800 pt-12 w-full max-w-4xl text-left">
          <div>
            <div className="text-4xl font-bold text-white mb-2">+65k</div>
            <div className="text-neutral-500 text-sm">Startups using our platform</div>
          </div>
          <div className="border-l border-neutral-800 pl-12">
            <div className="text-4xl font-bold text-white mb-2">+1.5b</div>
            <div className="text-neutral-500 text-sm">Events processed daily</div>
          </div>
          <div className="border-l border-neutral-800 pl-12">
            <div className="text-4xl font-bold text-white mb-2">+300k</div>
            <div className="text-neutral-500 text-sm">Active developers</div>
          </div>
        </div>
      </main>
    </div>
  )
}
