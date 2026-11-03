import React from "react";
import { FlatList, Pressable, StyleSheet, Text, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { api } from "../api";
import { errorMessage } from "../api/client";
import type { LikeReceived, SwipeAction } from "../api/types";
import { useMatch } from "../features/match/MatchContext";
import type { MainStackParams } from "../navigation/types";
import { radius, spacing, useTheme } from "../theme";
import { AuthImage } from "../ui/AuthImage";
import { EmptyState, ErrorState, Loading, Title } from "../ui/components";

export function LikesScreen() {
  const theme = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParams>>();
  const queryClient = useQueryClient();
  const { showMatch } = useMatch();
  const likes = useQuery({ queryKey: ["likes"], queryFn: api.likesReceived });

  const decide = useMutation({
    mutationFn: async ({ like, action }: { like: LikeReceived; action: SwipeAction }) => ({
      like,
      result: await api.swipe(like.user.id, action)
    }),
    onSuccess: ({ like, result }) => {
      if (result.matched && result.conversationId && result.matchId) {
        showMatch({ user: like.user, conversationId: result.conversationId, matchId: result.matchId });
      }
      void queryClient.invalidateQueries({ queryKey: ["likes"] });
      void queryClient.invalidateQueries({ queryKey: ["conversations"] });
    }
  });

  return (
    <View style={[styles.flex, { backgroundColor: theme.background }]}>
      <View style={styles.header}>
        <Title>Ils vous ont liké</Title>
        <Text style={{ color: theme.textMuted }}>Gratuit et illimité : répondez à qui vous voulez.</Text>
      </View>
      {likes.isLoading ? (
        <Loading />
      ) : likes.isError ? (
        <ErrorState message={errorMessage(likes.error)} onRetry={() => void likes.refetch()} />
      ) : (likes.data ?? []).length === 0 ? (
        <EmptyState title="Pas encore de like" message="Quand quelqu'un vous likera, il apparaîtra ici." />
      ) : (
        <FlatList
          data={likes.data}
          keyExtractor={(item) => item.user.id}
          numColumns={2}
          contentContainerStyle={styles.list}
          columnWrapperStyle={{ gap: spacing.md }}
          refreshing={likes.isRefetching}
          onRefresh={() => void likes.refetch()}
          renderItem={({ item }) => (
            <View style={[styles.tile, { backgroundColor: theme.surface }]}>
              <Pressable
                accessibilityRole="button"
                accessibilityLabel={`Voir le profil de ${item.user.firstName}`}
                onPress={() => navigation.navigate("UserProfile", { user: item.user })}
              >
                <AuthImage path={item.user.photos[0]?.url} style={styles.photo} />
                <Text style={[styles.name, { color: theme.text }]}>{`${item.user.firstName}, ${item.user.age}`}</Text>
              </Pressable>
              <View style={styles.row}>
                <Pressable
                  accessibilityRole="button"
                  accessibilityLabel={`Passer ${item.user.firstName}`}
                  onPress={() => decide.mutate({ like: item, action: "pass" })}
                  style={styles.action}
                >
                  <Text style={{ color: theme.danger, fontSize: 22, fontWeight: "800" }}>✕</Text>
                </Pressable>
                <Pressable
                  accessibilityRole="button"
                  accessibilityLabel={`Liker ${item.user.firstName}`}
                  onPress={() => decide.mutate({ like: item, action: "like" })}
                  style={styles.action}
                  testID={`like-back-${item.user.firstName}`}
                >
                  <Text style={{ color: theme.primary, fontSize: 22, fontWeight: "800" }}>♥</Text>
                </Pressable>
              </View>
            </View>
          )}
        />
      )}
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: { padding: spacing.xl, paddingBottom: spacing.sm, gap: spacing.xs },
  list: { padding: spacing.lg, gap: spacing.md },
  tile: { flex: 1, borderRadius: radius.md, overflow: "hidden" },
  photo: { width: "100%", aspectRatio: 0.8 },
  name: { fontSize: 16, fontWeight: "700", padding: spacing.sm },
  row: { flexDirection: "row" },
  action: { flex: 1, minHeight: 44, alignItems: "center", justifyContent: "center" }
});
