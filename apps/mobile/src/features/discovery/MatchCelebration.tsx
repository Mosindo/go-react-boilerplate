import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMyProfile } from "../../hooks/queries";
import { celebrateMatch, useCelebration } from "../../hooks/useRealtime";
import type { MainStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Avatar, Button, Text } from "../../ui";

export function MatchCelebration() {
  const match = useCelebration();
  const theme = useTheme();
  const me = useMyProfile();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  if (!match) return null;
  const close = () => celebrateMatch(null);
  return (
    <Modal animationType="fade" onRequestClose={close} transparent visible>
      <View style={[styles.backdrop, { backgroundColor: theme.colors.scrim }]}>
        <View
          accessibilityViewIsModal
          style={[styles.card, { backgroundColor: theme.colors.background, borderRadius: theme.radii.xl }]}
          testID="match-celebration"
        >
          <View style={styles.avatars}>
            <Avatar name={me.data?.profile?.firstName ?? "Moi"} photo={me.data?.photos[0]} ring size={96} />
            <View style={styles.overlap}>
              <Avatar name={match.profile.firstName} photo={match.profile.photos[0]} ring size={96} />
            </View>
          </View>
          <Text align="center" variant="display">
            C’est réciproque !
          </Text>
          <Text align="center" tone="muted">
            Vous et {match.profile.firstName} vous plaisez mutuellement. Lancez la conversation avec un mot sincère.
          </Text>
          <Button
            fullWidth
            label={`Écrire à ${match.profile.firstName}`}
            onPress={() => {
              close();
              navigation.navigate("Chat", { matchId: match.id, name: match.profile.firstName });
            }}
            testID="celebration-chat"
          />
          <Button fullWidth label="Continuer à découvrir" onPress={close} variant="ghost" />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: 24 },
  card: { width: "100%", maxWidth: 420, padding: 28, gap: 14, alignItems: "center" },
  avatars: { flexDirection: "row", marginBottom: 8 },
  overlap: { marginLeft: -18 }
});
