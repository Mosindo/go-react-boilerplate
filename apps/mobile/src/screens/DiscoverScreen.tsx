import React, { useCallback, useEffect, useMemo, useRef } from "react";
import { Animated, Image, Modal, PanResponder, Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { PublicProfile, SwipeAction } from "../api/types";
import { PhotoCarousel } from "../components/PhotoCarousel";
import { mediaUrl } from "../config";
import { distanceLabel } from "../lib/format";
import type { MainStackParamList } from "../navigation/types";
import { discoverStore, useDiscover } from "../state/discoverStore";
import { Button, EmptyView, ErrorView, LoadingView, Screen, Text, radii, spacing, useTheme, useToast } from "../ui";

type Nav = NativeStackNavigationProp<MainStackParamList>;
const SWIPE_THRESHOLD = 110;
const OFFSCREEN = 700;

type SwipeCardHandle = { fling: (action: SwipeAction) => void };

type CardProps = {
  profile: PublicProfile;
  handleRef: React.MutableRefObject<SwipeCardHandle | null>;
  onDecision: (action: SwipeAction) => void;
  onOpen: () => void;
};

function SwipeCard({ profile, handleRef, onDecision, onOpen }: CardProps) {
  const { colors } = useTheme();
  const position = useRef(new Animated.ValueXY()).current;

  const fling = useCallback(
    (action: SwipeAction) => {
      Animated.timing(position, {
        toValue: { x: action === "like" ? OFFSCREEN : -OFFSCREEN, y: 0 },
        duration: 220,
        useNativeDriver: false
      }).start(() => onDecision(action));
    },
    [onDecision, position]
  );

  useEffect(() => {
    handleRef.current = { fling };
    return () => {
      handleRef.current = null;
    };
  }, [fling, handleRef]);

  const responder = useMemo(
    () =>
      PanResponder.create({
        onMoveShouldSetPanResponder: (_e, g) => Math.abs(g.dx) > 8 && Math.abs(g.dx) > Math.abs(g.dy),
        onPanResponderMove: Animated.event([null, { dx: position.x, dy: position.y }], { useNativeDriver: false }),
        onPanResponderRelease: (_e, g) => {
          if (g.dx > SWIPE_THRESHOLD) fling("like");
          else if (g.dx < -SWIPE_THRESHOLD) fling("pass");
          else Animated.spring(position, { toValue: { x: 0, y: 0 }, useNativeDriver: false }).start();
        }
      }),
    [fling, position]
  );

  const rotate = position.x.interpolate({ inputRange: [-300, 0, 300], outputRange: ["-9deg", "0deg", "9deg"] });
  const likeOpacity = position.x.interpolate({
    inputRange: [20, SWIPE_THRESHOLD],
    outputRange: [0, 1],
    extrapolate: "clamp"
  });
  const passOpacity = position.x.interpolate({
    inputRange: [-SWIPE_THRESHOLD, -20],
    outputRange: [1, 0],
    extrapolate: "clamp"
  });
  const distance = distanceLabel(profile.distanceKm);

  return (
    <Animated.View
      {...responder.panHandlers}
      style={[styles.card, { transform: [{ translateX: position.x }, { translateY: position.y }, { rotate }] }]}
      testID="discover-card"
    >
      <PhotoCarousel photos={profile.photos} />
      <Animated.View
        style={[styles.stamp, styles.stampLike, { opacity: likeOpacity, borderColor: colors.success }]}
        pointerEvents="none"
      >
        <Text style={[styles.stampText, { color: colors.success }]}>J'AIME</Text>
      </Animated.View>
      <Animated.View
        style={[styles.stamp, styles.stampPass, { opacity: passOpacity, borderColor: colors.danger }]}
        pointerEvents="none"
      >
        <Text style={[styles.stampText, { color: colors.danger }]}>PASSER</Text>
      </Animated.View>
      <Pressable
        accessibilityLabel={`Voir le profil de ${profile.displayName}`}
        onPress={onOpen}
        style={[styles.info, { backgroundColor: colors.photoScrim }]}
      >
        <View style={styles.infoText}>
          <Text tone="inverse" variant="title" testID="discover-name">
            {profile.displayName}, {profile.age}
          </Text>
          <Text tone="inverse" variant="caption">
            {[profile.city, distance].filter(Boolean).join(" · ")}
          </Text>
        </View>
        <Ionicons color="#FFFFFF" name="information-circle-outline" size={28} />
      </Pressable>
    </Animated.View>
  );
}

function CircleButton({
  icon,
  label,
  onPress,
  color,
  testID,
  size = 64
}: {
  icon: React.ComponentProps<typeof Ionicons>["name"];
  label: string;
  onPress: () => void;
  color: string;
  testID: string;
  size?: number;
}) {
  const { colors } = useTheme();
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [
        styles.circle,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          backgroundColor: colors.surface,
          borderColor: colors.border,
          opacity: pressed ? 0.8 : 1
        }
      ]}
      testID={testID}
    >
      <Ionicons color={color} name={icon} size={size * 0.46} />
    </Pressable>
  );
}

export default function DiscoverScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<Nav>();
  const toast = useToast();
  const state = useDiscover();
  const handle = useRef<SwipeCardHandle | null>(null);
  const top = state.queue[0];
  const next = state.queue[1];

  useEffect(() => {
    if (state.queue.length === 0 && !state.loading && !state.exhausted && !state.error && !state.incomplete) {
      void discoverStore.load();
    }
  }, [state.queue.length, state.loading, state.exhausted, state.error, state.incomplete]);

  useEffect(() => {
    if (state.error && state.queue.length > 0) {
      toast.show(state.error, "error");
      discoverStore.clearError();
    }
  }, [state.error, state.queue.length, toast]);

  const decide = useCallback(
    (profile: PublicProfile, action: SwipeAction) => void discoverStore.swipe(profile, action),
    []
  );

  let body: React.ReactNode;
  if (top) {
    body = (
      <>
        <View style={styles.stack}>
          {next ? (
            <View style={[styles.card, styles.behind, { backgroundColor: colors.surfaceAlt }]}>
              {next.photos[0] ? (
                <Image
                  accessibilityIgnoresInvertColors
                  source={{ uri: mediaUrl(next.photos[0].url) }}
                  style={styles.behindImage}
                />
              ) : null}
            </View>
          ) : null}
          <SwipeCard
            handleRef={handle}
            key={top.userId}
            onDecision={(action) => decide(top, action)}
            onOpen={() => navigation.navigate("ProfileDetail", { userId: top.userId, showActions: true })}
            profile={top}
          />
        </View>
        <View style={styles.actions}>
          <CircleButton
            color={colors.danger}
            icon="close"
            label="Passer"
            onPress={() => handle.current?.fling("pass")}
            testID="discover-pass"
          />
          <CircleButton
            color={colors.textMuted}
            icon="person-outline"
            label="Voir le profil complet"
            onPress={() => navigation.navigate("ProfileDetail", { userId: top.userId, showActions: true })}
            size={50}
            testID="discover-details"
          />
          <CircleButton
            color={colors.primary}
            icon="heart"
            label="J'aime"
            onPress={() => handle.current?.fling("like")}
            testID="discover-like"
          />
        </View>
      </>
    );
  } else if (state.loading) {
    body = <LoadingView label="Recherche de profils…" />;
  } else if (state.error) {
    body = (
      <ErrorView
        message={state.error}
        onRetry={() => {
          discoverStore.clearError();
          void discoverStore.load();
        }}
      />
    );
  } else if (state.incomplete) {
    body = (
      <EmptyView
        icon="person-circle-outline"
        message="Ajoutez une photo, votre position et vos préférences pour découvrir des profils."
        title="Terminez votre profil"
      />
    );
  } else {
    body = (
      <EmptyView
        actionLabel="Actualiser"
        icon="compass-outline"
        message="Vous avez vu tout le monde pour le moment. Élargissez vos préférences ou revenez un peu plus tard."
        onAction={() => discoverStore.refresh()}
        testID="discover-empty"
        title="Plus personne à découvrir"
      />
    );
  }

  return (
    <Screen topInset testID="discover-screen">
      <View style={styles.header}>
        <Text tone="primary" variant="title">
          Alba
        </Text>
        <Pressable
          accessibilityLabel="Modifier mes critères de recherche"
          accessibilityRole="button"
          onPress={() => navigation.navigate("Preferences")}
          style={styles.filter}
        >
          <Ionicons color={colors.text} name="options-outline" size={24} />
        </Pressable>
      </View>
      <View style={styles.body}>{body}</View>

      <Modal
        animationType="fade"
        onRequestClose={discoverStore.dismissMatch}
        transparent
        visible={Boolean(state.match)}
      >
        <View style={[styles.matchBackdrop, { backgroundColor: colors.overlay }]}>
          {state.match ? (
            <View style={[styles.matchCard, { backgroundColor: colors.surface }]} testID="match-modal">
              <Ionicons color={colors.primary} name="heart-circle" size={64} />
              <Text variant="title">C'est un match !</Text>
              <Text style={styles.centerText} tone="muted">
                Vous et {state.match.user.displayName} vous êtes plu mutuellement.
              </Text>
              <View style={styles.matchPhoto}>
                <PhotoCarousel photos={state.match.user.photos.slice(0, 1)} tapToBrowse={false} />
              </View>
              <Button
                label="Envoyer un message"
                onPress={() => {
                  const match = state.match;
                  discoverStore.dismissMatch();
                  if (match)
                    navigation.navigate("Chat", {
                      conversationId: match.conversationId,
                      userId: match.user.userId,
                      name: match.user.displayName
                    });
                }}
                testID="match-message"
              />
              <Button
                label="Continuer à découvrir"
                onPress={discoverStore.dismissMatch}
                testID="match-continue"
                variant="ghost"
              />
            </View>
          ) : null}
        </View>
      </Modal>
    </Screen>
  );
}

const styles = StyleSheet.create({
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", marginBottom: spacing.md },
  filter: { width: 44, height: 44, alignItems: "center", justifyContent: "center" },
  body: { flex: 1 },
  stack: { flex: 1, marginBottom: spacing.lg },
  card: { ...StyleSheet.absoluteFillObject, borderRadius: radii.xl, overflow: "hidden" },
  behind: { transform: [{ scale: 0.95 }], opacity: 0.7 },
  behindImage: { width: "100%", height: "100%" },
  info: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    flexDirection: "row",
    alignItems: "center",
    padding: spacing.lg,
    gap: spacing.md
  },
  infoText: { flex: 1 },
  stamp: {
    position: "absolute",
    top: 56,
    paddingHorizontal: spacing.md,
    paddingVertical: spacing.xs,
    borderWidth: 4,
    borderRadius: radii.sm
  },
  stampLike: { left: spacing.lg, transform: [{ rotate: "-12deg" }] },
  stampPass: { right: spacing.lg, transform: [{ rotate: "12deg" }] },
  stampText: { fontSize: 26, fontWeight: "800", letterSpacing: 1 },
  actions: {
    flexDirection: "row",
    alignItems: "center",
    justifyContent: "center",
    gap: spacing.xl,
    paddingBottom: spacing.lg
  },
  circle: {
    alignItems: "center",
    justifyContent: "center",
    borderWidth: 1,
    shadowColor: "#000",
    shadowOpacity: 0.12,
    shadowRadius: 10,
    shadowOffset: { width: 0, height: 4 },
    elevation: 3
  },
  matchBackdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.xl },
  matchCard: {
    width: "100%",
    maxWidth: 380,
    borderRadius: radii.xl,
    padding: spacing.xl,
    alignItems: "center",
    gap: spacing.md
  },
  matchPhoto: { width: 150, height: 190, borderRadius: radii.lg, overflow: "hidden", marginVertical: spacing.sm },
  centerText: { textAlign: "center" }
});
