import { useState } from "react";
import { useRoutine } from "./hooks/useRoutine";
import { Header } from "./components/Header";
import { TimeContextBar } from "./components/TimeContextBar";
import { WorkTracker } from "./components/WorkTracker";
import { ActiveTaskCard } from "./components/ActiveTaskCard";
import { QueueConveyor } from "./components/QueueConveyor";
import { WeeklyScoreboard } from "./components/WeeklyScoreboard";
import { RulesSection } from "./components/RulesSection";
import { ManageModal } from "./components/ManageModal";

export function App() {
  const {
    state,
    currentActivity,
    weeklyStats,
    todayRecord,
    isWorking,
    isLinkedToFile,
    linkedFileName,
    toggleWork,
    addWorkSeconds,
    completeWorkDay,
    linkSharedFile,
    completeCurrent,
    skipCurrent,
    saveActivity,
    deleteActivity,
    saveRule,
    deleteRule,
    exportStateToFile,
    importStateFromFile,
  } = useRoutine();

  const [isManageOpen, setIsManageOpen] = useState(false);

  return (
    <div className="min-h-screen bg-neutral-950 text-neutral-100 font-sans antialiased selection:bg-amber-500 selection:text-black">
      <main className="mx-auto max-w-4xl p-4 sm:p-6 lg:p-8 space-y-5">
        {/* 1. Header (Ações de arquivo e configuração) */}
        <Header
          onExport={exportStateToFile}
          onImport={importStateFromFile}
          onOpenManage={() => setIsManageOpen(true)}
          onLinkSSD={linkSharedFile}
          isLinkedToFile={isLinkedToFile}
          linkedFileName={linkedFileName}
        />

        {/* 2. Barra de Contexto Temporal (Relógio + Dia da semana) */}
        <TimeContextBar />

        {/* 3. Expediente de Trabalho (Acumulador de 8h flexíveis) */}
        <WorkTracker
          record={todayRecord}
          isWorking={isWorking}
          onToggleWork={toggleWork}
          onAddSeconds={addWorkSeconds}
          onComplete={completeWorkDay}
          workDaysDoneThisWeek={weeklyStats.workDaysDone}
        />

        {/* 4. Card da Próxima Atividade com Cronômetro de Foco */}
        <ActiveTaskCard
          activity={currentActivity}
          onComplete={(duration) => completeCurrent("full", duration)}
          onMicroblock={() => completeCurrent("micro", 20)}
          onSkip={skipCurrent}
        />

        {/* 5. A Esteira Contínua */}
        <QueueConveyor queue={state.queue} activities={state.activities} />

        {/* 6. Placar Semanal */}
        <WeeklyScoreboard
          activities={state.activities}
          stats={weeklyStats.activityStats}
        />

        {/* 7. Manual de Manobras e Regras */}
        <RulesSection rules={state.rules} />

        {/* Modal de Gestão */}
        <ManageModal
          isOpen={isManageOpen}
          onClose={() => setIsManageOpen(false)}
          activities={state.activities}
          rules={state.rules}
          onSaveActivity={saveActivity}
          onDeleteActivity={deleteActivity}
          onSaveRule={saveRule}
          onDeleteRule={deleteRule}
        />
      </main>
    </div>
  );
}

export default App;
