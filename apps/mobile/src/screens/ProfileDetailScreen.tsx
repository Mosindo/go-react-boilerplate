import React, { useLayoutEffect, useState } from "react";
import { ScrollView, StyleSheet, View } from "react-native";
import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { ApiError } from "../api/client";
import { discoverApi, profileApi, safetyApi } from "../api/platform";
import { queryKeys } from "../api/queryClient";
import type { MatchSummary, ReportReason, SwipeAction } from "../api/types";
import { ActionSheet } from "../components/ActionSheet";
import { MatchModal } from "../components/MatchModal";
import { PhotoCarousel } from "../components/PhotoCarousel";
import { ReportSheet } from "../components/ReportSheet";
import type { RootScreenProps } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView, showToast } from "../shared/feedback";
import { Chip, IconButton, Text, spacing, useTheme } from "../shared/ui";
import { emitSwiped } from "../utils/deckEvents";
import { formatDistance } from "../utils/format";

export default function ProfileDetailScreen({ navigation, route }: RootScreenProps<"ProfileDetail">) {
  const { userId, profile: initial, canSwipe } = route.params;
  const { colors } = useTheme();
  const queryClient = useQueryClient();
  const [menuOpen, setMenuOpen] = useState(false);
  const [reportOpen, setReportOpen] = useState(false);
  const [reportError, setReportError] = useState<string | null>(null);
  const [match, setMatch] = useState<MatchSummary | null>(null);

  const query = useQuery({
    queryKey: queryKeys.publicProfile(userId),
    queryFn: () => profileApi.getPublic(userId),
    initialData: initial,
    staleTime: 60_000
  });
  const profile = query.data;

  const swipe = useMutation({
    mutationFn: (action: SwipeAction) => discoverApi.swipe(userId, action),
    onSuccess: (result) => {
      emitSwiped(userId);
      if (result.matched && result.match && !result.alreadySwiped) {
        setMatch(result.match);
        void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
        void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      } else {
        navigation.goBack();
      }
    },
    onError: (e) => {
      if (e instanceof ApiError && e.status === 404) {
        emitSwiped(userId);
        navigation.goBack();
      }
    }
  });

  const block = useMutation({
    mutationFn: () => safetyApi.block(userId),
    onSuccess: () => {
      emitSwiped(userId);
      void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
      void queryClient.invalidateQueries({ queryKey: queryKeys.conversations });
      showToast("Personne bloquée.");
      navigation.goBack();
    }
  });
  const report = useMutation({
    mutationFn: (v: { reason: ReportReason; details: string; block: boolean }) => safetyApi.report(userId, v.reason, v.details, v.block),
    onSuccess: (_, v) => {
      setReportOpen(false);
      showToast("Merci, votre signalement a été envoyé.");
      if (v.block) {
        emitSwiped(userId);
        void queryClient.invalidateQueries({ queryKey: queryKeys.matches });
        navigation.goBack();
      }
    },
    onError: (e) => setReportError(e instanceof ApiError ? e.message : "Envoi impossible.")
  });

  useLayoutEffect(() => {
    navigation.setOptions({
      title: profile?.firstName ?? "Profil",
      headerRight: () => (
        <IconButton icon="ellipsis-horizontal" label="Plus d'options" onPress={() => setMenuOpen(true)} size={40} style={styles.noBorder} />
      )
    });
  }, [navigation, profile?.firstName]);

  if (query.isLoading) {
    return <LoadingView fullScreen />;
  }
  if (query.error instanceof ApiError && query.error.status === 404) {
    return (
      <EmptyView
        icon="eye-off-outline"
        message="Ce profil n'est plus disponible."
        onAction={() => navigation.goBack()}
        actionLabel="Retour"
        title="Profil introuvable"
      />
    );
  }
  if (query.isError || !profile) {
    return <ErrorView message="Impossible de charger ce profil." onAction={() => void query.refetch()} />;
  }

  const distance = formatDistance(profile.distanceKm);

  return (
    <View style={styles.flex}>
      <ScrollView contentContainerStyle={styles.content}>
        <PhotoCarousel name={profile.firstName} photos={profile.photos} style={styles.carousel} />
        <View style={styles.info}>
          <Text accessibilityRole="header" variant="title">
            {profile.firstName}, {profile.age}
          </Text>
          {[profile.city, distance].filter(Boolean).length > 0 ? (
            <Text tone="muted">{[profile.city, distance].filter(Boolean).join(" · ")}</Text>
          ) : null}
          {profile.bio ? <Text>{profile.bio}</Text> : null}
          {profile.interests.length > 0 ? (
            <View style={styles.chips}>
              {profile.interests.map((i) => (
                <Chip key={i.slug} label={i.label} />
              ))}
            </View>
          ) : null}
        </View>
      </ScrollView>

      {canSwipe ? (
        <View style={[styles.actions, { backgroundColor: colors.background, borderTopColor: colors.border }]}>
          <IconButton
            color={colors.pass}
            disabled={swipe.isPending}
            icon="close"
            label="Passer"
            onPress={() => swipe.mutate("pass")}
            size={62}
          />
          <IconButton
            background={colors.primary}
            color={colors.primaryForeground}
            disabled={swipe.isPending}
            icon="heart"
            label="J'aime"
            onPress={() => swipe.mutate("like")}
            size={62}
          />
        </View>
      ) : null}

      <ActionSheet
        actions={[
          {
            label: "Signaler",
            destructive: true,
            onPress: () => {
              setReportError(null);
              setReportOpen(true);
            }
          },
          { label: "Bloquer", destructive: true, onPress: () => block.mutate() }
        ]}
        onClose={() => setMenuOpen(false)}
        title={profile.firstName}
        visible={menuOpen}
      />
      <ReportSheet
        busy={report.isPending}
        error={reportError}
        name={profile.firstName}
        onClose={() => setReportOpen(false)}
        onSubmit={(reason, details, alsoBlock) => report.mutate({ reason, details, block: alsoBlock })}
        visible={reportOpen}
      />
      <MatchModal
        match={match}
        onClose={() => {
          setMatch(null);
          navigation.goBack();
        }}
        onMessage={(m) => {
          setMatch(null);
          navigation.replace("Conversation", { conversationId: m.conversationId, user: m.user });
        }}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  content: { padding: spacing.lg, gap: spacing.lg, paddingBottom: spacing.xxxl },
  carousel: { width: "100%", aspectRatio: 3 / 4 },
  info: { gap: spacing.sm },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  actions: {
    flexDirection: "row",
    justifyContent: "center",
    gap: spacing.xxl,
    padding: spacing.md,
    borderTopWidth: StyleSheet.hairlineWidth
  },
  noBorder: { borderWidth: 0, backgroundColor: "transparent" }
});
