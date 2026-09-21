import { useState } from "react";
import { X, Plus, Trash2 } from "lucide-react";
import type { Activity, OperationalRule, ActivityCategory } from "../types";

interface ManageModalProps {
  isOpen: boolean;
  onClose: () => void;
  activities: Record<string, Activity>;
  rules: OperationalRule[];
  onSaveActivity: (act: Activity) => void;
  onDeleteActivity: (id: string) => void;
  onSaveRule: (rule: OperationalRule) => void;
  onDeleteRule: (id: string) => void;
}

export function ManageModal({
  isOpen,
  onClose,
  activities,
  rules,
  onSaveActivity,
  onDeleteActivity,
  onSaveRule,
  onDeleteRule,
}: ManageModalProps) {
  const [tab, setTab] = useState<"activities" | "rules">("activities");

  // Estado para nova atividade
  const [actName, setActName] = useState("");
  const [actCategory, setActCategory] = useState<ActivityCategory>("focus");
  const [actMinutes, setActMinutes] = useState(60);
  const [actTarget, setActTarget] = useState(3);

  // Estado para nova regra
  const [ruleTrigger, setRuleTrigger] = useState("");
  const [ruleInstruction, setRuleInstruction] = useState("");

  if (!isOpen) return null;

  const handleCreateActivity = (e: React.FormEvent) => {
    e.preventDefault();
    if (!actName.trim()) return;

    const id =
      actName.toLowerCase().replace(/\s+/g, "_") +
      "_" +
      Date.now().toString().slice(-4);
    onSaveActivity({
      id,
      name: actName,
      category: actCategory,
      recommendedMinutes: actMinutes,
      targetPerWeek: actTarget,
      weight: actTarget,
      color: {
        bg: "bg-neutral-900",
        text: "text-neutral-200",
        border: "border-neutral-700",
      },
    });

    setActName("");
    setActMinutes(60);
  };

  const handleCreateRule = (e: React.FormEvent) => {
    e.preventDefault();
    if (!ruleTrigger.trim() || !ruleInstruction.trim()) return;

    onSaveRule({
      id: `r-${Date.now()}`,
      trigger: ruleTrigger,
      instruction: ruleInstruction,
    });

    setRuleTrigger("");
    setRuleInstruction("");
  };

  return (
    <div className="fixed inset-0 z-50 flex items-center justify-center bg-black/70 p-4 backdrop-blur-xs">
      <div className="w-full max-w-2xl rounded-2xl border border-neutral-800 bg-neutral-900 p-6 shadow-2xl flex flex-col max-h-[85vh]">
        <div className="flex items-center justify-between border-b border-neutral-800 pb-3">
          <div className="flex gap-2">
            <button
              onClick={() => setTab("activities")}
              className={`rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors ${
                tab === "activities"
                  ? "bg-neutral-800 text-white"
                  : "text-neutral-400 hover:text-neutral-200"
              }`}
            >
              Atividades
            </button>
            <button
              onClick={() => setTab("rules")}
              className={`rounded-lg px-3 py-1.5 text-xs font-semibold transition-colors ${
                tab === "rules"
                  ? "bg-neutral-800 text-white"
                  : "text-neutral-400 hover:text-neutral-200"
              }`}
            >
              Regras & Contingências
            </button>
          </div>
          <button
            onClick={onClose}
            className="rounded-lg p-1 text-neutral-400 hover:text-white"
          >
            <X className="h-5 w-5" />
          </button>
        </div>

        <div className="flex-1 overflow-y-auto py-4 space-y-6">
          {tab === "activities" ? (
            <div className="space-y-6">
              {/* Formulário Adicionar Atividade */}
              <form
                onSubmit={handleCreateActivity}
                className="rounded-xl border border-neutral-800 bg-neutral-950 p-4 space-y-3"
              >
                <h4 className="text-xs font-semibold uppercase text-neutral-400">
                  Nova Atividade
                </h4>
                <div className="grid grid-cols-1 sm:grid-cols-2 gap-3 text-xs">
                  <div>
                    <label className="text-neutral-400">Nome:</label>
                    <input
                      type="text"
                      value={actName}
                      onChange={(e) => setActName(e.target.value)}
                      placeholder="Ex: Treino de Boxe"
                      className="mt-1 w-full rounded-lg border border-neutral-800 bg-neutral-900 px-3 py-2 text-white"
                      required
                    />
                  </div>
                  <div>
                    <label className="text-neutral-400">Categoria:</label>
                    <select
                      value={actCategory}
                      onChange={(e) =>
                        setActCategory(e.target.value as ActivityCategory)
                      }
                      className="mt-1 w-full rounded-lg border border-neutral-800 bg-neutral-900 px-3 py-2 text-white"
                    >
                      <option value="body">Corpo & Físico</option>
                      <option value="focus">Foco & Intelecto</option>
                      <option value="leisure">Lazer & Descompressão</option>
                    </select>
                  </div>
                  <div>
                    <label className="text-neutral-400">
                      Duração Sugerida (min):
                    </label>
                    <input
                      type="number"
                      value={actMinutes}
                      onChange={(e) => setActMinutes(Number(e.target.value))}
                      className="mt-1 w-full rounded-lg border border-neutral-800 bg-neutral-900 px-3 py-2 text-white"
                      min={10}
                      max={300}
                    />
                  </div>
                  <div>
                    <label className="text-neutral-400">
                      Meta Semanal (vezes):
                    </label>
                    <input
                      type="number"
                      value={actTarget}
                      onChange={(e) => setActTarget(Number(e.target.value))}
                      className="mt-1 w-full rounded-lg border border-neutral-800 bg-neutral-900 px-3 py-2 text-white"
                      min={1}
                      max={7}
                    />
                  </div>
                </div>
                <button
                  type="submit"
                  className="flex items-center gap-1.5 rounded-lg bg-amber-600 px-3 py-2 text-xs font-semibold text-white hover:bg-amber-500"
                >
                  <Plus className="h-3.5 w-3.5" /> Adicionar na Esteira
                </button>
              </form>

              {/* Lista Existente */}
              <div className="space-y-2">
                <h4 className="text-xs font-semibold uppercase text-neutral-400">
                  Atividades Cadastradas
                </h4>
                {Object.values(activities).map((act) => (
                  <div
                    key={act.id}
                    className="flex items-center justify-between rounded-lg border border-neutral-800 bg-neutral-950 px-3 py-2 text-xs"
                  >
                    <div>
                      <span className="font-semibold text-white">
                        {act.name}
                      </span>
                      <span className="text-neutral-500 ml-2">
                        ({act.recommendedMinutes}m • {act.targetPerWeek}x/sem)
                      </span>
                    </div>
                    <button
                      onClick={() => onDeleteActivity(act.id)}
                      className="text-neutral-500 hover:text-rose-400 p-1"
                      title="Excluir atividade"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          ) : (
            <div className="space-y-6">
              {/* Formulário Nova Regra */}
              <form
                onSubmit={handleCreateRule}
                className="rounded-xl border border-neutral-800 bg-neutral-950 p-4 space-y-3"
              >
                <h4 className="text-xs font-semibold uppercase text-neutral-400">
                  Nova Regra de Manobra
                </h4>
                <div className="space-y-2 text-xs">
                  <div>
                    <label className="text-neutral-400">
                      Gatilho / Situação:
                    </label>
                    <input
                      type="text"
                      value={ruleTrigger}
                      onChange={(e) => setRuleTrigger(e.target.value)}
                      placeholder="Ex: Tive médico e cheguei às 21h"
                      className="mt-1 w-full rounded-lg border border-neutral-800 bg-neutral-900 px-3 py-2 text-white"
                      required
                    />
                  </div>
                  <div>
                    <label className="text-neutral-400">
                      Instrução / O que fazer:
                    </label>
                    <textarea
                      value={ruleInstruction}
                      onChange={(e) => setRuleInstruction(e.target.value)}
                      placeholder="Ex: Não faça tarefas de foco. Vá direto para descanso ou leitura de 20 min."
                      className="mt-1 w-full rounded-lg border border-neutral-800 bg-neutral-900 px-3 py-2 text-white resize-none h-16"
                      required
                    />
                  </div>
                </div>
                <button
                  type="submit"
                  className="flex items-center gap-1.5 rounded-lg bg-amber-600 px-3 py-2 text-xs font-semibold text-white hover:bg-amber-500"
                >
                  <Plus className="h-3.5 w-3.5" /> Adicionar Regra
                </button>
              </form>

              {/* Lista Existente */}
              <div className="space-y-2">
                <h4 className="text-xs font-semibold uppercase text-neutral-400">
                  Regras Cadastradas
                </h4>
                {rules.map((r) => (
                  <div
                    key={r.id}
                    className="flex items-start justify-between rounded-lg border border-neutral-800 bg-neutral-950 p-3 text-xs gap-2"
                  >
                    <div>
                      <p className="font-semibold text-amber-400/90">
                        {r.trigger}
                      </p>
                      <p className="text-neutral-400 mt-1">{r.instruction}</p>
                    </div>
                    <button
                      onClick={() => onDeleteRule(r.id)}
                      className="text-neutral-500 hover:text-rose-400 p-1 shrink-0"
                      title="Excluir regra"
                    >
                      <Trash2 className="h-3.5 w-3.5" />
                    </button>
                  </div>
                ))}
              </div>
            </div>
          )}
        </div>
      </div>
    </div>
  );
}
