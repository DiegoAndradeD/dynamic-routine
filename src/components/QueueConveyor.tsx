import type { QueueItem, Activity } from "../types";

interface QueueConveyorProps {
  queue: QueueItem[];
  activities: Record<string, Activity>;
}

export function QueueConveyor({ queue, activities }: QueueConveyorProps) {
  // Exibe a partir do 2º item (já que o 1º está no card ativo) até os próximos 7 itens
  const upcoming = queue.slice(1, 8);

  if (upcoming.length === 0) return null;

  return (
    <section className="space-y-2">
      <h3 className="text-xs font-semibold uppercase tracking-wider text-neutral-500">
        Na Esteira (Sequência Imediata)
      </h3>
      <div className="flex gap-2 overflow-x-auto pb-2 scrollbar-none">
        {upcoming.map((item, index) => {
          const act = activities[item.activityId];
          if (!act) return null;

          return (
            <div
              key={item.instanceId}
              className={`flex shrink-0 items-center gap-2 rounded-xl border ${act.color.border} ${act.color.bg} px-3 py-2 text-xs backdrop-blur-sm`}
            >
              <span className="flex h-5 w-5 items-center justify-center rounded-full bg-neutral-900/80 text-[10px] font-mono text-neutral-400">
                +{index + 1}
              </span>
              <span className={`font-medium ${act.color.text}`}>
                {act.name}
              </span>
            </div>
          );
        })}
      </div>
    </section>
  );
}
