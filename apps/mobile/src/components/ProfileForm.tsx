import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import * as Location from "expo-location";
import type { Gender, OwnProfile, ProfileInput } from "../api/types";
import { GENDER_LABELS } from "../lib/format";
import { Button, FormField, Input, Notice, Text, spacing } from "../shared/ui";
import { Chip } from "./Chip";
import { InterestPicker } from "./InterestPicker";

export function toInput(profile: OwnProfile): ProfileInput {
  return {
    firstName: profile.firstName,
    gender: profile.gender,
    bio: profile.bio,
    city: profile.city,
    location: profile.location,
    interests: profile.interests.map((i) => i.slug),
    isDiscoverable: profile.exists ? profile.isDiscoverable : true,
    showDistance: profile.exists ? profile.showDistance : true
  };
}

type ProfileFormProps = {
  value: ProfileInput;
  onChange: (next: ProfileInput) => void;
  /** Hide the interests picker (used when it lives on a separate onboarding step). */
  sections?: Array<"identity" | "about" | "interests" | "location">;
};

export function ProfileForm({ value, onChange, sections = ["identity", "about", "interests", "location"] }: ProfileFormProps) {
  const [locating, setLocating] = useState(false);
  const [locationError, setLocationError] = useState<string | null>(null);
  const set = <K extends keyof ProfileInput>(key: K, v: ProfileInput[K]) => onChange({ ...value, [key]: v });

  const useMyLocation = async () => {
    setLocating(true);
    setLocationError(null);
    try {
      const permission = await Location.requestForegroundPermissionsAsync();
      if (permission.status !== "granted") {
        setLocationError("Autorisation refusée. Vous pouvez saisir votre ville manuellement.");
        return;
      }
      // Balanced accuracy is enough: the server only keeps coordinates rounded to ~1 km.
      const position = await Location.getCurrentPositionAsync({ accuracy: Location.Accuracy.Balanced });
      let city = value.city;
      try {
        const [place] = await Location.reverseGeocodeAsync(position.coords);
        city = place?.city ?? place?.subregion ?? city;
      } catch {
        // Reverse geocoding is optional.
      }
      onChange({ ...value, city, location: { latitude: position.coords.latitude, longitude: position.coords.longitude } });
    } catch {
      setLocationError("Position indisponible pour le moment.");
    } finally {
      setLocating(false);
    }
  };

  return (
    <View style={styles.container}>
      {sections.includes("identity") ? (
        <>
          <Input autoCapitalize="words" label="Prénom" maxLength={40} onChangeText={(t) => set("firstName", t)} value={value.firstName} />
          <FormField label="Je suis">
            <View style={styles.row}>
              {(Object.keys(GENDER_LABELS) as Gender[]).map((g) => (
                <Chip key={g} label={GENDER_LABELS[g]} onPress={() => set("gender", g)} selected={value.gender === g} />
              ))}
            </View>
          </FormField>
        </>
      ) : null}

      {sections.includes("about") ? (
        <Input
          helperText={`${value.bio.length}/500`}
          label="À propos de moi"
          maxLength={500}
          multiline
          onChangeText={(t) => set("bio", t)}
          placeholder="Quelques mots sur vous…"
          value={value.bio}
        />
      ) : null}

      {sections.includes("interests") ? (
        <FormField label="Centres d'intérêt">
          <InterestPicker onChange={(slugs) => set("interests", slugs)} value={value.interests} />
        </FormField>
      ) : null}

      {sections.includes("location") ? (
        <View style={styles.container}>
          <Input autoCapitalize="words" label="Ville" maxLength={80} onChangeText={(t) => set("city", t)} value={value.city} />
          <Button
            label={value.location ? "Actualiser ma position" : "Utiliser ma position"}
            loading={locating}
            onPress={useMyLocation}
            size="sm"
            variant="secondary"
          />
          <Text tone="muted" variant="caption">
            Votre position exacte n'est jamais affichée ni stockée : seule une zone d'environ 1 km est conservée et les autres voient une
            distance approximative.
          </Text>
          {value.location ? <Notice title="Position enregistrée (zone approximative)" tone="success" /> : null}
          {locationError ? <Notice title={locationError} tone="warning" /> : null}
        </View>
      ) : null}
    </View>
  );
}

const styles = StyleSheet.create({
  container: { gap: spacing.md },
  row: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
