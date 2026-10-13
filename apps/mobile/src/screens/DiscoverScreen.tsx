import React, { useRef, useState } from "react";
import { Modal, Pressable, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { PublicProfile, SwipeResult } from "../api/types";
import { ProfileCard } from "../features/discover/ProfileCard";
import { CardBackdrop, SwipeableCard, type SwipeableCardHandle } from "../features/discover/SwipeableCard";
import { useDeck } from "../features/discover/useDeck";
import { ProfileDetailView } from "../features/profile/ProfileDetailView";
import { SafetyMenu } from "../features/safety/SafetyMenu";
import { useOwnProfile } from "../hooks/queries";
import { markMatchAnnounced } from "../lib/matchSeen";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Button, PhotoImage, Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";

type Nav = NativeStackNavigationProp<RootStackParamList>;

type MatchInfo = { profile: PublicProfile; result: SwipeResult };

export default function DiscoverScreen() {
  const { colors } = useTheme();
  const navigation = useNavigation<Nav>();
  const own = useOwnProfile();
  const [match, setMatch] = useState<MatchInfo | null>(null);
  const [details, setDetails] = useState<PublicProfile | null>(null);
  const [safetyFor, setSafetyFor] = useState<PublicProfile | null>(null);
  const topCard = useRef<SwipeableCardHandle>(null);

  const deck = useDeck({
    onMatch: (profile, result) => {
      if (result.match) {
        markMatchAnnounced(result.match.conversationId);
      }
      setMatch({ profile, result });
    }
  });
  const [top, next] = deck.queue;

  function actFromDetails(action: "like" | "pass") {
    if (!details) {
      return;
    }
    const target = details;
    setDetails(null);
    void deck.act(target, action);
  }

  let content: React.ReactNode;
  if (deck.status === "loading") {
    content = <LoadingView label="Recherche de profils…" />;
  } else if (deck.status === "error") {
    content = <ErrorView message={deck.error ?? "Impossible de charger les profils."} onAction={deck.reload} />;
  } else if (deck.status === "incomplete") {
    content = (
      <EmptyView
        icon="person-circle-outline"
        message="Ajoutez une photo et vos préférences pour découvrir des profils."
        title="Terminez votre profil"
      />
    );
  } else if (!top) {
    content = (
      <EmptyView
        actionLabel="Actualiser"
        icon="compass-outline"
        message="Il n'y a plus personne pour le moment. Élargissez vos préférences ou revenez plus tard."
        onAction={deck.reload}
        title="Vous avez tout vu !"
      >
        <Button label="Modifier mes préférences" onPress={() => navigation.navigate("Preferences")} variant="ghost" />
      </EmptyView>
    );
  } else {
    content = (
      <View style={styles.deck}>
        <View style={styles.cards}>
          {next ? (
            <CardBackdrop key={next.id}>
              <ProfileCard onOpenDetails={() => undefined} profile={next} />
            </CardBackdrop>
          ) : null}
          <SwipeableCard key={top.id} onSwiped={(action) => void deck.act(top, action)} ref={topCard}>
            <ProfileCard onOpenDetails={() => setDetails(top)} profile={top} />
          </SwipeableCard>
        </View>
        <View style={styles.actions}>
          <RoundButton color={colors.pass} icon="close" label="Passer" onPress={() => topCard.current?.swipe("pass")} testID="swipe-pass" />
          <RoundButton color={colors.textMuted} icon="ellipsis-horizontal" label="Plus d'options" onPress={() => setSafetyFor(top)} small />
          <RoundButton color={colors.like} icon="heart" label="J'aime" onPress={() => topCard.current?.swipe("like")} testID="swipe-like" />
        </View>
      </View>
    );
  }

  const myPhoto = own.data?.photos[0]?.id;
  return (
    <ScreenContainer contentMaxWidth={480}>
      {content}

      <Modal animationType="slide" onRequestClose={() => setDetails(null)} visible={details !== null}>
        {details ? (
          <View style={[styles.detail, { backgroundColor: colors.background }]}>
            <ProfileDetailView profile={details} />
            <Pressable
              accessibilityLabel="Fermer"
              accessibilityRole="button"
              onPress={() => setDetails(null)}
              style={[styles.close, { backgroundColor: colors.surface }]}
            >
              <Ionicons color={colors.text} name="chevron-down" size={24} />
            </Pressable>
            <View style={styles.detailActions}>
              <RoundButton color={colors.pass} icon="close" label="Passer" onPress={() => actFromDetails("pass")} />
              <RoundButton color={colors.like} icon="heart" label="J'aime" onPress={() => actFromDetails("like")} />
            </View>
          </View>
        ) : null}
      </Modal>

      <Modal animationType="fade" onRequestClose={() => setMatch(null)} transparent visible={match !== null}>
        <View style={[styles.matchBackdrop, { backgroundColor: colors.overlay }]}>
          <View accessibilityViewIsModal style={[styles.matchCard, { backgroundColor: colors.surface }]} testID="match-modal">
            <Text tone="primary" variant="eyebrow" weight="bold">
              Réciproque
            </Text>
            <Text accessibilityRole="header" variant="title" weight="bold">
              C'est un match !
            </Text>
            <View style={styles.faces}>
              <PhotoImage photoId={myPhoto} size="thumb" style={[styles.face, { marginRight: -14, borderColor: colors.surface }]} />
              <PhotoImage photoId={match?.profile.photos[0]?.id} size="thumb" style={[styles.face, { borderColor: colors.surface }]} />
            </View>
            <Text style={styles.center} tone="muted">
              Vous et {match?.profile.firstName} vous êtes plu mutuellement.
            </Text>
            <Button
              fullWidth
              label={`Écrire à ${match?.profile.firstName ?? ""}`}
              onPress={() => {
                const current = match;
                setMatch(null);
                if (current?.result.match) {
                  navigation.navigate("Conversation", {
                    conversationId: current.result.match.conversationId,
                    userId: current.profile.id,
                    name: current.profile.firstName,
                    photoId: current.profile.photos[0]?.id ?? null
                  });
                }
              }}
              testID="match-send-message"
            />
            <Button fullWidth label="Continuer à découvrir" onPress={() => setMatch(null)} variant="ghost" testID="match-continue" />
          </View>
        </View>
      </Modal>

      {safetyFor ? (
        <SafetyMenu
          name={safetyFor.firstName}
          onBlocked={() => {
            deck.drop(safetyFor.id);
            setSafetyFor(null);
            setDetails(null);
          }}
          onClose={() => setSafetyFor(null)}
          userId={safetyFor.id}
          visible
        />
      ) : null}
    </ScreenContainer>
  );
}

type RoundButtonProps = {
  icon: keyof typeof Ionicons.glyphMap;
  color: string;
  label: string;
  onPress: () => void;
  small?: boolean;
  testID?: string;
};

function RoundButton({ icon, color, label, onPress, small, testID }: RoundButtonProps) {
  const { colors } = useTheme();
  const size = small ? 46 : 68;
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      onPress={onPress}
      style={({ pressed }) => [
        styles.round,
        {
          width: size,
          height: size,
          borderRadius: size / 2,
          backgroundColor: colors.surface,
          borderColor: colors.border,
          transform: [{ scale: pressed ? 0.94 : 1 }]
        }
      ]}
      testID={testID}
    >
      <Ionicons color={color} name={icon} size={small ? 20 : 32} />
    </Pressable>
  );
}

const styles = StyleSheet.create({
  deck: { flex: 1, gap: spacing.lg, paddingBottom: spacing.sm },
  cards: { flex: 1, minHeight: 320 },
  actions: { flexDirection: "row", alignItems: "center", justifyContent: "center", gap: spacing.xl, paddingBottom: spacing.xs },
  round: { alignItems: "center", justifyContent: "center", borderWidth: 1.5 },
  detail: { flex: 1 },
  close: {
    position: "absolute",
    top: spacing.lg,
    right: spacing.lg,
    width: 44,
    height: 44,
    borderRadius: 22,
    alignItems: "center",
    justifyContent: "center"
  },
  detailActions: {
    position: "absolute",
    bottom: spacing.xl,
    left: 0,
    right: 0,
    flexDirection: "row",
    justifyContent: "center",
    gap: spacing.xxl
  },
  matchBackdrop: { flex: 1, alignItems: "center", justifyContent: "center", padding: spacing.lg },
  matchCard: { width: "100%", maxWidth: 380, borderRadius: radii.xl, padding: spacing.xl, alignItems: "center", gap: spacing.md },
  faces: { flexDirection: "row", paddingVertical: spacing.md },
  face: { width: 96, height: 96, borderRadius: 48, borderWidth: 4 },
  center: { textAlign: "center" }
});
