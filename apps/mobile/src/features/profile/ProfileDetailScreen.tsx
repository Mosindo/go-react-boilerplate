import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import React, { useLayoutEffect, useRef, useState } from "react";
import { FlatList, ScrollView, StyleSheet, View, useWindowDimensions } from "react-native";
import type { PublicProfile } from "../../api/types";
import { usePublicProfile } from "../../hooks/useProfile";
import { formatDistance } from "../../lib/format";
import type { MainStackParamList } from "../../navigation/types";
import { pushDeckIntent } from "../../store/deckIntent";
import { spacing, useColors } from "../../theme";
import { Button, Chip, ErrorState, Icon, IconButton, LoadingState, ProfilePhoto, Screen, Text } from "../../ui";
import { useSafetyMenu } from "../safety/useSafetyMenu";
import { GENDER_LABELS } from "./labels";

type Props = NativeStackScreenProps<MainStackParamList, "ProfileDetail">;

export function ProfileDetailScreen({ navigation, route }: Props) {
  const { userId, source } = route.params;
  const query = usePublicProfile(userId);

  if (query.isPending) return <LoadingState />;
  if (query.isError || !query.data) {
    return (
      <Screen>
        <ErrorState error={query.error} onRetry={() => query.refetch()} title="This profile isn't available" />
        <Button label="Go back" onPress={() => navigation.goBack()} variant="ghost" />
      </Screen>
    );
  }
  return <Detail navigation={navigation} profile={query.data} source={source} />;
}

function Detail({
  profile,
  source,
  navigation
}: {
  profile: PublicProfile;
  source: Props["route"]["params"]["source"];
  navigation: Props["navigation"];
}) {
  const colors = useColors();
  const { width } = useWindowDimensions();
  const pageWidth = Math.min(width, 560);
  const [page, setPage] = useState(0);
  const listRef = useRef<FlatList>(null);
  const photos = [...profile.photos].sort((a, b) => a.position - b.position);
  const distance = formatDistance(profile.distanceKm);

  const safety = useSafetyMenu({
    target: { userId: profile.userId, displayName: profile.displayName },
    onBlocked: () => {
      pushDeckIntent({ userId: profile.userId, kind: "remove" });
      navigation.popToTop();
    }
  });

  useLayoutEffect(() => {
    navigation.setOptions({
      title: profile.displayName,
      headerRight: () => <IconButton icon="ellipsis-horizontal" label="More options" onPress={safety.open} />
    });
  });

  function decide(kind: "like" | "pass") {
    pushDeckIntent({ userId: profile.userId, kind });
    navigation.goBack();
  }

  return (
    <Screen edges={["left", "right", "bottom"]} padded={false}>
      <ScrollView>
        <View>
          <View style={{ width: pageWidth, alignSelf: "center" }}>
            <FlatList
              ref={listRef}
              data={photos.length ? photos : [null]}
              horizontal
              keyExtractor={(item, index) => item?.id ?? `empty-${index}`}
              onMomentumScrollEnd={(e) => setPage(Math.round(e.nativeEvent.contentOffset.x / pageWidth))}
              pagingEnabled
              renderItem={({ item }) => (
                <ProfilePhoto
                  name={profile.displayName}
                  photo={item}
                  size="full"
                  style={{ width: pageWidth, height: pageWidth * 1.25 }}
                />
              )}
              showsHorizontalScrollIndicator={false}
            />
            {photos.length > 1 ? (
              <View accessibilityLabel={`Photo ${page + 1} of ${photos.length}`} style={styles.dots}>
                {photos.map((photo, i) => (
                  <View
                    key={photo.id}
                    style={[styles.dot, { backgroundColor: i === page ? colors.primary : colors.border }]}
                  />
                ))}
              </View>
            ) : null}
          </View>
          <View style={styles.info}>
            <Text variant="heading">
              {profile.displayName}, {profile.age}
            </Text>
            <View style={styles.meta}>
              <Icon color={colors.textMuted} name="person-outline" size={16} />
              <Text tone="muted">{GENDER_LABELS[profile.gender]}</Text>
            </View>
            {profile.city || distance ? (
              <View style={styles.meta}>
                <Icon color={colors.textMuted} name="location-outline" size={16} />
                <Text tone="muted">{[profile.city, distance].filter(Boolean).join("  ·  ")}</Text>
              </View>
            ) : null}
            {profile.bio ? <Text style={styles.bio}>{profile.bio}</Text> : null}
            {profile.interests.length > 0 ? (
              <>
                <Text style={styles.section} weight="semibold">
                  Interests
                  {profile.commonInterests > 0 ? ` · ${profile.commonInterests} in common` : ""}
                </Text>
                <View style={styles.chips}>
                  {profile.interests.map((interest) => (
                    <Chip key={interest.slug} label={interest.label} />
                  ))}
                </View>
              </>
            ) : null}
          </View>
        </View>
      </ScrollView>
      {source === "discover" ? (
        <View style={[styles.bar, { backgroundColor: colors.background, borderColor: colors.border }]}>
          <View style={styles.barButton}>
            <Button icon="close" label="Pass" onPress={() => decide("pass")} variant="ghost" />
          </View>
          <View style={styles.barButton}>
            <Button icon="heart" label="Like" onPress={() => decide("like")} />
          </View>
        </View>
      ) : null}
      {safety.element}
    </Screen>
  );
}

const styles = StyleSheet.create({
  dots: { flexDirection: "row", justifyContent: "center", gap: 6, paddingTop: spacing.xs },
  dot: { width: 8, height: 8, borderRadius: 4 },
  info: { padding: spacing.md, gap: spacing.xs, maxWidth: 560, width: "100%", alignSelf: "center" },
  meta: { flexDirection: "row", alignItems: "center", gap: spacing.xxs + 2 },
  bio: { marginTop: spacing.xs },
  section: { marginTop: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", columnGap: spacing.xs },
  bar: {
    flexDirection: "row",
    gap: spacing.sm,
    padding: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  barButton: { flex: 1 }
});
