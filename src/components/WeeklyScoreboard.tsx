import type { Activity } from "../types";

interface WeeklyScoreboardProps {
  activities: Record<string, Activity>;
  stats: Record<string, { full: number; micro: number; total: number }>;
}

export function WeeklyScoreboard({ activities, stats }: WeeklyScoreboardProps) {
  const activityList = Object.values(activities);

  return (
    <section className="rounded-2xl border border-neutral-800 bg-neutral-900/40 p-5 backdrop-blur-sm space-y-4">
      <div className="flex items-center justify-between">
        <div>
          <h3 className="text-sm font-semibold text-neutral-200">
            Placar Semanal
          </h3>
          <p className="text-xs text-neutral-500">
            Média corrente de Segunda a Domingo
          </p>
        </div>
      </div>

      <div className="grid grid-cols-1 md:grid-cols-2 lg:grid-cols-3 gap-3">
        {activityList.map((act) => {
          const actStat = stats[act.id] || { full: 0, micro: 0, total: 0 };
          const progressPercent = Math.min(
            100,
            Math.round((actStat.total / act.targetPerWeek) * 100),
          );

          return (
            <div
              key={act.id}
              className="rounded-xl border border-neutral-800/80 bg-neutral-900/60 p-3.5 space-y-2.5"
            >
              <div className="flex items-center justify-between text-xs">
                <span className="font-medium text-neutral-300">{act.name}</span>
                <span className="font-mono text-neutral-400">
                  <strong className="text-white">{actStat.total}</strong> /{" "}
                  {act.targetPerWeek}x
                </span>
              </div>

              {/* Barra de Progresso Suave */}
              <div className="h-2 w-full overflow-hidden rounded-full bg-neutral-800">
                <div
                  className="h-full rounded-full bg-neutral-400 transition-all duration-300"
                  style={{
                    width: `${progressPercent}%`,
                    backgroundColor:
                      actStat.total >= act.targetPerWeek
                        ? "#10b981"
                        : "#f59e0b",
                  }}
                />
              </div>

              <div className="flex items-center justify-between text-[11px] text-neutral-500">
                <span>{act.recommendedMinutes} min/sessão</span>
                {actStat.micro > 0 && (
                  <span className="text-amber-500/90 font-mono">
                    ({actStat.micro} micro)
                  </span>
                )}
              </div>
            </div>
          );
        })}
      </div>
    </section>
  );
}
