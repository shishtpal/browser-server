export interface AITTSVoice {
  id: string;
  label: string;
}

export interface AITTSModel {
  id: string;
  label: string;
  default: boolean;
  voices?: AITTSVoice[];
}

export interface AITTSConfig {
  enabled: boolean;
  default_provider: string;
  providers: Record<string, { models: AITTSModel[] }>;
}

export interface GeneratedSpeech {
  id: string;
  text: string;
  provider: string;
  model: string;
  voice: string;
  content_type: string;
  filename: string;
  size_bytes: number;
  generation_id?: string;
  created_at: string;
}

export interface GenerateSpeechInput {
  text: string;
  provider?: string;
  model?: string;
  voice?: string;
}

export interface GenerateSpeechResponse {
  speech: GeneratedSpeech;
  url: string;
}
