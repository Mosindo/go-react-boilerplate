import * as ImagePicker from "expo-image-picker";
import { ImageManipulator, SaveFormat } from "expo-image-manipulator";
import * as Location from "expo-location";
import { Platform } from "react-native";

export class PermissionDeniedError extends Error {}

/**
 * Lets the user pick a photo, then resizes it (1440 px max) and re-encodes
 * it as JPEG before upload to save bandwidth. The server re-validates and
 * re-encodes again, stripping any metadata.
 */
export async function pickPhoto(source: "library" | "camera"): Promise<string | null> {
  if (source === "camera") {
    const perm = await ImagePicker.requestCameraPermissionsAsync();
    if (!perm.granted) throw new PermissionDeniedError("Autorisez l'accès à l'appareil photo dans les réglages.");
  } else if (Platform.OS !== "web") {
    const perm = await ImagePicker.requestMediaLibraryPermissionsAsync();
    if (!perm.granted) throw new PermissionDeniedError("Autorisez l'accès à vos photos dans les réglages.");
  }
  const options: ImagePicker.ImagePickerOptions = {
    mediaTypes: ["images"],
    allowsEditing: Platform.OS !== "web",
    aspect: [4, 5],
    quality: 1,
    exif: false
  };
  const result = source === "camera" ? await ImagePicker.launchCameraAsync(options) : await ImagePicker.launchImageLibraryAsync(options);
  if (result.canceled || !result.assets?.[0]) return null;
  const asset = result.assets[0];
  const context = ImageManipulator.manipulate(asset.uri);
  if ((asset.width ?? 0) > 1440) context.resize({ width: 1440 });
  const image = await context.renderAsync();
  const saved = await image.saveAsync({ compress: 0.82, format: SaveFormat.JPEG });
  return saved.uri;
}

export type ApproxLocation = { latitude: number; longitude: number; city: string };

/**
 * Reads a coarse, one-off position (no background tracking) and the city
 * name. Coordinates are coarsened again server-side to ~1 km.
 */
export async function readApproximateLocation(): Promise<ApproxLocation> {
  const perm = await Location.requestForegroundPermissionsAsync();
  if (!perm.granted) throw new PermissionDeniedError("Autorisez la localisation pour voir des profils proches.");
  const pos = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Low });
  const latitude = Math.round(pos.coords.latitude * 100) / 100;
  const longitude = Math.round(pos.coords.longitude * 100) / 100;
  let city = "";
  if (Platform.OS !== "web") {
    try {
      const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
      city = place?.city ?? place?.subregion ?? place?.region ?? "";
    } catch {
      city = "";
    }
  }
  return { latitude, longitude, city: city.slice(0, 80) };
}
