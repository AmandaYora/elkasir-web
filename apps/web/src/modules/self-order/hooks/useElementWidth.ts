import { useEffect, useRef, useState } from "react";

// Mengukur lebar elemen secara live (ResizeObserver) — dipakai agar QR/barcode
// di-render pada resolusi asli yang pas dengan container, bukan di-scale via CSS
// (yang bisa buram/kurang presisi saat dipindai scanner kasir).
export function useElementWidth<T extends HTMLElement>() {
  const ref = useRef<T>(null);
  const [width, setWidth] = useState(0);

  useEffect(() => {
    const el = ref.current;
    if (!el) return;
    const update = () => setWidth(el.getBoundingClientRect().width);
    update();
    const observer = new ResizeObserver(update);
    observer.observe(el);
    return () => observer.disconnect();
  }, []);

  return { ref, width };
}
