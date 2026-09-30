import React, { useState } from "react";
import { ScrollView, StyleSheet, View, useWindowDimensions } from "react-native";
import { Image } from "expo-image";
import { useQuery } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { Chip, ErrorState, IconButton, LoadingState, Text } from "../../design/components";
import { useTheme } from "../../design/ThemeProvider";
import { ApiError, errorMessage } from "../../lib/api/client";
import { mediaUrl } from "../../lib/api/config";
import { discoveryApi, profileApi } from "../../lib/api/endpoints";
import { formatDistance, genderLabels, goalLabels } from "../../lib/format";
import { queryKeys } from "../../lib/queryClient";
import { showToast } from "../../lib/toast";
import type { AppStackParamList } from "../../navigation/types";
import { useProfile } from "../profile/hooks";
import { SafetySheet } from "../safety/SafetySheet";
import { swipeEvents } from "./swipeEvents";

type Props = NativeStackScreenProps<AppStackParamList, "ProfileDetail">;

export default function ProfileDetailScreen({ route, navigation }: Props) {
  const { userId, fromDiscovery } = route.params;
  const { colors, radii } = useTheme();
  const { width } = useWindowDimensions();
  const { data: me } = useProfile();
  const [safetyOpen, setSafetyOpen] = useState(false);
  const [busy, setBusy] = useState(false);
  const { data: profile, isLoading, error, refetch } = useQuery({
    queryKey: queryKeys.publicProfile(userId),
    queryFn: () => profileApi.publicProfile(userId)
  });

  if (isLoading) return <LoadingState />;
  if (error || !profile) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;

  const myInterests = new Set(me?.interests.map((i) => i.id));
  const photoWidth = Math.min(width, 640);
  const distance = formatDistance(profile.distanceKm);

  const swipe = async (action: "like" | "pass") => {
    setBusy(true);
    try {
      const result = await discoveryApi.swipe(userId, action);
      swipeEvents.emit(userId, result);
      navigation.goBack();
    } catch (e) {
      if (e instanceof ApiError && e.code === "already_swiped") {
        swipeEvents.emit(userId, null);
        navigation.goBack();
      } else {
        showToast(errorMessage(e), "error");
      }
    } finally {
      setBusy(false);
    }
  };

  return (
    <View style={[styles.fill, { backgroundColor: colors.background }]} testID="profile-detail">
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={{ width: photoWidth, alignSelf: "center" }}>
          <ScrollView horizontal pagingEnabled showsHorizontalScrollIndicator={false}>
            {profile.photos.map((photo, i) => (
              <Image
                accessibilityLabel={`Photo ${i + 1} de ${profile.firstName}`}
                cachePolicy="memory-disk"
                contentFit="cover"
                key={photo.id}
                source={{ uri: mediaUrl(photo.url) }}
                style={{ width: photoWidth, height: photoWidth * 1.25 }}
              />
            ))}
          </ScrollView>
        </View>
        <View style={styles.body}>
          <View style={styles.titleRow}>
            <Text style={styles.flex} variant="title">
              {profile.firstName}, {profile.age}
            </Text>
            <IconButton icon="shield-outline" label={`Signaler ou bloquer ${profile.firstName}`} onPress={() => setSafetyOpen(true)} testID="profile-safety" />
          </View>
          <View style={styles.facts}>
            <Fact label={genderLabels[profile.gender]} />
            {profile.jobTitle ? <Fact label={profile.jobTitle} /> : null}
            {distance ? <Fact label={distance} /> : null}
            {profile.city ? <Fact label={profile.city} /> : null}
          </View>
          {profile.relationshipGoal ? (
            <View style={[styles.goal, { backgroundColor: colors.accentSoft, borderRadius: radii.md }]}>
              <Text tone="muted" variant="overline">
                RECHERCHE
              </Text>
              <Text variant="label">{goalLabels[profile.relationshipGoal]}</Text>
            </View>
          ) : null}
          {profile.bio ? (
            <View style={styles.section}>
              <Text tone="muted" variant="overline">
                À PROPOS
              </Text>
              <Text>{profile.bio}</Text>
            </View>
          ) : null}
          {profile.interests.length ? (
            <View style={styles.section}>
              <Text tone="muted" variant="overline">
                CENTRES D’INTÉRÊT
              </Text>
              <View style={styles.chips}>
                {profile.interests.map((interest) => (
                  <Chip highlighted={myInterests.has(interest.id)} key={interest.id} label={interest.label} />
                ))}
              </View>
            </View>
          ) : null}
        </View>
      </ScrollView>
      {fromDiscovery ? (
        <View style={[styles.actions, { backgroundColor: colors.background, borderTopColor: colors.border }]}>
          <IconButton disabled={busy} filled icon="close" label="Passer" onPress={() => void swipe("pass")} size={60} tone="muted" />
          <IconButton disabled={busy} filled icon="heart" label="J'aime" onPress={() => void swipe("like")} size={60} testID="detail-like" tone="primary" />
        </View>
      ) : null}
      <SafetySheet
        name={profile.firstName}
        onBlocked={() => {
          swipeEvents.emit(userId, null);
          navigation.popToTop();
        }}
        onClose={() => setSafetyOpen(false)}
        userId={userId}
        visible={safetyOpen}
      />
    </View>
  );
}

function Fact({ label }: { label: string }) {
  const { colors, radii } = useTheme();
  return (
    <View style={{ paddingHorizontal: 12, paddingVertical: 6, borderRadius: radii.pill, backgroundColor: colors.surfaceMuted }}>
      <Text variant="caption">{label}</Text>
    </View>
  );
}

const styles = StyleSheet.create({
  fill: { flex: 1 },
  flex: { flex: 1 },
  scroll: { paddingBottom: 120 },
  body: { padding: 20, gap: 18, width: "100%", maxWidth: 640, alignSelf: "center" },
  titleRow: { flexDirection: "row", alignItems: "center" },
  facts: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  goal: { padding: 14, gap: 4 },
  section: { gap: 8 },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: 8 },
  actions: {
    position: "absolute",
    bottom: 0,
    left: 0,
    right: 0,
    flexDirection: "row",
    justifyContent: "center",
    gap: 32,
    paddingTop: 12,
    paddingBottom: 24,
    borderTopWidth: StyleSheet.hairlineWidth
  }
});
