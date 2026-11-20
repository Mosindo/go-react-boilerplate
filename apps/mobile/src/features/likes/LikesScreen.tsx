import React, { useMemo, useState } from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { LinearGradient } from "expo-linear-gradient";
import { useNavigation } from "@react-navigation/native";
import { useQuery } from "@tanstack/react-query";
import { mediaUrl } from "../../api/client";
import { matchingApi } from "../../api/endpoints";
import type { SwipeResponse } from "../../api/types";
import { errorMessage, formatDistance } from "../../lib/format";
import { keys } from "../../providers/queryClient";
import { useTheme } from "../../theme";
import { EmptyState, ErrorState, Header, IconButton, LoadingState, Screen, Text } from "../../ui";
import { MatchModal } from "../discovery/MatchModal";
import { useDecidedProfiles } from "../discovery/swipeStore";
import { useSwipe } from "../discovery/useSwipe";
import { useMyProfile } from "../profile/queries";

export function LikesScreen() {
  const navigation = useNavigation();
  const { colors, radii, spacing } = useTheme();
  const me = useMyProfile();
  const decided = useDecidedProfiles();
  const likes = useQuery({ queryKey: keys.likes, queryFn: () => matchingApi.likesReceived(50) });
  const [match, setMatch] = useState<{ user: NonNullable<SwipeResponse["matchedUser"]>; conversationId: string } | null>(null);
  const swipe = useSwipe((r) => r.matchedUser && r.conversationId && setMatch({ user: r.matchedUser, conversationId: r.conversationId }));
  const visible = useMemo(() => (likes.data ?? []).filter((c) => !decided.has(c.userId)), [likes.data, decided]);

  return (
    <Screen>
      <Header large subtitle="Gratuit, comme tout le reste." title="Ils t’ont liké" />
      {likes.isLoading ? (
        <LoadingState />
      ) : likes.isError ? (
        <ErrorState message={errorMessage(likes.error)} onRetry={() => void likes.refetch()} />
      ) : (
        <FlatList
          columnWrapperStyle={{ gap: spacing.sm }}
          contentContainerStyle={{ padding: spacing.lg, gap: spacing.sm, flexGrow: 1 }}
          data={visible}
          keyExtractor={(c) => c.userId}
          ListEmptyComponent={
            <EmptyState
              icon="heart-outline"
              message="Quand quelqu’un t’aime, son profil apparaît ici. Tu pourras liker en retour pour créer un match."
              title="Pas encore de like"
            />
          }
          numColumns={2}
          onRefresh={() => void likes.refetch()}
          refreshing={likes.isRefetching}
          renderItem={({ item }) => (
            <Pressable
              accessibilityLabel={`${item.firstName}, ${item.age} ans`}
              onPress={() => navigation.navigate("ProfileDetail", { userId: item.userId, fromDiscovery: true })}
              style={[styles.tile, { borderRadius: radii.lg, backgroundColor: colors.surfaceAlt }]}
              testID="like-tile"
            >
              {item.photos[0] ? (
                <Image cachePolicy="memory-disk" contentFit="cover" source={{ uri: mediaUrl(item.photos[0].url) }} style={StyleSheet.absoluteFill} />
              ) : null}
              <LinearGradient colors={["transparent", "rgba(12,8,14,0.9)"]} style={[styles.overlay, { padding: spacing.sm }]}>
                <Text style={styles.white} variant="bodyStrong">
                  {item.firstName}, {item.age}
                </Text>
                {formatDistance(item.distanceKm) ? (
                  <Text style={styles.whiteSoft} variant="caption">
                    {formatDistance(item.distanceKm)}
                  </Text>
                ) : null}
                <View style={styles.actions}>
                  <IconButton
                    filled
                    icon="close"
                    label={`Passer ${item.firstName}`}
                    onPress={() => swipe.mutate({ userId: item.userId, action: "pass" })}
                    size={40}
                  />
                  <IconButton
                    filled
                    icon="heart"
                    label={`Liker ${item.firstName} en retour`}
                    onPress={() => swipe.mutate({ userId: item.userId, action: "like" })}
                    size={40}
                    tone="primary"
                  />
                </View>
              </LinearGradient>
            </Pressable>
          )}
        />
      )}
      <MatchModal
        match={match}
        myName={me.data?.firstName ?? ""}
        myPhotoUrl={me.data?.photos[0]?.url}
        onClose={() => setMatch(null)}
        onMessage={(conversationId) => {
          setMatch(null);
          navigation.navigate("Chat", { conversationId });
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  tile: { flex: 1, aspectRatio: 3 / 4, overflow: "hidden", maxWidth: "50%" },
  overlay: { position: "absolute", left: 0, right: 0, bottom: 0, paddingTop: 40, gap: 2 },
  actions: { flexDirection: "row", justifyContent: "space-between", marginTop: 6 },
  white: { color: "#FFFFFF" },
  whiteSoft: { color: "rgba(255,255,255,0.8)" }
});
