import React from "react";
import { FlatList, Pressable, RefreshControl, StyleSheet, View } from "react-native";
import type { CompositeScreenProps } from "@react-navigation/native";
import type { BottomTabScreenProps } from "@react-navigation/bottom-tabs";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { LinearGradient } from "expo-linear-gradient";
import { errorMessage } from "../../api/client";
import { useLikes } from "../../hooks/queries";
import type { AppStackParamList, TabParamList } from "../../navigation/types";
import { radii, spacing, useTheme } from "../../theme";
import { EmptyState, ErrorState, Header, LoadingState, RemoteImage, Screen, Text } from "../../ui";

type Props = CompositeScreenProps<BottomTabScreenProps<TabParamList, "Likes">, NativeStackScreenProps<AppStackParamList>>;

export function LikesScreen({ navigation }: Props) {
  const { colors } = useTheme();
  const likes = useLikes();

  return (
    <Screen testID="likes-screen">
      <Header large title="Ils vous ont liké" />
      <Text style={styles.sub} tone="muted">
        Gratuit et sans flou : likez en retour pour lancer la conversation.
      </Text>
      {likes.isLoading ? (
        <LoadingState />
      ) : likes.isError ? (
        <ErrorState message={errorMessage(likes.error)} onRetry={() => void likes.refetch()} />
      ) : (
        <FlatList
          columnWrapperStyle={styles.row}
          contentContainerStyle={styles.list}
          data={likes.data ?? []}
          keyExtractor={(p) => p.userId}
          ListEmptyComponent={
            <EmptyState icon="heart-outline" message="Quand quelqu'un vous likera, son profil apparaîtra ici." title="Pas encore de like" />
          }
          numColumns={2}
          refreshControl={
            <RefreshControl onRefresh={() => void likes.refetch()} refreshing={likes.isRefetching} tintColor={colors.primary} />
          }
          renderItem={({ item }) => (
            <Pressable
              accessibilityLabel={`${item.firstName}, ${item.age} ans`}
              accessibilityRole="button"
              onPress={() => navigation.navigate("ProfileDetail", { userId: item.userId, source: "likes" })}
              style={[styles.tile, { backgroundColor: colors.surfaceMuted }]}
              testID={`like-tile-${item.userId}`}
            >
              {item.photos[0] ? <RemoteImage style={StyleSheet.absoluteFill} url={item.photos[0].url} /> : null}
              <LinearGradient colors={["transparent", "rgba(15,8,12,0.85)"]} style={styles.tileInfo}>
                <Text numberOfLines={1} tone="inverse" variant="bodyStrong">
                  {item.firstName}, {item.age}
                </Text>
              </LinearGradient>
            </Pressable>
          )}
        />
      )}
      <View />
    </Screen>
  );
}

const styles = StyleSheet.create({
  sub: { marginBottom: spacing.md },
  list: { paddingBottom: spacing.xxl, flexGrow: 1 },
  row: { gap: spacing.sm },
  tile: { flex: 1, aspectRatio: 3 / 4, borderRadius: radii.lg, overflow: "hidden", marginBottom: spacing.sm },
  tileInfo: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.sm, paddingTop: spacing.xl }
});
