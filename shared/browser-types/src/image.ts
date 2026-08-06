export interface AIImageModel {
  id: string;
  label: string;
  default: boolean;
  supports_editing: boolean;
  image_sizes: string[];
  aspect_ratios?: string[];
  max_images?: number;
  supports_seed?: boolean;
}

export interface AIImageConfig {
  enabled: boolean;
  default_provider: string;
  providers: Record<string, { models: AIImageModel[] }>;
}

export interface GeneratedImage {
  id: string;
  prompt: string;
  provider: string;
  model: string;
  image_size: string;
  content_type: string;
  filename: string;
  size_bytes: number;
  created_at: string;
}

export interface GenerateImageInput {
  prompt: string;
  provider?: string;
  model?: string;
  image_size?: string;
  aspect_ratio?: string;
  n?: number;
  seed?: number;
  source_image_ids?: string[];
}

export interface GenerateImageResponse {
  image: GeneratedImage;
  images: GeneratedImage[];
  url: string;
}
