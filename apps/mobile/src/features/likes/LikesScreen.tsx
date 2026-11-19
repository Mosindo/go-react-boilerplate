import React from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { discoveryApi } from "../../api/endpoints";
import { queryKeys } from "../../api/queryKeys";
import { formatDistance } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { radii, spacing } from "../../theme/tokens";
import { EmptyState, ErrorState, Header, LoadingState, RemoteImage, Screen, Text } from "../../ui";
import { useTheme } from "../../theme/ThemeProvider";

type Nav = NativeStackNavigationProp<AppStackParamList>;

/** Everyone sees who liked them — no paywall. */
export function LikesScreen() {
  const navigation = useNavigation<Nav>();
  const { colors } = useTheme();
  const query = useQuery({ queryKey: queryKeys.likesReceived, queryFn: discoveryApi.likesReceived });

  return (
    <Screen>
      <Header title="Vous ont liké" large />
      {query.isLoading ? (
        <LoadingState />
      ) : query.error ? (
        <ErrorState message={errorMessage(query.error)} onRetry={() => void query.refetch()} />
      ) : (
        <FlatList
          data={query.data}
          keyExtractor={(c) => c.userId}
          numColumns={2}
          columnWrapperStyle={styles.row}
          contentContainerStyle={styles.list}
          onRefresh={() => void query.refetch()}
          refreshing={query.isRefetching}
          ListHeaderComponent={
            query.data?.length ? (
              <Text tone="muted" style={styles.intro}>
                Ces personnes vous ont liké. Likez en retour pour matcher.
              </Text>
            ) : null
          }
          ListEmptyComponent={
            <EmptyState
              icon="heart-outline"
              title="Pas encore de like"
              message="Complétez votre profil avec plusieurs photos et une bio pour attirer l'attention."
            />
          }
          renderItem={({ item }) => (
            <Pressable
              accessibilityRole="button"
              accessibilityLabel={`${item.firstName}, ${item.age} ans. Voir le profil`}
              onPress={() => navigation.navigate("ProfileDetail", { userId: item.userId, canSwipe: true })}
              style={[styles.tile, { backgroundColor: colors.surfaceMuted }]}
              testID={`like-${item.userId}`}
            >
              <RemoteImage path={item.photos[0]?.url} style={StyleSheet.absoluteFill} />
              <View style={[styles.caption, { backgroundColor: colors.scrim }]}>
                <Text variant="label" tone="onPhoto" numberOfLines={1}>
                  {item.firstName}, {item.age}
                </Text>
                {formatDistance(item.distanceKm) ? (
                  <Text variant="caption" tone="onPhoto" numberOfLines={1}>
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
  list: { paddingBottom: spacing.xl, flexGrow: 1 },
  row: { gap: spacing.sm, marginBottom: spacing.sm },
  intro: { marginBottom: spacing.sm },
  tile: { flex: 1, maxWidth: "50%", aspectRatio: 3 / 4, borderRadius: radii.md, overflow: "hidden", justifyContent: "flex-end" },
  caption: { padding: spacing.sm }
});
