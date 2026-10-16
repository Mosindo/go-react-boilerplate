import React, { useState } from "react";
import { Pressable, ScrollView, StyleSheet, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useQueryClient } from "@tanstack/react-query";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { friendlyError } from "../api/errors";
import { swipe } from "../api/platform";
import type { SwipeAction } from "../api/types";
import { ActionSheet, type SheetAction } from "../components/ActionSheet";
import { Chip, ChipRow } from "../components/Chip";
import { PhotoCarousel } from "../components/PhotoCarousel";
import { ReportSheet } from "../components/ReportSheet";
import { usePublicProfile, queryKeys } from "../hooks/queries";
import { useRealtime } from "../hooks/useRealtime";
import { useSafetyActions } from "../hooks/useSafetyActions";
import { formatDistance } from "../lib/format";
import { lookingForLabel } from "../lib/labels";
import { markDecided } from "../lib/decisions";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Button, Text, colors, radii, spacing } from "../shared/ui";
import { showAlert } from "../lib/alert";

type Props = NativeStackScreenProps<RootStackParamList, "ProfileDetail">;

export default function ProfileDetailScreen({ navigation, route }: Props) {
  const { userId, source, matchId } = route.params;
  const query = usePublicProfile(userId);
  const queryClient = useQueryClient();
  const { announceMatch } = useRealtime();
  const [menuOpen, setMenuOpen] = useState(false);
  const [deciding, setDeciding] = useState(false);

  const profile = query.data;
  const safety = useSafetyActions({
    userId,
    name: profile?.firstName ?? "cette personne",
    matchId,
    onGone: () => navigation.goBack()
  });

  const decide = async (action: SwipeAction) => {
    if (!profile) {
      return;
    }
    setDeciding(true);
    try {
      const result = await swipe(profile.id, action);
      markDecided(profile.id);
      void queryClient.invalidateQueries({ queryKey: queryKeys.likes });
      if (result.matched && result.matchId) {
        void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
        announceMatch({ matchId: result.matchId, profile });
      }
      navigation.goBack();
    } catch (error) {
      showAlert("Action impossible", friendlyError(error));
    } finally {
      setDeciding(false);
    }
  };

  const menu: SheetAction[] = [
    { label: "Signaler", onPress: safety.openReport },
    { label: "Bloquer", destructive: true, onPress: safety.confirmBlock },
    ...(matchId ? [{ label: "Annuler le match", destructive: true, onPress: safety.confirmUnmatch }] : [])
  ];

  if (query.isLoading) {
    return (
      <SafeAreaLayout>
        <LoadingView fullScreen label="Chargement du profil…" />
      </SafeAreaLayout>
    );
  }
  if (query.isError || !profile) {
    return (
      <SafeAreaLayout style={styles.padded}>
        <ErrorView actionLabel="Retour" message="Ce profil n'est plus disponible." onAction={() => navigation.goBack()} />
      </SafeAreaLayout>
    );
  }

  const distance = formatDistance(profile.distanceKm);

  return (
    <SafeAreaLayout edges={["top", "left", "right", "bottom"]}>
      <View style={styles.topBar}>
        <IconButton icon="chevron-back" label="Retour" onPress={() => navigation.goBack()} />
        <IconButton icon="ellipsis-horizontal" label="Plus d'options" onPress={() => setMenuOpen(true)} testID="profile-menu" />
      </View>
      <ScrollView contentContainerStyle={styles.scroll} testID="screen-profile-detail">
        <View style={styles.photo}>
          <PhotoCarousel name={profile.firstName} photos={profile.photos} />
        </View>

        <View style={styles.block}>
          <Text variant="title" weight="bold">
            {profile.firstName}, {profile.age}
          </Text>
          {profile.occupation ? <Text tone="secondary">{profile.occupation}</Text> : null}
          <View style={styles.meta}>
            {profile.city ? <Meta icon="location-outline" text={profile.city} /> : null}
            {distance ? <Meta icon="navigate-outline" text={distance} /> : null}
            <Meta icon="search-outline" text={lookingForLabel(profile.lookingFor)} />
          </View>
        </View>

        {profile.bio ? (
          <View style={styles.block}>
            <Text tone="muted" variant="eyebrow" weight="bold">
              À propos
            </Text>
            <Text>{profile.bio}</Text>
          </View>
        ) : null}

        {profile.interests.length > 0 ? (
          <View style={styles.block}>
            <Text tone="muted" variant="eyebrow" weight="bold">
              Centres d'intérêt
            </Text>
            <ChipRow>
              {profile.interests.map((interest) => (
                <Chip key={interest.slug} label={interest.label} tone="highlight" />
              ))}
            </ChipRow>
          </View>
        ) : null}
      </ScrollView>

      {source === "discover" || source === "likes" ? (
        <View style={styles.footer}>
          <Button
            disabled={deciding}
            label="Passer"
            onPress={() => void decide("pass")}
            style={styles.footerButton}
            testID="detail-pass"
            variant="outline"
          />
          <Button loading={deciding} label="J'aime" onPress={() => void decide("like")} style={styles.footerButton} testID="detail-like" />
        </View>
      ) : null}
      {source === "match" && matchId ? (
        <View style={styles.footer}>
          <Button
            label={`Écrire à ${profile.firstName}`}
            onPress={() => navigation.navigate("Thread", { matchId, profile })}
            style={styles.footerButton}
            testID="detail-message"
          />
        </View>
      ) : null}

      <ActionSheet actions={menu} onClose={() => setMenuOpen(false)} title={profile.firstName} visible={menuOpen} />
      <ReportSheet
        name={profile.firstName}
        onClose={safety.closeReport}
        onSubmit={safety.submitReport}
        submitting={safety.reporting}
        visible={safety.reportOpen}
      />
    </SafeAreaLayout>
  );
}

function IconButton({
  icon,
  label,
  onPress,
  testID
}: {
  icon: React.ComponentProps<typeof Ionicons>["name"];
  label: string;
  onPress: () => void;
  testID?: string;
}) {
  return (
    <Pressable
      accessibilityLabel={label}
      accessibilityRole="button"
      hitSlop={8}
      onPress={onPress}
      style={styles.iconButton}
      testID={testID}
    >
      <Ionicons color={colors.text} name={icon} size={22} />
    </Pressable>
  );
}

function Meta({ icon, text }: { icon: React.ComponentProps<typeof Ionicons>["name"]; text: string }) {
  return (
    <View style={styles.metaItem}>
      <Ionicons color={colors.textMuted} name={icon} size={16} />
      <Text tone="muted" variant="label">
        {text}
      </Text>
    </View>
  );
}

const styles = StyleSheet.create({
  padded: { padding: spacing.lg },
  topBar: { flexDirection: "row", justifyContent: "space-between", paddingHorizontal: spacing.lg, paddingVertical: spacing.sm },
  iconButton: {
    width: 40,
    height: 40,
    borderRadius: radii.pill,
    backgroundColor: colors.backgroundElevated,
    borderWidth: 1,
    borderColor: colors.border,
    alignItems: "center",
    justifyContent: "center"
  },
  scroll: { paddingHorizontal: spacing.lg, paddingBottom: spacing.xxl, gap: spacing.lg },
  photo: { aspectRatio: 4 / 5, width: "100%" },
  block: { gap: spacing.sm },
  meta: { flexDirection: "row", flexWrap: "wrap", gap: spacing.md },
  metaItem: { flexDirection: "row", alignItems: "center", gap: spacing.xs },
  footer: {
    flexDirection: "row",
    gap: spacing.md,
    padding: spacing.lg,
    borderTopWidth: 1,
    borderTopColor: colors.border,
    backgroundColor: colors.background
  },
  footerButton: { flex: 1 }
});
