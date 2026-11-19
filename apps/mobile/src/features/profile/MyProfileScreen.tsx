import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { AppStackParamList } from "../../navigation/types";
import { useSession } from "../../state/session";
import { useTheme } from "../../theme/ThemeProvider";
import { spacing } from "../../theme/tokens";
import { Avatar, Card, Header, ListRow, LoadingState, Screen, SectionTitle, Text } from "../../ui";

type Nav = NativeStackNavigationProp<AppStackParamList>;

function completion(p: NonNullable<ReturnType<typeof useSession>["profile"]>): { score: number; tip: string | null } {
  const checks: [boolean, string][] = [
    [p.photos.length >= 3, "Ajoutez au moins 3 photos."],
    [p.bio.trim().length >= 40, "Écrivez une bio de quelques lignes."],
    [p.interests.length >= 3, "Choisissez au moins 3 centres d'intérêt."],
    [p.hasLocation, "Activez la localisation pour des rencontres proches."]
  ];
  const done = checks.filter(([ok]) => ok).length;
  return { score: Math.round(((done + 1) / (checks.length + 1)) * 100), tip: checks.find(([ok]) => !ok)?.[1] ?? null };
}

export function MyProfileScreen() {
  const navigation = useNavigation<Nav>();
  const { colors } = useTheme();
  const { profile } = useSession();
  if (!profile) return <LoadingState />;
  const { score, tip } = completion(profile);

  return (
    <Screen scroll>
      <Header title="Mon profil" large />
      <View style={styles.hero}>
        <Avatar path={profile.photos[0]?.url} name={profile.firstName} size={112} ring />
        <Text variant="title">
          {profile.firstName}, {profile.age}
        </Text>
        {!profile.preferences.isDiscoverable ? (
          <Text tone="danger" variant="label">
            Profil masqué de la découverte
          </Text>
        ) : null}
        <View style={[styles.meter, { backgroundColor: colors.border }]}>
          <View style={[styles.meterFill, { width: `${score}%`, backgroundColor: colors.accent }]} />
        </View>
        <Text variant="caption" tone="muted" align="center">
          Profil complété à {score} %{tip ? ` · ${tip}` : ""}
        </Text>
      </View>
      <SectionTitle>Profil</SectionTitle>
      <Card>
        <ListRow
          icon="create-outline"
          label="Modifier mes informations"
          onPress={() => navigation.navigate("EditProfile")}
          testID="me-edit"
        />
        <ListRow
          icon="images-outline"
          label="Mes photos"
          value={`${profile.photos.length}/6`}
          onPress={() => navigation.navigate("Photos")}
        />
        <ListRow
          icon="eye-outline"
          label="Aperçu de mon profil"
          onPress={() => navigation.navigate("ProfileDetail", { userId: profile.userId })}
        />
      </Card>
      <SectionTitle>Rencontres</SectionTitle>
      <Card>
        <ListRow
          icon="options-outline"
          label="Préférences et confidentialité"
          onPress={() => navigation.navigate("Preferences")}
          testID="me-preferences"
        />
      </Card>
      <SectionTitle>Compte</SectionTitle>
      <Card>
        <ListRow icon="settings-outline" label="Paramètres" onPress={() => navigation.navigate("Settings")} testID="me-settings" />
      </Card>
      <Text variant="caption" tone="subtle" align="center" style={styles.free}>
        Lueur est entièrement gratuit. Aucune option payante, jamais.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: spacing.xs, marginVertical: spacing.md },
  meter: { width: "70%", height: 6, borderRadius: 3, overflow: "hidden", marginTop: spacing.sm },
  meterFill: { height: 6 },
  free: { marginTop: spacing.xl }
});
