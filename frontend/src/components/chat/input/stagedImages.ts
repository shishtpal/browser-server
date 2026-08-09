/** A staged (picked/pasted) image waiting to be attached to the next message. */
export interface StagedImageInput {
  id: string;
  file: File;
  previewUrl: string;
  uploading: boolean;
  error?: string;
}

export const ALLOWED_IMAGE_TYPES = ['image/png', 'image/jpeg', 'image/webp', 'image/gif'];

export const isAllowedImage = (file: File): boolean => ALLOWED_IMAGE_TYPES.includes(file.type);

/**
 * Validate picked/pasted files against the server's attachment config.
 * Returns the files to accept and the rejection reasons (first is shown).
 */
export function validateImageFiles(
  files: File[],
  cfg: { max_image_bytes?: number; max_images?: number } | null | undefined,
  formatBytes: (n: number) => string,
): { valid: File[]; rejected: string[] } {
  const valid: File[] = [];
  const rejected: string[] = [];
  const maxBytes = cfg?.max_image_bytes ?? 5 * 1024 * 1024;
  const maxCount = cfg?.max_images ?? 5;
  for (const file of files) {
    if (!isAllowedImage(file)) {
      rejected.push(`${file.name || 'file'} is not a supported image`);
      continue;
    }
    if (file.size > maxBytes) {
      rejected.push(`${file.name || 'file'} exceeds the ${formatBytes(maxBytes)} image limit`);
      continue;
    }
    if (valid.length >= maxCount) {
      rejected.push(`Only ${maxCount} images are allowed per message`);
      break;
    }
    valid.push(file);
  }
  return { valid, rejected };
}
