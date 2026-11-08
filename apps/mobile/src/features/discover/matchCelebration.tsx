import React, { createContext, useCallback, useContext, useMemo, useState } from "react";
import { Modal, StyleSheet, View } from "react-native";
import { LinearGradient } from "expo-linear-gradient";
import type { Match } from "../../api/types";
import { navigationRef } from "../../navigation/ref";
import { spacing, useTheme } from "../../theme/theme";
import { Button } from "../../ui/Button";
import { Avatar } from "../../ui/Photo";
import { Text } from "../../ui/Text";

const celebrated = new Set<string>();
export const wasCelebrated = (matchId: string) => celebrated.has(matchId);

type Value = { show: (match: Match) => void };
const Ctx = createContext<Value>({ show: () => undefined });

/** Full-screen "It's a match" moment shown right after the user's own like creates a match. */
export function MatchCelebrationProvider({ children }: { children: React.ReactNode }) {
  const { colors } = useTheme();
  const [match, setMatch] = useState<Match | null>(null);
  const show = useCallback((m: Match) => {
    celebrated.add(m.id);
    setMatch(m);
  }, []);
  const value = useMemo(() => ({ show }), [show]);

  const close = () => setMatch(null);
  const openChat = () => {
    if (!match) return;
    const { conversationId, user, id: matchId } = match;
    setMatch(null);
    if (navigationRef.isReady()) navigationRef.navigate("Chat", { conversationId, matchId, peer: user });
  };

  return (
    <Ctx.Provider value={value}>
      {children}
      <Modal visible={!!match} transparent animationType="fade" onRequestClose={close}>
        <LinearGradient colors={[colors.primary, "#5A1E14"]} style={styles.fill}>
          {match ? (
            <View style={styles.content} accessibilityViewIsModal>
              <Text variant="display" style={styles.title} accessibilityRole="header">
                C&apos;est un match !
              </Text>
              <Text style={styles.subtitle}>Toi et {match.user.firstName} vous êtes plu mutuellement.</Text>
              <View style={styles.avatar}>
                <Avatar path={match.user.photos[0]?.url} name={match.user.firstName} size={140} />
              </View>
              <Button
                title={`Écrire à ${match.user.firstName}`}
                onPress={openChat}
                variant="secondary"
                testID="match-send-message"
              />
              <Button
                title="Continuer à découvrir"
                onPress={close}
                variant="ghost"
                style={styles.keep}
                testID="match-keep-going"
              />
            </View>
          ) : null}
        </LinearGradient>
      </Modal>
    </Ctx.Provider>
  );
}

export const useMatchCelebration = () => useContext(Ctx);

const styles = StyleSheet.create({
  fill: { flex: 1, justifyContent: "center" },
  content: { padding: spacing.lg, alignItems: "stretch" },
  title: { color: "#FFFFFF", textAlign: "center" },
  subtitle: { color: "#F8E4DC", textAlign: "center", marginTop: spacing.xs },
  avatar: { alignSelf: "center", marginVertical: spacing.xl },
  keep: { marginTop: spacing.xs }
});
