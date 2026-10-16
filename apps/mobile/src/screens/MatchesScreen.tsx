import React from "react";
import { FlatList, Image, Pressable, RefreshControl, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { absoluteUrl } from "../api/client";
import type { Match, PublicProfile } from "../api/types";
import { useLikesReceived, useMatches } from "../hooks/queries";
import { pluralize } from "../lib/format";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { Header, ScreenContainer } from "../shared/layout";
import { Text, colors, radii, spacing } from "../shared/ui";

/** Matches tab: who liked you (free, always visible) and the people you matched with. */
export default function MatchesScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const likes = useLikesReceived();
  const matches = useMatches();

  const refreshing = likes.isRefetching || matches.isRefetching;
  const refresh = () => {
    void likes.refetch();
    void matches.refetch();
  };

  const likeCount = likes.data?.total ?? 0;

  return (
    <ScreenContainer testID="screen-matches">
      <Header subtitle="Tout est gratuit : voyez qui vous aime, sans limite." title="Matchs" />
      {matches.isLoading ? <LoadingView label="Chargement…" /> : null}
      {matches.isError && !matches.data ? <ErrorView message="Impossible de charger vos matchs." onAction={refresh} /> : null}

      {matches.data ? (
        <ScrollView
          contentContainerStyle={styles.scroll}
          refreshControl={<RefreshControl onRefresh={refresh} refreshing={refreshing} tintColor={colors.primary} />}
          showsVerticalScrollIndicator={false}
        >
          <View style={styles.section}>
            <Text variant="heading" weight="bold">
              {likeCount > 0 ? `${likeCount} ${pluralize(likeCount, "personne vous a", "personnes vous ont")} liké` : "Ils vous ont liké"}
            </Text>
            {likes.data && likes.data.profiles.length > 0 ? (
              <FlatList<PublicProfile>
                contentContainerStyle={styles.likesRow}
                data={likes.data.profiles}
                horizontal
                keyExtractor={(item) => item.id}
                renderItem={({ item }) => (
                  <PhotoTile
                    onPress={() => navigation.navigate("ProfileDetail", { userId: item.id, source: "likes" })}
                    profile={item}
                    testID={`like-${item.id}`}
                    width={132}
                  />
                )}
                showsHorizontalScrollIndicator={false}
              />
            ) : (
              <Text tone="muted">Personne en attente pour le moment. Continuez à découvrir des profils !</Text>
            )}
          </View>

          <View style={styles.section}>
            <Text variant="heading" weight="bold">
              Vos matchs
            </Text>
            {matches.data.length === 0 ? (
              <EmptyView
                message="Quand l'intérêt est réciproque, votre match apparaît ici."
                testID="matches-empty"
                title="Pas encore de match"
              />
            ) : (
              <View style={styles.grid}>
                {matches.data.map((match: Match) => (
                  <PhotoTile
                    key={match.id}
                    onPress={() => navigation.navigate("Thread", { matchId: match.id, profile: match.profile })}
                    profile={match.profile}
                    testID={`match-${match.id}`}
                    width="48%"
                  />
                ))}
              </View>
            )}
          </View>
        </ScrollView>
      ) : null}
    </ScreenContainer>
  );
}

function PhotoTile({
  profile,
  onPress,
  width,
  testID
}: {
  profile: PublicProfile;
  onPress: () => void;
  width: number | `${number}%`;
  testID: string;
}) {
  const photo = profile.photos[0];
  return (
    <Pressable
      accessibilityLabel={`${profile.firstName}, ${profile.age} ans`}
      accessibilityRole="button"
      onPress={onPress}
      style={[styles.tile, { width }]}
      testID={testID}
    >
      {photo ? <Image source={{ uri: absoluteUrl(photo.url) }} style={styles.tileImage} /> : <View style={styles.tileImage} />}
      <View style={styles.tileCaption}>
        <Text tone="inverse" variant="label" weight="bold">
          {profile.firstName}, {profile.age}
        </Text>
      </View>
    </Pressable>
  );
}

const styles = StyleSheet.create({
  scroll: { paddingTop: spacing.lg, paddingBottom: spacing.xxxl, gap: spacing.xl },
  section: { gap: spacing.md },
  likesRow: { gap: spacing.md },
  grid: { flexDirection: "row", flexWrap: "wrap", gap: spacing.md },
  tile: { aspectRatio: 4 / 5, borderRadius: radii.lg, overflow: "hidden", backgroundColor: colors.surfaceSubtle },
  tileImage: { ...StyleSheet.absoluteFillObject, width: "100%", height: "100%" },
  tileCaption: { position: "absolute", left: 0, right: 0, bottom: 0, padding: spacing.sm, backgroundColor: colors.scrim }
});
