import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import type { PublicProfile } from "../api/types";
import { Button, Text, colors, radii, spacing } from "../shared/ui";
import { PhotoImage } from "./PhotoImage";

type Props = {
  profile: PublicProfile | null;
  onMessage: () => void;
  onClose: () => void;
};

export function MatchModal({ onClose, onMessage, profile }: Props) {
  return (
    <Modal animationType="fade" onRequestClose={onClose} transparent visible={Boolean(profile)}>
      <View style={styles.scrim}>
        {profile ? (
          <View accessibilityViewIsModal style={styles.card} testID="match-modal">
            <PhotoImage fallbackLabel={profile.firstName} path={profile.photos[0]?.url} style={styles.photo} />
            <Text style={styles.center} variant="title" weight="bold">
              It's a match!
            </Text>
            <Text style={styles.center} tone="muted">
              You and {profile.firstName} liked each other.
            </Text>
            <Button label={`Message ${profile.firstName}`} onPress={onMessage} testID="match-message" />
            <Button label="Keep swiping" onPress={onClose} testID="match-close" variant="ghost" />
          </View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  scrim: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl, backgroundColor: colors.scrim },
  card: { width: "100%", maxWidth: 380, gap: spacing.md, padding: spacing.xl, borderRadius: radii.xl, backgroundColor: colors.background, alignItems: "stretch" },
  photo: { alignSelf: "center", width: 140, height: 140, borderRadius: 70 },
  center: { textAlign: "center" }
});
