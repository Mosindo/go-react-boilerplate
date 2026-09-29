import React, { useState } from "react";
import { View } from "react-native";
import { useDeviceLocation } from "../../hooks/useDeviceLocation";
import { useSaveLocation } from "../../hooks/useProfileData";
import { messageFromError } from "../../lib/errors";
import { strings } from "../../lib/strings";
import { Button } from "../ui/Button";
import { Card } from "../ui/Card";
import { Notice } from "../ui/Notice";
import { Text } from "../ui/Text";
import { type Theme } from "../ui/theme";
import { useThemedStyles } from "../ui/useThemedStyles";

type Props = {
  hasLocation: boolean;
  /** Called after the location has been saved on the server. */
  onSaved?: () => void;
  primaryLabel?: string;
};

const makeStyles = (t: Theme) => ({
  card: { gap: t.spacing.md },
  actions: { gap: t.spacing.sm }
});

/** Explains why we need location, asks for foreground permission, and stores a coarse position. */
export function LocationCard({ hasLocation, onSaved, primaryLabel }: Props) {
  const styles = useThemedStyles(makeStyles);
  const device = useDeviceLocation();
  const save = useSaveLocation();
  const [saved, setSaved] = useState(false);
  const [saveError, setSaveError] = useState<string | null>(null);

  const busy = device.status === "requesting" || save.isPending;

  async function handleShare() {
    setSaveError(null);
    setSaved(false);
    const coords = await device.request();
    if (!coords) {
      return;
    }
    try {
      await save.mutateAsync(coords);
      setSaved(true);
      onSaved?.();
    } catch (error) {
      setSaveError(messageFromError(error));
    }
  }

  return (
    <Card style={styles.card}>
      <Text weight="semibold">Why we ask</Text>
      <Text tone="muted">{strings.onboarding.locationWhy}</Text>

      {saved || (hasLocation && device.status === "idle") ? (
        <Notice
          description="You can update it any time, for example after you move."
          title={saved ? "Location saved" : "Your location is set"}
          tone="success"
        />
      ) : null}

      {device.status === "denied" ? (
        <Notice
          description={
            device.canAskAgain
              ? "Alba needs location to show people near you. Tap the button to try again."
              : `${strings.onboarding.locationDenied} On most phones: Settings, then Alba, then Location, then "While using the app".`
          }
          title="Location is turned off"
          tone="warning"
        />
      ) : null}
      {device.status === "unavailable" ? (
        <Notice description={strings.onboarding.locationUnavailable} title="Location unavailable" tone="warning" />
      ) : null}
      {saveError ? <Notice description={saveError} title="Could not save your location" tone="danger" /> : null}

      <View style={styles.actions}>
        <Button
          label={
            primaryLabel ??
            (device.status === "denied" || device.status === "unavailable"
              ? "Try again"
              : hasLocation || saved
                ? "Update my location"
                : "Share my approximate location")
          }
          loading={busy}
          onPress={() => void handleShare()}
          testID="location-share-button"
        />
        {device.status === "denied" && !device.canAskAgain ? (
          <Button label="Open settings" onPress={() => void device.openSettings()} variant="outline" />
        ) : null}
      </View>
    </Card>
  );
}
