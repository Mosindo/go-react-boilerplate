import React from "react";
import { FlatList, Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { MatchItem } from "../api/types";
import { useMatches } from "../hooks/data";
import type { MainStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Header, SafeAreaLayout } from "../shared/layout";
import { PhotoImage, Text, radii, spacing } from "../shared/ui";
import { useTheme } from "../theme";

function MatchTile({ match, onPress }: { match: MatchItem; onPress: () => void }) {
  const { colors } = useTheme();
  const isNew = !match.hasMessages;
  return (
    <Pressable
      accessibilityLabel={`${match.user.firstName}, ${match.user.age} ans${isNew ? ", nouveau match" : ""}`}
      accessibilityRole="button"
      onPress={onPress}
      style={styles.tile}
    >
      <PhotoImage path={match.user.photos[0]?.thumbUrl} style={styles.photo} />
      <View pointerEvents="none" style={styles.caption}>
        <Text style={styles.name} variant="label">
          {match.user.firstName}, {match.user.age}
        </Text>
      </View>
      {isNew ? (
        <View style={[styles.tag, { backgroundColor: colors.primary }]}>
          <Text style={{ color: colors.onPrimary }} variant="caption" weight="bold">
            Nouveau
          </Text>
        </View>
      ) : null}
    </Pressable>
  );
}

export default function MatchesScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const query = useMatches();
  const matches = query.data?.pages.flatMap((page) => page.matches) ?? [];

  return (
    <SafeAreaLayout edges={["top"]}>
      <Header subtitle={matches.length ? `${matches.length} match${matches.length > 1 ? "s" : ""}` : undefined} title="Matchs" />
      {query.isLoading ? (
        <LoadingView fullScreen />
      ) : query.isError ? (
        <ErrorView message="Impossible de charger vos matchs." onAction={() => void query.refetch()} />
      ) : matches.length === 0 ? (
        <EmptyView
          icon="heart-outline"
          message="Quand vous et une autre personne vous plairez mutuellement, votre match apparaîtra ici."
          title="Pas encore de match"
        />
      ) : (
        <FlatList
          columnWrapperStyle={styles.columns}
          contentContainerStyle={styles.list}
          data={matches}
          keyExtractor={(match) => match.id}
          numColumns={2}
          onEndReached={() => query.hasNextPage && void query.fetchNextPage()}
          onRefresh={() => void query.refetch()}
          refreshing={query.isRefetching && !query.isFetchingNextPage}
          renderItem={({ item }) => (
            <MatchTile
              match={item}
              onPress={() =>
                navigation.navigate("Chat", {
                  conversationId: item.conversationId,
                  matchId: item.id,
                  user: { id: item.user.id, firstName: item.user.firstName, photoPath: item.user.photos[0]?.thumbUrl }
                })
              }
            />
          )}
        />
      )}
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  list: { padding: spacing.lg, gap: spacing.md, width: "100%", maxWidth: 560, alignSelf: "center" },
  columns: { gap: spacing.md },
  tile: { flex: 1, aspectRatio: 3 / 4, borderRadius: radii.lg, overflow: "hidden" },
  photo: { flex: 1 },
  caption: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.sm, backgroundColor: "rgba(20,8,16,0.45)" },
  name: { color: "#FFFFFF" },
  tag: { position: "absolute", top: spacing.sm, left: spacing.sm, borderRadius: radii.pill, paddingHorizontal: 8, paddingVertical: 2 }
});
