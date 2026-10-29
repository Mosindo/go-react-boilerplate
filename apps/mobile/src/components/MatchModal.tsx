import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import React, { useEffect, useRef } from "react";
import { Animated, Modal, StyleSheet, Text, View } from "react-native";
import type { RootStackParamList } from "../navigation/types";
import { useRealtime } from "../state/RealtimeProvider";
import { fontSizes, spacing } from "../theme/palette";
import { useTheme } from "../theme/ThemeProvider";
import { Avatar } from "./Avatar";
import { Button } from "./Button";

export function MatchModal() {
  const theme = useTheme();
  const { newMatch, dismissMatch } = useRealtime();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const scale = useRef(new Animated.Value(0.8)).current;

  useEffect(() => {
    if (!newMatch) return;
    scale.setValue(0.8);
    Animated.spring(scale, { toValue: 1, friction: 5, useNativeDriver: true }).start();
  }, [newMatch, scale]);

  if (!newMatch) return null;
  const { user, conversationId } = newMatch;

  return (
    <Modal animationType="fade" onRequestClose={dismissMatch} transparent visible>
      <View style={[styles.backdrop, { backgroundColor: theme.overlay }]}>
        <Animated.View
          style={[styles.card, { backgroundColor: theme.background, transform: [{ scale }] }]}
        >
          <Text accessibilityRole="header" style={[styles.title, { color: theme.primary }]}>
            C&apos;est un match !
          </Text>
          <Text style={[styles.body, { color: theme.textMuted }]}>
            Vous et {user.firstName} vous êtes plu mutuellement.
          </Text>
          <View style={styles.avatar}>
            <Avatar name={user.firstName} size={120} url={user.photoUrl} />
          </View>
          <Button
            onPress={() => {
              dismissMatch();
              navigation.navigate("Chat", { conversationId, user });
            }}
            testID="match-send-message"
            title={`Écrire à ${user.firstName}`}
          />
          <Button
            onPress={dismissMatch}
            style={styles.later}
            testID="match-dismiss"
            title="Continuer à découvrir"
            variant="ghost"
          />
        </Animated.View>
      </View>
    </Modal>
  );
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  card: {
    width: "100%",
    maxWidth: 380,
    borderRadius: 28,
    padding: spacing.xl,
    alignItems: "stretch"
  },
  title: { fontSize: 32, fontWeight: "800", textAlign: "center", fontStyle: "italic" },
  body: { fontSize: fontSizes.md, textAlign: "center", marginTop: spacing.sm },
  avatar: { alignItems: "center", marginVertical: spacing.xl },
  later: { marginTop: spacing.sm }
});
