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
  instanceId: string;
  activityId: string;
}

export interface ActivityLog {
  id: string;
  activityId: string;
  completedAt: string; // ISO Date
  durationMinutes?: number;
  type: "full" | "micro" | "skipped";
  notes?: string;
}

export interface OperationalRule {
  id: string;
  trigger: string;
  instruction: string;
  category?: "time" | "energy" | "delay";
}

// Histórico de trabalho diário (8h)
export interface DailyWorkRecord {
  date: string; // YYYY-MM-DD
  secondsWorked: number;
  isCompleted: boolean;
}

export interface RoutineState {
  version: number;
  queue: QueueItem[];
  activities: Record<string, Activity>;
  logs: ActivityLog[];
  rules: OperationalRule[];
  workRecords: Record<string, DailyWorkRecord>; // Chave: 'YYYY-MM-DD'
}
