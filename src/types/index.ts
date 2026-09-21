export type ActivityCategory = "body" | "focus" | "leisure";

export interface Activity {
  id: string;
  name: string;
  category: ActivityCategory;
  targetPerWeek: number;
  recommendedMinutes: number;
  weight: number;
  color: {
    bg: string;
    text: string;
    border: string;
  };
}

export interface QueueItem {
  instanceId: string; // ID único da ficha na fila
  activityId: string;
}

export interface ActivityLog {
  id: string;
  activityId: string;
  completedAt: string; // Data ISO
  type: "full" | "micro" | "skipped";
  notes?: string;
}

export interface OperationalRule {
  id: string;
  trigger: string;
  instruction: string;
  category?: "time" | "energy" | "delay";
}

export interface RoutineState {
  version: number;
  queue: QueueItem[];
  activities: Record<string, Activity>;
  logs: ActivityLog[];
  rules: OperationalRule[];
}
