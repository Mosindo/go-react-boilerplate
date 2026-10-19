import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import type { Match, OwnProfile } from "../api/types";
import { Avatar, Button, Text } from "../ui";
import { radius, spacing } from "../theme/tokens";
import { useTheme } from "../theme/ThemeProvider";

type Props = { match: Match | null; onClose: () => void; onMessage: (match: Match) => void };

export function MatchModal({ match, onClose, onMessage }: Props) {
  const { colors } = useTheme();
  const own = useQueryClient().getQueryData<OwnProfile | null>(["profile"]);
  if (!match) return null;
  return (
    <Modal transparent animationType="fade" visible onRequestClose={onClose} accessibilityViewIsModal>
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]} testID="match-modal">
        <View style={[styles.card, { backgroundColor: colors.surface }]}>
          <Text variant="display" tone="accent" style={styles.center}>
            C&apos;est un match !
          </Text>
          <View style={styles.avatars}>
            <Avatar photoId={own?.photos[0]?.id} name={own?.firstName} size={88} />
            <Text variant="title" tone="accent" accessibilityElementsHidden>
              ♥
            </Text>
            <Avatar photoId={match.profile.photos[0]?.id} name={match.profile.firstName} size={88} />
          </View>
          <Text style={styles.center} tone="muted">
            Vous et {match.profile.firstName} vous plaisez mutuellement. Lancez la conversation !
          </Text>
          <Button
            label={`Écrire à ${match.profile.firstName}`}
            onPress={() => onMessage(match)}
            testID="match-message"
          />
          <Button label="Continuer" variant="ghost" onPress={onClose} testID="match-close" />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  card: { width: "100%", maxWidth: 420, borderRadius: radius.lg, padding: spacing.xl, gap: spacing.lg },
  center: { textAlign: "center" },
  avatars: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: spacing.lg }
});
