import { useState } from "react";
import { View } from "react-native";
import * as Location from "expo-location";
import { useToast } from "./feedback";
import { Button, Text, TextField } from "./ui";
import { spacing } from "../theme";

export type Place = { city: string; latitude: number; longitude: number };

/**
 * Localisation minimale: position approximative uniquement lorsque l'utilisateur
 * le demande (jamais en arriere-plan). Le serveur arrondit a ~1 km avant stockage.
 */
export function LocationPicker({ value, onChange }: { value: Place | null; onChange: (place: Place) => void }) {
  const toast = useToast();
  const [busy, setBusy] = useState(false);
  const [query, setQuery] = useState("");

  const useMyLocation = async () => {
    setBusy(true);
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (!permission.granted) {
        toast.show("Position refusée : saisissez votre ville ci-dessous.", "error");
        return;
      }
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      const [address] = await Location.reverseGeocodeAsync(position.coords);
      onChange({
        city: address?.city ?? address?.subregion ?? address?.region ?? "Ma position",
        latitude: position.coords.latitude,
        longitude: position.coords.longitude,
      });
    } catch {
      toast.show("Impossible de déterminer votre position.", "error");
    } finally {
      setBusy(false);
    }
  };

  const search = async () => {
    const text = query.trim();
    if (text.length < 2) {
      return;
    }
    setBusy(true);
    try {
      const [result] = await Location.geocodeAsync(text);
      if (!result) {
        toast.show("Ville introuvable.", "error");
        return;
      }
      onChange({ city: text, latitude: result.latitude, longitude: result.longitude });
      setQuery("");
    } catch {
      toast.show("Recherche impossible pour le moment.", "error");
    } finally {
      setBusy(false);
    }
  };

  return (
    <View>
      {value ? (
        <Text style={{ marginBottom: spacing.md }} variant="label">
          📍 {value.city}
        </Text>
      ) : null}
      <Button icon="locate" loading={busy} onPress={useMyLocation} title="Utiliser ma position" variant="secondary" />
      <View style={{ height: spacing.md }} />
      <TextField
        label="Ou saisir une ville"
        onChangeText={setQuery}
        onSubmitEditing={search}
        returnKeyType="search"
        value={query}
      />
      <Button disabled={query.trim().length < 2} loading={busy} onPress={search} title="Rechercher" variant="ghost" />
      <Text muted variant="caption">
        Seule une position approximative (~1 km) est conservée. Les autres voient une distance arrondie, jamais votre
        position.
      </Text>
    </View>
  );
}
