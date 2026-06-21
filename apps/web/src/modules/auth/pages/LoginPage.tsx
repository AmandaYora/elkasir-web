import { useState, type FormEvent } from "react";
import { useNavigate } from "react-router-dom";
import { Mail, Lock, Eye, EyeOff, ArrowRight, CircleAlert } from "lucide-react";
import { toast } from "sonner";
import { Button } from "@/shared/components/ui/button";
import { Label } from "@/shared/components/ui/label";
import { Card } from "@/shared/components/ui/card";
import { useAuthStore } from "@/shared/stores/auth.store";
import { ROUTE_PATHS } from "@/app/routes/route-paths";
import { loginSchema } from "@/modules/auth/schemas/login.schema";

// Admin sign-in, rendered in the dashboard's own card + token language.
export default function LoginPage() {
  const navigate = useNavigate();
  const login = useAuthStore((s) => s.login);
  const [email, setEmail] = useState("");
  const [password, setPassword] = useState("");
  const [showPassword, setShowPassword] = useState(false);
  const [error, setError] = useState<string | null>(null);
  const [submitting, setSubmitting] = useState(false);

  const onSubmit = async (e: FormEvent) => {
    e.preventDefault();
    const parsed = loginSchema.safeParse({ email, password });
    if (!parsed.success) {
      setError(parsed.error.issues[0]?.message ?? "Periksa input Anda.");
      return;
    }
    setSubmitting(true);
    const res = await login(parsed.data.email, parsed.data.password);
    setSubmitting(false);
    if (!res.ok) {
      setError(res.error);
      return;
    }
    setError(null);
    toast.success("Berhasil masuk");
    navigate(ROUTE_PATHS.dashboard, { replace: true });
  };

  return (
    <Card className="p-7 sm:p-8">
      <div>
        <h1 className="font-display text-2xl font-bold tracking-tight text-text">Masuk</h1>
        <p className="mt-1.5 text-sm text-muted">Masuk ke dasbor admin untuk mengelola tokomu.</p>
      </div>

      <form onSubmit={onSubmit} noValidate className="mt-7 space-y-4">
        <Field
          id="email"
          label="Email atau username"
          icon={<Mail className="h-4 w-4" />}
          type="text"
          autoComplete="username"
          value={email}
          onChange={setEmail}
          placeholder="admin"
        />

        <div className="space-y-1.5">
          <Label htmlFor="password">Password</Label>
          <div className="relative">
            <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted">
              <Lock className="h-4 w-4" />
            </span>
            <input
              id="password"
              type={showPassword ? "text" : "password"}
              autoComplete="current-password"
              value={password}
              onChange={(e) => setPassword(e.target.value)}
              placeholder="••••••••"
              className="h-11 w-full rounded-md border border-border bg-surface pl-10 pr-10 text-sm text-text shadow-sm outline-none transition-colors placeholder:text-muted focus:border-primary focus:ring-2 focus:ring-primary/30"
            />
            <button
              type="button"
              onClick={() => setShowPassword((v) => !v)}
              className="absolute right-2.5 top-1/2 -translate-y-1/2 rounded text-muted transition-colors hover:text-text focus-visible:outline-none focus-visible:ring-2 focus-visible:ring-primary/40"
              aria-label={showPassword ? "Sembunyikan password" : "Tampilkan password"}
            >
              {showPassword ? <EyeOff className="h-4 w-4" /> : <Eye className="h-4 w-4" />}
            </button>
          </div>
        </div>

        {error && (
          <div
            role="alert"
            className="flex items-start gap-2 rounded-md border border-danger/30 bg-danger-soft/60 px-3 py-2.5 text-sm text-danger"
          >
            <CircleAlert className="mt-0.5 h-4 w-4 shrink-0" />
            <span>{error}</span>
          </div>
        )}

        <Button type="submit" size="lg" loading={submitting} className="mt-1 w-full">
          Masuk ke Dasbor
          {!submitting && <ArrowRight className="h-4 w-4" />}
        </Button>
      </form>

      <p className="mt-6 text-center text-sm text-muted">
        Lupa akses? <span className="font-medium text-text">Hubungi pemilik toko.</span>
      </p>
    </Card>
  );
}

// A labelled input with a leading icon, in the app's token language.
function Field({
  id,
  label,
  icon,
  value,
  onChange,
  ...rest
}: {
  id: string;
  label: string;
  icon: React.ReactNode;
  value: string;
  onChange: (v: string) => void;
} & Omit<React.InputHTMLAttributes<HTMLInputElement>, "value" | "onChange" | "id">) {
  return (
    <div className="space-y-1.5">
      <Label htmlFor={id}>{label}</Label>
      <div className="relative">
        <span className="pointer-events-none absolute left-3 top-1/2 -translate-y-1/2 text-muted">
          {icon}
        </span>
        <input
          id={id}
          value={value}
          onChange={(e) => onChange(e.target.value)}
          className="h-11 w-full rounded-md border border-border bg-surface pl-10 pr-3 text-sm text-text shadow-sm outline-none transition-colors placeholder:text-muted focus:border-primary focus:ring-2 focus:ring-primary/30"
          {...rest}
        />
      </div>
    </div>
  );
}
