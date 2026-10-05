import { Modal, StyleSheet, View } from "react-native";
import type { MatchPreview } from "../api/types";
import { Avatar, Button, Text } from "../shared/ui";
import { radii, spacing, useTheme } from "../theme";

export function MatchModal({ match, myName, myPhotoPath, onMessage, onClose }: { match: MatchPreview | null; myName: string; myPhotoPath?: string; onMessage: (match: MatchPreview) => void; onClose: () => void }) {
  const { colors } = useTheme();
  return (
    <Modal visible={!!match} transparent animationType="fade" onRequestClose={onClose} accessibilityViewIsModal>
      <View style={[styles.backdrop, { backgroundColor: colors.overlay }]}>
        {match ? (
          <View style={[styles.sheet, { backgroundColor: colors.surface }]} testID="match-modal">
            <Text variant="hero" tone="primary" center>
              C'est un match !
            </Text>
            <Text tone="muted" center>
              Vous et {match.user.firstName} vous plaisez mutuellement.
            </Text>
            <View style={styles.avatars}>
              <Avatar name={myName} path={myPhotoPath} size={96} />
              <Avatar name={match.user.firstName} path={match.user.photos[0]?.url} size={96} />
            </View>
            <Button testID="match-message" label={`Écrire à ${match.user.firstName}`} icon="chatbubble-ellipses-outline" onPress={() => onMessage(match)} />
            <Button testID="match-continue" label="Continuer à découvrir" variant="ghost" onPress={onClose} />
          </View>
        ) : null}
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  sheet: { width: "100%", maxWidth: 420, borderRadius: radii.xl, padding: spacing.xl, gap: spacing.lg },
  avatars: { flexDirection: "row", justifyContent: "center", gap: spacing.lg, marginVertical: spacing.md }
});
