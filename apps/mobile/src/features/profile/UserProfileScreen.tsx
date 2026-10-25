import React, { useState } from "react";
import {
  FlatList,
  StyleSheet,
  View,
  useWindowDimensions,
  type NativeScrollEvent,
  type NativeSyntheticEvent
} from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useSafeAreaInsets } from "react-native-safe-area-context";
import { useQuery } from "@tanstack/react-query";
import { profileApi, queryKeys, type Photo } from "../../api";
import { formatDistance } from "../../shared/format";
import {
  ActionSheet,
  AuthImage,
  Chip,
  ErrorState,
  IconButton,
  Screen,
  Skeleton,
  Text,
  type ActionSheetOption
} from "../../shared/ui";
import { spacing, useTheme } from "../../theme";
import type { MainStackParamList } from "../../navigation/types";
import { usePersonSafety } from "../safety/usePersonSafety";

type Props = NativeStackScreenProps<MainStackParamList, "UserProfile">;

export function UserProfileScreen({ navigation, route }: Props) {
  const { userId } = route.params;
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const insets = useSafeAreaInsets();
  const [page, setPage] = useState(0);
  const [menuOpen, setMenuOpen] = useState(false);
  const query = useQuery({
    queryKey: queryKeys.publicProfile(userId),
    queryFn: () => profileApi.publicProfile(userId)
  });
  const profile = query.data;

  const safety = usePersonSafety({
    userId,
    firstName: profile?.firstName ?? "",
    onBlocked: () => navigation.popToTop()
  });

  const onScrollEnd = (event: NativeSyntheticEvent<NativeScrollEvent>) => {
    setPage(Math.round(event.nativeEvent.contentOffset.x / width));
  };

  const menu: ActionSheetOption[] = [
    { label: "Signaler", destructive: true, onPress: safety.openReport, testID: "profile-report" },
    { label: "Bloquer", destructive: true, onPress: safety.confirmBlock, testID: "profile-block" }
  ];

  const header = (
    <View style={[styles.topBar, { top: insets.top + spacing.sm }]}>
      <IconButton
        color="onScrim"
        icon="chevron-back"
        label="Retour"
        onPress={() => navigation.goBack()}
        style={{ backgroundColor: colors.scrim }}
      />
      {profile ? (
        <IconButton
          color="onScrim"
          icon="ellipsis-horizontal"
          label="Plus d'options"
          onPress={() => setMenuOpen(true)}
          style={{ backgroundColor: colors.scrim }}
          testID="profile-menu-button"
        />
      ) : null}
    </View>
  );

  if (query.isError) {
    return (
      <Screen edges={["bottom"]}>
        {header}
        <ErrorState message="Ce profil n'est plus disponible." onRetry={() => void query.refetch()} />
      </Screen>
    );
  }

  if (!profile) {
    return (
      <Screen edges={["bottom"]} padded={false}>
        <Skeleton height={width * 1.25} style={styles.fullWidth} />
        {header}
      </Screen>
    );
  }

  const distance = formatDistance(profile.distanceKm);

  return (
    <Screen edges={["bottom"]} padded={false} scroll testID="user-profile-screen">
      <View>
        <FlatList<Photo>
          data={profile.photos}
          getItemLayout={(_data, index) => ({ length: width, offset: width * index, index })}
          horizontal
          keyExtractor={(photo) => photo.id}
          onMomentumScrollEnd={onScrollEnd}
          pagingEnabled
          renderItem={({ item }) => <AuthImage path={item.url} style={{ width, height: width * 1.25 }} />}
          showsHorizontalScrollIndicator={false}
        />
        {profile.photos.length > 1 ? (
          <View pointerEvents="none" style={styles.dots}>
            {profile.photos.map((photo, index) => (
              <View
                key={photo.id}
                style={[styles.dot, { backgroundColor: index === page ? colors.onScrim : "rgba(255,255,255,0.45)" }]}
              />
            ))}
          </View>
        ) : null}
        {header}
      </View>
      <View style={styles.body}>
        <Text variant="title">{`${profile.firstName}, ${profile.age}`}</Text>
        <Text color="textMuted">{[profile.city, distance].filter(Boolean).join(" · ")}</Text>
        {profile.bio ? <Text>{profile.bio}</Text> : null}
        {profile.interests.length > 0 ? (
          <View style={styles.interests}>
            <Text variant="heading">Centres d&apos;intérêt</Text>
            <View style={styles.chips}>
              {profile.interests.map((interest) => (
                <Chip
                  highlighted={profile.sharedInterests.includes(interest.slug)}
                  key={interest.slug}
                  label={interest.label}
                />
              ))}
            </View>
            {profile.sharedInterests.length > 0 ? (
              <Text color="textMuted" variant="caption">
                Les centres d&apos;intérêt en commun sont mis en avant.
              </Text>
            ) : null}
          </View>
        ) : null}
      </View>
      <ActionSheet onClose={() => setMenuOpen(false)} options={menu} title={profile.firstName} visible={menuOpen} />
      {safety.reportModal}
    </Screen>
  );
}

const styles = StyleSheet.create({
  fullWidth: { borderRadius: 0 },
  topBar: {
    position: "absolute",
    left: spacing.lg,
    right: spacing.lg,
    flexDirection: "row",
    justifyContent: "space-between"
  },
  dots: {
    position: "absolute",
    bottom: spacing.md,
    left: 0,
    right: 0,
    flexDirection: "row",
    justifyContent: "center",
    gap: 6
  },
  dot: { width: 8, height: 8, borderRadius: 4 },
  body: { paddingHorizontal: spacing.xl, paddingTop: spacing.lg, gap: spacing.md },
  interests: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
