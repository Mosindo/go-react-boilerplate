import React, { useLayoutEffect, useRef, useState } from "react";
import { FlatList, Pressable, StyleSheet, View, useWindowDimensions, type NativeScrollEvent, type NativeSyntheticEvent } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { describeError, isApiError } from "../api/client";
import { useUserActions } from "../components/UserActions";
import { useMe, useProfile, useSwipe } from "../hooks/data";
import { useRealtime } from "../hooks/useRealtime";
import type { MainStackParamList } from "../navigation/types";
import { ErrorView, LoadingView, showToast } from "../shared/feedback";
import { formatDistance, GENDER_LABELS, LOOKING_FOR_LABELS } from "../shared/format";
import { Button, Chip, IconButton, PhotoImage, Sheet, Text, spacing } from "../shared/ui";
import { markSwiped, unmarkSwiped } from "../state/swiped";
import { useTheme } from "../theme";

const MAX_WIDTH = 560;

export default function ProfileDetailScreen({ navigation, route }: NativeStackScreenProps<MainStackParamList, "ProfileDetail">) {
  const { userId, matchId, conversationId, canSwipe } = route.params;
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const pageWidth = Math.min(width, MAX_WIDTH);
  const { data: profile, isLoading, error, refetch } = useProfile(userId);
  const swipe = useSwipe();
  const { data: me } = useMe();
  const mine = new Set(me?.profile?.interests.map((i) => i.slug) ?? []);
  const actions = useUserActions();
  const { announceMatch } = useRealtime();
  const [menu, setMenu] = useState(false);
  const [page, setPage] = useState(0);
  const listRef = useRef<FlatList>(null);

  useLayoutEffect(() => {
    navigation.setOptions({
      title: profile?.firstName ?? "",
      headerRight: () => <IconButton icon="ellipsis-horizontal" label="Plus d'options" onPress={() => setMenu(true)} />
    });
  }, [navigation, profile?.firstName]);

  if (isLoading) return <LoadingView fullScreen />;
  if (error || !profile) {
    const gone = isApiError(error) && error.status === 404;
    return (
      <ErrorView
        actionLabel={gone ? "Retour" : "Réessayer"}
        message={gone ? "Ce profil n'est plus disponible." : describeError(error)}
        onAction={() => (gone ? navigation.goBack() : void refetch())}
        title={gone ? "Profil introuvable" : undefined}
      />
    );
  }

  const decide = async (action: "like" | "pass") => {
    markSwiped(profile.id);
    try {
      const result = await swipe.mutateAsync({ userId: profile.id, action });
      if (result.matched && !result.alreadyProcessed && result.matchId && result.conversationId) {
        announceMatch({ matchId: result.matchId, conversationId: result.conversationId, userId: profile.id });
      }
      navigation.goBack();
    } catch (e) {
      unmarkSwiped(profile.id);
      showToast(describeError(e), "error");
    }
  };

  const onScroll = (event: NativeSyntheticEvent<NativeScrollEvent>) => setPage(Math.round(event.nativeEvent.contentOffset.x / pageWidth));
  const distance = formatDistance(profile.distanceKm);

  const menuActions = [
    ...(matchId
      ? [
          {
            label: "Annuler le match",
            icon: "heart-dislike-outline" as const,
            destructive: true,
            onPress: () => void actions.unmatchUser(matchId, profile.firstName).then((done) => done && navigation.popToTop())
          }
        ]
      : []),
    { label: "Signaler", icon: "flag-outline" as const, onPress: () => navigation.navigate("Report", { userId: profile.id, firstName: profile.firstName }) },
    {
      label: "Bloquer",
      icon: "ban-outline" as const,
      destructive: true,
      onPress: () => void actions.blockUser(profile.id, profile.firstName).then((done) => done && navigation.popToTop())
    }
  ];

  return (
    <View style={[styles.flex, { backgroundColor: colors.background }]}>
      <FlatList
        ListHeaderComponent={
          <View style={{ width: pageWidth, alignSelf: "center" }}>
            <FlatList
              data={profile.photos}
              getItemLayout={(_, index) => ({ length: pageWidth, offset: pageWidth * index, index })}
              horizontal
              keyExtractor={(photo) => photo.id}
              onMomentumScrollEnd={onScroll}
              pagingEnabled
              ref={listRef}
              renderItem={({ item, index }) => (
                <PhotoImage
                  label={`Photo ${index + 1} sur ${profile.photos.length}`}
                  path={item.url}
                  style={{ width: pageWidth, height: pageWidth * (4 / 3) }}
                />
              )}
              showsHorizontalScrollIndicator={false}
            />
            {profile.photos.length > 1 ? (
              <View style={styles.dots}>
                {profile.photos.map((photo, index) => (
                  <Pressable
                    accessibilityLabel={`Voir la photo ${index + 1}`}
                    accessibilityRole="button"
                    hitSlop={8}
                    key={photo.id}
                    onPress={() => {
                      listRef.current?.scrollToIndex({ index, animated: true });
                      setPage(index);
                    }}
                    style={[styles.dot, { backgroundColor: index === page ? colors.primary : colors.border }]}
                  />
                ))}
              </View>
            ) : null}
          </View>
        }
        contentContainerStyle={styles.content}
        data={[]}
        renderItem={null}
        ListFooterComponent={
          <View style={[styles.info, { maxWidth: MAX_WIDTH }]}>
            <Text accessibilityRole="header" variant="title">
              {profile.firstName}, {profile.age}
            </Text>
            {profile.occupation ? <Text tone="muted">{profile.occupation}</Text> : null}
            <View style={styles.row}>
              <Ionicons color={colors.textMuted} name="location-outline" size={16} />
              <Text tone="muted" variant="bodySmall">
                {[profile.city, distance].filter(Boolean).join(" · ") || "Lieu non renseigné"}
              </Text>
            </View>
            <View style={styles.row}>
              <Ionicons color={colors.textMuted} name="person-outline" size={16} />
              <Text tone="muted" variant="bodySmall">
                {GENDER_LABELS[profile.gender]} · Cherche : {LOOKING_FOR_LABELS[profile.lookingFor].toLowerCase()}
              </Text>
            </View>
            {profile.bio ? (
              <View style={styles.section}>
                <Text tone="muted" variant="eyebrow">
                  À propos
                </Text>
                <Text>{profile.bio}</Text>
              </View>
            ) : null}
            {profile.interests.length > 0 ? (
              <View style={styles.section}>
                <Text tone="muted" variant="eyebrow">
                  Centres d'intérêt{profile.sharedInterests > 0 ? ` · ${profile.sharedInterests} en commun` : ""}
                </Text>
                <View style={styles.chips}>
                  {profile.interests.map((interest) => (
                    <Chip highlighted={mine.has(interest.slug)} key={interest.slug} label={interest.label} />
                  ))}
                </View>
              </View>
            ) : null}
          </View>
        }
      />

      {canSwipe ? (
        <View style={[styles.bar, { backgroundColor: colors.background, borderTopColor: colors.border }]}>
          <Button disabled={swipe.isPending} icon="close" onPress={() => void decide("pass")} style={styles.barButton} variant="outline">
            Passer
          </Button>
          <Button disabled={swipe.isPending} icon="heart" onPress={() => void decide("like")} style={styles.barButton}>
            J'aime
          </Button>
        </View>
      ) : conversationId ? (
        <View style={[styles.bar, { backgroundColor: colors.background, borderTopColor: colors.border }]}>
          <Button
            icon="chatbubble-ellipses-outline"
            onPress={() =>
              navigation.navigate("Chat", {
                conversationId,
                matchId,
                user: { id: profile.id, firstName: profile.firstName, photoPath: profile.photos[0]?.thumbUrl }
              })
            }
            style={styles.barButton}
          >
            Envoyer un message
          </Button>
        </View>
      ) : null}

      <Sheet actions={menuActions} onClose={() => setMenu(false)} title={profile.firstName} visible={menu} />
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { paddingBottom: spacing.xxl },
  dots: { flexDirection: "row", justifyContent: "center", gap: spacing.xs, paddingVertical: spacing.sm },
  dot: { width: 28, height: 6, borderRadius: 3 },
  info: { width: "100%", alignSelf: "center", padding: spacing.xl, gap: spacing.sm },
  row: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  section: { gap: spacing.xs, marginTop: spacing.md },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  bar: { flexDirection: "row", gap: spacing.md, padding: spacing.lg, borderTopWidth: StyleSheet.hairlineWidth, justifyContent: "center" },
  barButton: { flex: 1, maxWidth: 260 }
});
