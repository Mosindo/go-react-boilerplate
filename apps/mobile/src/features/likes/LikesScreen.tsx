import React from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useInfiniteQuery } from "@tanstack/react-query";
import { LinearGradient } from "expo-linear-gradient";
import { matchingApi } from "../../api/matching";
import { queryKeys } from "../../api/queryKeys";
import type { ReceivedLike } from "../../api/types";
import { errorMessage } from "../../lib/errors";
import { formatDistance, relativeTime } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { radii, spacing, useTheme } from "../../theme";
import { EmptyState, ErrorState, Header, Loading, PhotoImage, Screen, Text } from "../../ui";
import { swipedIds } from "../discovery/hooks";

type Nav = NativeStackNavigationProp<AppStackParamList>;

export function LikesScreen() {
  const navigation = useNavigation<Nav>();
  const { colors } = useTheme();
  const likes = useInfiniteQuery({
    queryKey: queryKeys.likes,
    queryFn: ({ pageParam }) => matchingApi.likes(pageParam),
    initialPageParam: null as string | null,
    getNextPageParam: (last) => last.nextCursor
  });
  const items = (likes.data?.pages.flatMap((p) => p.likes) ?? []).filter((l) => !swipedIds.has(l.user.userId));

  const renderItem = ({ item }: { item: ReceivedLike }) => (
    <Pressable
      style={[styles.tile, { backgroundColor: colors.surface }]}
      onPress={() => navigation.navigate("ProfileDetail", { userId: item.user.userId, source: "likes" })}
      accessibilityRole="button"
      accessibilityLabel={`${item.user.firstName}, ${item.user.age} ans, vous a liké`}
      testID="like-tile"
    >
      <PhotoImage photo={item.user.photos[0]} style={StyleSheet.absoluteFill} />
      <LinearGradient colors={["transparent", "rgba(12,8,6,0.85)"]} style={styles.shade} />
      <View style={styles.caption}>
        <Text variant="heading" style={styles.white} numberOfLines={1}>
          {item.user.firstName}, {item.user.age}
        </Text>
        <Text variant="caption" style={styles.muted} numberOfLines={1}>
          {formatDistance(item.user.distanceKm) ?? relativeTime(item.likedAt)}
        </Text>
      </View>
    </Pressable>
  );

  return (
    <Screen testID="likes-screen">
      <Header title="Likes reçus" subtitle="Gratuit : découvrez qui vous a liké, et likez en retour." large />
      {likes.isPending ? (
        <Loading />
      ) : likes.isError ? (
        <ErrorState message={errorMessage(likes.error)} onRetry={() => void likes.refetch()} />
      ) : (
        <FlatList
          data={items}
          keyExtractor={(item) => item.user.userId}
          renderItem={renderItem}
          numColumns={2}
          columnWrapperStyle={styles.row}
          contentContainerStyle={[styles.list, items.length === 0 && styles.flex]}
          onEndReached={() => likes.hasNextPage && !likes.isFetchingNextPage && void likes.fetchNextPage()}
          onEndReachedThreshold={0.5}
          refreshing={likes.isRefetching}
          onRefresh={() => void likes.refetch()}
          ListEmptyComponent={
            <EmptyState
              icon="heart-outline"
              title="Pas encore de likes"
              message="Complétez votre profil et ajoutez plusieurs photos pour être plus visible."
            />
          }
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flexGrow: 1 },
  list: { paddingBottom: spacing.xl, gap: spacing.sm },
  row: { gap: spacing.sm },
  tile: { flex: 1, aspectRatio: 3 / 4, borderRadius: radii.lg, overflow: "hidden", maxWidth: "50%" },
  shade: { position: "absolute", left: 0, right: 0, bottom: 0, height: "50%" },
  caption: { position: "absolute", left: spacing.sm, right: spacing.sm, bottom: spacing.sm },
  white: { color: "#F7EFE9" },
  muted: { color: "rgba(247,239,233,0.8)" }
});
