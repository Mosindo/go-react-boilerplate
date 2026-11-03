import React, { createContext, useCallback, useContext, useMemo, useState } from "react";
import { Modal, StyleSheet, Text, View } from "react-native";
import type { Card } from "../../api/types";
import { radius, spacing, useTheme } from "../../theme";
import { AuthImage } from "../../ui/AuthImage";
import { Button } from "../../ui/components";

type MatchInfo = { user: Card; conversationId: string; matchId: string };

type MatchApi = {
  showMatch: (info: MatchInfo) => void;
  /** Registered by the navigator so the overlay can open the chat. */
  setOpenChat: (open: ((info: MatchInfo) => void) | null) => void;
};

const MatchContext = createContext<MatchApi | null>(null);

export function MatchProvider({ children }: { children: React.ReactNode }) {
  const theme = useTheme();
  const [info, setInfo] = useState<MatchInfo | null>(null);
  const [openChat, setOpenChatState] = useState<((info: MatchInfo) => void) | null>(null);

  const showMatch = useCallback((next: MatchInfo) => setInfo(next), []);
  const setOpenChat = useCallback((open: ((info: MatchInfo) => void) | null) => setOpenChatState(() => open), []);
  const api = useMemo(() => ({ showMatch, setOpenChat }), [showMatch, setOpenChat]);

  return (
    <MatchContext.Provider value={api}>
      {children}
      <Modal visible={info !== null} transparent animationType="fade" onRequestClose={() => setInfo(null)}>
        <View style={[styles.backdrop, { backgroundColor: theme.overlay }]}>
          {info ? (
            <View accessibilityViewIsModal style={[styles.card, { backgroundColor: theme.surface }]} testID="match-overlay">
              <Text style={[styles.heading, { color: theme.primary }]}>C&apos;est un match !</Text>
              <AuthImage path={info.user.photos[0]?.url} label={`Photo de ${info.user.firstName}`} style={styles.photo} />
              <Text style={[styles.text, { color: theme.text }]}>{`Vous et ${info.user.firstName} vous plaisez mutuellement.`}</Text>
              <Button
                title="Envoyer un message"
                testID="match-send-message"
                onPress={() => {
                  const target = info;
                  setInfo(null);
                  openChat?.(target);
                }}
              />
              <Button title="Continuer" variant="ghost" onPress={() => setInfo(null)} testID="match-continue" />
            </View>
          ) : null}
        </View>
      </Modal>
    </MatchContext.Provider>
  );
}

export function useMatch(): MatchApi {
  const ctx = useContext(MatchContext);
  if (!ctx) {
    throw new Error("useMatch must be used inside MatchProvider");
  }
  return ctx;
}

const styles = StyleSheet.create({
  backdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  card: { width: "100%", maxWidth: 380, borderRadius: radius.lg, padding: spacing.xl, alignItems: "center", gap: spacing.lg },
  heading: { fontSize: 28, fontWeight: "800" },
  photo: { width: 160, height: 160, borderRadius: 80 },
  text: { fontSize: 16, textAlign: "center" }
});
