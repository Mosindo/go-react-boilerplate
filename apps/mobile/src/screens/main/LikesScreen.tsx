import React from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import { useInfiniteQuery } from "@tanstack/react-query";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { receivedLikes } from "../../api/social";
import type { MainStackParams, TabParams } from "../../navigation/types";
import { EmptyState, ErrorState, Loading, PhotoImage, Text } from "../../ui";
import { radius, spacing } from "../../theme/tokens";
import { useTheme } from "../../theme/ThemeProvider";

type Props = CompositeScreenProps<
  BottomTabScreenProps<TabParams, "Likes">,
  NativeStackScreenProps<MainStackParams>
>;

export default function LikesScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const query = useInfiniteQuery({
    queryKey: ["likes"],
    queryFn: ({ pageParam }) => receivedLikes(pageParam),
    initialPageParam: undefined as string | undefined,
    getNextPageParam: (last) => last.nextCursor
  });

  if (query.isPending) return <Loading />;
  if (query.isError) return <ErrorState error={query.error} onRetry={() => void query.refetch()} />;

  const profiles = query.data.pages.flatMap((page) => page.profiles);
  if (profiles.length === 0) {
    return (
      <EmptyState
        icon="♥"
        title="Pas encore de like"
        message="Les personnes qui vous ont déjà liké apparaîtront ici. C'est gratuit, et sans limite."
      />
    );
  }

  return (
    <FlatList
      testID="likes-list"
      data={profiles}
      numColumns={2}
      keyExtractor={(p) => p.userId}
      contentContainerStyle={styles.list}
      columnWrapperStyle={styles.row}
      refreshing={query.isRefetching}
      onRefresh={() => void query.refetch()}
      onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
      onEndReachedThreshold={0.5}
      renderItem={({ item }) => (
        <Pressable
          accessibilityRole="button"
          accessibilityLabel={`${item.firstName}, ${item.age} ans, vous a liké`}
          style={[styles.tile, { backgroundColor: colors.surface }]}
          onPress={() =>
            navigation.navigate("ProfileDetail", { userId: item.userId, profile: item, canSwipe: true })
          }
        >
          <PhotoImage photoId={item.photos[0]?.id} name={item.firstName} style={styles.photo} />
          <View style={[styles.caption, { backgroundColor: colors.scrim }]}>
            <Text bold style={styles.white}>
              {item.firstName}, {item.age}
            </Text>
          </View>
        </Pressable>
      )}
    />
  );
}

const styles = StyleSheet.create({
  list: { padding: spacing.md, gap: spacing.md },
  row: { gap: spacing.md },
  tile: { flex: 1, aspectRatio: 3 / 4, borderRadius: radius.md, overflow: "hidden" },
  photo: { width: "100%", height: "100%" },
  caption: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.sm },
  white: { color: "#fff" }
});
