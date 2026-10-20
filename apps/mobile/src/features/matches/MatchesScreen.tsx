import React from "react";
import { Alert, FlatList, Pressable, StyleSheet, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useInfiniteQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { fetchMatches, unmatch } from "../../api/social";
import { queryKeys } from "../../api/keys";
import { flattenPages } from "../../lib/pagination";
import type { MainStackParamList, MainTabParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Avatar, EmptyState, ErrorState, LoadingState, Screen, Text, useToast } from "../../ui";

type Props = CompositeScreenProps<
  BottomTabScreenProps<MainTabParamList, "Matches">,
  NativeStackScreenProps<MainStackParamList>
>;

export function MatchesScreen({ navigation }: Props) {
  const qc = useQueryClient();
  const toast = useToast();
  const remove = useMutation({
    mutationFn: (matchId: string) => unmatch(matchId),
    onSuccess: async () => {
      await Promise.all([
        qc.invalidateQueries({ queryKey: queryKeys.matches }),
        qc.invalidateQueries({ queryKey: queryKeys.conversations })
      ]);
    },
    onError: (e) => toast.show(e instanceof Error ? e.message : "Could not unmatch.", "error")
  });
  const query = useInfiniteQuery({
    queryKey: queryKeys.matches,
    queryFn: ({ pageParam }) => fetchMatches(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });
  const matches = flattenPages(
    query.data?.pages.map((p) => ({ items: p.matches, nextCursor: p.nextCursor }))
  );

  return (
    <Screen>
      <Text variant="title">Matches</Text>
      {query.isLoading ? (
        <LoadingState />
      ) : query.isError ? (
        <ErrorState error={query.error} onRetry={() => void query.refetch()} />
      ) : matches.length === 0 ? (
        <EmptyState
          title="No matches yet"
          message="When you and someone else like each other, they show up here."
        />
      ) : (
        <FlatList
          data={matches}
          numColumns={2}
          keyExtractor={(m) => m.id}
          contentContainerStyle={{ paddingTop: spacing.lg }}
          columnWrapperStyle={{ justifyContent: "space-between" }}
          onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
          refreshing={query.isRefetching}
          onRefresh={() => void query.refetch()}
          renderItem={({ item }) => (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={`Open conversation with ${item.user.firstName}`}
              style={styles.item}
              onLongPress={() =>
                Alert.alert(`Unmatch with ${item.user.firstName}?`, "You will not see each other again.", [
                  { text: "Cancel", style: "cancel" },
                  { text: "Unmatch", style: "destructive", onPress: () => remove.mutate(item.id) }
                ])
              }
              onPress={() =>
                navigation.navigate("Thread", {
                  conversationId: item.conversationId,
                  userId: item.user.userId,
                  name: item.user.firstName
                })
              }
            >
              <View style={styles.center}>
                <Avatar url={item.user.photoUrl} name={item.user.firstName} size={96} />
                <Text variant="label" style={{ marginTop: spacing.sm }}>
                  {item.user.firstName}
                </Text>
              </View>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  item: { width: "48%", marginBottom: spacing.xl },
  center: { alignItems: "center" }
});
