import React, { useCallback, useEffect, useRef, useState } from "react";
import { Pressable, View } from "react-native";
import { useFocusEffect, useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { AppText, Banner, EmptyState, ErrorState, Loading, Row, Screen } from "../../components/ui";
import SwipeCard from "../../components/SwipeCard";
import { MatchModal } from "../../components/Modals";
import { discoveryApi } from "../../api/services";
import { ApiError, errorMessage } from "../../api/http";
import { useDeck, useUnreadNotifications } from "../../hooks";
import { radius, spacing, useTheme } from "../../theme/theme";
import type { MainStackParams } from "../../navigation/types";
import type { Card, MatchView } from "../../api/types";

export default function DiscoverScreen() {
  const { colors } = useTheme();
  const nav = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const qc = useQueryClient();
  const deck = useDeck();
  const unread = useUnreadNotifications();
  const [match, setMatch] = useState<MatchView | null>(null);
  const [notice, setNotice] = useState<string | null>(null);
  const decideRef = useRef<((a: "like" | "pass") => void) | null>(null);
  const busy = useRef(false);

  // Coming back to an exhausted deck (e.g. after changing preferences) looks for new profiles.
  // Only focus *gains* matter, so the latest value is read through a ref.
  const { exhausted, reset } = deck;
  const exhaustedRef = useRef(exhausted);
  useEffect(() => {
    exhaustedRef.current = exhausted;
  }, [exhausted]);
  useFocusEffect(
    useCallback(() => {
      if (exhaustedRef.current) reset();
    }, [reset]),
  );

  const top = deck.queue[0];

  const decide = async (card: Card, action: "like" | "pass") => {
    if (busy.current) return;
    busy.current = true;
    setNotice(null);
    deck.shift();
    try {
      const res = await discoveryApi.swipe(card.userId, action);
      if (res.matched && res.match) {
        setMatch(res.match);
        void qc.invalidateQueries({ queryKey: ["matches"] });
        void qc.invalidateQueries({ queryKey: ["conversations"] });
      }
      void qc.invalidateQueries({ queryKey: ["likes"] });
    } catch (e) {
      if (e instanceof ApiError && (e.status === 409 || e.status === 404)) {
        // already decided, or the person is no longer available: simply move on
      } else {
        deck.restore(card);
        setNotice(errorMessage(e));
      }
    } finally {
      busy.current = false;
    }
  };

  return (
    <Screen>
      <Row style={{ justifyContent: "space-between", paddingVertical: spacing.sm }}>
        <AppText variant="title">Aube</AppText>
        <Pressable
          testID="open-notifications"
          accessibilityRole="button"
          accessibilityLabel={`Notifications${unread ? `, ${unread} non lues` : ""}`}
          onPress={() => nav.navigate("Notifications")}
          style={{
            paddingHorizontal: spacing.md,
            paddingVertical: spacing.sm,
            borderRadius: radius.pill,
            backgroundColor: colors.surfaceAlt,
          }}
        >
          <AppText variant="small" style={{ fontWeight: "700" }}>
            🔔 {unread > 0 ? unread : ""}
          </AppText>
        </Pressable>
      </Row>
      {notice ? <Banner message={notice} /> : null}
      <View style={{ flex: 1, justifyContent: "center", gap: spacing.lg }}>
        {top ? (
          <>
            <SwipeCard
              key={top.userId}
              card={top}
              decideRef={decideRef}
              onDecide={(a) => void decide(top, a)}
              onOpen={() => nav.navigate("ProfileDetail", { userId: top.userId, card: top, canAct: false })}
            />
            <Row style={{ justifyContent: "center", gap: spacing.xl }}>
              <RoundButton
                testID="pass-button"
                label="✕"
                accessibilityLabel="Passer"
                color={colors.textMuted}
                onPress={() => decideRef.current?.("pass")}
              />
              <RoundButton
                testID="like-button"
                label="♥"
                accessibilityLabel="Liker"
                color={colors.primary}
                filled
                onPress={() => decideRef.current?.("like")}
              />
            </Row>
          </>
        ) : deck.loading ? (
          <Loading label="Recherche de profils…" />
        ) : deck.error ? (
          <ErrorState message={deck.error} onRetry={deck.reset} />
        ) : (
          <EmptyState
            title="Vous avez tout vu pour l'instant"
            message="Revenez plus tard, ou élargissez vos préférences (distance, tranche d'âge) pour voir plus de monde."
            actionLabel="Modifier mes préférences"
            onAction={() => nav.navigate("EditProfile")}
          />
        )}
      </View>
      <MatchModal
        match={match}
        onClose={() => setMatch(null)}
        onChat={() => {
          const m = match;
          setMatch(null);
          if (m)
            nav.navigate("Chat", {
              conversationId: m.conversationId,
              peerName: m.user.firstName,
              peerId: m.user.userId,
            });
        }}
      />
    </Screen>
  );
}

function RoundButton({
  label,
  color,
  filled,
  onPress,
  testID,
  accessibilityLabel,
}: {
  label: string;
  color: string;
  filled?: boolean;
  onPress: () => void;
  testID: string;
  accessibilityLabel: string;
}) {
  const { colors } = useTheme();
  return (
    <Pressable
      testID={testID}
      accessibilityRole="button"
      accessibilityLabel={accessibilityLabel}
      onPress={onPress}
      style={({ pressed }) => ({
        width: 68,
        height: 68,
        borderRadius: 34,
        alignItems: "center",
        justifyContent: "center",
        backgroundColor: filled ? color : colors.surface,
        borderWidth: 1,
        borderColor: filled ? color : colors.border,
        opacity: pressed ? 0.8 : 1,
      })}
    >
      <AppText style={{ fontSize: 28, color: filled ? colors.primaryText : color, fontWeight: "800" }}>
        {label}
      </AppText>
    </Pressable>
  );
}
