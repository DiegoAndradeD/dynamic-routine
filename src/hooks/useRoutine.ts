import { useState, useEffect, useCallback, useMemo, useRef } from "react";
import type {
  RoutineState,
  Activity,
  OperationalRule,
  ActivityLog,
  QueueItem,
} from "../types";
import { initialRoutineState } from "../data/initialState";

const STORAGE_KEY = "dynamic_routine_state_v1";

export function useRoutine() {
  const [state, setState] = useState<RoutineState>(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        return JSON.parse(saved) as RoutineState;
      }
    } catch (e) {
      console.error("Falha ao carregar estado do localStorage:", e);
    }
    return initialRoutineState;
  });

  // Handle do arquivo no SSD (File System Access API)
  const [fileHandle, setFileHandle] = useState<any | null>(null);
  const [linkedFileName, setLinkedFileName] = useState<string | null>(null);
  const isInitialMount = useRef(true);

  // 1. Sempre sincroniza com o localStorage (como fallback instantâneo)
  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    } catch (e) {
      console.error("Falha ao sincronizar com localStorage:", e);
    }
  }, [state]);

  // 2. Sempre que o estado mudar, se houver arquivo vinculado no SSD, grava direto nele
  useEffect(() => {
    if (isInitialMount.current) {
      isInitialMount.current = false;
      return;
    }

    if (!fileHandle) return;

    const writeToDisk = async () => {
      try {
        const writable = await fileHandle.createWritable();
        await writable.write(JSON.stringify(state, null, 2));
        await writable.close();
      } catch (err) {
        console.error("Falha ao gravar no arquivo do SSD compartilhado:", err);
      }
    };

    writeToDisk();
  }, [state, fileHandle]);

  // 3. Função para conectar ao arquivo routine_state.json no SSD
  const linkSharedFile = useCallback(async () => {
    if (!("showOpenFilePicker" in window)) {
      alert(
        "Seu navegador atual não suporta a File System Access API nativa. Recomendamos usar um navegador baseado em Chromium (Brave, Chrome, Edge) ou utilizar os botões Importar/Exportar.",
      );
      return;
    }

    try {
      // @ts-ignore - API nativa Chromium
      const [handle] = await window.showOpenFilePicker({
        types: [
          {
            description: "Estado da Rotina (JSON)",
            accept: { "application/json": [".json"] },
          },
        ],
        multiple: false,
      });

      const file = await handle.getFile();
      const content = await file.text();

      if (content.trim()) {
        const parsed = JSON.parse(content) as RoutineState;
        if (parsed.activities && parsed.queue) {
          setState(parsed);
        }
      }

      setFileHandle(handle);
      setLinkedFileName(handle.name);
    } catch (err: any) {
      // Usuário cancelou ou fechou a janela
      if (err.name !== "AbortError") {
        console.error("Erro ao vincular arquivo:", err);
      }
    }
  }, []);

  // --- Ações da Fila ---

  const completeCurrent = useCallback((type: "full" | "micro" = "full") => {
    setState((prev) => {
      if (prev.queue.length === 0) return prev;

      const [current, ...rest] = prev.queue;
      const logEntry: ActivityLog = {
        id: `log-${Date.now()}`,
        activityId: current.activityId,
        completedAt: new Date().toISOString(),
        type,
      };

      const updatedQueue: QueueItem[] = [
        ...rest,
        { instanceId: `q-${Date.now()}`, activityId: current.activityId },
      ];

      return {
        ...prev,
        queue: updatedQueue,
        logs: [logEntry, ...prev.logs],
      };
    });
  }, []);

  const skipCurrent = useCallback(() => {
    setState((prev) => {
      if (prev.queue.length <= 1) return prev;

      const [current, next, ...rest] = prev.queue;
      return {
        ...prev,
        queue: [next, current, ...rest],
      };
    });
  }, []);

  // --- Gestão de Atividades ---

  const saveActivity = useCallback((activity: Activity) => {
    setState((prev) => {
      const isNew = !prev.activities[activity.id];
      const updatedActivities = { ...prev.activities, [activity.id]: activity };

      let updatedQueue = [...prev.queue];
      if (isNew) {
        updatedQueue.push({
          instanceId: `q-${Date.now()}`,
          activityId: activity.id,
        });
      }

      return {
        ...prev,
        activities: updatedActivities,
        queue: updatedQueue,
      };
    });
  }, []);

  const deleteActivity = useCallback((activityId: string) => {
    setState((prev) => {
      const { [activityId]: _, ...remainingActivities } = prev.activities;
      const filteredQueue = prev.queue.filter(
        (item) => item.activityId !== activityId,
      );

      return {
        ...prev,
        activities: remainingActivities,
        queue: filteredQueue,
      };
    });
  }, []);

  // --- Gestão de Regras ---

  const saveRule = useCallback((rule: OperationalRule) => {
    setState((prev) => {
      const exists = prev.rules.some((r) => r.id === rule.id);
      const updatedRules = exists
        ? prev.rules.map((r) => (r.id === rule.id ? rule : r))
        : [...prev.rules, rule];

      return {
        ...prev,
        rules: updatedRules,
      };
    });
  }, []);

  const deleteRule = useCallback((ruleId: string) => {
    setState((prev) => ({
      ...prev,
      rules: prev.rules.filter((r) => r.id !== ruleId),
    }));
  }, []);

  // --- Import / Export Manual ---

  const exportStateToFile = useCallback(() => {
    const dataStr = JSON.stringify(state, null, 2);
    const blob = new Blob([dataStr], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `routine_state_${new Date().toISOString().slice(0, 10)}.json`;
    document.body.appendChild(link);
    link.click();
    document.body.removeChild(link);
    URL.revokeObjectURL(url);
  }, [state]);

  const importStateFromFile = useCallback((file: File) => {
    const reader = new FileReader();
    reader.onload = (event) => {
      try {
        const json = JSON.parse(event.target?.result as string) as RoutineState;
        if (json.activities && json.queue && Array.isArray(json.rules)) {
          setState(json);
        } else {
          alert("Estrutura de JSON incompatível.");
        }
      } catch (err) {
        alert("Falha ao processar arquivo JSON.");
      }
    };
    reader.readAsText(file);
  }, []);

  // --- Placar Semanal ---

  const weeklyStats = useMemo(() => {
    const now = new Date();
    const dayOfWeek = now.getDay();
    const distanceToMonday = dayOfWeek === 0 ? -6 : 1 - dayOfWeek;
    const monday = new Date(now);
    monday.setDate(now.getDate() + distanceToMonday);
    monday.setHours(0, 0, 0, 0);

    const counts: Record<
      string,
      { full: number; micro: number; total: number }
    > = {};

    Object.keys(state.activities).forEach((id) => {
      counts[id] = { full: 0, micro: 0, total: 0 };
    });

    state.logs.forEach((log) => {
      const logDate = new Date(log.completedAt);
      if (logDate >= monday && counts[log.activityId]) {
        if (log.type === "full") counts[log.activityId].full += 1;
        if (log.type === "micro") counts[log.activityId].micro += 1;
        counts[log.activityId].total += 1;
      }
    });

    return counts;
  }, [state.logs, state.activities]);

  const currentQueueItem = state.queue[0] || null;
  const currentActivity = currentQueueItem
    ? state.activities[currentQueueItem.activityId]
    : null;

  return {
    state,
    currentQueueItem,
    currentActivity,
    weeklyStats,
    linkedFileName,
    isLinkedToFile: Boolean(fileHandle),
    linkSharedFile,
    completeCurrent,
    skipCurrent,
    saveActivity,
    deleteActivity,
    saveRule,
    deleteRule,
    exportStateToFile,
    importStateFromFile,
  };
}
