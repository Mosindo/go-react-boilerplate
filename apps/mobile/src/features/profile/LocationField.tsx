import React, { useState } from "react";
import { Linking, View } from "react-native";
import { strings } from "../../i18n/strings";
import { Button, Input, Notice, Text, spacing } from "../../shared/ui";
import { requestDeviceLocation } from "./location";

type Coords = { latitude: number; longitude: number };

type LocationFieldProps = {
  coords: Coords | null;
  city: string;
  cityError?: string | null;
  onCoords: (coords: Coords) => void;
  onCity: (city: string) => void;
};

/** "Use my location" with permission handling, plus the manual city fallback. */
export function LocationField({ city, cityError, coords, onCity, onCoords }: LocationFieldProps) {
  const [busy, setBusy] = useState(false);
  const [problem, setProblem] = useState<"denied" | "blocked" | "unavailable" | null>(null);

  async function locate() {
    setBusy(true);
    setProblem(null);
    const result = await requestDeviceLocation();
    setBusy(false);
    if (!result.ok) {
      setProblem(result.reason);
      return;
    }
    onCoords({ latitude: result.latitude, longitude: result.longitude });
    if (result.city && city.trim().length === 0) {
      onCity(result.city);
    }
  }

  return (
    <View style={{ gap: spacing.md }}>
      <Button
        label={coords ? strings.location.update : strings.location.use}
        loading={busy}
        onPress={() => void locate()}
        variant={coords ? "outline" : "primary"}
      />
      {coords ? (
        <Notice description={strings.location.savedBody} title={strings.location.savedTitle} tone="success" />
      ) : null}
      {problem ? (
        <Notice
          description={
            problem === "blocked"
              ? strings.location.blocked
              : problem === "denied"
                ? strings.location.denied
                : strings.location.unavailable
          }
          title={strings.location.problemTitle}
          tone="warning"
        />
      ) : null}
      {problem === "blocked" ? (
        <Button
          label={strings.common.openSettings}
          onPress={() => void Linking.openSettings()}
          size="sm"
          variant="outline"
        />
      ) : null}
      <Text tone="muted" variant="caption">
        {strings.location.manualHelp}
      </Text>
      <Input
        autoCapitalize="words"
        error={cityError}
        label={strings.location.cityLabel}
        onChangeText={onCity}
        value={city}
      />
    </View>
  );
}
