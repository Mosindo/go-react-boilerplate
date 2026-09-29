import { useCallback } from "react";
import * as ImagePicker from "expo-image-picker";
import type { UploadableFile } from "../api/photos";
import { guessImageMime, isTooLarge, uploadFileName } from "../lib/photos";

export type PickedPhoto = { file: UploadableFile; tooLarge: boolean };

/** Opens the system photo library. Resolves to null when the user cancels. */
export function usePhotoPicker() {
  return useCallback(async (): Promise<PickedPhoto | null> => {
    const result = await ImagePicker.launchImageLibraryAsync({
      mediaTypes: ["images"],
      allowsEditing: true,
      aspect: [4, 5],
      quality: 0.8
    });
    if (result.canceled) {
      return null;
    }
    const asset = result.assets[0];
    if (!asset) {
      return null;
    }
    return {
      file: {
        uri: asset.uri,
        name: uploadFileName(asset.uri, asset.fileName),
        type: guessImageMime(asset.uri, asset.mimeType)
      },
      tooLarge: isTooLarge(asset.fileSize)
    };
  }, []);
}
