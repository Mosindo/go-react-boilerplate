import React from "react";
import { StyleSheet, Switch, View } from "react-native";
import { useQueryClient } from "@tanstack/react-query";
import { profileApi, safetyApi } from "../api/endpoints";
import type { OwnProfile } from "../api/types";
import { errorMessage, keys, useBlocked, useProfile } from "../hooks/queries";
import { discoverStore } from "../state/discoverStore";
import { Button, ErrorView, LoadingView, Screen, Text, radii, spacing, useTheme, useToast } from "../ui";

function ToggleRow({
  title,
  description,
  value,
  onChange,
  testID
}: {
  title: string;
  description: string;
  value: boolean;
  onChange: (v: boolean) => void;
  testID: string;
}) {
  const { colors } = useTheme();
  return (
    <View style={[styles.toggle, { backgroundColor: colors.surface, borderColor: colors.border }]}>
      <View style={styles.toggleText}>
        <Text variant="label">{title}</Text>
        <Text tone="muted" variant="caption">
          {description}
        </Text>
      </View>
      <Switch
        accessibilityLabel={title}
        onValueChange={onChange}
        testID={testID}
        thumbColor="#FFFFFF"
        trackColor={{ true: colors.primary, false: colors.border }}
        value={value}
      />
    </View>
  );
}

function Content({ profile }: { profile: OwnProfile }) {
  const toast = useToast();
  const queryClient = useQueryClient();
  const blocked = useBlocked();

  const update = async (patch: { isDiscoverable?: boolean; showDistance?: boolean }) => {
    try {
      await profileApi.save({
        displayName: profile.displayName,
        birthDate: profile.birthDate,
        gender: profile.gender,
        bio: profile.bio,
        city: profile.city,
        interests: profile.interests.map((i) => i.slug),
        isDiscoverable: profile.isDiscoverable,
        showDistance: profile.showDistance,
        ...patch
      });
      await queryClient.invalidateQueries({ queryKey: keys.profile });
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  const unblock = async (userId: string) => {
    try {
      await safetyApi.unblock(userId);
      await queryClient.invalidateQueries({ queryKey: keys.blocked });
      discoverStore.refresh();
    } catch (error) {
      toast.show(errorMessage(error), "error");
    }
  };

  return (
    <View style={styles.content}>
      <ToggleRow
        description="Désactivez pour mettre votre profil en pause : personne ne pourra vous découvrir, vos matchs restent actifs."
        onChange={(v) => void update({ isDiscoverable: v })}
        testID="toggle-discoverable"
        title="Profil visible dans la découverte"
        value={profile.isDiscoverable}
      />
      <ToggleRow
        description="Les autres voient une distance approximative, jamais votre position exacte."
        onChange={(v) => void update({ showDistance: v })}
        testID="toggle-distance"
        title="Afficher ma distance"
        value={profile.showDistance}
      />

      <Text style={styles.sectionTitle} variant="heading">
        Personnes bloquées
      </Text>
      {blocked.isLoading ? <LoadingView /> : null}
      {blocked.data?.length === 0 ? <Text tone="muted">Vous n'avez bloqué personne.</Text> : null}
      {blocked.data?.map((b) => (
        <View key={b.userId} style={styles.blockedRow}>
          <Text style={styles.toggleText}>{b.displayName || "Utilisateur"}</Text>
          <Button label="Débloquer" onPress={() => void unblock(b.userId)} small variant="secondary" />
        </View>
      ))}
    </View>
  );
}

export default function PrivacyScreen() {
  const query = useProfile();
  if (query.isLoading) return <LoadingView />;
  if (!query.data) return <ErrorView message={errorMessage(query.error)} onRetry={() => void query.refetch()} />;
  return (
    <Screen scroll testID="privacy-screen">
      <Content profile={query.data} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { gap: spacing.md, paddingTop: spacing.lg },
  toggle: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.md,
    padding: spacing.lg,
    borderRadius: radii.md,
    borderWidth: 1
  },
  toggleText: { flex: 1, gap: 2 },
  sectionTitle: { marginTop: spacing.lg },
  blockedRow: { flexDirection: "row", alignItems: "center", gap: spacing.md, minHeight: 52 }
});
