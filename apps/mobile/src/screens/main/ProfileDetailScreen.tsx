import React, { useState } from "react";
import { ScrollView, StyleSheet, View, useWindowDimensions } from "react-native";
import { SafeAreaView } from "react-native-safe-area-context";
import { useQueryClient } from "@tanstack/react-query";
import { discoveryApi } from "../../api/endpoints";
import { errorMessage } from "../../api/errors";
import type { SwipeDecision } from "../../api/types";
import { PhotoCarousel } from "../../components/PhotoCarousel";
import { SafetySheet } from "../../components/SafetySheet";
import { useFeedback } from "../../feedback";
import { useProfileCard } from "../../hooks/queries";
import { discoveryEvents } from "../../hooks/discoveryEvents";
import { queryKeys } from "../../hooks/queryClient";
import type { AppScreenProps } from "../../navigation/types";
import { layout, makeStyles, radii, spacing } from "../../theme";
import { Button, Chip, ErrorState, Icon, IconButton, LoadingState, Text, type IconName } from "../../ui";
import { genderLabels, goalLabels } from "../../utils/labels";

export default function ProfileDetailScreen({ navigation, route }: AppScreenProps<"ProfileDetail">) {
  const { userId, source } = route.params;
  const styles = useStyles();
  const qc = useQueryClient();
  const { toast } = useFeedback();
  const { width } = useWindowDimensions();
  const query = useProfileCard(userId);
  const [safetyOpen, setSafetyOpen] = useState(false);
  const [deciding, setDeciding] = useState(false);

  if (query.isLoading) {
    return (
      <SafeAreaView style={styles.root}>
        <LoadingState />
      </SafeAreaView>
    );
  }
  if (!query.data) {
    return (
      <SafeAreaView style={styles.root}>
        <ErrorState message="Ce profil n’est plus disponible." onRetry={() => navigation.goBack()} />
      </SafeAreaView>
    );
  }

  const profile = query.data;
  const isSelf = source === "self";
  const photoHeight = Math.min(width, layout.maxContentWidth) * 1.2;

  const decide = async (decision: SwipeDecision) => {
    setDeciding(true);
    try {
      const result = await discoveryApi.swipe(userId, decision);
      discoveryEvents.emit({ userId, match: result.match });
      if (result.matched) {
        void qc.invalidateQueries({ queryKey: queryKeys.matches });
      }
      navigation.goBack();
    } catch (e) {
      toast(errorMessage(e), "error");
      setDeciding(false);
    }
  };

  const facts: { icon: IconName; label: string }[] = [
    profile.jobTitle ? { icon: "briefcase-outline", label: profile.jobTitle } : null,
    profile.city ? { icon: "home-outline", label: profile.city } : null,
    profile.distanceKm != null ? { icon: "navigate-outline", label: `À ${profile.distanceKm} km` } : null,
    profile.heightCm ? { icon: "resize-outline", label: `${profile.heightCm} cm` } : null,
    { icon: "person-outline", label: genderLabels[profile.gender] },
    { icon: "heart-circle-outline", label: goalLabels[profile.relationshipGoal] }
  ].filter((f): f is { icon: IconName; label: string } => f !== null);

  return (
    <SafeAreaView edges={["top", "bottom"]} style={styles.root} testID="profile-detail-screen">
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={styles.column}>
          <PhotoCarousel name={profile.firstName} photos={profile.photos} style={[styles.photos, { height: photoHeight }]} />
          <View style={styles.topBar}>
            <IconButton icon="chevron-down" label="Fermer" onPress={() => navigation.goBack()} tone="surface" />
            {!isSelf ? (
              <IconButton
                icon="ellipsis-horizontal"
                label="Options de sécurité"
                onPress={() => setSafetyOpen(true)}
                testID="profile-safety"
                tone="surface"
              />
            ) : null}
          </View>
          <View style={styles.body}>
            <View style={styles.nameRow}>
              <Text variant="display">
                {profile.firstName}, {profile.age}
              </Text>
              {profile.isDemo ? <Chip label="Profil de démo" /> : null}
            </View>
            <View style={styles.facts}>
              {facts.map((f) => (
                <View key={f.label} style={styles.fact}>
                  <Icon color="textMuted" name={f.icon} size={16} />
                  <Text color="textMuted" variant="label">
                    {f.label}
                  </Text>
                </View>
              ))}
            </View>
            {profile.bio ? (
              <View style={styles.section}>
                <Text color="textSubtle" variant="overline">
                  À propos
                </Text>
                <Text>{profile.bio}</Text>
              </View>
            ) : null}
            {profile.interests.length > 0 ? (
              <View style={styles.section}>
                <Text color="textSubtle" variant="overline">
                  Centres d’intérêt{profile.sharedInterests > 0 ? ` · ${profile.sharedInterests} en commun` : ""}
                </Text>
                <View style={styles.chips}>
                  {profile.interests.map((i) => (
                    <Chip key={i.slug} highlighted={i.shared} icon={i.shared ? "sparkles" : undefined} label={i.label} />
                  ))}
                </View>
              </View>
            ) : null}
            {isSelf ? (
              <Text align="center" color="textSubtle" variant="caption">
                C’est ainsi que les autres membres voient votre profil.
              </Text>
            ) : null}
          </View>
        </View>
      </ScrollView>
      {!isSelf ? (
        <View style={styles.bottomBar}>
          {profile.isMatch ? (
            <Button
              icon="chatbubble-ellipses-outline"
              label={`Écrire à ${profile.firstName}`}
              onPress={() => navigation.navigate("Chat", { userId, firstName: profile.firstName, photoUrl: profile.photos[0]?.url })}
            />
          ) : source === "discovery" ? (
            <View style={styles.decisions}>
              <IconButton
                color="textMuted"
                disabled={deciding}
                icon="close"
                label="Passer"
                onPress={() => void decide("pass")}
                size={60}
                testID="detail-pass"
                tone="surface"
              />
              <IconButton
                disabled={deciding}
                icon="heart"
                label="Aimer"
                onPress={() => void decide("like")}
                size={60}
                testID="detail-like"
                tone="accent"
              />
            </View>
          ) : null}
        </View>
      ) : null}
      <SafetySheet
        firstName={profile.firstName}
        isMatch={profile.isMatch}
        onClose={() => setSafetyOpen(false)}
        onRemoved={() => {
          discoveryEvents.emit({ userId });
          navigation.goBack();
        }}
        userId={userId}
        visible={safetyOpen}
      />
    </SafeAreaView>
  );
}

const useStyles = makeStyles((c) => ({
  root: { flex: 1, backgroundColor: c.background },
  scroll: { alignItems: "center", paddingBottom: 120 },
  column: { width: "100%", maxWidth: layout.maxContentWidth },
  photos: { width: "100%", borderBottomLeftRadius: radii.xl, borderBottomRightRadius: radii.xl },
  topBar: {
    position: "absolute",
    top: spacing.sm,
    left: spacing.md,
    right: spacing.md,
    flexDirection: "row",
    justifyContent: "space-between",
    zIndex: 5
  },
  body: { padding: spacing.lg, gap: spacing.lg },
  nameRow: { flexDirection: "row", alignItems: "center", gap: spacing.sm, flexWrap: "wrap" },
  facts: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  fact: {
    flexDirection: "row",
    alignItems: "center",
    gap: 6,
    paddingVertical: 6,
    paddingHorizontal: spacing.sm,
    borderRadius: radii.pill,
    backgroundColor: c.surfaceAlt
  },
  section: { gap: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  bottomBar: {
    position: "absolute",
    left: 0,
    right: 0,
    bottom: 0,
    paddingHorizontal: spacing.lg,
    paddingTop: spacing.sm,
    paddingBottom: spacing.xl,
    backgroundColor: c.background,
    borderTopWidth: StyleSheet.hairlineWidth,
    borderTopColor: c.border,
    alignItems: "center"
  },
  decisions: { flexDirection: "row", gap: spacing.xxl }
}));
