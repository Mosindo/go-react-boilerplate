import React, { useState } from "react";
import { StyleSheet, View } from "react-native";
import { getErrorMessage } from "../../api/client";
import { LocationError, requestApproximateLocation, type Coordinates } from "../../lib/location";
import { spacing } from "../../theme";
import { Button, Input, Text, toast } from "../../ui";

type Props = {
  city: string;
  onCityChange: (city: string) => void;
  error?: string | null;
  coordinates: Coordinates | null;
  onCoordinates: (value: Coordinates | null) => void;
  /** True when the server already holds a location for this profile. */
  hasSavedLocation?: boolean;
};

/**
 * City input plus an optional "Use my location" button. Permission is requested only when the
 * user taps it. The coordinates are returned to the parent, which sends them to PUT /profile/location.
 */
export function LocationField({ city, onCityChange, error, coordinates, onCoordinates, hasSavedLocation }: Props) {
  const [busy, setBusy] = useState(false);

  async function useMyLocation() {
    setBusy(true);
    try {
      onCoordinates(await requestApproximateLocation());
      toast.success("Location saved. Distances will be shown to people nearby.");
    } catch (e) {
      toast.error(e instanceof LocationError ? e.message : getErrorMessage(e));
    } finally {
      setBusy(false);
    }
  }

  const shared = coordinates !== null || hasSavedLocation;
  return (
    <View style={styles.wrap}>
      <Input
        autoCapitalize="words"
        error={error}
        label="City"
        maxLength={80}
        onChangeText={onCityChange}
        textContentType="addressCity"
        value={city}
      />
      <Button
        icon="locate-outline"
        label={shared ? "Update my location" : "Use my location"}
        loading={busy}
        onPress={useMyLocation}
        variant="secondary"
      />
      <Text tone="muted" variant="small">
        Optional. Lets Lumen show people near you. We store an approximate spot (about 1 km), never your exact position.
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({ wrap: { gap: spacing.xs } });
