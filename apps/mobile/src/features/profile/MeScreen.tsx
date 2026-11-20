import React from "react";
import { Pressable, StyleSheet, View } from "react-native";
import Ionicons from "@expo/vector-icons/Ionicons";
import { useNavigation } from "@react-navigation/native";
import { Image } from "expo-image";
import { mediaUrl } from "../../api/client";
import type { AppStackParamList } from "../../navigation/types";
import { errorMessage, genderLabels } from "../../lib/format";
import { useAuth } from "../../providers/AuthProvider";
import { useTheme } from "../../theme";
import { Badge, Card, ErrorState, Header, LoadingState, Screen, Text } from "../../ui";
import { useMyProfile } from "./queries";

type Entry = { icon: keyof typeof Ionicons.glyphMap; label: string; route: keyof AppStackParamList; testID?: string };

const entries: Entry[] = [
  { icon: "create-outline", label: "Modifier mon profil", route: "EditProfile", testID: "me-edit" },
  { icon: "images-outline", label: "Mes photos", route: "EditPhotos" },
  { icon: "options-outline", label: "Préférences de rencontre", route: "Preferences" },
  { icon: "lock-closed-outline", label: "Confidentialité et compte", route: "Settings", testID: "me-settings" }
];

export function MeScreen() {
  const navigation = useNavigation();
  const { user } = useAuth();
  const { colors, radii, spacing } = useTheme();
  const profile = useMyProfile();

  if (profile.isLoading) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  if (profile.isError || !profile.data) {
    return (
      <Screen>
        <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />
      </Screen>
    );
  }
  const p = profile.data;
  return (
    <Screen scroll header={<Header large title="Mon profil" />}>
      <View style={styles.hero}>
        <View style={[styles.photo, { borderRadius: radii.xl, backgroundColor: colors.surfaceAlt, borderColor: colors.primary }]}>
          {p.photos[0] ? <Image contentFit="cover" source={{ uri: mediaUrl(p.photos[0].url) }} style={StyleSheet.absoluteFill} /> : null}
        </View>
        <Text variant="title">
          {p.firstName}, {p.age}
        </Text>
        <Text tone="muted">{[genderLabels[p.gender], p.city].filter(Boolean).join(" · ")}</Text>
        <View style={{ flexDirection: "row", gap: spacing.xs }}>
          {!p.discoverable ? <Badge label="Profil en pause" tone="accent" /> : <Badge label="Visible" />}
          {user?.isDemo ? <Badge label="Compte démo" tone="accent" /> : null}
        </View>
      </View>
      <Card style={{ padding: 0, gap: 0 }}>
        {entries.map((entry, i) => (
          <Pressable
            accessibilityRole="button"
            key={entry.route}
            onPress={() => navigation.navigate(entry.route as never)}
            style={({ pressed }) => [
              styles.row,
              {
                padding: spacing.md,
                backgroundColor: pressed ? colors.surfaceAlt : "transparent",
                borderTopWidth: i ? StyleSheet.hairlineWidth : 0,
                borderTopColor: colors.border
              }
            ]}
            testID={entry.testID}
          >
            <Ionicons color={colors.primary} name={entry.icon} size={22} />
            <Text style={{ flex: 1 }} variant="bodyStrong">
              {entry.label}
            </Text>
            <Ionicons color={colors.textSubtle} name="chevron-forward" size={18} />
          </Pressable>
        ))}
      </Card>
      <Card style={{ backgroundColor: colors.accentSoft, borderColor: "transparent" }}>
        <Text tone="accent" variant="bodyStrong">
          Lueur est entièrement gratuit
        </Text>
        <Text tone="muted" variant="caption">
          Pas d’abonnement, pas de boost payant, pas de likes limités. Toutes les fonctionnalités sont ouvertes à tout le monde.
        </Text>
      </Card>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: 6 },
  photo: { width: 132, height: 176, overflow: "hidden", borderWidth: 2, marginBottom: 8 },
  row: { flexDirection: "row", alignItems: "center", gap: 12 }
});
