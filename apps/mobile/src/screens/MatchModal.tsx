import React from "react";
import { Modal, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import type { Conversation } from "../api/types";
import { chatApi } from "../api/endpoints";
import { queryKeys } from "../lib/queryKeys";
import type { MainStackParams } from "../navigation/types";
import { useRealtime } from "../realtime/RealtimeProvider";
import { radii, useTheme } from "../theme";
import { Button, Text } from "../ui";

/** Full-screen celebration shown when a match happens (from a swipe or live). */
export function MatchModal() {
  const t = useTheme();
  const qc = useQueryClient();
  const { pendingMatch, dismissMatch } = useRealtime();
  const stack = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  if (!pendingMatch) return null;

  const openChat = async () => {
    const m = pendingMatch;
    dismissMatch();
    const convos = await qc.fetchQuery({
      queryKey: queryKeys.conversations,
      queryFn: chatApi.conversations,
      staleTime: 0
    });
    const c: Conversation | undefined = convos.find((x) => x.id === m.conversationId);
    stack.navigate("Chat", {
      conversationId: m.conversationId,
      peerId: m.peerId,
      peerName: c?.peer.firstName ?? m.peerName,
      peerPhotoId: c?.peer.mainPhotoId ?? null
    });
  };

  return (
    <Modal visible transparent animationType="fade" onRequestClose={dismissMatch}>
      <View style={[styles.backdrop, { backgroundColor: t.overlay }]}>
        <View style={[styles.card, { backgroundColor: t.surface }]} accessibilityViewIsModal>
          <View style={[styles.icon, { backgroundColor: t.primarySoft }]}>
            <Ionicons name="heart" size={40} color={t.primary} />
          </View>
          <Text variant="display" tone="primary" style={styles.center}>
            C'est un match !
          </Text>
          <Text tone="muted" style={styles.center}>
            Vous et {pendingMatch.peerName} vous plaisez mutuellement.
          </Text>
          <Button label={`Écrire à ${pendingMatch.peerName}`} onPress={() => void openChat()} />
          <Button label="Continuer à découvrir" variant="ghost" onPress={dismissMatch} />
        </View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, justifyContent: "center", padding: 24 },
  card: { borderRadius: radii.xl, padding: 24, gap: 12, alignItems: "stretch" },
  icon: {
    width: 84,
    height: 84,
    borderRadius: 42,
    alignItems: "center",
    justifyContent: "center",
    alignSelf: "center"
  },
  center: { textAlign: "center" },
  hidden: { height: 0 }
});
