export type Tutorial = 'users' | 'branding' | 'agents' | 'registry';

export interface TutorialProgressEvent {
  stepId: string;
  taskId: string;
  value?: any;
}

export interface TutorialProgressRequest extends TutorialProgressEvent {
  markCompleted?: boolean;
}

export interface TutorialProgress {
  tutorial: Tutorial;
  createdAt?: string;
  completedAt?: string;
  events?: TutorialProgressEvent[];
}
