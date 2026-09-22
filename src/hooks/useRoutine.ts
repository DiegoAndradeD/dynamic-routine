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

const getTodayKey = () => new Date().toISOString().slice(0, 10);

export function useRoutine() {
  const [state, setState] = useState<RoutineState>(() => {
    try {
      const saved = localStorage.getItem(STORAGE_KEY);
      if (saved) {
        const parsed = JSON.parse(saved);
        return {
          ...initialRoutineState,
          ...parsed,
          workRecords: parsed.workRecords || {},
        };
      }
    } catch (e) {
      console.error("Falha ao carregar localStorage:", e);
    }
    return initialRoutineState;
  });

  const [fileHandle, setFileHandle] = useState<any | null>(null);
  const [linkedFileName, setLinkedFileName] = useState<string | null>(null);
  const isInitialMount = useRef(true);

  // Timer do expediente de trabalho em tempo real
  const [isWorking, setIsWorking] = useState(false);

  // 1. Sincronização localStorage
  useEffect(() => {
    try {
      localStorage.setItem(STORAGE_KEY, JSON.stringify(state));
    } catch (e) {
      console.error("Falha ao sincronizar localStorage:", e);
    }
  }, [state]);

  // 2. Sincronização SSD compartilhado
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
        console.error("Falha ao gravar no SSD:", err);
      }
    };
    writeToDisk();
  }, [state, fileHandle]);

  // 3. Tick do cronômetro de trabalho (incrementa a cada segundo se ativo)
  useEffect(() => {
    if (!isWorking) return;

    const timer = setInterval(() => {
      const today = getTodayKey();
      setState((prev) => {
        const currentRecord = prev.workRecords[today] || {
          date: today,
          secondsWorked: 0,
          isCompleted: false,
        };

        const updatedSeconds = currentRecord.secondsWorked + 1;
        const isCompleted = updatedSeconds >= 8 * 3600;

        return {
          ...prev,
          workRecords: {
            ...prev.workRecords,
            [today]: {
              ...currentRecord,
              secondsWorked: updatedSeconds,
              isCompleted,
            },
          },
        };
      });
    }, 1000);

    return () => clearInterval(timer);
  }, [isWorking]);

  // Ações de Trabalho (Work)
  const toggleWork = useCallback(() => {
    setIsWorking((prev) => !prev);
  }, []);

  const addWorkSeconds = useCallback((seconds: number) => {
    const today = getTodayKey();
    setState((prev) => {
      const currentRecord = prev.workRecords[today] || {
        date: today,
        secondsWorked: 0,
        isCompleted: false,
      };
      const updatedSeconds = Math.max(0, currentRecord.secondsWorked + seconds);
      return {
        ...prev,
        workRecords: {
          ...prev.workRecords,
          [today]: {
            ...currentRecord,
            secondsWorked: updatedSeconds,
            isCompleted: updatedSeconds >= 8 * 3600,
          },
        },
      };
    });
  }, []);

  const completeWorkDay = useCallback(() => {
    const today = getTodayKey();
    setIsWorking(false);
    setState((prev) => ({
      ...prev,
      workRecords: {
        ...prev.workRecords,
        [today]: {
          date: today,
          secondsWorked: 8 * 3600,
          isCompleted: true,
        },
      },
    }));
  }, []);

  // Conectar com arquivo SSD
  const linkSharedFile = useCallback(async () => {
    if (!("showOpenFilePicker" in window)) {
      alert(
        "Navegador não suporta File System Access API. Use Chromium/Chrome/Brave ou use Importar/Exportar.",
      );
      return;
    }

    try {
      // @ts-ignore
      const [handle] = await window.showOpenFilePicker({
        types: [
          {
            description: "JSON de Estado",
            accept: { "application/json": [".json"] },
          },
        ],
        multiple: false,
      });
      const file = await handle.getFile();
      const content = await file.text();
      if (content.trim()) {
        const parsed = JSON.parse(content);
        setState(parsed);
      }
      setFileHandle(handle);
      setLinkedFileName(handle.name);
    } catch (err: any) {
      if (err.name !== "AbortError") console.error(err);
    }
  }, []);

  // Ações da Fila
  const completeCurrent = useCallback(
    (type: "full" | "micro" = "full", durationMinutes?: number) => {
      setState((prev) => {
        if (prev.queue.length === 0) return prev;
        const [current, ...rest] = prev.queue;
        const logEntry: ActivityLog = {
          id: `log-${Date.now()}`,
          activityId: current.activityId,
          completedAt: new Date().toISOString(),
          durationMinutes,
          type,
        };

        return {
          ...prev,
          queue: [
            ...rest,
            { instanceId: `q-${Date.now()}`, activityId: current.activityId },
          ],
          logs: [logEntry, ...prev.logs],
        };
      });
    },
    [],
  );

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

  // CRUD
  const saveActivity = useCallback((activity: Activity) => {
    setState((prev) => {
      const isNew = !prev.activities[activity.id];
      return {
        ...prev,
        activities: { ...prev.activities, [activity.id]: activity },
        queue: isNew
          ? [
              ...prev.queue,
              { instanceId: `q-${Date.now()}`, activityId: activity.id },
            ]
          : prev.queue,
      };
    });
  }, []);

  const deleteActivity = useCallback((activityId: string) => {
    setState((prev) => {
      const { [activityId]: _, ...remaining } = prev.activities;
      return {
        ...prev,
        activities: remaining,
        queue: prev.queue.filter((i) => i.activityId !== activityId),
      };
    });
  }, []);

  const saveRule = useCallback((rule: OperationalRule) => {
    setState((prev) => {
      const exists = prev.rules.some((r) => r.id === rule.id);
      return {
        ...prev,
        rules: exists
          ? prev.rules.map((r) => (r.id === rule.id ? rule : r))
          : [...prev.rules, rule],
      };
    });
  }, []);

  const deleteRule = useCallback((ruleId: string) => {
    setState((prev) => ({
      ...prev,
      rules: prev.rules.filter((r) => r.id !== ruleId),
    }));
  }, []);

  const exportStateToFile = useCallback(() => {
    const dataStr = JSON.stringify(state, null, 2);
    const blob = new Blob([dataStr], { type: "application/json" });
    const url = URL.createObjectURL(blob);
    const link = document.createElement("a");
    link.href = url;
    link.download = `routine_state_${getTodayKey()}.json`;
    link.click();
    URL.revokeObjectURL(url);
  }, [state]);

  const importStateFromFile = useCallback((file: File) => {
    const reader = new FileReader();
    reader.onload = (e) => {
      try {
        const json = JSON.parse(e.target?.result as string);
        if (json.activities && json.queue) setState(json);
      } catch {
        alert("Falha ao processar arquivo JSON.");
      }
    };
    reader.readAsText(file);
  }, []);

  // Métricas da Semana Vigente
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

    // Contagem de dias de trabalho cumpridos na semana
    let workDaysDone = 0;
    for (let i = 0; i < 7; i++) {
      const d = new Date(monday);
      d.setDate(monday.getDate() + i);
      const key = d.toISOString().slice(0, 10);
      if (
        state.workRecords[key]?.isCompleted ||
        (state.workRecords[key]?.secondsWorked || 0) >= 7.5 * 3600
      ) {
        workDaysDone++;
      }
    }

    return { activityStats: counts, workDaysDone };
  }, [state.logs, state.activities, state.workRecords]);

  const todayRecord = state.workRecords[getTodayKey()] || {
    date: getTodayKey(),
    secondsWorked: 0,
    isCompleted: false,
  };

  const currentQueueItem = state.queue[0] || null;
  const currentActivity = currentQueueItem
    ? state.activities[currentQueueItem.activityId]
    : null;

  return {
    state,
    currentActivity,
    weeklyStats,
    todayRecord,
    isWorking,
    isLinkedToFile: Boolean(fileHandle),
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
  };
}
