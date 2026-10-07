import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { fetchMyMainPhoto } from "../../api/discover";
import { Button, Text, radii, spacing } from "../../shared/ui";
import type { Match } from "../types";
import { PhotoImage } from "../ui/PhotoImage";
import { usePalette } from "../ui/palette";

type Props = {
  match: Match | null;
  onMessage: (match: Match) => void;
  onDismiss: () => void;
};

export function MatchOverlay({ match, onMessage, onDismiss }: Props) {
  const p = usePalette();
  const mine = useQuery({
    queryKey: ["my-main-photo"],
    queryFn: fetchMyMainPhoto,
    staleTime: 5 * 60_000,
    enabled: match !== null
  });
  return (
    <Modal animationType="fade" onRequestClose={onDismiss} transparent visible={match !== null}>
      <View style={[styles.backdrop, { backgroundColor: p.scrim }]} testID="match-overlay">
        {match ? (
          <View style={[styles.sheet, { backgroundColor: p.surface }]}>
            <Text style={styles.title} variant="title" weight="bold">
              It's a match
            </Text>
            <Text style={styles.sub} tone="muted">
              You and {match.user.firstName} like each other.
            </Text>
            <View style={styles.photos}>
              <View
                style={[
                  styles.photoWrap,
                  styles.left,
                  { borderColor: p.surface, backgroundColor: p.surfaceMuted }
                ]}
              >
                <PhotoImage accessibilityLabel="Your photo" photo={mine.data ?? null} style={styles.photo} />
              </View>
              <View
                style={[
                  styles.photoWrap,
                  styles.right,
                  { borderColor: p.surface, backgroundColor: p.surfaceMuted }
                ]}
              >
                <PhotoImage
                  accessibilityLabel={`${match.user.firstName}'s photo`}
                  photo={match.user.photo}
                  style={styles.photo}
                />
              </View>
            </View>
            <Button
              accessibilityLabel={`Send a message to ${match.user.firstName}`}
              fullWidth
              label="Send a message"
              onPress={() => onMessage(match)}
              testID="match-send-message"
            />
            <Button
              accessibilityLabel="Keep swiping"
              fullWidth
              label="Keep swiping"
              onPress={onDismiss}
              testID="match-keep-swiping"
              variant="ghost"
            />
          </View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  sheet: {
    width: "100%",
    maxWidth: 420,
    borderRadius: radii.xl,
    padding: spacing.xl,
    gap: spacing.md,
    alignItems: "center"
  },
  title: { textAlign: "center" },
  sub: { textAlign: "center" },
  photos: { flexDirection: "row", justifyContent: "center", marginVertical: spacing.lg },
  photoWrap: { width: 120, height: 150, borderRadius: radii.lg, borderWidth: 4, overflow: "hidden" },
  left: { transform: [{ rotate: "-6deg" }], marginRight: -14 },
  right: { transform: [{ rotate: "6deg" }] },
  photo: { width: "100%", height: "100%" }
});
