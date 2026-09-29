import React, { useCallback, useState } from "react";
import { ActivityIndicator, Alert, Modal, ScrollView, StyleSheet, View } from "react-native";
import { useQuery } from "@tanstack/react-query";
import { SafeAreaProvider, useSafeAreaInsets } from "react-native-safe-area-context";
import { fetchPublicProfile } from "../api/discovery";
import type { PublicProfile } from "../api/models";
import { blockUser } from "../api/safety";
import { formatDistance, formatNameAge, humanizeSlug } from "../lib/dating/format";
import { useTheme } from "../shared/ui/theme";
import { errorMessage } from "./errors";
import { AppButton, Banner, Chip, IconButton, Txt } from "./kit";
import { PhotoCarousel } from "./PhotoCarousel";
import { ReportSheet } from "./ReportSheet";

type Props = {
  visible: boolean;
  userId: string | null;
  /** Already-known profile (from the deck); otherwise it is fetched with GET /profiles/:userId. */
  initialProfile?: PublicProfile | null;
  onClose: () => void;
  /** Called after the person was blocked (directly or through the report sheet). */
  onBlocked?: (userId: string) => void;
};

function ProfileBody({ userId, initialProfile, onClose, onBlocked }: Omit<Props, "visible">) {
  const { colors, spacing } = useTheme();
  const insets = useSafeAreaInsets();
  const [reportOpen, setReportOpen] = useState(false);
  const [blocking, setBlocking] = useState(false);
  const [actionError, setActionError] = useState<string | null>(null);

  const query = useQuery({
    queryKey: ["profile", userId],
    queryFn: () => fetchPublicProfile(userId ?? ""),
    enabled: Boolean(userId) && !initialProfile,
    staleTime: 60_000
  });
  const profile = initialProfile ?? query.data ?? null;

  const doBlock = useCallback(async () => {
    if (!profile) return;
    setBlocking(true);
    setActionError(null);
    try {
      await blockUser(profile.userId);
      onBlocked?.(profile.userId);
      onClose();
    } catch (e) {
      setActionError(errorMessage(e, "We couldn't block this person. Please try again."));
    } finally {
      setBlocking(false);
    }
  }, [onBlocked, onClose, profile]);

  const confirmBlock = useCallback(() => {
    if (!profile) return;
    Alert.alert(
      `Block ${profile.firstName}?`,
      "You won't see each other anymore, and any match or conversation will be removed.",
      [
        { text: "Cancel", style: "cancel" },
        { text: "Block", style: "destructive", onPress: () => void doBlock() }
      ]
    );
  }, [doBlock, profile]);

  const distance = profile ? formatDistance(profile.distanceKm) : null;

  return (
    <View style={[styles.flex, { backgroundColor: colors.background }]} testID="profile-modal">
      {profile ? (
        <ScrollView contentContainerStyle={{ paddingBottom: insets.bottom + spacing.xxl }}>
          <PhotoCarousel photos={profile.photos} name={profile.firstName} />
          <View style={{ padding: spacing.xl, gap: spacing.md }}>
            <Txt variant="title" accessibilityRole="header">
              {formatNameAge(profile.firstName, profile.age)}
            </Txt>
            {profile.city || distance ? (
              <Txt tone="muted">{[profile.city, distance].filter(Boolean).join(" · ")}</Txt>
            ) : null}
            {profile.bio ? (
              <View style={{ gap: spacing.xs }}>
                <Txt variant="label" tone="muted" weight="semibold">
                  About
                </Txt>
                <Txt>{profile.bio}</Txt>
              </View>
            ) : null}
            {profile.interests.length > 0 ? (
              <View style={{ gap: spacing.xs }}>
                <Txt variant="label" tone="muted" weight="semibold">
                  Interests
                </Txt>
                <View style={[styles.chips, { gap: spacing.xs }]}>
                  {profile.interests.map((slug) => (
                    <Chip key={slug} label={humanizeSlug(slug)} />
                  ))}
                </View>
              </View>
            ) : null}
            {actionError ? <Banner tone="danger" message={actionError} /> : null}
            <View style={[styles.actions, { gap: spacing.sm, marginTop: spacing.md }]}>
              <AppButton
                label="Report"
                variant="outline"
                onPress={() => setReportOpen(true)}
                testID="profile-report-button"
                style={styles.flex}
              />
              <AppButton
                label="Block"
                variant="danger"
                onPress={confirmBlock}
                loading={blocking}
                testID="profile-block-button"
                style={styles.flex}
              />
            </View>
          </View>
        </ScrollView>
      ) : query.isError ? (
        <View style={styles.center}>
          <Txt tone="muted" style={styles.centerText}>
            {errorMessage(query.error, "This profile isn't available anymore.")}
          </Txt>
          <AppButton label="Try again" variant="soft" onPress={() => void query.refetch()} />
        </View>
      ) : (
        <View style={styles.center}>
          <ActivityIndicator color={colors.primary} />
        </View>
      )}

      <View style={[styles.close, { top: insets.top + spacing.sm, right: spacing.md }]}>
        <IconButton
          glyph="✕"
          label="Close profile"
          onPress={onClose}
          backgroundColor={colors.photoScrim}
          color={colors.text}
          testID="profile-close"
        />
      </View>

      {profile ? (
        <ReportSheet
          visible={reportOpen}
          userId={profile.userId}
          firstName={profile.firstName}
          onClose={() => setReportOpen(false)}
          onReported={({ blocked }) => {
            if (blocked) onBlocked?.(profile.userId);
          }}
        />
      ) : null}
    </View>
  );
}

/** Full-screen profile viewer: all photos, bio, interests, distance, plus Report / Block. */
export function ProfileModal({ visible, ...rest }: Props) {
  return (
    <Modal
      visible={visible}
      animationType="slide"
      onRequestClose={rest.onClose}
      statusBarTranslucent
    >
      <SafeAreaProvider>
        <ProfileBody {...rest} />
      </SafeAreaProvider>
    </Modal>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  chips: { flexDirection: "row", flexWrap: "wrap" },
  actions: { flexDirection: "row" },
  center: {
    flex: 1,
    alignItems: "center",
    justifyContent: "center",
    gap: 12,
    padding: 24
  },
  centerText: { textAlign: "center" },
  close: { position: "absolute" }
});
