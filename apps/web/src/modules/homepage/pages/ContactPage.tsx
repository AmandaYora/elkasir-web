import { Mail, Clock, Globe, MapPin, Phone } from "lucide-react";
import { HomepageNav } from "@/modules/homepage/components/HomepageNav";
import { HomepageFooter } from "@/modules/homepage/components/HomepageFooter";
import "@/modules/homepage/styles.css";

const CHANNELS = [
  { icon: Mail, label: "Email", value: "cs@elcodelabs.com", href: "mailto:cs@elcodelabs.com" },
  { icon: Phone, label: "Telepon", value: "0851-7347-1146", href: "tel:+6285173471146" },
  {
    icon: Globe,
    label: "Website",
    value: "elkasir.elcodelabs.com",
    href: "https://elkasir.elcodelabs.com",
  },
  {
    icon: MapPin,
    label: "Alamat Kantor",
    value: "Gland Ciwastra Park, Jl. Morinda X No. 26, Bojongsoang, Kab. Bandung",
    href: null,
  },
  { icon: Clock, label: "Jam Layanan", value: "Setiap Hari, 08.00–21.00 WIB", href: null },
];

export default function ContactPage() {
  return (
    <div className="hp-root">
      <HomepageNav />

      <section className="mx-auto max-w-2xl px-5 py-16 sm:py-24">
        <p className="hp-font-mono text-[11px] tracking-[0.3em] text-(--hp-glow-deep) uppercase">
          Bantuan
        </p>
        <h1 className="hp-font-display mt-3 text-3xl font-bold tracking-tight text-(--hp-ink) sm:text-4xl">
          Kontak Customer Service
        </h1>
        <p className="mt-4 max-w-md text-[15px] leading-relaxed text-(--hp-ink-soft)">
          Ada pertanyaan soal produk, langganan, atau kendala teknis? Tim kami siap membantu melalui
          kanal berikut.
        </p>

        <div className="mt-10 divide-y divide-(--hp-line) rounded-2xl border border-(--hp-line) bg-(--hp-surface)">
          {CHANNELS.map((c) => (
            <div key={c.label} className="flex items-start gap-4 px-6 py-5">
              <div className="flex h-10 w-10 shrink-0 items-center justify-center rounded-xl bg-[linear-gradient(135deg,var(--hp-glow),var(--hp-glow-deep))]">
                <c.icon className="h-5 w-5 text-(--hp-dark)" aria-hidden="true" />
              </div>
              <div>
                <p className="hp-font-mono text-[11px] tracking-[0.12em] text-(--hp-ink-soft) uppercase">
                  {c.label}
                </p>
                {c.href ? (
                  <a
                    href={c.href}
                    className="text-[15px] font-semibold text-(--hp-ink) underline decoration-(--hp-line) underline-offset-2 hover:text-(--hp-glow-deep)"
                  >
                    {c.value}
                  </a>
                ) : (
                  <p className="max-w-[38ch] text-[15px] leading-snug font-semibold text-(--hp-ink)">
                    {c.value}
                  </p>
                )}
              </div>
            </div>
          ))}
        </div>

        <p className="mt-6 text-[13px] text-(--hp-ink-soft)">
          Kami biasanya membalas email dalam 1×24 jam pada hari kerja.
        </p>
      </section>

      <HomepageFooter />
    </div>
  );
}
