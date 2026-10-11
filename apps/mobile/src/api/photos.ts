import { Platform } from 'react-native';
import { http } from './client';
import type { Photo } from './types';

/** A local image chosen with expo-image-picker. */
export interface LocalImage {
  uri: string;
  fileName?: string | null;
  mimeType?: string | null;
}

/** React Native's FormData accepts `{uri, name, type}` objects; the web needs a real Blob. */
async function buildForm(image: LocalImage): Promise<FormData> {
  const type = image.mimeType === 'image/png' ? 'image/png' : 'image/jpeg';
  const name = image.fileName ?? (type === 'image/png' ? 'photo.png' : 'photo.jpg');
  const form = new FormData();
  if (Platform.OS === 'web') {
    const blob = await (await fetch(image.uri)).blob();
    form.append('file', blob, name);
  } else {
    form.append('file', { uri: image.uri, name, type } as unknown as Blob);
  }
  return form;
}

export const photosApi = {
  async upload(image: LocalImage): Promise<Photo> {
    return http.request<Photo>('POST', '/me/photos', { form: await buildForm(image) });
  },

  async replace(photoId: string, image: LocalImage): Promise<Photo | undefined> {
    return http.request<Photo | undefined>('PUT', `/me/photos/${encodeURIComponent(photoId)}`, {
      form: await buildForm(image),
    });
  },

  async reorder(photoIds: string[]): Promise<Photo[]> {
    const res = await http.request<{ photos: Photo[] }>('PUT', '/me/photos/order', {
      body: { photoIds },
    });
    return res.photos;
  },

  remove(photoId: string): Promise<void> {
    return http.request<void>('DELETE', `/me/photos/${encodeURIComponent(photoId)}`);
  },
};
