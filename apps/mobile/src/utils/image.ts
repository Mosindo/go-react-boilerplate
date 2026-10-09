import * as ImageManipulator from 'expo-image-manipulator';

export const MAX_PHOTO_SIDE = 1600;

/** Returns a resize action keeping the aspect ratio, or null when already small enough. */
export function computeResize(
  width: number,
  height: number,
  maxSide = MAX_PHOTO_SIDE,
): { width: number } | { height: number } | null {
  if (!width || !height || Math.max(width, height) <= maxSide) return null;
  return width >= height ? { width: maxSide } : { height: maxSide };
}

/** Resizes (long side <= 1600px) and re-encodes as JPEG (quality 0.8) before upload. */
export async function prepareImageForUpload(
  uri: string,
  width: number,
  height: number,
): Promise<string> {
  const resize = computeResize(width, height);
  const result = await ImageManipulator.manipulateAsync(uri, resize ? [{ resize }] : [], {
    compress: 0.8,
    format: ImageManipulator.SaveFormat.JPEG,
  });
  return result.uri;
}
