import { Mail, Clock, Globe, MapPin, Phone, MessageSquareText } from "lucide-react";
import { HomepageNav } from "@/modules/homepage/components/HomepageNav";
import { HomepageFooter } from "@/modules/homepage/components/HomepageFooter";
import "@/modules/homepage/styles.css";

const CHANNELS = [
  { icon: Mail, label: "Email", value: "cs@elcodelabs.com", href: "mailto:cs@elcodelabs.com", colSpan: "sm:col-span-1" },
  { icon: Phone, label: "Telepon", value: "0851-7347-1146", href: "tel:+6285173471146", colSpan: "sm:col-span-1" },
  {
    icon: Globe,
    label: "Website",
    value: "elkasir.elcodelabs.com",
    href: "https://elkasir.elcodelabs.com",
    colSpan: "sm:col-span-2 lg:col-span-1"
  },
  {
    icon: MapPin,
    label: "Alamat Kantor",
    value: "Gland Ciwastra Park, Jl. Morinda X No. 26, Bojongsoang, Kab. Bandung",
    href: null,
    colSpan: "sm:col-span-2"
  },
  { 
    icon: Clock, 
    label: "Jam Layanan", 
    value: "Setiap Hari, 08.00–21.00 WIB", 
    href: null,
    colSpan: "sm:col-span-2 lg:col-span-1"
  },
];

export default function ContactPage() {
  return (
    <div className="hp-root relative min-h-screen flex flex-col bg-(--hp-surface-raised)">
      <HomepageNav />

      <main className="relative z-10 flex-grow px-6 py-24 sm:py-32">
        <div className="mx-auto max-w-4xl hp-animate-fade">
          {/* Header */}
          <div className="text-center mb-16">
            <div className="mx-auto mb-6 flex h-16 w-16 items-center justify-center rounded-2xl bg-white border border-(--hp-line) shadow-sm">
              <MessageSquareText className="h-8 w-8 text-(--hp-primary)" aria-hidden="true" />
            </div>
            <p className="hp-font-mono text-sm font-semibold tracking-widest text-(--hp-ink-soft) uppercase">
              Pusat Bantuan
            </p>
            <h1 className="hp-font-display mt-4 text-4xl font-bold tracking-tight text-(--hp-ink) sm:text-5xl">
              Kontak Tim Kami
            </h1>
            <p className="mt-6 max-w-xl mx-auto text-lg leading-relaxed text-(--hp-ink-soft)">
              Ada pertanyaan soal produk, langganan, atau kendala teknis? Tim kami siap membantu melalui kanal-kanal berikut.
            </p>
          </div>

          {/* Cards Grid */}
          <div className="grid gap-6 sm:grid-cols-2 lg:grid-cols-3">
            {CHANNELS.map((c) => (
              <div 
                key={c.label} 
                className={`hp-card p-6 flex flex-col items-start ${c.colSpan}`}
              >
                <div className="flex h-12 w-12 items-center justify-center rounded-xl bg-(--hp-surface-raised) border border-(--hp-line) text-(--hp-primary)">
                  <c.icon className="h-5 w-5" aria-hidden="true" />
                </div>
                
                <div className="mt-6 flex-grow">
                  <p className="hp-font-mono text-[11px] font-semibold tracking-[0.15em] text-(--hp-ink-soft) uppercase">
                    {c.label}
                  </p>
                  
                  {c.href ? (
                    <a
                      href={c.href}
                      className="hp-font-display mt-2 inline-block text-xl font-semibold text-(--hp-ink) hover:text-(--hp-primary) transition-colors"
                    >
                      {c.value}
                    </a>
                  ) : (
                    <p className="hp-font-display mt-2 text-xl font-semibold leading-tight text-(--hp-ink)">
                      {c.value}
                    </p>
                  )}
                </div>
              </div>
            ))}
          </div>

          <div className="mt-16 text-center">
            <div className="inline-flex items-center gap-3 rounded-md border border-(--hp-line) bg-white px-5 py-2.5 shadow-sm">
              <span className="flex h-2 w-2 rounded-full bg-(--hp-primary)" aria-hidden="true" />
              <p className="text-sm text-(--hp-ink-soft)">
                Kami biasanya membalas email dalam <strong className="text-(--hp-ink) font-semibold">1×24 jam</strong> pada hari kerja.
              </p>
            </div>
          </div>
        </div>
      </main>

      <div className="relative z-10 border-t border-(--hp-line) bg-white mt-auto">
        <HomepageFooter />
      </div>
    </div>
  );
}
