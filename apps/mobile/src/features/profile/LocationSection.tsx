import React, { useState } from "react";
import { View } from "react-native";
import { profileApi } from "../../api/profile";
import { useProfileMutation } from "../../state/queries";
import { useTheme } from "../../theme/ThemeProvider";
import { Button, Text, TextField } from "../../ui";
import { describeError } from "../../utils/errors";
import { LocationError, locateCity, locateMe, type ResolvedLocation } from "./location";

type Props = { city: string; hasLocation: boolean; onDone?: () => void };

export function LocationSection({ city, hasLocation, onDone }: Props) {
  const { spacing } = useTheme();
  const [query, setQuery] = useState("");
  const [cityLabel, setCityLabel] = useState(city);
  const [error, setError] = useState<string | null>(null);
  const [locating, setLocating] = useState(false);
  const save = useProfileMutation((loc: ResolvedLocation) =>
    profileApi.updateLocation(loc.latitude, loc.longitude, loc.city)
  );

  const resolve = async (resolver: () => Promise<ResolvedLocation>) => {
    setError(null);
    setLocating(true);
    try {
      const loc = await resolver();
      const label = loc.city || cityLabel.trim();
      await save.mutateAsync({ ...loc, city: label });
      setCityLabel(label);
      onDone?.();
    } catch (e) {
      setError(e instanceof LocationError ? e.message : describeError(e, "Localisation impossible. Réessayez."));
    } finally {
      setLocating(false);
    }
  };

  return (
    <View style={{ gap: spacing.md }}>
      {hasLocation ? (
        <Text tone="muted" testID="location-current">
          Position enregistrée{cityLabel ? ` : ${cityLabel}` : ""}. Seule une distance approximative est montrée aux
          autres.
        </Text>
      ) : null}
      <Button
        label={hasLocation ? "Mettre à jour ma position" : "Utiliser ma position"}
        icon="navigate-outline"
        onPress={() => resolve(locateMe)}
        loading={locating}
        testID="location-use-current"
      />
      <TextField
        label="Ville affichée"
        value={cityLabel}
        onChangeText={setCityLabel}
        placeholder="Ex. Lyon"
        hint="Visible sur votre profil. Sur mobile, vous pouvez aussi rechercher une ville."
        testID="location-city-label"
      />
      <View style={{ flexDirection: "row", gap: spacing.sm, alignItems: "flex-end" }}>
        <View style={{ flex: 1 }}>
          <TextField label="Rechercher une ville" value={query} onChangeText={setQuery} placeholder="Ex. Nantes" />
        </View>
        <Button
          label="OK"
          size="md"
          variant="secondary"
          disabled={query.trim().length < 2 || locating}
          onPress={() => resolve(() => locateCity(query))}
          style={{ marginBottom: 2 }}
        />
      </View>
      {error ? (
        <Text tone="danger" accessibilityLiveRegion="polite">
          {error}
        </Text>
      ) : null}
    </View>
  );
}
