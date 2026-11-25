import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMyProfile } from "../../hooks/queries";
import { useSession } from "../../hooks/useSession";
import type { AppStackParamList } from "../../navigation/types";
import { spacing } from "../../theme";
import { Avatar, Header, ListRow, LoadingState, Screen, Text } from "../../ui";

export function MeScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const { user } = useSession();
  const me = useMyProfile();

  if (!me.data) {
    return (
      <Screen edges={["top"]}>
        <LoadingState />
      </Screen>
    );
  }
  const { profile, photos, privacy } = me.data;

  return (
    <Screen scroll edges={["top"]} testID="me-screen">
      <Header title="Mon profil" large />
      <View style={styles.identity}>
        <Avatar path={photos[0]?.url} name={profile?.firstName ?? "?"} size={96} ring />
        <Text variant="title">
          {profile?.firstName}
          {profile ? `, ${profile.age}` : ""}
        </Text>
        <Text variant="small" color="textSubtle">
          {user?.email}
        </Text>
        {!privacy.discoverable ? (
          <Text variant="caption" color="gold">
            Profil masqué de la découverte
          </Text>
        ) : null}
      </View>
      <View style={styles.group}>
        <ListRow
          icon="eye-outline"
          title="Aperçu de mon profil"
          subtitle="Tel que les autres le voient"
          onPress={() => user && navigation.navigate("ProfileDetail", { userId: user.id })}
          testID="me-preview"
        />
        <ListRow
          icon="create-outline"
          title="Modifier mon profil"
          subtitle="Prénom, bio, centres d'intérêt…"
          onPress={() => navigation.navigate("EditProfile")}
          testID="me-edit"
        />
        <ListRow
          icon="images-outline"
          title="Mes photos"
          subtitle={`${photos.length}/6`}
          onPress={() => navigation.navigate("EditPhotos")}
          testID="me-photos"
        />
        <ListRow
          icon="options-outline"
          title="Préférences de rencontre"
          subtitle="Genres, âge, distance"
          onPress={() => navigation.navigate("Preferences")}
          testID="me-preferences"
        />
      </View>
      <View style={styles.group}>
        <ListRow
          icon="lock-closed-outline"
          title="Confidentialité et sécurité"
          subtitle="Visibilité, distance, blocages"
          onPress={() => navigation.navigate("Privacy")}
          testID="me-privacy"
        />
        <ListRow
          icon="person-circle-outline"
          title="Compte"
          subtitle="Mot de passe, déconnexion, suppression"
          onPress={() => navigation.navigate("Account")}
          testID="me-account"
        />
      </View>
      <Text variant="caption" color="textSubtle" align="center" style={styles.footer}>
        Lueur est gratuite, pour tout le monde, sans limite.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  identity: {
    alignItems: "center",
    gap: spacing.xs,
    paddingVertical: spacing.lg
  },
  group: {
    gap: spacing.xs,
    marginBottom: spacing.lg
  },
  footer: {
    marginBottom: spacing.lg
  }
});
