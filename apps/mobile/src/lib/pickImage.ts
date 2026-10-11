import * as ImagePicker from 'expo-image-picker';
import type { LocalImage } from '../api';

const MAX_BYTES = 8 * 1024 * 1024;

export type PickResult =
  | { status: 'picked'; image: LocalImage }
  | { status: 'cancelled' }
  | { status: 'too_large' }
  | { status: 'unavailable' };

/** Opens the system photo library. Quality < 1 makes iOS re-encode HEIC to JPEG. */
export async function pickImage(): Promise<PickResult> {
  try {
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ['images'],
      allowsMultipleSelection: false,
      quality: 0.85,
      exif: false,
    });
    if (result.canceled || result.assets.length === 0) return { status: 'cancelled' };
    const asset = result.assets[0];
    if (asset.fileSize !== undefined && asset.fileSize > MAX_BYTES) return { status: 'too_large' };
    return {
      status: 'picked',
      image: { uri: asset.uri, fileName: asset.fileName, mimeType: asset.mimeType },
    };
  } catch {
    return { status: 'unavailable' };
  }
}
