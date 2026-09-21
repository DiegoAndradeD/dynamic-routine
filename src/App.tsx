import { useState } from "react";
import { useRoutine } from "./hooks/useRoutine";
import { Header } from "./components/Header";
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
    isLinkedToFile,
    linkedFileName,
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
      <main className="mx-auto max-w-4xl p-4 sm:p-6 lg:p-8 space-y-6">
        {/* Topo com ações de SSD, JSON e Configurações */}
        <Header
          onExport={exportStateToFile}
          onImport={importStateFromFile}
          onOpenManage={() => setIsManageOpen(true)}
          onLinkSSD={linkSharedFile}
          isLinkedToFile={isLinkedToFile}
          linkedFileName={linkedFileName}
        />

        {/* Card do Próximo Passo */}
        <ActiveTaskCard
          activity={currentActivity}
          onComplete={() => completeCurrent("full")}
          onMicroblock={() => completeCurrent("micro")}
          onSkip={skipCurrent}
        />

        {/* Fila Horizontal Visível */}
        <QueueConveyor queue={state.queue} activities={state.activities} />

        {/* Placar Semanal Sem Cobrança Rígida */}
        <WeeklyScoreboard activities={state.activities} stats={weeklyStats} />

        {/* Manual de Regras de Contingência */}
        <RulesSection rules={state.rules} />

        {/* Modal de Gestão de Atividades e Regras */}
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
