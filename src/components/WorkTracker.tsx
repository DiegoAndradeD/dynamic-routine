import { Play, Pause, CheckCircle2, Briefcase, Plus } from "lucide-react";
import type { DailyWorkRecord } from "../types";

interface WorkTrackerProps {
  record: DailyWorkRecord;
  isWorking: boolean;
  onToggleWork: () => void;
  onAddSeconds: (sec: number) => void;
  onComplete: () => void;
  workDaysDoneThisWeek: number;
}

export function WorkTracker({
  record,
  isWorking,
  onToggleWork,
  onAddSeconds,
  onComplete,
  workDaysDoneThisWeek,
}: WorkTrackerProps) {
  const totalTargetSeconds = 8 * 3600;
  const progressPercent = Math.min(
    100,
    Math.round((record.secondsWorked / totalTargetSeconds) * 100),
  );

  const hours = Math.floor(record.secondsWorked / 3600);
  const minutes = Math.floor((record.secondsWorked % 3600) / 60);
  const seconds = record.secondsWorked % 60;

  const formattedTime = `${String(hours).padStart(2, "0")}:${String(minutes).padStart(2, "0")}:${String(seconds).padStart(2, "0")}`;

  return (
    <section className="rounded-2xl border border-neutral-800 bg-neutral-900/50 p-4 sm:p-5 backdrop-blur-sm space-y-3">
      <div className="flex flex-wrap items-center justify-between gap-2">
        <div className="flex items-center gap-2.5">
          <div
            className={`p-2 rounded-xl border ${isWorking ? "border-amber-500/50 bg-amber-500/10 text-amber-400 animate-pulse" : "border-neutral-800 bg-neutral-900 text-neutral-400"}`}
          >
            <Briefcase className="h-4 w-4" />
          </div>
          <div>
            <div className="flex items-center gap-2">
              <h3 className="text-sm font-bold text-neutral-100">
                Expediente de Trabalho
              </h3>
              <span className="text-xs font-mono px-2 py-0.5 rounded-full bg-neutral-800 text-neutral-400">
                {workDaysDoneThisWeek}/5 dias na semana
              </span>
            </div>
            <p className="text-xs text-neutral-500">
              Meta diária: 8 horas (intercalável livremente com as tarefas)
            </p>
          </div>
        </div>

        {/* Display do Cronômetro de Trabalho */}
        <div className="flex items-center gap-3">
          <div className="text-right font-mono">
            <span className="text-xl font-bold tracking-tight text-white">
              {formattedTime}
            </span>
            <span className="text-xs text-neutral-500"> / 08:00:00</span>
          </div>

          <button
            onClick={onToggleWork}
            className={`flex items-center gap-1.5 rounded-xl px-3.5 py-2 text-xs font-semibold shadow-md transition-all ${
              isWorking
                ? "bg-amber-600 text-white hover:bg-amber-500 shadow-amber-950"
                : "bg-neutral-800 text-neutral-200 hover:bg-neutral-700 hover:text-white"
            }`}
          >
            {isWorking ? (
              <Pause className="h-3.5 w-3.5" />
            ) : (
              <Play className="h-3.5 w-3.5" />
            )}
            <span>{isWorking ? "Pausar" : "Entrar no Trabalho"}</span>
          </button>
        </div>
      </div>

      {/* Barra de Progresso do Trabalho */}
      <div className="h-2 w-full overflow-hidden rounded-full bg-neutral-800/80">
        <div
          className="h-full rounded-full transition-all duration-300"
          style={{
            width: `${progressPercent}%`,
            backgroundColor: record.isCompleted ? "#10b981" : "#f59e0b",
          }}
        />
      </div>

      {/* Controles Rápidos Manuais */}
      <div className="flex flex-wrap items-center justify-between gap-2 pt-1 text-xs text-neutral-400">
        <div className="flex items-center gap-1.5">
          <span>Ajuste rápido:</span>
          <button
            onClick={() => onAddSeconds(3600)}
            className="flex items-center gap-0.5 rounded-md border border-neutral-800 bg-neutral-900 px-2 py-1 hover:bg-neutral-800 hover:text-white"
          >
            <Plus className="h-3 w-3" /> 1h
          </button>
          <button
            onClick={() => onAddSeconds(4 * 3600)}
            className="flex items-center gap-0.5 rounded-md border border-neutral-800 bg-neutral-900 px-2 py-1 hover:bg-neutral-800 hover:text-white"
          >
            <Plus className="h-3 w-3" /> 4h
          </button>
        </div>

        {!record.isCompleted && (
          <button
            onClick={onComplete}
            className="flex items-center gap-1 text-emerald-400/90 hover:text-emerald-300 transition-colors"
          >
            <CheckCircle2 className="h-3.5 w-3.5" />
            <span>Marcar 8h Concluídas Hoje</span>
          </button>
        )}
      </div>
    </section>
  );
}
