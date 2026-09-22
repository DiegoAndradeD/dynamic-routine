import { useState, useEffect } from "react";
import { Calendar as CalendarIcon, Clock } from "lucide-react";

export function TimeContextBar() {
  const [time, setTime] = useState(new Date());

  useEffect(() => {
    const timer = setInterval(() => setTime(new Date()), 1000);
    return () => clearInterval(timer);
  }, []);

  // Formata hora e data
  const timeString = time.toLocaleTimeString("pt-BR", {
    hour: "2-digit",
    minute: "2-digit",
    second: "2-digit",
  });
  const dateString = time.toLocaleDateString("pt-BR", {
    weekday: "long",
    day: "numeric",
    month: "long",
    year: "numeric",
  });

  // Dias da semana corrente (Segunda a Domingo)
  const dayOfWeek = time.getDay(); // 0 = Domingo
  const mondayOffset = dayOfWeek === 0 ? -6 : 1 - dayOfWeek;
  const monday = new Date(time);
  monday.setDate(time.getDate() + mondayOffset);

  const daysLabels = ["Seg", "Ter", "Qua", "Qui", "Sex", "Sáb", "Dom"];
  const currentDayIndex = dayOfWeek === 0 ? 6 : dayOfWeek - 1;

  return (
    <div className="flex flex-wrap items-center justify-between gap-4 rounded-2xl border border-neutral-800/80 bg-neutral-900/30 px-5 py-3 text-xs text-neutral-400">
      {/* Relógio e Data */}
      <div className="flex items-center gap-4">
        <div className="flex items-center gap-2 font-mono text-sm font-semibold text-neutral-100">
          <Clock className="h-4 w-4 text-amber-500" />
          <span>{timeString}</span>
        </div>
        <div className="hidden sm:flex items-center gap-1.5 capitalize text-neutral-400 border-l border-neutral-800 pl-4">
          <CalendarIcon className="h-3.5 w-3.5 text-neutral-500" />
          <span>{dateString}</span>
        </div>
      </div>

      {/* Navegador da semana */}
      <div className="flex items-center gap-1.5 font-mono">
        {daysLabels.map((day, idx) => {
          const isToday = idx === currentDayIndex;
          const isPast = idx < currentDayIndex;
          return (
            <div
              key={day}
              className={`flex h-7 w-8 items-center justify-center rounded-lg text-[11px] transition-all ${
                isToday
                  ? "bg-amber-500 text-black font-bold shadow-md shadow-amber-500/20"
                  : isPast
                    ? "bg-neutral-900 border border-neutral-800 text-neutral-300"
                    : "bg-neutral-950/40 text-neutral-600"
              }`}
            >
              {day}
            </div>
          );
        })}
      </div>
    </div>
  );
}
