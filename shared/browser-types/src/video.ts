export type VideoParamType = 'text' | 'textarea' | 'number' | 'select' | 'boolean' | 'image_urls';

/** One tweakable generation option, declared by a provider's model config. The
 * UI renders a field for every spec, so provider-specific option sets require
 * no UI changes. */
export interface VideoParamSpec {
  key: string;
  label: string;
  type: VideoParamType;
  group?: string;
  default?: unknown;
  required?: boolean;
  min?: number;
  max?: number;
  step?: number;
  options?: string[];
  help?: string;
}

export interface AIVideoModel {
  id: string;
  label: string;
  default: boolean;
  parameters: VideoParamSpec[];
}

export interface AIVideoConfig {
  enabled: boolean;
  default_provider: string;
  providers: Record<string, { models: AIVideoModel[] }>;
}

export type VideoStatus = 'queued' | 'in_progress' | 'completed' | 'failed';

export interface GeneratedVideo {
  id: string;
  task_id?: string;
  prompt: string;
  provider: string;
  model: string;
  params: string;
  content_type: string;
  filename: string;
  size_bytes: number;
  status: VideoStatus;
  progress: number;
  video_url?: string;
  error_message?: string;
  seconds?: number;
  size?: string;
  created_at: string;
  completed_at?: string;
}

export interface GenerateVideoInput {
  prompt: string;
  provider?: string;
  model?: string;
  params?: Record<string, unknown>;
}

export interface GenerateVideoResponse {
  video: GeneratedVideo;
  url: string;
}
