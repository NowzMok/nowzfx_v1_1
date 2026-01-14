import { motion, AnimatePresence } from 'framer-motion'
import { useState, useEffect } from 'react'

// Bible verses about money and wealth
const bibleVerses = [
  {
    verse: '箴言 10:2',
    text: '不义之财，毫无益处',
    subtitle: '恶人的财富终必归于义人，罪人的家产终必归于善人',
    emphasis: '不义之财',
  },
  {
    verse: '箴言 11:4',
    text: '财富在忿怒的日子毫无益处',
    subtitle: '惟有公义能救人脱离死亡',
    emphasis: '忿怒的日子',
  },
  {
    verse: '箴言 13:11',
    text: '发财致富，速速积攒',
    subtitle: '但渐富的必致贫穷',
    emphasis: '速速积攒',
  },
  {
    verse: '箴言 15:16',
    text: '家中有财，不如公义',
    subtitle: '敬畏耶和华，强如拥有金银',
    emphasis: '公义',
  },
  {
    verse: '箴言 16:8',
    text: '多有财利，行事不义',
    subtitle: '不如少有财利，行事公义',
    emphasis: '行事公义',
  },
  {
    verse: '箴言 28:20',
    text: '诚实人必多得福',
    subtitle: '想要急速发财的，不免受罚',
    emphasis: '诚实',
  },
  {
    verse: '提摩太前书 6:10',
    text: '贪财是万恶之根',
    subtitle: '有人贪恋钱财，就被引诱离了真道',
    emphasis: '贪财',
  },
  {
    verse: '马太福音 6:24',
    text: '你们不能又事奉神，又事奉玛门',
    subtitle: '玛门即钱财',
    emphasis: '事奉神',
  },
]

const goodReminders = [
  '财富如水，善用则润物无声',
  '公义的财宝，永不朽坏',
  '施比受更为有福',
  '诚实的劳作，必得饱足',
  '贪财是万恶之根',
  '知足常乐，乃是大益',
  '义人的财富，必蒙保守',
  '善用钱财，积攒天家',
]

export default function TerminalHero() {
  // Bible verse rotation state
  const [currentVerseIndex, setCurrentVerseIndex] = useState(0)

  // Auto-rotate verses every 5 seconds
  useEffect(() => {
    const timer = setInterval(() => {
      setCurrentVerseIndex((prev) => (prev + 1) % bibleVerses.length)
    }, 5000)
    return () => clearInterval(timer)
  }, [])

  // Real-time price state (kept but unused in display)
  const [, setPrices] = useState<Record<string, string>>({
    BTC: '...',
    ETH: '...',
    SOL: '...',
    BNB: '...',
    XRP: '...',
    DOGE: '...',
    ADA: '...',
    AVAX: '...',
  })

  useEffect(() => {
    const fetchPrices = async () => {
      const symbols = ['BTC', 'ETH', 'SOL', 'BNB', 'XRP', 'DOGE', 'ADA', 'AVAX']

      // We use Promise.all to fetch them in parallel for now, or sequentially if rate limited.
      // Parallel is better for UI responsiveness.
      try {
        const results = await Promise.all(
          symbols.map(async (sym) => {
            try {
              // Use native fetch to bypass global error handlers (toasts) in httpClient
              const response = await fetch(
                `/api/klines?symbol=${sym}USDT&interval=1m&limit=1`
              )
              if (!response.ok) return null

              const res = await response.json()
              // Check for standard API response structure or direct array
              const klineData = res.data || res

              if (Array.isArray(klineData) && klineData.length > 0) {
                const closePrice = parseFloat(
                  klineData[0].close || klineData[0][4]
                ) // Handle object or array format
                if (isNaN(closePrice)) return null

                // Format price: < 1 use 4 decimals, > 1 use 2
                const formatted =
                  closePrice < 1
                    ? closePrice.toFixed(4)
                    : closePrice.toLocaleString('en-US', {
                        minimumFractionDigits: 2,
                        maximumFractionDigits: 2,
                      })
                return { symbol: sym, price: formatted }
              }
            } catch (err) {
              // Silent failure for background polling
            }
            return null
          })
        )

        const newPrices: Record<string, string> = {}
        results.forEach((r) => {
          if (r) newPrices[r.symbol] = r.price
        })

        setPrices((prev) => ({ ...prev, ...newPrices }))
      } catch (e) {
        console.error('Failed to fetch market prices', e)
      }
    }

    // Only fetch once on mount, cache the result
    fetchPrices()
  }, [])

  return (
    <section className="relative w-full min-h-screen bg-nofx-bg text-nofx-text overflow-hidden flex flex-col pt-20">
      {/* BACKGROUND LAYERS */}
      {/* 1. Grid */}
      <div className="absolute inset-0 bg-[url('https://grainy-gradients.vercel.app/noise.svg')] opacity-20 mix-blend-soft-light pointer-events-none"></div>
      <div
        className="absolute inset-x-0 bottom-0 h-[50vh] bg-[linear-gradient(to_right,#80808012_1px,transparent_1px),linear-gradient(to_bottom,#80808012_1px,transparent_1px)] bg-[size:40px_40px] [mask-image:radial-gradient(ellipse_60%_50%_at_50%_0%,#000_70%,transparent_100%)] pointer-events-none md:hidden"
        style={{
          transform:
            'perspective(500px) rotateX(60deg) translateY(100px) scale(2)',
        }}
      ></div>
      <div className="absolute inset-0 bg-grid-pattern opacity-[0.03] pointer-events-none"></div>

      {/* 2. World Map / Data Viz Background (Abstract) - Removed borders */}
      <div className="absolute inset-0 flex items-center justify-center opacity-10 pointer-events-none">
        <div className="w-[80vw] h-[80vw] rounded-full border border-nofx-gold/10 animate-pulse-slow"></div>
        <div className="absolute w-[60vw] h-[60vw] rounded-full border border-dashed border-nofx-accent/10 animate-[spin_60s_linear_infinite]"></div>
      </div>

      {/* 3. Gradient Spots - Intensified for Mobile */}
      <div className="absolute top-[-10%] left-[-10%] w-[40vw] h-[40vw] bg-nofx-gold/20 rounded-full blur-[120px] pointer-events-none mix-blend-screen"></div>
      <div className="absolute bottom-[-10%] right-[-10%] w-[40vw] h-[40vw] bg-nofx-accent/10 rounded-full blur-[120px] pointer-events-none mix-blend-screen"></div>

      {/* Mobile Bottom Fade */}
      <div className="absolute bottom-0 left-0 w-full h-32 bg-gradient-to-t from-nofx-bg to-transparent z-20 pointer-events-none md:hidden" />

      {/* CONTENT GRID */}
      <div className="relative z-10 flex-1 grid grid-cols-1 lg:grid-cols-12 gap-0 lg:gap-8 max-w-[1800px] mx-auto w-full px-6 h-full pb-20 pt-10 pointer-events-none">
        {/* LEFT COLUMN: Empty */}
        <div className="hidden lg:flex col-span-3 flex-col justify-between h-full pr-8 py-10 pointer-events-auto">
          {/* Empty - removed all system diagnostics and borders */}
        </div>

        {/* CENTER COLUMN: MAIN ACTION */}
        <div className="col-span-1 lg:col-span-6 flex flex-col items-center justify-center text-center relative z-20 pointer-events-auto">
          {/* Bible Verse Cards - Auto Rotating */}
          <div className="relative z-20 w-full max-w-4xl mx-auto p-8 md:p-12 h-[500px] flex items-center justify-center">
            <AnimatePresence mode="wait">
              <motion.div
                key={currentVerseIndex}
                initial={{ opacity: 0, x: 100, scale: 0.9 }}
                animate={{ opacity: 1, x: 0, scale: 1 }}
                exit={{ opacity: 0, x: -100, scale: 0.9 }}
                transition={{ duration: 0.6, ease: 'easeInOut' }}
                className="absolute w-full"
              >
                {/* Decorative Frame - Removed border, only gradient */}
                <div className="absolute inset-0 bg-gradient-to-br from-nofx-gold/5 via-transparent to-nofx-accent/5 rounded-2xl pointer-events-none"></div>

                {/* Main Bible Verse */}
                <div className="relative flex flex-col items-center justify-center space-y-6 py-12">
                  {/* Ornamental Cross */}
                  <div className="text-nofx-gold/40 text-6xl md:text-8xl font-serif leading-none opacity-30 animate-pulse-slow">
                    ✞
                  </div>

                  {/* Verse Text - Ultra Prominent */}
                  <div className="text-center space-y-3">
                    <div className="text-4xl md:text-6xl lg:text-7xl xl:text-8xl font-black tracking-tighter leading-tight bg-clip-text text-transparent bg-gradient-to-b from-white via-white to-zinc-500 drop-shadow-2xl filter brightness-125">
                      {bibleVerses[currentVerseIndex].text}
                    </div>

                    <div className="text-2xl md:text-4xl lg:text-5xl font-bold text-nofx-gold tracking-widest opacity-95 drop-shadow-lg">
                      {bibleVerses[currentVerseIndex].verse}
                    </div>

                    {/* Subtitle */}
                    <div className="text-lg md:text-2xl text-zinc-300 font-light leading-relaxed opacity-80 max-w-2xl mx-auto mt-2">
                      {bibleVerses[currentVerseIndex].subtitle}
                    </div>
                  </div>

                  {/* Emphasis Highlight */}
                  <div className="text-sm md:text-base text-nofx-gold/80 font-mono tracking-wider opacity-90">
                    重点: {bibleVerses[currentVerseIndex].emphasis}
                  </div>
                </div>
              </motion.div>
            </AnimatePresence>
          </div>

          {/* Empty placeholder to maintain layout */}
          <div className="h-16"></div>
        </div>
      </div>

      {/* RIGHT COLUMN: Empty - All agentic content removed */}
      <div className="absolute top-20 md:top-0 right-0 h-[50vh] md:h-full w-full lg:w-[45vw] flex pointer-events-none items-center justify-center z-0 opacity-80 lg:opacity-100 mix-blend-normal">
        {/* Empty space - removed all agentic elements */}
      </div>

      {/* FOOTER:善意提醒滚动条 */}
      <div className="absolute bottom-0 w-full bg-black/80 backdrop-blur-md z-30 overflow-hidden py-3 flex items-center">
        <div className="flex animate-marquee whitespace-nowrap gap-16 text-sm font-mono text-zinc-300 px-6">
          {goodReminders.map((reminder, index) => (
            <span
              key={index}
              className="flex items-center gap-3 text-nofx-gold whitespace-nowrap"
            >
              <span className="text-lg">●</span>
              <span className="font-light">{reminder}</span>
            </span>
          ))}
          {/* Duplicate for seamless loop */}
          {goodReminders.map((reminder, index) => (
            <span
              key={`dup-${index}`}
              className="flex items-center gap-3 text-nofx-gold whitespace-nowrap"
            >
              <span className="text-lg">●</span>
              <span className="font-light">{reminder}</span>
            </span>
          ))}
        </div>
      </div>

      {/* CRT OVERLAY (Global) */}
      <div className="absolute inset-0 crt-overlay pointer-events-none z-50 opacity-40"></div>
    </section>
  )
}
