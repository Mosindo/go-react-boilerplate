import React, { useState } from "react";
import { ScrollView, StyleSheet, View, useWindowDimensions, type NativeScrollEvent, type NativeSyntheticEvent } from "react-native";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { SafeAreaView } from "react-native-safe-area-context";
import { ApiError, errorMessage } from "../../api/client";
import { swipe } from "../../api/matching";
import type { SwipeAction } from "../../api/types";
import { useCelebrateMatch } from "../../components/MatchCelebration";
import { DemoBadge, ProfileFacts } from "../../components/ProfileBadges";
import { SafetyDialogs } from "../../components/SafetyActions";
import { usePublicProfile } from "../../hooks/queries";
import { queryClient, queryKeys } from "../../hooks/queryClient";
import type { AppStackParamList } from "../../navigation/types";
import { layout, radii, spacing, useTheme } from "../../theme";
import { ActionSheet, Chip, EmptyState, IconButton, LoadingState, RemoteImage, Text, showToast } from "../../ui";
import { genderLabels } from "../../utils/format";

type Props = NativeStackScreenProps<AppStackParamList, "ProfileDetail">;

export function ProfileDetailScreen({ navigation, route }: Props) {
  const { userId, source } = route.params;
  const { colors } = useTheme();
  const { width } = useWindowDimensions();
  const celebrate = useCelebrateMatch();
  const query = usePublicProfile(userId);
  const [photoIndex, setPhotoIndex] = useState(0);
  const [menu, setMenu] = useState(false);
  const [safety, setSafety] = useState<"block" | "report" | null>(null);
  const [busy, setBusy] = useState(false);
  const pageWidth = Math.min(width, layout.maxContentWidth);
  const canDecide = source === "discover" || source === "likes";

  const decide = async (action: SwipeAction) => {
    setBusy(true);
    try {
      const result = await swipe(userId, action);
      void queryClient.invalidateQueries({ queryKey: queryKeys.likes });
      navigation.goBack();
      if (result.matched && result.match) celebrate(result.match);
    } catch (e) {
      if (e instanceof ApiError && e.code === "already_swiped") navigation.goBack();
      else showToast(errorMessage(e), "error");
    } finally {
      setBusy(false);
    }
  };

  const onScroll = (e: NativeSyntheticEvent<NativeScrollEvent>) => setPhotoIndex(Math.round(e.nativeEvent.contentOffset.x / pageWidth));

  if (query.isLoading) {
    return (
      <SafeAreaView style={[styles.root, { backgroundColor: colors.background }]}>
        <LoadingState />
      </SafeAreaView>
    );
  }
  if (!query.data) {
    return (
      <SafeAreaView style={[styles.root, { backgroundColor: colors.background }]}>
        <IconButton icon="chevron-back" label="Retour" onPress={navigation.goBack} style={styles.floatingBack} />
        <EmptyState
          icon="person-remove-outline"
          message={errorMessage(query.error, "Ce profil n'est plus disponible.")}
          title="Profil indisponible"
        />
      </SafeAreaView>
    );
  }
  const profile = query.data;

  return (
    <View style={[styles.root, { backgroundColor: colors.background }]} testID="profile-detail">
      <ScrollView contentContainerStyle={styles.scroll}>
        <View style={[styles.column, { width: pageWidth }]}>
          <View style={{ width: pageWidth, height: pageWidth * 1.2 }}>
            <ScrollView horizontal onMomentumScrollEnd={onScroll} pagingEnabled showsHorizontalScrollIndicator={false}>
              {profile.photos.map((p) => (
                <RemoteImage
                  accessibilityLabel={`Photo de ${profile.firstName}`}
                  key={p.id}
                  style={{ width: pageWidth, height: pageWidth * 1.2 }}
                  url={p.url}
                />
              ))}
            </ScrollView>
            {profile.photos.length > 1 ? (
              <View style={styles.dots}>
                {profile.photos.map((p, i) => (
                  <View key={p.id} style={[styles.dot, { backgroundColor: i === photoIndex ? "#FFFFFF" : "rgba(255,255,255,0.45)" }]} />
                ))}
              </View>
            ) : null}
          </View>

          <View style={styles.body}>
            <View style={styles.nameRow}>
              <Text style={styles.name} variant="display">
                {profile.firstName}, {profile.age}
              </Text>
              {profile.isDemo ? <DemoBadge /> : null}
            </View>
            <Text tone="muted" variant="label">
              {genderLabels[profile.gender]}
            </Text>
            <ProfileFacts profile={profile} />
            {profile.bio ? (
              <View style={[styles.block, { backgroundColor: colors.surface, borderColor: colors.border }]}>
                <Text tone="subtle" variant="overline">
                  À propos
                </Text>
                <Text>{profile.bio}</Text>
              </View>
            ) : null}
            {profile.interests.length > 0 ? (
              <View style={[styles.block, { backgroundColor: colors.surface, borderColor: colors.border }]}>
                <Text tone="subtle" variant="overline">
                  Centres d’intérêt
                </Text>
                <View style={styles.chips}>
                  {profile.interests.map((i) => (
                    <Chip key={i.slug} label={i.label} />
                  ))}
                </View>
              </View>
            ) : null}
            {source !== "self" ? (
              <View style={styles.safetyLinks}>
                <Text onPress={() => setSafety("block")} style={styles.link} tone="muted" variant="label">
                  Bloquer {profile.firstName}
                </Text>
                <Text onPress={() => setSafety("report")} style={styles.link} testID="profile-report" tone="danger" variant="label">
                  Signaler
                </Text>
              </View>
            ) : null}
          </View>
        </View>
      </ScrollView>

      <SafeAreaView edges={["top"]} pointerEvents="box-none" style={styles.topBar}>
        <IconButton icon="chevron-down" label="Fermer" onPress={navigation.goBack} testID="profile-close" variant="surface" />
        {source !== "self" ? (
          <IconButton icon="ellipsis-horizontal" label="Plus d'options" onPress={() => setMenu(true)} variant="surface" />
        ) : null}
      </SafeAreaView>

      {canDecide ? (
        <SafeAreaView edges={["bottom"]} pointerEvents="box-none" style={styles.decision}>
          <IconButton
            color={colors.textMuted}
            disabled={busy}
            icon="close"
            label="Passer"
            onPress={() => decide("pass")}
            size={64}
            testID="profile-pass"
            variant="surface"
          />
          <IconButton
            disabled={busy}
            icon="heart"
            label="Liker"
            onPress={() => decide("like")}
            size={72}
            testID="profile-like"
            variant="primary"
          />
        </SafeAreaView>
      ) : null}

      <ActionSheet
        actions={[
          { label: `Bloquer ${profile.firstName}`, icon: "hand-left-outline", onPress: () => setSafety("block") },
          { label: "Signaler", icon: "flag-outline", destructive: true, onPress: () => setSafety("report") }
        ]}
        onClose={() => setMenu(false)}
        visible={menu}
      />
      <SafetyDialogs
        firstName={profile.firstName}
        mode={safety}
        onClose={() => setSafety(null)}
        onDone={() => {
          setSafety(null);
          navigation.popToTop();
        }}
        userId={profile.userId}
      />
    </View>
  );
}

const styles = StyleSheet.create({
  root: { flex: 1 },
  scroll: { alignItems: "center", paddingBottom: 140 },
  column: { maxWidth: layout.maxContentWidth },
  dots: { position: "absolute", bottom: spacing.md, alignSelf: "center", flexDirection: "row", gap: 6 },
  dot: { width: 8, height: 8, borderRadius: 4 },
  body: { padding: spacing.lg, gap: spacing.sm },
  nameRow: { flexDirection: "row", alignItems: "center", gap: spacing.sm },
  name: { flexShrink: 1 },
  block: { borderWidth: 1, borderRadius: radii.lg, padding: spacing.md, gap: spacing.xs, marginTop: spacing.xs },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.xs },
  safetyLinks: { flexDirection: "row", justifyContent: "center", gap: spacing.xl, marginTop: spacing.lg },
  link: { padding: spacing.xs },
  topBar: { position: "absolute", top: 0, left: 0, right: 0, flexDirection: "row", justifyContent: "space-between", padding: spacing.md },
  floatingBack: { margin: spacing.md },
  decision: {
    position: "absolute",
    bottom: 0,
    left: 0,
    right: 0,
    flexDirection: "row",
    justifyContent: "center",
    gap: spacing.xl,
    paddingBottom: spacing.lg
  }
});
