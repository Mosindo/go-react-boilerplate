import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useAuth } from "../auth/AuthContext";
import { Chip } from "../components/Chip";
import { Photo } from "../components/Photo";
import { Screen } from "../components/Screen";
import { ErrorState, LoadingState } from "../components/States";
import { Text } from "../components/Text";
import { useOwnProfile } from "../hooks/useQueries";
import type { RootStackParamList } from "../navigation/types";
import { radius, spacing, useTheme } from "../theme";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export function MeScreen() {
  const theme = useTheme();
  const navigation = useNavigation<Nav>();
  const { user } = useAuth();
  const profile = useOwnProfile();

  if (profile.isPending)
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  if (profile.isError || !profile.data)
    return (
      <Screen>
        <ErrorState onRetry={() => void profile.refetch()} />
      </Screen>
    );
  const p = profile.data;

  const items: { label: string; hint: string; target: keyof RootStackParamList; testID: string }[] = [
    {
      label: "Modifier mon profil",
      hint: "Infos, bio, centres d'intérêt",
      target: "EditProfile",
      testID: "menu-edit"
    },
    {
      label: "Mes photos",
      hint: `${p.photos.length}/6 · ordre et photo principale`,
      target: "Photos",
      testID: "menu-photos"
    },
    {
      label: "Mes préférences",
      hint: "Genre, âge, distance",
      target: "Preferences",
      testID: "menu-preferences"
    },
    {
      label: "Confidentialité",
      hint: "Visibilité, distance, position",
      target: "Privacy",
      testID: "menu-privacy"
    },
    { label: "Personnes bloquées", hint: "Gérer les blocages", target: "Blocked", testID: "menu-blocked" },
    { label: "Compte et sécurité", hint: user?.email ?? "", target: "Account", testID: "menu-account" }
  ];

  return (
    <Screen scroll contentStyle={styles.content}>
      <View style={styles.hero}>
        <Photo url={p.photos[0]?.url} label="Ma photo principale" style={styles.avatar} />
        <Text variant="title" testID="me-name">
          {p.firstName}, {p.age}
        </Text>
        <Text muted>{[p.city, p.jobTitle].filter(Boolean).join(" · ")}</Text>
        {!p.isVisible ? <Chip label="Profil en pause" /> : null}
      </View>

      <View style={[styles.card, { backgroundColor: theme.surface, borderColor: theme.border }]}>
        {items.map((item, i) => (
          <Pressable
            key={item.label}
            onPress={() => navigation.navigate(item.target as never)}
            accessibilityRole="button"
            accessibilityLabel={item.label}
            testID={item.testID}
            style={({ pressed }) => [
              styles.row,
              i > 0 ? { borderTopWidth: StyleSheet.hairlineWidth, borderTopColor: theme.border } : null,
              { backgroundColor: pressed ? theme.surfaceAlt : "transparent" }
            ]}
          >
            <View style={styles.rowText}>
              <Text variant="bodyStrong">{item.label}</Text>
              <Text variant="caption" muted>
                {item.hint}
              </Text>
            </View>
            <Text muted>›</Text>
          </Pressable>
        ))}
      </View>
      <Text variant="caption" muted center>
        Lien est 100 % gratuit : pas d&apos;abonnement, pas de limite de likes, pas de publicité.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { maxWidth: 560, width: "100%", alignSelf: "center" },
  hero: { alignItems: "center", gap: spacing.xs },
  avatar: { width: 120, height: 120, borderRadius: 60, overflow: "hidden", marginBottom: spacing.sm },
  card: { borderRadius: radius.lg, borderWidth: 1, overflow: "hidden" },
  row: { flexDirection: "row", alignItems: "center", padding: spacing.lg, gap: spacing.md },
  rowText: { flex: 1, gap: 2 }
});
