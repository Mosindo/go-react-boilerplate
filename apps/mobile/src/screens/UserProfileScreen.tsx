import React, { useCallback, useState } from "react";
import {
  Alert,
  Pressable,
  ScrollView,
  StyleSheet,
  View,
  useWindowDimensions,
  type NativeScrollEvent,
  type NativeSyntheticEvent
} from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { StackActions } from "@react-navigation/native";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { fetchUserProfile, postSwipe } from "../api/discover";
import { ErrorView } from "../shared/feedback";
import { Button, Text, radii, spacing } from "../shared/ui";
import { formatDistance } from "../features/discover/distance";
import { emitSwipeOutcome } from "../features/discover/swipeBus";
import { errorMessage, isApiErrorCode } from "../features/errors";
import { useAppNavigation, useUserProfileParams } from "../features/nav";
import { queryKeys } from "../features/queryKeys";
import { ReportModal } from "../features/safety/ReportModal";
import { useSafetyActions } from "../features/safety/useSafetyActions";
import type { SwipeAction } from "../features/types";
import { Chip } from "../features/ui/Chip";
import { PhotoImage } from "../features/ui/PhotoImage";
import { ScreenHeader } from "../features/ui/ScreenHeader";
import { Skeleton } from "../features/ui/Skeleton";
import { usePalette } from "../features/ui/palette";

export default function UserProfileScreen() {
  const p = usePalette();
  const navigation = useAppNavigation();
  const queryClient = useQueryClient();
  const { userId, conversationId } = useUserProfileParams();
  const { width } = useWindowDimensions();
  const [photoPage, setPhotoPage] = useState(0);
  const [swiping, setSwiping] = useState(false);

  const query = useQuery({
    queryKey: queryKeys.userProfile(userId),
    queryFn: () => fetchUserProfile(userId),
    retry: false
  });
  const profile = query.data;
  const isMatchContext = conversationId !== undefined;

  const leave = useCallback(() => {
    if (isMatchContext) {
      navigation.dispatch(StackActions.popToTop());
    } else if (navigation.canGoBack()) {
      navigation.goBack();
    }
  }, [isMatchContext, navigation]);

  const safety = useSafetyActions({ userId, name: profile?.firstName ?? "this person", onDone: leave });

  const swipe = useCallback(
    async (action: SwipeAction) => {
      setSwiping(true);
      try {
        const result = await postSwipe(userId, action);
        emitSwipeOutcome({ userId, action, matched: result.matched, match: result.match });
        void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
        navigation.goBack();
      } catch (err) {
        if (isApiErrorCode(err, "already_swiped")) {
          emitSwipeOutcome({ userId, action, matched: false, match: null });
          navigation.goBack();
        } else {
          const status = err instanceof ApiError ? err.status : 0;
          Alert.alert(
            "Couldn't save your choice",
            status === 0 || status >= 500 ? "Please try again." : errorMessage(err, "Please try again.")
          );
        }
      } finally {
        setSwiping(false);
      }
    },
    [navigation, queryClient, userId]
  );

  const onScroll = useCallback(
    (e: NativeSyntheticEvent<NativeScrollEvent>) => {
      setPhotoPage(Math.round(e.nativeEvent.contentOffset.x / Math.max(1, width)));
    },
    [width]
  );

  const unavailable = query.error instanceof ApiError && query.error.status === 404;
  const distance = profile ? formatDistance(profile.distanceKm) : null;
  const shared = new Set(profile?.sharedInterests ?? []);
  const photoHeight = Math.min(width * 1.25, 560);

  return (
    <SafeAreaView
      edges={["top", "bottom"]}
      style={[styles.root, { backgroundColor: p.background }]}
      testID="user-profile-screen"
    >
      <ScreenHeader onBack={() => navigation.goBack()} title={profile?.firstName ?? "Profile"} />
      {query.isLoading ? (
        <View style={styles.pad}>
          <Skeleton style={{ height: photoHeight, borderRadius: radii.xl }} />
          <Skeleton style={{ height: 28, width: "50%", marginTop: spacing.lg }} />
          <Skeleton style={{ height: 16, width: "80%", marginTop: spacing.sm }} />
        </View>
      ) : query.isError || !profile ? (
        <View style={styles.pad}>
          <ErrorView
            actionLabel={unavailable ? "Go back" : "Try again"}
            message={
              unavailable
                ? "This profile isn't available anymore."
                : errorMessage(query.error, "Couldn't load this profile.")
            }
            onAction={unavailable ? () => navigation.goBack() : () => void query.refetch()}
            title={unavailable ? "Profile unavailable" : "Something went wrong"}
          />
        </View>
      ) : (
        <>
          <ScrollView contentContainerStyle={styles.content}>
            {profile.photos.length > 0 ? (
              <View>
                <ScrollView
                  horizontal
                  onScroll={onScroll}
                  pagingEnabled
                  scrollEventThrottle={16}
                  showsHorizontalScrollIndicator={false}
                  testID="profile-photos"
                >
                  {profile.photos.map((photo, i) => (
                    <PhotoImage
                      accessibilityLabel={`Photo ${i + 1} of ${profile.photos.length} of ${profile.firstName}`}
                      key={photo.id}
                      photo={photo}
                      style={{ width, height: photoHeight }}
                    />
                  ))}
                </ScrollView>
                {profile.photos.length > 1 ? (
                  <View pointerEvents="none" style={styles.dots}>
                    {profile.photos.map((photo, i) => (
                      <View
                        key={photo.id}
                        style={[
                          styles.dot,
                          { backgroundColor: i === photoPage ? "#ffffff" : "rgba(255,255,255,0.45)" }
                        ]}
                      />
                    ))}
                  </View>
                ) : null}
              </View>
            ) : (
              <View style={[styles.noPhoto, { height: photoHeight / 2, backgroundColor: p.surfaceMuted }]}>
                <Text tone="muted">No photos yet</Text>
              </View>
            )}

            <View style={styles.pad}>
              <Text variant="title" weight="bold">
                {profile.firstName}, {profile.age}
              </Text>
              {profile.city ? <Text tone="muted">{profile.city}</Text> : null}
              {distance ? <Text tone="muted">{distance}</Text> : null}

              {profile.bio ? (
                <View style={styles.section}>
                  <Text tone="muted" variant="eyebrow" weight="bold">
                    About
                  </Text>
                  <Text>{profile.bio}</Text>
                </View>
              ) : null}

              {profile.interests.length > 0 ? (
                <View style={styles.section}>
                  <Text tone="muted" variant="eyebrow" weight="bold">
                    Interests
                  </Text>
                  <View style={styles.chips}>
                    {profile.interests.map((interest) => (
                      <Chip
                        highlighted={shared.has(interest.slug)}
                        key={interest.slug}
                        label={interest.label}
                      />
                    ))}
                  </View>
                  {shared.size > 0 ? (
                    <Text tone="muted" variant="caption">
                      Highlighted interests are ones you share.
                    </Text>
                  ) : null}
                </View>
              ) : null}

              <View style={styles.safety}>
                {isMatchContext ? (
                  <Button
                    label="Unmatch"
                    loading={safety.busy}
                    onPress={safety.confirmUnmatch}
                    testID="profile-unmatch"
                    variant="outline"
                  />
                ) : null}
                <Button
                  disabled={safety.busy}
                  label="Report"
                  onPress={safety.openReport}
                  testID="profile-report"
                  variant="outline"
                />
                <Button
                  disabled={safety.busy}
                  label="Block"
                  onPress={safety.confirmBlock}
                  testID="profile-block"
                  variant="destructive"
                />
              </View>
            </View>
          </ScrollView>

          {!isMatchContext ? (
            <View style={[styles.actions, { backgroundColor: p.background, borderTopColor: p.border }]}>
              <Pressable
                accessibilityLabel={`Pass on ${profile.firstName}`}
                accessibilityRole="button"
                disabled={swiping}
                onPress={() => void swipe("pass")}
                style={[styles.round, { borderColor: p.nope, backgroundColor: p.surface }]}
                testID="profile-pass-button"
              >
                <Text style={{ color: p.nope, fontSize: 26, lineHeight: 30 }} weight="bold">
                  {"✕"}
                </Text>
              </Pressable>
              <Pressable
                accessibilityLabel={`Like ${profile.firstName}`}
                accessibilityRole="button"
                disabled={swiping}
                onPress={() => void swipe("like")}
                style={[styles.round, { borderColor: p.like, backgroundColor: p.like }]}
                testID="profile-like-button"
              >
                <Text style={{ color: "#ffffff", fontSize: 28, lineHeight: 32 }} weight="bold">
                  {"♥"}
                </Text>
              </Pressable>
            </View>
          ) : null}
        </>
      )}
      <ReportModal
        busy={safety.busy}
        name={profile?.firstName ?? "this person"}
        onClose={safety.closeReport}
        onSubmit={(r, d) => void safety.submitReport(r, d)}
        visible={safety.reportVisible}
      />
    </SafeAreaView>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
  content: { paddingBottom: spacing.xxl },
  pad: { padding: spacing.lg, gap: spacing.xs },
  section: { marginTop: spacing.lg, gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  dots: {
    position: "absolute",
    bottom: spacing.md,
    left: 0,
    right: 0,
    flexDirection: "row",
    justifyContent: "center",
    gap: 6
  },
  dot: { width: 7, height: 7, borderRadius: 4 },
  noPhoto: { alignItems: "center", justifyContent: "center" },
  safety: { marginTop: spacing.xl, gap: spacing.sm },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    gap: spacing.xl,
    paddingVertical: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  round: {
    width: 64,
    height: 64,
    borderRadius: 32,
    borderWidth: 2,
    alignItems: "center",
    justifyContent: "center"
  }
});
