import React from "react";
import { StyleSheet, View } from "react-native";
import { useMyProfile } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import type { TabScreenProps } from "../../navigation/types";
import { spacing } from "../../theme";
import { Avatar, Button, Card, ErrorState, LoadingState, ListRow, Screen, Text } from "../../ui";

export default function MeScreen({ navigation }: TabScreenProps<"Me">) {
  const { user, signOut } = useAuth();
  const query = useMyProfile();

  if (query.isLoading) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  if (!query.data?.profile) {
    return (
      <Screen>
        <ErrorState message="Impossible de charger votre profil." onRetry={() => void query.refetch()} />
      </Screen>
    );
  }
  const { profile, photos, interests } = query.data;
  const completion = [photos.length >= 3, !!profile.bio, interests.length >= 3, !!profile.jobTitle, profile.hasLocation].filter(
    Boolean
  ).length;

  return (
    <Screen scroll testID="me-screen">
      <View style={styles.hero}>
        <Avatar name={profile.firstName} ring size={112} uri={photos[0]?.url} />
        <Text variant="title">
          {profile.firstName}, {profile.age}
        </Text>
        <Text color="textMuted" variant="caption">
          {user?.email}
        </Text>
        <Button
          fullWidth={false}
          icon="eye-outline"
          label="Aperçu de mon profil"
          onPress={() => user && navigation.navigate("ProfileDetail", { userId: user.id, source: "self" })}
          size="sm"
          testID="me-preview"
          variant="secondary"
        />
      </View>
      {completion < 5 ? (
        <Card>
          <Text variant="bodyStrong">Profil complété à {Math.round((completion / 5) * 100)} %</Text>
          <Text color="textMuted" variant="caption">
            3 photos, une bio, quelques centres d’intérêt et votre métier aident les autres à engager la conversation.
          </Text>
        </Card>
      ) : null}
      {!profile.isDiscoverable ? (
        <Card>
          <Text variant="bodyStrong">Profil en pause</Text>
          <Text color="textMuted" variant="caption">
            Personne ne peut vous découvrir. Réactivez-le dans Confidentialité.
          </Text>
        </Card>
      ) : null}
      <Card>
        <ListRow icon="create-outline" onPress={() => navigation.navigate("EditProfile")} testID="me-edit" title="Modifier mon profil" />
        <ListRow
          icon="images-outline"
          onPress={() => navigation.navigate("Photos")}
          subtitle={`${photos.length}/6 photos`}
          testID="me-photos"
          title="Mes photos"
        />
        <ListRow
          icon="options-outline"
          onPress={() => navigation.navigate("Preferences")}
          testID="me-preferences"
          title="Critères de rencontre"
        />
        <ListRow
          icon="shield-checkmark-outline"
          onPress={() => navigation.navigate("Privacy")}
          testID="me-privacy"
          title="Confidentialité et sécurité"
        />
        <ListRow icon="person-circle-outline" onPress={() => navigation.navigate("Account")} testID="me-account" title="Compte" />
      </Card>
      <Button icon="log-out-outline" label="Se déconnecter" onPress={() => void signOut()} testID="me-logout" variant="secondary" />
      <Text align="center" color="textSubtle" variant="caption">
        Lueur est entièrement gratuite : pas d’abonnement, pas de publicité, pas de limite de likes.
      </Text>
      <View style={styles.bottomSpace} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: spacing.xs, paddingTop: spacing.md },
  bottomSpace: { height: 80 }
});
