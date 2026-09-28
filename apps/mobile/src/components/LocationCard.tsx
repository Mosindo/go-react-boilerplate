import React, { useState } from "react";
import { Linking, StyleSheet, View } from "react-native";
import { errorMessage } from "../api/client";
import { requestDeviceLocation } from "../lib/deviceLocation";
import { useSaveLocation } from "../hooks/useProfile";
import { showToast } from "../shared/feedback";
import { Button, Input, Notice, Text } from "../shared/ui";
import { spacing } from "../theme";

type Coordinates = { latitude: number; longitude: number };
type Phase =
  | { kind: "idle" }
  | { kind: "locating" }
  | { kind: "ready"; coords: Coordinates }
  | { kind: "denied"; canAskAgain: boolean }
  | { kind: "unavailable" };

export type LocationCardProps = { currentLabel?: string; onSaved?: () => void; actionLabel?: string };

/**
 * Location is required for discovery: coordinates always come from the device. The label is only a
 * display name, so when reverse geocoding fails the user types a city while the coordinates are kept.
 */
export function LocationCard({ currentLabel, onSaved, actionLabel = "Use my location" }: LocationCardProps) {
  const [phase, setPhase] = useState<Phase>({ kind: "idle" });
  const [label, setLabel] = useState(currentLabel ?? "");
  const [labelError, setLabelError] = useState<string | null>(null);
  const save = useSaveLocation();

  const locate = async () => {
    setPhase({ kind: "locating" });
    const result = await requestDeviceLocation();
    if (result.ok) {
      if (result.label) {
        setLabel(result.label);
      }
      setPhase({ kind: "ready", coords: { latitude: result.latitude, longitude: result.longitude } });
    } else if (result.reason === "denied") {
      setPhase({ kind: "denied", canAskAgain: result.canAskAgain });
    } else {
      setPhase({ kind: "unavailable" });
    }
  };

  const submit = (coords: Coordinates) => {
    const trimmed = label.trim();
    if (!trimmed) {
      setLabelError("Enter the name of your city.");
      return;
    }
    setLabelError(null);
    save.mutate(
      { ...coords, label: trimmed },
      {
        onSuccess: () => {
          showToast("Location saved.", "success");
          setPhase({ kind: "idle" });
          onSaved?.();
        }
      }
    );
  };

  return (
    <View style={styles.wrap}>
      {currentLabel && phase.kind === "idle" ? <Text>Current location: {currentLabel}</Text> : null}
      <Text tone="muted">
        Your exact position is never shown. Other people only see an approximate distance, and coordinates are rounded
        to about 1 km.
      </Text>
      {phase.kind === "denied" ? (
        <Notice
          kind="error"
          message="Location is required for discovery: we need it to show you people nearby. Allow location access to continue."
        />
      ) : null}
      {phase.kind === "unavailable" ? (
        <Notice
          kind="error"
          message="We could not read your position. Check that location services are on and try again."
        />
      ) : null}
      {phase.kind === "ready" ? (
        <>
          <Input
            error={labelError}
            hint="Shown on your profile. Use a city or area name."
            label="City"
            onChangeText={setLabel}
            value={label}
          />
          {save.isError ? <Notice kind="error" message={errorMessage(save.error)} /> : null}
          <Button label="Save location" loading={save.isPending} onPress={() => submit(phase.coords)} />
        </>
      ) : (
        <>
          <Button
            label={phase.kind === "denied" || phase.kind === "unavailable" ? "Try again" : actionLabel}
            loading={phase.kind === "locating"}
            onPress={locate}
          />
          {phase.kind === "denied" && !phase.canAskAgain ? (
            <Button label="Open settings" onPress={() => void Linking.openSettings()} variant="secondary" />
          ) : null}
        </>
      )}
    </View>
  );
}

const styles = StyleSheet.create({ wrap: { gap: spacing.md } });
