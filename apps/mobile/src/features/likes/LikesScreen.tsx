import React, { useState } from "react";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { errorMessage } from "../../api/client";
import { discoveryApi } from "../../api/endpoints";
import type { PublicProfile, SwipeAction, SwipeResponse } from "../../api/types";
import { keys } from "../../hooks/queries";
import type { MainStackParams } from "../../navigation/types";
import { Screen } from "../../shared/layout";
import { Avatar, Button, EmptyState, ErrorState, ListSkeleton, Text, spacing, useColors, useToast } from "../../shared/ui";
import { MatchModal } from "../discover/MatchModal";

type Likes = { profiles: PublicProfile[] };

export function LikesScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const qc = useQueryClient();
  const toast = useToast();
  const c = useColors();
  const [match, setMatch] = useState<SwipeResponse["match"] | null>(null);
  const query = useQuery({ queryKey: keys.likes, queryFn: discoveryApi.likesReceived });

  const swipe = useMutation({
    mutationFn: ({ profile, action }: { profile: PublicProfile; action: SwipeAction }) => discoveryApi.swipe(profile.userId, action).then((res) => ({ res, profile })),
    onMutate: ({ profile }) => {
      qc.setQueryData<Likes>(keys.likes, (old) => (old ? { profiles: old.profiles.filter((p) => p.userId !== profile.userId) } : old));
    },
    onSuccess: ({ res, profile }) => {
      if (res.result === "matched" && res.match) setMatch({ ...res.match, profile: res.match.profile ?? profile });
      void qc.invalidateQueries({ queryKey: keys.conversations });
    },
    onError: (err) => {
      toast.show(errorMessage(err), "error");
      void qc.invalidateQueries({ queryKey: keys.likes });
    },
  });

  let body: React.ReactNode;
  if (query.isPending) body = <ListSkeleton />;
  else if (query.isError) body = <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  else if (query.data.profiles.length === 0) {
    body = <EmptyState title="Personne en attente" message="Les personnes qui vous ont liké apparaîtront ici, gratuitement et sans limite." />;
  } else {
    body = (
      <FlatList
        data={query.data.profiles}
        keyExtractor={(p) => p.userId}
        refreshControl={<RefreshControl refreshing={query.isRefetching} onRefresh={() => void query.refetch()} />}
        contentContainerStyle={{ padding: spacing.lg, gap: spacing.md }}
        renderItem={({ item }) => (
          <View style={[styles.row, { backgroundColor: c.surface, borderColor: c.border }]}>
            <Pressable style={styles.who} accessibilityRole="button" accessibilityLabel={`Voir le profil de ${item.firstName}`} onPress={() => navigation.navigate("ProfileDetail", { userId: item.userId })}>
              <Avatar path={item.photos[0]?.url} name={item.firstName} size={56} />
              <View style={{ flex: 1 }}>
                <Text variant="label">{item.firstName}, {item.age}</Text>
                {item.distanceKm !== null ? <Text variant="caption" tone="muted">à ~{item.distanceKm} km</Text> : null}
              </View>
            </Pressable>
            <Button title="Passer" variant="secondary" onPress={() => swipe.mutate({ profile: item, action: "pass" })} accessibilityLabel={`Passer ${item.firstName}`} />
            <Button title="Liker" variant="accent" onPress={() => swipe.mutate({ profile: item, action: "like" })} accessibilityLabel={`Liker ${item.firstName}`} testID={`like-back-${item.userId}`} />
          </View>
        )}
      />
    );
  }

  return (
    <Screen padded={false}>
      <View style={{ padding: spacing.lg, paddingBottom: 0 }}>
        <Text variant="heading">Ils vous ont liké</Text>
      </View>
      {body}
      <MatchModal
        match={match ?? null}
        onClose={() => setMatch(null)}
        onChat={() => {
          const m = match;
          setMatch(null);
          if (m) navigation.navigate("Chat", { conversationId: m.conversationId, name: m.profile?.firstName ?? "Conversation", userId: m.profile?.userId ?? "" });
        }}
      />
    </Screen>
  );
}

const styles = StyleSheet.create({
  row: { flexDirection: "row", alignItems: "center", gap: spacing.sm, borderWidth: 1, borderRadius: 18, padding: spacing.md },
  who: { flex: 1, flexDirection: "row", alignItems: "center", gap: spacing.md, minHeight: 56 },
});
