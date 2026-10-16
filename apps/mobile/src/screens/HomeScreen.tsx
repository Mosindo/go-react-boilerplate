import React, { useCallback, useRef } from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { PublicProfile } from "../api/types";
import { SwipeDeck, type SwipeDeckHandle } from "../components/SwipeDeck";
import { useDiscoverDeck } from "../hooks/useDiscoverDeck";
import { useRealtime } from "../hooks/useRealtime";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Notice, Text, colors, radii, shadows, spacing } from "../shared/ui";

/** Discovery tab: one profile at a time, like or pass. */
export default function HomeScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { announceMatch } = useRealtime();
  const deck = useRef<SwipeDeckHandle>(null);

  const { queue, status, error, actionError, exhausted, reload, decide } = useDiscoverDeck(
    useCallback(
      (profile, result) => {
        if (result.matchId) {
          announceMatch({ matchId: result.matchId, profile });
        }
      },
      [announceMatch]
    )
  );

  const openProfile = (profile: PublicProfile) => navigation.navigate("ProfileDetail", { userId: profile.id, source: "discover" });

  return (
    <ScreenContainer contentStyle={styles.content} testID="screen-discover">
      <Header subtitle="Des rencontres près de chez vous, sans limite ni abonnement." title="Découvrir" />

      {actionError ? <Notice description={actionError} title="Action non enregistrée" tone="danger" /> : null}

      <View style={styles.stage}>
        {status === "loading" ? <LoadingView label="Recherche de profils…" /> : null}
        {status === "error" ? <ErrorView message={error ?? "Impossible de charger les profils."} onAction={reload} /> : null}
        {status === "ready" && queue.length === 0 ? (
          <EmptyView
            actionLabel="Actualiser"
            message={
              exhausted
                ? "Vous avez fait le tour pour le moment. Élargissez vos préférences ou revenez un peu plus tard."
                : "Chargement de nouveaux profils…"
            }
            onAction={reload}
            testID="discover-empty"
            title="Plus personne dans votre zone"
          />
        ) : null}
        {status === "ready" && queue.length > 0 ? (
          <SwipeDeck onOpen={openProfile} onSwipe={(profile, action) => void decide(profile, action)} profiles={queue} ref={deck} />
        ) : null}
      </View>

      {status === "ready" && queue.length > 0 ? (
        <View style={styles.actions}>
          <RoundButton color={colors.pass} icon="close" label="Passer" onPress={() => deck.current?.swipe("pass")} testID="action-pass" />
          <RoundButton
            color={colors.secondary}
            icon="information"
            label="Voir le profil"
            onPress={() => queue[0] && openProfile(queue[0])}
            size={48}
            testID="action-info"
          />
          <RoundButton color={colors.like} icon="heart" label="J'aime" onPress={() => deck.current?.swipe("like")} testID="action-like" />
        </View>
      ) : null}
    </ScreenContainer>
  );
}

type RoundButtonProps = {
  icon: React.ComponentProps<typeof Ionicons>["name"];
  label: string;
  color: string;
  onPress: () => void;
  size?: number;
  testID: string;
};

function RoundButton({ icon, label, color, onPress, size = 64, testID }: RoundButtonProps) {
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [styles.round, { width: size, height: size, borderColor: color }, pressed ? styles.pressed : null]}
      testID={testID}
    >
      <Ionicons color={color} name={icon} size={size * 0.46} />
      <Text style={styles.srOnly}>{label}</Text>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  content: { gap: spacing.md },
  stage: { flex: 1, justifyContent: "center" },
  actions: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: spacing.xl, paddingBottom: spacing.xs },
  round: {
    borderRadius: radii.pill,
    borderWidth: 2,
    backgroundColor: colors.backgroundElevated,
    alignItems: "center",
    justifyContent: "center",
    ...shadows.card
  },
  pressed: { transform: [{ scale: 0.94 }] },
  srOnly: { position: "absolute", width: 1, height: 1, opacity: 0 }
});
