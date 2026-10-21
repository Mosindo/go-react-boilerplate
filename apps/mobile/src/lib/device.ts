import * as ImageManipulator from "expo-image-manipulator";
import * as ImagePicker from "expo-image-picker";
import * as Location from "expo-location";

export type ApproxLocation = { latitude: number; longitude: number; city?: string };

const round2 = (v: number) => Math.round(v * 100) / 100;

/**
 * Asks for the foreground position and returns it rounded to ~1 km: the exact position never
 * leaves the device. Returns null when the permission is refused or unavailable.
 */
export async function getApproxLocation(): Promise<ApproxLocation | null> {
  try {
    const permission = await Location.requestForegroundPermissionsAsync();
    if (permission.status !== "granted") return null;
    const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
    const latitude = round2(position.coords.latitude);
    const longitude = round2(position.coords.longitude);
    let city: string | undefined;
    try {
      const [place] = await Location.reverseGeocodeAsync({ latitude, longitude });
      city = place?.city ?? place?.subregion ?? undefined;
    } catch {
      city = undefined;
    }
    return { latitude, longitude, city };
  } catch {
    return null;
  }
}

/**
 * Lets the user pick a photo, then resizes / compresses it on the device (1080 px, JPEG) so uploads
 * stay small. Returns null if cancelled.
 */
export async function pickPhoto(): Promise<string | null> {
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [3, 4],
    quality: 1
  });
  const asset = result.canceled ? undefined : result.assets[0];
  if (!asset) return null;
  const processed = await ImageManipulator.manipulateAsync(asset.uri, [{ resize: { width: Math.min(asset.width, 1080) } }], {
    compress: 0.85,
    format: ImageManipulator.SaveFormat.JPEG
  });
  return processed.uri;
}
