import { Ionicons } from "@expo/vector-icons";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useRef } from "react";
import { Pressable, StyleSheet, Text, View } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { EmptyView, ErrorView, LoadingView } from "../../components/StateViews";
import { SwipeCard, type SwipeCardHandle } from "../../components/SwipeCard";
import { useDeck } from "../../hooks/useDeck";
import type { MainTabParamList, RootStackParamList } from "../../navigation/types";
import { useRealtime } from "../../state/RealtimeProvider";
import { useToast } from "../../state/ToastProvider";
import { fontSizes, spacing } from "../../theme/palette";
import { useTheme } from "../../theme/ThemeProvider";
import { Wordmark } from "../../components/Wordmark";

type Props = CompositeScreenProps<
  BottomTabScreenProps<MainTabParamList, "Discover">,
  NativeStackScreenProps<RootStackParamList>
>;

export default function DiscoverScreen({ navigation }: Props) {
  const theme = useTheme();
  const toast = useToast();
  const { announceMatch } = useRealtime();
  const { deck, status, error, act, reload } = useDeck(announceMatch, (m) =>
    toast.show(m, "error")
  );
  const topCard = useRef<SwipeCardHandle>(null);

  const top = deck[0];
  const next = deck[1];

  return (
    <SafeAreaView edges={["top"]} style={[styles.flex, { backgroundColor: theme.background }]}>
      <View style={styles.header}>
        <Wordmark size={30} />
        <Pressable
          accessibilityLabel="Préférences de rencontre"
          accessibilityRole="button"
          hitSlop={10}
          onPress={() => navigation.navigate("Preferences")}
        >
          <Ionicons color={theme.text} name="options-outline" size={26} />
        </Pressable>
      </View>

      <View style={styles.stage}>
        {status === "loading" && !top ? (
          <LoadingView label="Recherche de profils…" />
        ) : status === "error" && !top ? (
          <ErrorView message={error ?? undefined} onRetry={reload} />
        ) : !top ? (
          <EmptyView
            actionLabel="Actualiser"
            icon="compass-outline"
            message="Vous avez fait le tour pour le moment. Élargissez vos préférences ou revenez un peu plus tard."
            onAction={reload}
            title="Plus de profils près de vous"
          />
        ) : (
          <View style={styles.deck}>
            {next ? (
              <View key={next.id} pointerEvents="none" style={styles.behind}>
                <SwipeCard
                  active={false}
                  onOpen={() => undefined}
                  onSwipe={() => undefined}
                  profile={next}
                />
              </View>
            ) : null}
            <SwipeCard
              active
              key={top.id}
              onOpen={() =>
                navigation.navigate("ProfileDetail", {
                  userId: top.id,
                  firstName: top.firstName,
                  allowActions: true
                })
              }
              onSwipe={(action) => void act(top, action)}
              profile={top}
              ref={topCard}
            />
          </View>
        )}
      </View>

      {top ? (
        <View style={styles.actions}>
          <Pressable
            accessibilityLabel="Passer"
            accessibilityRole="button"
            onPress={() => topCard.current?.swipe("pass")}
            style={[styles.round, { backgroundColor: theme.surface, borderColor: theme.border }]}
            testID="discover-pass"
          >
            <Ionicons color={theme.textMuted} name="close" size={32} />
          </Pressable>
          <Pressable
            accessibilityLabel="Voir le profil complet"
            accessibilityRole="button"
            onPress={() =>
              navigation.navigate("ProfileDetail", {
                userId: top.id,
                firstName: top.firstName,
                allowActions: true
              })
            }
            style={[styles.small, { backgroundColor: theme.surface, borderColor: theme.border }]}
          >
            <Ionicons color={theme.accent} name="information" size={24} />
          </Pressable>
          <Pressable
            accessibilityLabel="J'aime"
            accessibilityRole="button"
            onPress={() => topCard.current?.swipe("like")}
            style={[styles.round, { backgroundColor: theme.primary, borderColor: theme.primary }]}
            testID="discover-like"
          >
            <Ionicons color="#FFFFFF" name="heart" size={30} />
          </Pressable>
        </View>
      ) : null}
      {top ? (
        <Text style={[styles.tip, { color: theme.textMuted }]}>
          Glissez à droite pour aimer, à gauche pour passer.
        </Text>
      ) : null}
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "space-between",
    paddingHorizontal: spacing.lg,
    paddingVertical: spacing.sm
  },
  stage: { flex: 1, paddingHorizontal: spacing.md },
  deck: { flex: 1, marginVertical: spacing.sm },
  behind: { ...StyleSheet.absoluteFillObject, transform: [{ scale: 0.95 }], opacity: 0.9 },
  actions: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.xl,
    paddingVertical: spacing.md
  },
  round: {
    width: 68,
    height: 68,
    borderRadius: 34,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center"
  },
  small: {
    width: 48,
    height: 48,
    borderRadius: 24,
    borderWidth: 1,
    alignItems: "center",
    justifyContent: "center"
  },
  tip: { fontSize: fontSizes.xs, textAlign: "center", paddingBottom: spacing.sm }
});
