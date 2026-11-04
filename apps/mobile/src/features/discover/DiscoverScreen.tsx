import React, { useEffect, useRef } from "react";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQueryClient } from "@tanstack/react-query";
import { StyleSheet, View } from "react-native";
import { keys } from "../../hooks/queries";
import type { MainStackParams } from "../../navigation/types";
import { Screen } from "../../shared/layout";
import { Button, EmptyState, ErrorState, Loader, Text, spacing, useToast } from "../../shared/ui";
import { MatchModal } from "./MatchModal";
import { SwipeCard, type SwipeCardHandle } from "./SwipeCard";
import { useDeck } from "./useDeck";

export function DiscoverScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const qc = useQueryClient();
  const toast = useToast();
  const { deck, loading, exhausted, error, act, reload, match, dismissMatch, actionError } = useDeck(() => {
    void qc.invalidateQueries({ queryKey: keys.conversations });
    void qc.invalidateQueries({ queryKey: keys.notifications });
  });
  const cardRef = useRef<SwipeCardHandle>(null);
  const top = deck[0];
  const next = deck[1];

  useEffect(() => {
    if (actionError) toast.show(actionError, "error");
  }, [actionError, toast]);

  const openChat = () => {
    if (!match) return;
    const name = match.profile?.firstName ?? "Conversation";
    const userId = match.profile?.userId ?? "";
    dismissMatch();
    navigation.navigate("Chat", { conversationId: match.conversationId, name, userId });
  };

  let body: React.ReactNode;
  if (top) {
    body = (
      <>
        <View style={styles.stack}>
          {next ? <View style={styles.behind} pointerEvents="none"><View style={styles.placeholder} /></View> : null}
          <SwipeCard
            key={top.userId}
            ref={cardRef}
            profile={top}
            onSwiped={act}
            onOpen={() => navigation.navigate("ProfileDetail", { userId: top.userId })}
          />
        </View>
        <View style={styles.actions}>
          <ActionButton label="Passer" variant="secondary" testID="pass" onPress={() => cardRef.current?.swipe("pass")} />
          <ActionButton label="Liker" variant="accent" testID="like" onPress={() => cardRef.current?.swipe("like")} />
        </View>
      </>
    );
  } else if (error) {
    body = <ErrorState message={error} onRetry={reload} />;
  } else if (loading || !exhausted) {
    body = <Loader label="Recherche de profils…" />;
  } else {
    body = (
      <EmptyState
        testID="deck-empty"
        title="Vous avez tout vu pour le moment"
        message="Élargissez votre tranche d'âge ou votre distance dans votre profil, ou revenez un peu plus tard : de nouvelles personnes arrivent régulièrement."
        actionLabel="Actualiser"
        onAction={reload}
      />
    );
  }

  return (
    <Screen padded={false}>
      <View style={styles.header}>
        <Text variant="heading">Découvrir</Text>
      </View>
      {body}
      <MatchModal match={match} onChat={openChat} onClose={dismissMatch} />
    </Screen>
  );
}

function ActionButton({ label, variant, onPress, testID }: { label: string; variant: "secondary" | "accent"; onPress: () => void; testID: string }) {
  return <Button title={label} variant={variant} onPress={onPress} testID={testID} accessibilityLabel={label === "Liker" ? "Liker ce profil" : "Passer ce profil"} style={{ flex: 1 }} />;
}

const styles = StyleSheet.create({
  header: { paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  stack: { flex: 1, marginHorizontal: spacing.lg, marginBottom: spacing.md },
  behind: { ...StyleSheet.absoluteFillObject, transform: [{ scale: 0.95 }, { translateY: 10 }], opacity: 0.7 },
  placeholder: { flex: 1, borderRadius: 22, backgroundColor: "rgba(120,100,130,0.25)" },
  actions: { flexDirection: "row", gap: spacing.md, paddingHorizontal: spacing.lg, paddingBottom: spacing.lg },
});
