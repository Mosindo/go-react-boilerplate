import { Ionicons } from "@expo/vector-icons";
import type { NativeStackScreenProps } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import React, { useLayoutEffect, useState } from "react";
import { Pressable, ScrollView, StyleSheet, View } from "react-native";
import { fetchPublicProfile } from "../../api/profile";
import { blockUser } from "../../api/safety";
import { PhotoCarousel } from "../../components/PhotoCarousel";
import { keys } from "../../hooks/queries";
import { useSwipe } from "../../hooks/useSwipe";
import { formatDistance } from "../../lib/format";
import { labelOf, LOOKING_FOR } from "../../lib/labels";
import { confirmAction } from "../../lib/platform";
import type { RootStackParamList } from "../../navigation/types";
import { ErrorView, LoadingView, showError, showToast } from "../../shared/feedback";
import { SafeAreaLayout } from "../../shared/layout";
import { ActionSheet, Button, Chip, Text } from "../../shared/ui";
import { MIN_TOUCH, spacing, useTheme } from "../../theme";

type Props = NativeStackScreenProps<RootStackParamList, "ProfileDetail">;

export function ProfileDetailScreen({ navigation, route }: Props) {
  const { userId, canSwipe } = route.params;
  const theme = useTheme();
  const queryClient = useQueryClient();
  const swipe = useSwipe();
  const [menuOpen, setMenuOpen] = useState(false);
  const query = useQuery({ queryKey: keys.publicProfile(userId), queryFn: () => fetchPublicProfile(userId) });
  const profile = query.data;

  useLayoutEffect(() => {
    navigation.setOptions({
      title: profile?.firstName ?? "Profil",
      headerRight: () => (
        <Pressable
          testID="profile-menu"
          accessibilityRole="button"
          accessibilityLabel="Plus d'options"
          onPress={() => setMenuOpen(true)}
          style={styles.menuButton}
        >
          <Ionicons name="ellipsis-horizontal" size={24} color={theme.text} />
        </Pressable>
      )
    });
  }, [navigation, profile?.firstName, theme.text]);

  if (query.isPending) {
    return (
      <SafeAreaLayout edges={["bottom"]}>
        <LoadingView />
      </SafeAreaLayout>
    );
  }
  if (query.isError || !profile) {
    return (
      <SafeAreaLayout edges={["bottom"]}>
        <ErrorView title="Profil indisponible" message="Ce profil n'est plus accessible." onRetry={() => void query.refetch()} />
      </SafeAreaLayout>
    );
  }

  const decide = async (action: "like" | "pass") => {
    const result = await swipe(profile, action);
    if (result?.matched && result.match) navigation.replace("Match", { match: result.match });
    else if (result) navigation.goBack();
  };

  const block = async () => {
    if (
      !(await confirmAction(
        `Bloquer ${profile.firstName} ?`,
        "Vous ne vous verrez plus mutuellement et la conversation éventuelle sera supprimée.",
        "Bloquer"
      ))
    )
      return;
    try {
      await blockUser(userId);
      queryClient.setQueryData(keys.discover, (deck: unknown) =>
        Array.isArray(deck) ? deck.filter((p: { userId: string }) => p.userId !== userId) : deck
      );
      void queryClient.invalidateQueries({ queryKey: keys.conversations });
      showToast(`${profile.firstName} a été bloqué·e.`, "info");
      navigation.popToTop();
    } catch (err) {
      showError(err instanceof Error ? err.message : "Blocage impossible.");
    }
  };

  const distance = formatDistance(profile.distanceKm);
  const meta = [profile.city, distance].filter(Boolean).join(" · ");

  return (
    <SafeAreaLayout edges={["bottom"]}>
      <ScrollView contentContainerStyle={styles.scroll} testID="profile-detail">
        <View style={styles.column}>
          <PhotoCarousel photos={profile.photos} name={profile.firstName} />
          <View style={styles.body}>
            <Text variant="title" testID="detail-name">
              {profile.firstName}, {profile.age}
            </Text>
            {meta ? <Text tone="muted">{meta}</Text> : null}
            {profile.occupation ? <Text>{profile.occupation}</Text> : null}
            {profile.bio ? <Text>{profile.bio}</Text> : null}

            <View style={styles.tags}>
              <Chip label={labelOf(LOOKING_FOR, profile.lookingFor)} selected />
              {profile.heightCm ? <Chip label={`${profile.heightCm} cm`} /> : null}
            </View>
            {profile.interests.length > 0 ? (
              <View style={styles.tags}>
                {profile.interests.map((i) => (
                  <Chip key={i.slug} label={i.label} />
                ))}
              </View>
            ) : null}
          </View>
        </View>
      </ScrollView>

      {canSwipe ? (
        <View style={[styles.actions, { borderTopColor: theme.border, backgroundColor: theme.background }]}>
          <Button testID="detail-pass" label="Passer" variant="secondary" style={styles.flex} onPress={() => void decide("pass")} />
          <Button testID="detail-like" label="J'aime" style={styles.flex} onPress={() => void decide("like")} />
        </View>
      ) : null}

      <ActionSheet
        visible={menuOpen}
        title={profile.firstName}
        onClose={() => setMenuOpen(false)}
        actions={[
          { label: "Signaler", onPress: () => navigation.navigate("Report", { userId, name: profile.firstName }), testID: "menu-report" },
          { label: "Bloquer", destructive: true, onPress: () => void block(), testID: "menu-block" }
        ]}
      />
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  scroll: { alignItems: "center", paddingBottom: spacing.xl },
  column: { width: "100%", maxWidth: 560 },
  body: { padding: spacing.lg, gap: spacing.md },
  tags: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm },
  actions: { flexDirection: "row", gap: spacing.md, padding: spacing.lg, borderTopWidth: StyleSheet.hairlineWidth },
  flex: { flex: 1 },
  menuButton: { width: MIN_TOUCH, height: MIN_TOUCH, alignItems: "center", justifyContent: "center" }
});
