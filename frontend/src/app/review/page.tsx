"use client";

import { useRouter } from "next/navigation";
import { useEffect, useState, useCallback } from "react";
import { api } from "@/lib/auth";

interface Case {
  id: number;
  contact_name: string | null;
  appointment_at: string | null;
  flow_source: string;
  ai_reason: string | null;
  created_at: string;
  skip_reason: string | null;
  chat_context: unknown[];
  suggested_message: string | null;
}

function formatAppointment(iso: string | null) {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "";
  const fecha = d.toLocaleDateString("es-AR", { day: "2-digit", month: "short" });
  const hora = d.toLocaleTimeString("es-AR", { hour: "2-digit", minute: "2-digit" });
  return `${fecha} · ${hora} hs`;
}

function formatShortTime(iso: string) {
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "";
  return d.toLocaleTimeString("es-AR", { hour: "2-digit", minute: "2-digit" });
}

export default function ReviewPage() {
  const router = useRouter();
  const [tab, setTab] = useState<"actionable" | "informative">("actionable");
  const [cases, setCases] = useState<Case[]>([]);
  const [loading, setLoading] = useState(true);
  const [error, setError] = useState("");
  const [busy, setBusy] = useState(false);

  const load = useCallback(async () => {
    setLoading(true);
    setError("");
    try {
      const kind = tab === "informative" ? "informative" : "actionable";
      const data = await api(`/api/cases/pending?kind=${kind}`);
      setCases(data.items || []);
    } catch (err: unknown) {
      setError(err instanceof Error ? err.message : "Error al cargar");
    } finally {
      setLoading(false);
    }
  }, [tab]);

  useEffect(() => {
    load();
  }, [load, router]);

  async function act(id: number, endpoint: string, label: string) {
    if (busy) return;
    setBusy(true);
    try {
      await api(`/api/cases/${id}/${endpoint}`, { method: "POST" });
      setCases((prev) => prev.filter((c) => c.id !== id));
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Error";
      setError(msg);
    } finally {
      setBusy(false);
    }
  }

  if (loading) {
    return (
      <main className="flex-1 flex flex-col items-center justify-center min-h-dvh text-muted gap-4">
        <div className="spinner" />
        <p>Cargando…</p>
      </main>
    );
  }

  return (
    <>
      <header className="sticky top-0 z-10 bg-surface border-b border-border px-4 py-3" style={{ paddingTop: "max(12px, env(safe-area-inset-top))" }}>
        <div className="flex items-center justify-between">
          <div className="flex items-center gap-2">
            <h1 className="text-lg font-semibold">Confirmaciones</h1>
            <span className="bg-accent text-white text-xs font-bold px-2 py-0.5 rounded-full min-w-[22px] text-center">
              {cases.length}
            </span>
          </div>
          <div className="flex items-center gap-2">
            <button onClick={load} className="p-2 rounded-full hover:bg-surface-2 transition">
              <svg viewBox="0 0 24 24" width="20" height="20" fill="currentColor">
                <path d="M17.65 6.35A7.96 7.96 0 0 0 12 4a8 8 0 1 0 7.45 11h-2.1A6 6 0 1 1 12 6c1.66 0 3.14.69 4.22 1.78L13 11h7V4l-2.35 2.35Z"/>
              </svg>
            </button>
          </div>
        </div>
      </header>

      <nav className="sticky top-[52px] z-9 flex bg-surface border-b border-border" style={{ top: "max(52px, calc(env(safe-area-inset-top) + 52px))" }}>
        {(["actionable", "informative"] as const).map((t) => (
          <button
            key={t}
            onClick={() => setTab(t)}
            className={`flex-1 py-3 text-sm font-semibold border-b-2 transition ${
              tab === t ? "text-accent border-accent" : "text-muted border-transparent"
            }`}
          >
            {t === "actionable" ? "Por revisar" : "Informe del día"}
          </button>
        ))}
      </nav>

      <main className="flex-1 overflow-auto">
        {error && (
          <div className="m-4 bg-red/10 border border-red/30 text-red text-sm rounded-lg px-3 py-2 flex justify-between items-center">
            <span>{error}</span>
            <button onClick={() => setError("")} className="text-red font-bold ml-2">&times;</button>
          </div>
        )}

        {cases.length === 0 ? (
          <div className="flex flex-col items-center justify-center min-h-[60vh] text-muted px-4 text-center">
            <svg viewBox="0 0 24 24" width="64" height="64" fill="currentColor" className="text-accent opacity-80 mb-4">
              <path d="M9 16.17 4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41L9 16.17z"/>
            </svg>
            <h2 className="text-xl text-[#e9edef] mb-1">Sin pendientes</h2>
            <p>{tab === "actionable" ? "No hay confirmaciones para revisar." : "Hoy todas las citas pasaron al flujo normal."}</p>
          </div>
        ) : tab === "informative" ? (
          <ul className="p-3 space-y-3">
            {cases.map((c) => (
              <li key={c.id} className="bg-surface border border-border rounded-xl p-3 space-y-2">
                <div className="flex justify-between items-baseline">
                  <span className="font-semibold text-sm truncate flex-1">{c.contact_name || "(sin nombre)"}</span>
                  <span className="text-xs text-muted ml-2">{formatShortTime(c.created_at)}</span>
                </div>
                <div className="bg-yellow/10 border border-yellow/30 text-yellow text-xs rounded-md px-2 py-1.5">
                  {c.ai_reason || c.skip_reason || "Sin motivo informado"}
                </div>
                <div className="flex gap-2 text-xs text-muted flex-wrap">
                  <span className="bg-surface-2 px-2 py-0.5 rounded">{c.flow_source === "mañana" ? "Mañana" : "Citas"}</span>
                  {c.appointment_at && <span>{formatAppointment(c.appointment_at)}</span>}
                  {c.skip_reason && <span className="bg-surface-2 px-2 py-0.5 rounded">{c.skip_reason}</span>}
                </div>
              </li>
            ))}
          </ul>
        ) : (
          <div className="p-4 space-y-3">
            {cases.map((c) => (
              <button
                key={c.id}
                onClick={() => router.push(`/review/${c.id}`)}
                className="w-full text-left bg-surface border border-border rounded-xl p-4 hover:bg-surface-2 transition space-y-3"
              >
                <div className="flex justify-between items-start">
                  <div className="min-w-0 flex-1">
                    <div className="font-semibold text-base truncate">{c.contact_name || "(sin nombre)"}</div>
                    <div className="text-xs text-muted mt-0.5">{formatAppointment(c.appointment_at)}</div>
                  </div>
                  <span className="bg-surface-2 text-muted text-xs font-semibold uppercase px-2 py-1 rounded-full border border-border ml-2">
                    {c.flow_source === "mañana" ? "Mañana" : "Citas"}
                  </span>
                </div>
                <div className="bg-yellow/10 border border-yellow/30 text-yellow text-sm rounded-lg px-3 py-2">
                  {c.ai_reason || "Sin motivo informado"}
                </div>
                <div className="text-sm text-muted bg-bubble-in rounded-lg px-3 py-2 line-clamp-2">
                  {c.suggested_message || "(sin mensaje sugerido)"}
                </div>
              </button>
            ))}
          </div>
        )}
      </main>
    </>
  );
}
