import React from "react";
import { Modal, View } from "react-native";
import type { UserSummary } from "../../api/types";
import { makeStyles } from "../../theme";
import { Avatar, Button, modalAnimation, Text } from "../../ui";

type Props = {
  match: { user: UserSummary } | null;
  myPhotoUrl: string | null;
  myName: string;
  onMessage: () => void;
  onClose: () => void;
};

export function MatchModal({ match, myPhotoUrl, myName, onMessage, onClose }: Props) {
  const styles = useStyles();
  return (
    <Modal visible={match !== null} transparent animationType={modalAnimation("fade")} onRequestClose={onClose}>
      <View style={styles.backdrop} accessibilityViewIsModal>
        {match ? (
          <View style={styles.card}>
            <Text variant="micro" tone="primary">
              NOUVEAU MATCH
            </Text>
            <Text variant="display" align="center" accessibilityRole="header">
              C&apos;est réciproque
            </Text>
            <Text tone="muted" align="center">
              Vous et {match.user.firstName} vous plaisez. Lancez la conversation !
            </Text>
            <View style={styles.avatars}>
              <View style={styles.left}>
                <Avatar uri={myPhotoUrl} name={myName} size={104} ring />
              </View>
              <View style={styles.right}>
                <Avatar uri={match.user.photoUrl} name={match.user.firstName} size={104} ring />
              </View>
            </View>
            <Button
              testID="match-message"
              label={`Écrire à ${match.user.firstName}`}
              icon="chatbubble-ellipses-outline"
              size="lg"
              fullWidth
              onPress={onMessage}
            />
            <Button testID="match-continue" label="Continuer à découvrir" variant="ghost" onPress={onClose} />
          </View>
        ) : null}
      </View>
    </Modal>
  );
}

const useStyles = makeStyles((t) => ({
  backdrop: { flex: 1, backgroundColor: t.colors.overlay, alignItems: "center", justifyContent: "center", padding: t.spacing.xl },
  card: {
    width: "100%",
    maxWidth: 420,
    alignItems: "center",
    gap: t.spacing.md,
    backgroundColor: t.colors.surfaceRaised,
    borderRadius: t.radii.xl,
    padding: t.spacing.xl
  },
  avatars: { flexDirection: "row", marginVertical: t.spacing.lg },
  left: { transform: [{ rotate: "-6deg" }], marginRight: -18 },
  right: { transform: [{ rotate: "6deg" }] }
}));
