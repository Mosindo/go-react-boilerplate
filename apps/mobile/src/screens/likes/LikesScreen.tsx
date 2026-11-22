import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import { Image } from "expo-image";
import { useNavigation, type CompositeNavigationProp } from "@react-navigation/native";
import type { BottomTabNavigationProp } from "@react-navigation/bottom-tabs";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { errorMessage, mediaUrl } from "../../api/client";
import { useLikesReceived } from "../../hooks/queries";
import { formatDistance } from "../../lib/format";
import type { MainStackParamList, TabParamList } from "../../navigation/types";
import { EmptyState, ErrorState, Header, LoadingState, Screen, Text, colors, radii, spacing } from "../../ui";

type Nav = CompositeNavigationProp<BottomTabNavigationProp<TabParamList, "Likes">, NativeStackNavigationProp<MainStackParamList>>;

export default function LikesScreen() {
  const navigation = useNavigation<Nav>();
  const likes = useLikesReceived();

  return (
    <Screen testID="likes-screen">
      <Header subtitle="Gratuit, comme tout le reste." title="Ils vous ont liké" />
      {likes.isPending ? (
        <LoadingState />
      ) : likes.isError ? (
        <ErrorState message={errorMessage(likes.error)} onAction={() => void likes.refetch()} />
      ) : (
        <FlatList
          columnWrapperStyle={styles.row}
          contentContainerStyle={likes.data.length === 0 ? styles.emptyContainer : styles.list}
          data={likes.data}
          keyExtractor={(p) => p.userId}
          ListEmptyComponent={
            <EmptyState
              icon="heart-outline"
              message="Quand quelqu'un vous like, son profil apparaît ici. Likez-le en retour pour matcher."
              title="Pas encore de like"
            />
          }
          numColumns={2}
          refreshControl={
            <RefreshControl onRefresh={() => void likes.refetch()} refreshing={likes.isRefetching} tintColor={colors.accent} />
          }
          renderItem={({ item }) => (
            <Pressable
              accessibilityLabel={`${item.firstName}, ${item.age} ans, vous a liké`}
              accessibilityRole="button"
              onPress={() => navigation.navigate("ProfileDetail", { userId: item.userId, context: "likes" })}
              style={({ pressed }) => [styles.tile, pressed ? styles.pressed : null]}
              testID={`like-${item.userId}`}
            >
              {item.photos[0] ? (
                <Image
                  cachePolicy="memory-disk"
                  contentFit="cover"
                  source={{ uri: mediaUrl(item.photos[0].url) }}
                  style={StyleSheet.absoluteFill}
                />
              ) : null}
              <View style={styles.caption}>
                <Text variant="label">
                  {item.firstName}, {item.age}
                </Text>
                {item.distanceKm !== undefined ? (
                  <Text tone="muted" variant="caption">
                    {formatDistance(item.distanceKm)}
                  </Text>
                ) : null}
              </View>
            </Pressable>
          )}
        />
      )}
    </Screen>
  );
}

const styles = StyleSheet.create({
  list: {
    gap: spacing.sm,
    paddingBottom: spacing.xl
  },
  emptyContainer: {
    flexGrow: 1
  },
  row: {
    gap: spacing.sm
  },
  tile: {
    flex: 1,
    aspectRatio: 4 / 5,
    borderRadius: radii.lg,
    overflow: "hidden",
    backgroundColor: colors.surfaceHigh,
    justifyContent: "flex-end"
  },
  pressed: {
    opacity: 0.85
  },
  caption: {
    padding: spacing.sm,
    backgroundColor: colors.scrimBottom
  }
});
