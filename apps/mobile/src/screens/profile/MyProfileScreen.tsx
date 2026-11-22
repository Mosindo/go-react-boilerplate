import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation, type CompositeNavigationProp } from "@react-navigation/native";
import type { BottomTabNavigationProp } from "@react-navigation/bottom-tabs";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { errorMessage } from "../../api/client";
import { useMyProfile } from "../../hooks/queries";
import type { MainStackParamList, TabParamList } from "../../navigation/types";
import { Avatar, Chip, ErrorState, Header, ListRow, LoadingState, Screen, Section, Text, colors, radii, spacing } from "../../ui";

type Nav = CompositeNavigationProp<BottomTabNavigationProp<TabParamList, "Me">, NativeStackNavigationProp<MainStackParamList>>;

const missingLabels: Record<string, string> = {
  firstName: "prénom",
  birthdate: "date de naissance",
  gender: "genre",
  interestedIn: "qui vous souhaitez rencontrer",
  location: "position",
  photos: "une photo"
};

export default function MyProfileScreen() {
  const navigation = useNavigation<Nav>();
  const profile = useMyProfile();

  if (profile.isPending) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  if (profile.isError) {
    return (
      <Screen>
        <ErrorState message={errorMessage(profile.error)} onAction={() => void profile.refetch()} />
      </Screen>
    );
  }
  const p = profile.data;

  return (
    <Screen scroll testID="me-screen">
      <Header title="Mon profil" />
      <View style={styles.hero}>
        <Avatar name={p.firstName} ring size={112} url={p.photos[0]?.url} />
        <Text variant="title">
          {p.firstName}
          {p.age ? `, ${p.age}` : ""}
        </Text>
        {p.city ? <Text tone="muted">{p.city}</Text> : null}
        {!p.discoverable ? (
          <View style={styles.pill}>
            <Text tone="accent" variant="caption">
              Profil en pause
            </Text>
          </View>
        ) : null}
      </View>

      {!p.isComplete ? (
        <View style={styles.warning}>
          <Text variant="label">Profil incomplet</Text>
          <Text tone="muted" variant="caption">
            Il manque : {p.missingFields.map((f) => missingLabels[f] ?? f).join(", ")}. Vous restez invisible tant que ce n’est pas
            complété.
          </Text>
        </View>
      ) : null}

      {p.interests.length ? (
        <View style={styles.chips}>
          {p.interests.map((i) => (
            <Chip key={i.slug} label={i.label} />
          ))}
        </View>
      ) : null}

      <Section title="Profil">
        <ListRow
          icon="create-outline"
          onPress={() => navigation.navigate("EditProfile")}
          subtitle="Photos, bio, centres d'intérêt, position"
          testID="me-edit"
          title="Modifier mon profil"
        />
        <ListRow
          icon="options-outline"
          onPress={() => navigation.navigate("Preferences")}
          subtitle={`${p.preferences.minAge}–${p.preferences.maxAge} ans · ${p.preferences.maxDistanceKm} km`}
          testID="me-preferences"
          title="Préférences et confidentialité"
        />
        {p.userId ? (
          <ListRow
            icon="eye-outline"
            onPress={() => navigation.navigate("ProfileDetail", { userId: p.userId, context: "match" })}
            title="Aperçu public"
          />
        ) : null}
      </Section>
      <Section title="Compte">
        <ListRow icon="settings-outline" onPress={() => navigation.navigate("Settings")} testID="me-settings" title="Réglages du compte" />
      </Section>
      <Text align="center" tone="subtle" variant="caption">
        Lueur est 100 % gratuit : pas d’abonnement, pas de boost, pas de publicité.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: {
    alignItems: "center",
    gap: spacing.xs,
    marginBottom: spacing.lg
  },
  pill: {
    paddingHorizontal: spacing.sm,
    paddingVertical: 4,
    borderRadius: radii.pill,
    backgroundColor: colors.accentSoft
  },
  warning: {
    gap: 4,
    padding: spacing.md,
    borderRadius: radii.md,
    backgroundColor: colors.accentSoft,
    marginBottom: spacing.lg
  },
  chips: {
    flexDirection: "row",
    flexWrap: "wrap",
    justifyContent: "center",
    gap: spacing.xs,
    marginBottom: spacing.lg
  }
});
