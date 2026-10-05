import { useFocusEffect, useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useCallback, useEffect, useRef, useState } from "react";
import { StyleSheet, View } from "react-native";
import type { MatchPreview } from "../api/types";
import { MatchModal } from "../components/MatchModal";
import { SwipeDeck, type SwipeDeckHandle } from "../components/SwipeDeck";
import { useDeck } from "../hooks/useDeck";
import { useMyProfile, useNotifications } from "../hooks/queries";
import { consumeDeckStale } from "../lib/deckSignal";
import type { RootStackParamList } from "../navigation/types";
import { useRealtime } from "../realtime/RealtimeProvider";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { IconButton, Text } from "../shared/ui";
import { SafeAreaLayout } from "../shared/layout";
import { layout, spacing } from "../theme";

/** "Découvrir" tab: the swipe deck. */
export default function HomeScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { markMatchAnnounced } = useRealtime();
  const { data: my } = useMyProfile();
  const { data: notifications } = useNotifications();
  const [match, setMatch] = useState<MatchPreview | null>(null);
  const deckRef = useRef<SwipeDeckHandle>(null);

  const { cards, status, error, act, reload, recheck } = useDeck((m) => {
    markMatchAnnounced(m.id); // the realtime push for this match must not show a second banner
    setMatch(m);
  });

  // Coming back to the tab: reload when settings changed, or when the deck was empty (new people may have joined).
  const statusRef = useRef(status);
  useEffect(() => {
    statusRef.current = status;
  });
  useFocusEffect(
    useCallback(() => {
      if (consumeDeckStale() || statusRef.current === "empty") void reload();
      // While the deck is empty, look again every 30 s so new people appear without a manual refresh.
      const timer = setInterval(() => {
        if (statusRef.current === "empty") void recheck();
      }, 30_000);
      return () => clearInterval(timer);
    }, [reload, recheck])
  );

  const openChat = (m: MatchPreview) => {
    setMatch(null);
    navigation.navigate("Chat", { conversationId: m.conversationId, matchId: m.id, userId: m.user.id, name: m.user.firstName, photoPath: m.user.photos[0]?.url });
  };

  return (
    <SafeAreaLayout edges={["top"]}>
      <View style={styles.page}>
        <View style={styles.header}>
          <Text variant="title" tone="primary">
            Lueur
          </Text>
          <IconButton testID="open-notifications" icon="notifications-outline" label="Notifications" badge={notifications?.unreadCount} onPress={() => navigation.navigate("Notifications")} />
        </View>

        <View style={styles.deck}>
          {status === "loading" ? <LoadingView label="On cherche des personnes pour vous…" /> : null}
          {status === "error" ? <ErrorView error={error} onRetry={() => void reload()} /> : null}
          {status === "incomplete" ? <EmptyView icon="person-circle-outline" title="Terminez votre profil" message="Ajoutez une photo, vos préférences et votre position pour découvrir des personnes." /> : null}
          {status === "empty" ? (
            <EmptyView
              icon="compass-outline"
              title="Vous avez fait le tour pour le moment"
              message="Revenez plus tard : de nouvelles personnes s'inscrivent chaque jour. Vous pouvez aussi élargir la distance ou la tranche d'âge."
              actionLabel="Actualiser"
              onAction={() => void reload()}
            />
          ) : null}
          {status === "ready" && cards.length > 0 ? (
            <SwipeDeck
              ref={deckRef}
              cards={cards}
              onSwipe={(card, action) => void act(card, action)}
              onOpenDetails={(card) => navigation.navigate("ProfileDetail", { userId: card.id })}
            />
          ) : null}
        </View>
      </View>

      <MatchModal match={match} myName={my?.profile?.firstName ?? ""} myPhotoPath={my?.profile?.photos[0]?.url} onMessage={openChat} onClose={() => setMatch(null)} />
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  page: { flex: 1, width: "100%", maxWidth: layout.maxContentWidth, alignSelf: "center", paddingHorizontal: spacing.lg, paddingBottom: spacing.sm },
  header: { flexDirection: "row", alignItems: "center", justifyContent: "space-between", minHeight: 52 },
  deck: { flex: 1 }
});
