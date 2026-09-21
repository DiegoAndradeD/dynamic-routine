import { CheckCircle2, FastForward, Sparkles, Clock } from "lucide-react";
import type { Activity } from "../types";

interface ActiveTaskCardProps {
  activity: Activity | null;
  onComplete: () => void;
  onMicroblock: () => void;
  onSkip: () => void;
}

export function ActiveTaskCard({
  activity,
  onComplete,
  onMicroblock,
  onSkip,
}: ActiveTaskCardProps) {
  if (!activity) {
    return (
      <div className="rounded-2xl border border-dashed border-neutral-800 p-8 text-center text-neutral-500">
        Nenhuma atividade na fila. Adicione atividades na configuração.
      </div>
    );
  }

  const categoryLabel = {
    body: "Corpo & Físico",
    focus: "Foco & Intelecto",
    leisure: "Lazer & Descompressão",
  }[activity.category];

  return (
    <section className="relative overflow-hidden rounded-2xl border border-neutral-800 bg-neutral-900/60 p-6 shadow-xl backdrop-blur-sm">
      <div className="flex items-center justify-between text-xs text-neutral-400 mb-3">
        <span className="flex items-center gap-1.5 font-medium tracking-wide uppercase text-[11px] text-amber-500">
          <Sparkles className="h-3.5 w-3.5" /> Próximo da Fila
        </span>
        <span className="rounded-full border border-neutral-800 bg-neutral-900 px-2.5 py-0.5 font-mono">
          {categoryLabel}
        </span>
      </div>

      <div className="mb-6">
        <h2 className="text-2xl font-bold tracking-tight text-white mb-1.5">
          {activity.name}
        </h2>
        <div className="flex items-center gap-1.5 text-xs text-neutral-400">
          <Clock className="h-3.5 w-3.5 text-neutral-500" />
          <span>
            Sessão sugerida:{" "}
            <strong className="text-neutral-200">
              {activity.recommendedMinutes} min
            </strong>
          </span>
          <span className="text-neutral-600">•</span>
          <span>Meta: {activity.targetPerWeek}x na semana</span>
        </div>
      </div>

      {/* Botões de Decisão */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-2.5">
        <button
          onClick={onComplete}
          className="flex items-center justify-center gap-2 rounded-xl bg-emerald-600 px-4 py-3 text-sm font-semibold text-white shadow-lg shadow-emerald-950/50 hover:bg-emerald-500 active:scale-[0.98] transition-all"
        >
          <CheckCircle2 className="h-4 w-4" />
          <span>Concluir Sessão</span>
        </button>

        <button
          onClick={onMicroblock}
          className="flex items-center justify-center gap-2 rounded-xl border border-amber-800/60 bg-amber-950/40 px-4 py-3 text-sm font-medium text-amber-300 hover:bg-amber-900/50 active:scale-[0.98] transition-all"
          title="Fazer versão curta (20 minutos) para manter o hábito sem sobrecarregar"
        >
          <Sparkles className="h-4 w-4 text-amber-400" />
          <span>Microbloco (20m)</span>
        </button>

        <button
          onClick={onSkip}
          className="flex items-center justify-center gap-2 rounded-xl border border-neutral-800 bg-neutral-800/80 px-4 py-3 text-sm font-medium text-neutral-300 hover:bg-neutral-800 hover:text-white active:scale-[0.98] transition-all"
          title="Sem tempo ou energia agora? Mova para trás na fila sem culpa"
        >
          <FastForward className="h-4 w-4 text-neutral-400" />
          <span>Pular / Desviar</span>
        </button>
      </div>
    </section>
  );
}
