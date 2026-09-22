import { useState, useEffect } from "react";
import {
  CheckCircle2,
  FastForward,
  Sparkles,
  Clock,
  Play,
  Pause,
  RotateCcw,
} from "lucide-react";
import type { Activity } from "../types";

interface ActiveTaskCardProps {
  activity: Activity | null;
  onComplete: (durationMinutes?: number) => void;
  onMicroblock: () => void;
  onSkip: () => void;
}

export function ActiveTaskCard({
  activity,
  onComplete,
  onMicroblock,
  onSkip,
}: ActiveTaskCardProps) {
  const [timerSeconds, setTimerSeconds] = useState(0);
  const [isRunning, setIsRunning] = useState(false);

  // Reseta o timer sempre que mudar a atividade no topo
  useEffect(() => {
    setTimerSeconds(0);
    setIsRunning(false);
  }, [activity?.id]);

  // Loop do timer
  useEffect(() => {
    if (!isRunning) return;
    const interval = setInterval(
      () => setTimerSeconds((prev) => prev + 1),
      1000,
    );
    return () => clearInterval(interval);
  }, [isRunning]);

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

  const timerMin = Math.floor(timerSeconds / 60);
  const timerSec = timerSeconds % 60;
  const formattedTimer = `${String(timerMin).padStart(2, "0")}:${String(timerSec).padStart(2, "0")}`;

  const handleFinish = () => {
    const elapsedMinutes = Math.max(1, Math.round(timerSeconds / 60));
    onComplete(elapsedMinutes);
  };

  return (
    <section className="relative overflow-hidden rounded-2xl border border-neutral-800 bg-neutral-900/60 p-6 shadow-xl backdrop-blur-sm space-y-6">
      <div className="flex items-center justify-between text-xs text-neutral-400">
        <span className="flex items-center gap-1.5 font-medium tracking-wide uppercase text-[11px] text-amber-500">
          <Sparkles className="h-3.5 w-3.5" /> Próximo da Fila
        </span>
        <span className="rounded-full border border-neutral-800 bg-neutral-900 px-2.5 py-0.5 font-mono">
          {categoryLabel}
        </span>
      </div>

      <div className="flex flex-wrap items-center justify-between gap-4">
        <div>
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

        {/* Cronômetro Dedicado da Atividade */}
        <div className="flex items-center gap-2 rounded-xl border border-neutral-800 bg-neutral-950 px-3.5 py-2">
          <div className="font-mono text-xl font-bold text-amber-400 min-w-[65px] text-center">
            {formattedTimer}
          </div>
          <button
            onClick={() => setIsRunning(!isRunning)}
            className={`p-1.5 rounded-lg text-xs transition-colors ${
              isRunning
                ? "bg-amber-500 text-black font-semibold"
                : "bg-neutral-800 text-neutral-300 hover:text-white"
            }`}
            title={isRunning ? "Pausar sessão" : "Iniciar cronômetro de foco"}
          >
            {isRunning ? (
              <Pause className="h-4 w-4" />
            ) : (
              <Play className="h-4 w-4" />
            )}
          </button>
          {timerSeconds > 0 && (
            <button
              onClick={() => {
                setTimerSeconds(0);
                setIsRunning(false);
              }}
              className="p-1.5 rounded-lg text-neutral-500 hover:text-neutral-300"
              title="Zerar cronômetro"
            >
              <RotateCcw className="h-3.5 w-3.5" />
            </button>
          )}
        </div>
      </div>

      {/* Botões de Ação */}
      <div className="grid grid-cols-1 sm:grid-cols-3 gap-2.5 pt-2">
        <button
          onClick={handleFinish}
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
          title="Mova para trás na esteira sem culpa"
        >
          <FastForward className="h-4 w-4 text-neutral-400" />
          <span>Pular / Desviar</span>
        </button>
      </div>
    </section>
  );
}
