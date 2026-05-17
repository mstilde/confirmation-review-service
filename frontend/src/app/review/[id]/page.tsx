"use client";

import { useRouter, useParams } from "next/navigation";
import { useEffect, useState } from "react";
import { api } from "@/lib/auth";

interface Message {
  text?: string;
  timestamp?: string;
  is_sender?: boolean;
  attachments?: { type?: string }[];
}

interface Case {
  id: number;
  contact_name: string | null;
  appointment_at: string | null;
  flow_source: string;
  ai_reason: string | null;
  chat_context: Message[];
  suggested_message: string | null;
  status: string;
}

function formatAppointment(iso: string | null) {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "";
  const fecha = d.toLocaleDateString("es-AR", { day: "2-digit", month: "short" });
  const hora = d.toLocaleTimeString("es-AR", { hour: "2-digit", minute: "2-digit" });
  return `${fecha} · ${hora} hs ART`;
}

function formatMessageTime(iso: string | undefined) {
  if (!iso) return "";
  const d = new Date(iso);
  if (isNaN(d.getTime())) return "";
  return d.toLocaleString("es-AR", { day: "2-digit", month: "2-digit", hour: "2-digit", minute: "2-digit" });
}

export default function CaseDetailPage() {
  const router = useRouter();
  const params = useParams();
  const id = params.id as string;

  const [cc, setCc] = useState<Case | null>(null);
  const [loading, setLoading] = useState(true);
  const [busy, setBusy] = useState(false);
  const [toast, setToast] = useState<{ msg: string; kind: string } | null>(null);
  const [error, setError] = useState("");

  useEffect(() => {
    (async () => {
      try {
        const data = await api(`/api/cases/${id}`);
        setCc(data.item);
      } catch (err: unknown) {
        setError(err instanceof Error ? err.message : "Error al cargar");
      } finally {
        setLoading(false);
      }
    })();
  }, [id, router]);

  async function act(endpoint: string, label: string) {
    if (busy || !cc) return;
    setBusy(true);
    try {
      await api(`/api/cases/${id}/${endpoint}`, { method: "POST" });
      setToast({ msg: label, kind: endpoint === "skip" ? "" : "success" });
      setTimeout(() => router.push("/review"), 800);
    } catch (err: unknown) {
      const msg = err instanceof Error ? err.message : "Error";
      setToast({ msg: msg, kind: "error" });
    } finally {
      setBusy(false);
    }
  }

  if (loading) {
    return (
      <main className="flex-1 flex items-center justify-center min-h-dvh">
        <div className="spinner" />
      </main>
    );
  }

  if (error || !cc) {
    return (
      <main className="flex-1 flex flex-col items-center justify-center min-h-dvh gap-4 px-4 text-center">
        <p className="text-red">{error || "Caso no encontrado"}</p>
        <button onClick={() => router.push("/review")} className="text-accent text-sm font-semibold">Volver a la lista</button>
      </main>
    );
  }

  const chatMessages: Message[] = cc.chat_context || [];

  return (
    <>
      <header className="sticky top-0 z-10 bg-surface border-b border-border px-4 py-3 flex items-center gap-3" style={{ paddingTop: "max(12px, env(safe-area-inset-top))" }}>
        <button onClick={() => router.push("/review")} className="p-1 -ml-1 text-muted hover:text-[#e9edef]">
          <svg viewBox="0 0 24 24" width="24" height="24" fill="currentColor"><path d="M20 11H7.83l5.59-5.59L12 4l-8 8 8 8 1.41-1.41L7.83 13H20v-2z"/></svg>
        </button>
        <h1 className="text-lg font-semibold truncate flex-1">{cc.contact_name || "(sin nombre)"}</h1>
      </header>

      <main className="flex-1 overflow-auto p-4 space-y-3">
        <div className="flex justify-between items-start">
          <div>
            <div className="text-xs text-muted">{formatAppointment(cc.appointment_at)}</div>
          </div>
          <span className="bg-surface-2 text-muted text-xs font-semibold uppercase px-2 py-1 rounded-full border border-border">
            {cc.flow_source === "mañana" ? "Mañana" : "Citas"}
          </span>
        </div>

        <div className="bg-yellow/10 border border-yellow/30 text-yellow text-sm rounded-xl p-3 leading-relaxed">
          {cc.ai_reason || "Sin motivo informado"}
        </div>

        <section>
          <div className="text-xs font-bold uppercase tracking-wider text-muted mb-2">Conversación</div>
          <div className="bg-surface border border-border rounded-xl p-3 max-h-[45vh] overflow-y-auto space-y-1.5">
            {chatMessages.length === 0 ? (
              <p className="text-center text-muted text-sm py-6">Sin mensajes en caché</p>
            ) : (
              [...chatMessages].reverse().map((msg, i) => (
                <div key={i} className={`max-w-[80%] px-3 py-2 rounded-lg text-sm leading-relaxed whitespace-pre-wrap ${
                  msg.is_sender ? "bg-bubble-out ml-auto rounded-tr-sm" : "bg-bubble-in rounded-tl-sm"
                }`}>
                  {msg.text && <div>{msg.text}</div>}
                  {(msg.attachments || []).map((att, j) => (
                    <span key={j} className="inline-block bg-white/5 text-muted text-xs px-2 py-0.5 rounded mt-1">
                      [{att.type || "archivo"}]
                    </span>
                  ))}
                  <div className={`text-[10px] text-muted mt-1 opacity-80 ${msg.is_sender ? "text-right" : ""}`}>
                    {formatMessageTime(msg.timestamp)}
                  </div>
                </div>
              ))
            )}
          </div>
        </section>

        <section>
          <div className="text-xs font-bold uppercase tracking-wider text-muted mb-2">Mensaje sugerido</div>
          <div className="bg-bubble-out text-[#e9edef] rounded-xl rounded-tr-sm p-3 text-sm leading-relaxed whitespace-pre-wrap">
            {cc.suggested_message || "(sin mensaje sugerido)"}
          </div>
        </section>
      </main>

      <footer className="sticky bottom-0 bg-surface border-t border-border px-4 py-3 flex justify-around items-center gap-3" style={{ paddingBottom: "max(12px, env(safe-area-inset-bottom))" }}>
        <button
          disabled={busy}
          onClick={() => {
            if (confirm("¿Cancelar la cita en Notion?")) act("cancel", "Cita cancelada");
          }}
          className="flex-1 max-w-[110px] h-[60px] bg-red text-white rounded-2xl flex items-center justify-center hover:opacity-90 disabled:opacity-50 shadow-lg"
        >
          <svg viewBox="0 0 24 24" width="32" height="32" fill="currentColor"><path d="M19 6.41 17.59 5 12 10.59 6.41 5 5 6.41 10.59 12 5 17.59 6.41 19 12 13.41 17.59 19 19 17.59 13.41 12z"/></svg>
        </button>
        <button
          disabled={busy}
          onClick={() => act("skip", "Skipeado")}
          className="flex-1 max-w-[110px] h-[60px] bg-[#5a6b75] text-white rounded-2xl flex items-center justify-center hover:opacity-90 disabled:opacity-50 shadow-lg"
        >
          <svg viewBox="0 0 24 24" width="32" height="32" fill="currentColor"><path d="M11.07 12.85c.77-1.39 2.25-2.21 3.11-3.44.91-1.29.4-3.7-2.18-3.7-1.69 0-2.52 1.28-2.87 2.34L6.54 6.96C7.25 4.83 9.18 3 11.99 3c2.35 0 3.96 1.07 4.78 2.41.7 1.15 1.11 3.3.03 4.9-1.2 1.77-2.35 2.31-2.97 3.45-.25.46-.35.76-.35 2.24h-2.89c-.01-.78-.13-2.05.48-3.15zM14 20c0 1.1-.9 2-2 2s-2-.9-2-2 .9-2 2-2 2 .9 2 2z"/></svg>
        </button>
        <button
          disabled={busy}
          onClick={() => act("approve", "Confirmación enviada")}
          className="flex-1 max-w-[110px] h-[60px] bg-green text-white rounded-2xl flex items-center justify-center hover:opacity-90 disabled:opacity-50 shadow-lg"
        >
          <svg viewBox="0 0 24 24" width="32" height="32" fill="currentColor"><path d="M9 16.17 4.83 12l-1.42 1.41L9 19 21 7l-1.41-1.41L9 16.17z"/></svg>
        </button>
      </footer>

      {toast && (
        <div className={`toast ${toast.kind}`}>
          {toast.msg}
        </div>
      )}
    </>
  );
}
