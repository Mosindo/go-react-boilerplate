import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMyProfile } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import type { AppStackParamList } from "../../navigation/types";
import { ErrorView, LoadingView } from "../../shared/feedback";
import { Screen } from "../../shared/layout";
import { Card, ListRow, Photo, Text, colors, radii, spacing } from "../../shared/ui";

type Nav = NativeStackNavigationProp<AppStackParamList>;

export default function MeScreen() {
  const navigation = useNavigation<Nav>();
  const { user } = useAuth();
  const profile = useMyProfile();

  if (profile.isLoading) {
    return <LoadingView />;
  }
  if (!profile.data) {
    return <ErrorView onRetry={() => void profile.refetch()} />;
  }
  const p = profile.data;

  return (
    <Screen testID="me-screen">
      <View style={styles.hero}>
        <Photo placeholderLabel={p.firstName.slice(0, 1)} style={styles.avatar} url={p.photos[0]?.url} />
        <View style={styles.heroText}>
          <Text variant="title">
            {p.firstName}
            {p.age ? `, ${p.age}` : ""}
          </Text>
          <Text tone="muted" variant="label">
            {user?.email}
          </Text>
        </View>
      </View>
      {!p.settings.isDiscoverable ? (
        <Card tone="primary">
          <Text weight="semibold">Ton profil est en pause</Text>
          <Text tone="muted" variant="label">
            Personne ne te voit dans Découvrir. Tes matchs peuvent toujours t&apos;écrire.
          </Text>
        </Card>
      ) : null}
      <Card padded={false} style={styles.group}>
        <ListRow
          onPress={() => navigation.navigate("ProfileDetail", { userId: p.userId })}
          subtitle="Tel que les autres le voient"
          title="Aperçu de mon profil"
        />
        <ListRow
          onPress={() => navigation.navigate("EditProfile")}
          subtitle="Prénom, bio, centres d'intérêt"
          testID="me-edit"
          title="Modifier mon profil"
        />
        <ListRow onPress={() => navigation.navigate("Photos")} subtitle={`${p.photos.length}/${p.maxPhotos} photos`} title="Mes photos" />
        <ListRow
          onPress={() => navigation.navigate("Preferences")}
          subtitle={`${p.preferences.minAge}–${p.preferences.maxAge} ans · ${p.preferences.maxDistanceKm} km`}
          title="Critères de rencontre"
        />
        <ListRow
          onPress={() => navigation.navigate("Settings")}
          subtitle="Confidentialité, sécurité, compte"
          testID="me-settings"
          title="Réglages"
        />
      </Card>
      <Text align="center" tone="subtle" variant="caption">
        Hearth est entièrement gratuit : pas d&apos;abonnement, pas de pub, pas de limite.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: {
    flexDirection: "row",
    alignItems: "center",
    gap: spacing.lg
  },
  avatar: {
    width: 88,
    height: 88,
    borderRadius: 44,
    borderWidth: 2,
    borderColor: colors.primary
  },
  heroText: {
    flex: 1,
    gap: 2
  },
  group: {
    paddingHorizontal: spacing.lg,
    borderRadius: radii.lg
  }
});
