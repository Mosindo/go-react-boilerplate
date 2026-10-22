import React, { useState } from "react";
import { Pressable, ScrollView, StyleSheet, View } from "react-native";
import { useNavigation, useRoute, type RouteProp } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery, useQueryClient } from "@tanstack/react-query";
import { errorMessage } from "../api/client";
import { fetchPublicProfile } from "../api/profile";
import { blockUser, reportUser } from "../api/safety";
import { ActionSheet, ConfirmDialog, ReportDialog } from "../components/Dialogs";
import { PhotoCarousel } from "../components/PhotoCarousel";
import { ProfileSummary } from "../components/ProfileSummary";
import type { RootStackParamList } from "../navigation/types";
import { EmptyView, ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { Text, colors, spacing } from "../shared/ui";

type Dialog = "menu" | "report" | "block" | null;

/** Full profile of another person: all photos, bio, interests, and safety actions. */
export default function ProfileDetailScreen() {
  const route = useRoute<RouteProp<RootStackParamList, "ProfileDetail">>();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const queryClient = useQueryClient();
  const { userId } = route.params;
  const profile = useQuery({ queryKey: ["profile", userId], queryFn: () => fetchPublicProfile(userId) });

  const [dialog, setDialog] = useState<Dialog>(null);
  const [busy, setBusy] = useState(false);
  const [error, setError] = useState<string | null>(null);

  const finish = async () => {
    await Promise.all([
      queryClient.invalidateQueries({ queryKey: ["conversations"] }),
      queryClient.invalidateQueries({ queryKey: ["matches"] }),
      queryClient.invalidateQueries({ queryKey: ["discover"] })
    ]);
    navigation.navigate("Tabs", { screen: "Discover" });
  };

  const run = async (task: () => Promise<void>) => {
    setBusy(true);
    setError(null);
    try {
      await task();
      setDialog(null);
      await finish();
    } catch (err) {
      setError(errorMessage(err));
    } finally {
      setBusy(false);
    }
  };

  const name = profile.data?.firstName ?? "cette personne";

  return (
    <SafeAreaLayout edges={["top", "left", "right"]}>
      <View style={styles.bar}>
        <Pressable
          accessibilityLabel="Retour"
          accessibilityRole="button"
          onPress={() => navigation.goBack()}
          style={styles.barButton}
          testID="screen-back"
        >
          <Text variant="heading">←</Text>
        </Pressable>
        {profile.data ? (
          <Pressable
            accessibilityLabel="Plus d'options"
            accessibilityRole="button"
            onPress={() => setDialog("menu")}
            style={styles.barButton}
            testID="profile-menu"
          >
            <Text variant="heading">⋯</Text>
          </Pressable>
        ) : null}
      </View>

      {profile.isLoading ? (
        <LoadingView label="Chargement du profil…" />
      ) : profile.isError ? (
        profile.error instanceof Error && "status" in profile.error && (profile.error as { status: number }).status === 404 ? (
          <EmptyView message="Ce profil n'est plus disponible." title="Profil introuvable" />
        ) : (
          <ErrorView actionLabel="Réessayer" message={errorMessage(profile.error)} onAction={() => void profile.refetch()} />
        )
      ) : profile.data ? (
        <ScrollView contentContainerStyle={styles.content} testID="profile-detail">
          <PhotoCarousel name={profile.data.firstName} photos={profile.data.photos} style={styles.photo} />
          <ProfileSummary
            age={profile.data.age}
            bio={profile.data.bio}
            city={profile.data.city}
            distanceKm={profile.data.distanceKm}
            firstName={profile.data.firstName}
            interests={profile.data.interests}
          />
        </ScrollView>
      ) : null}

      <ActionSheet
        onClose={() => setDialog(null)}
        options={[
          { label: "Signaler", onPress: () => setDialog("report"), destructive: true, testID: "profile-report" },
          { label: "Bloquer", onPress: () => setDialog("block"), destructive: true, testID: "profile-block" }
        ]}
        title={name}
        visible={dialog === "menu"}
      />
      <ConfirmDialog
        confirmLabel="Bloquer"
        destructive
        error={error}
        loading={busy}
        message={`${name} ne pourra plus vous voir ni vous écrire. Vous ne verrez plus ce profil non plus.`}
        onCancel={() => setDialog(null)}
        onConfirm={() => run(() => blockUser(userId))}
        title={`Bloquer ${name} ?`}
        visible={dialog === "block"}
      />
      <ReportDialog
        error={error}
        loading={busy}
        name={name}
        onCancel={() => setDialog(null)}
        onSubmit={(reason, details, alsoBlock) => run(() => reportUser(userId, reason, details, alsoBlock))}
        visible={dialog === "report"}
      />
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  bar: { flexDirection: "row", justifyContent: "space-between", paddingHorizontal: spacing.md },
  barButton: { width: 44, height: 44, alignItems: "center", justifyContent: "center" },
  content: { padding: spacing.lg, gap: spacing.lg, paddingBottom: spacing.xxxl, width: "100%", maxWidth: 560, alignSelf: "center" },
  photo: { width: "100%", aspectRatio: 0.8, backgroundColor: colors.surfaceSubtle }
});
