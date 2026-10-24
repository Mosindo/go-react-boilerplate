import { Platform } from "react-native";
import * as ImageManipulator from "expo-image-manipulator";
import * as ImagePicker from "expo-image-picker";

const MAX_EDGE = 1440;

export type PickResult = { form: FormData } | { cancelled: true } | { error: string };

/**
 * Lets the user choose a photo, then shrinks and re-encodes it as JPEG on the device
 * (this also bakes in the EXIF rotation) so uploads stay small even on slow networks.
 * The server independently re-validates and re-encodes every upload.
 */
export async function pickPhotoForm(): Promise<PickResult> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted && permission.canAskAgain === false) {
    return { error: "Autorisez l'accès à vos photos dans les réglages de votre téléphone." };
  }

  const picked = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [3, 4],
    quality: 1,
  });
  if (picked.canceled || !picked.assets[0]) return { cancelled: true };

  const asset = picked.assets[0];
  const longest = Math.max(asset.width, asset.height);
  const actions =
    longest > MAX_EDGE
      ? [{ resize: asset.width >= asset.height ? { width: MAX_EDGE } : { height: MAX_EDGE } }]
      : [];
  const result = await ImageManipulator.manipulateAsync(asset.uri, actions, {
    compress: 0.85,
    format: ImageManipulator.SaveFormat.JPEG,
  });

  const form = new FormData();
  if (Platform.OS === "web") {
    const blob = await (await fetch(result.uri)).blob();
    form.append("photo", blob, "photo.jpg");
  } else {
    form.append("photo", {
      uri: result.uri,
      name: "photo.jpg",
      type: "image/jpeg",
    } as unknown as Blob);
  }
  return { form };
}
