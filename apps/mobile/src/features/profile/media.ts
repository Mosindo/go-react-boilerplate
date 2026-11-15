import * as ImagePicker from "expo-image-picker";
import { ImageManipulator, SaveFormat } from "expo-image-manipulator";
import * as Location from "expo-location";

export type PickResult = { uri: string } | { error: string } | null;

/**
 * Lets the user pick a photo, then resizes and re-encodes it on device
 * (smaller upload, orientation applied). The server re-validates anyway.
 */
export async function pickPhoto(): Promise<PickResult> {
  const permission = await ImagePicker.requestMediaLibraryPermissionsAsync();
  if (!permission.granted) {
    return { error: "Autorisez l'accès à vos photos dans les réglages pour en ajouter." };
  }
  const result = await ImagePicker.launchImageLibraryAsync({
    mediaTypes: ["images"],
    allowsEditing: true,
    aspect: [3, 4],
    quality: 1,
    exif: false
  });
  if (result.canceled || !result.assets[0]) return null;
  const asset = result.assets[0];
  try {
    const context = ImageManipulator.manipulate(asset.uri);
    if (asset.width > 1440) context.resize({ width: 1440 });
    const rendered = await context.renderAsync();
    const saved = await rendered.saveAsync({ compress: 0.82, format: SaveFormat.JPEG });
    return { uri: saved.uri };
  } catch {
    return { uri: asset.uri };
  }
}

export type LocationResult = { latitude: number; longitude: number; city: string } | { error: string };

/** One-shot, foreground-only, balanced-accuracy position plus city name. */
export async function currentApproximateLocation(): Promise<LocationResult> {
  const permission = await Location.requestForegroundPermissionsAsync();
  if (!permission.granted) {
    return { error: "Sans la localisation, indiquez simplement votre ville." };
  }
  try {
    const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
    let city = "";
    try {
      const [place] = await Location.reverseGeocodeAsync(position.coords);
      city = place?.city ?? place?.subregion ?? place?.region ?? "";
    } catch {
      city = "";
    }
    return { latitude: position.coords.latitude, longitude: position.coords.longitude, city };
  } catch {
    return { error: "Position indisponible pour le moment." };
  }
}
