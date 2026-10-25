import React from "react";
import { Alert, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { APP_NAME } from "../../config";
import { Avatar, Button, ErrorState, ListRow, Screen, Skeleton, Text } from "../../shared/ui";
import { spacing } from "../../theme";
import type { MainStackParamList } from "../../navigation/types";
import { useAuth } from "../auth/AuthProvider";
import { useProfileState } from "./hooks";

type Navigation = NativeStackNavigationProp<MainStackParamList>;

export function ProfileScreen() {
  const navigation = useNavigation<Navigation>();
  const { signOut, user } = useAuth();
  const query = useProfileState();
  const profile = query.data?.profile;

  const confirmSignOut = () => {
    Alert.alert("Se déconnecter ?", undefined, [
      { text: "Annuler", style: "cancel" },
      { text: "Se déconnecter", onPress: () => void signOut() }
    ]);
  };

  if (query.isError) {
    return (
      <Screen edges={["top"]}>
        <ErrorState onRetry={() => void query.refetch()} />
      </Screen>
    );
  }

  return (
    <Screen edges={["top"]} scroll testID="profile-screen">
      <Text variant="title">Profil</Text>
      {profile ? (
        <View style={styles.identity}>
          <Avatar name={profile.firstName} size={88} thumbUrl={profile.photos[0]?.thumbUrl} />
          <View style={styles.identityText}>
            <Text variant="heading">{`${profile.firstName}, ${profile.age}`}</Text>
            <Text color="textMuted" numberOfLines={1}>
              {user?.email}
            </Text>
          </View>
        </View>
      ) : (
        <Skeleton height={88} />
      )}
      <Button
        icon="create-outline"
        label="Modifier mon profil et mes photos"
        onPress={() => navigation.navigate("EditProfile")}
        testID="profile-edit"
        variant="secondary"
      />
      <View>
        <ListRow
          chevron
          onPress={() => navigation.navigate("EditPreferences")}
          testID="profile-preferences"
          title="Mes préférences de rencontre"
        />
        <ListRow
          chevron
          onPress={() => navigation.navigate("Privacy")}
          testID="profile-privacy"
          title="Confidentialité et position"
        />
        <ListRow
          chevron
          onPress={() => navigation.navigate("BlockedUsers")}
          testID="profile-blocked"
          title="Utilisateurs bloqués"
        />
        <ListRow
          chevron
          onPress={() => navigation.navigate("ChangePassword")}
          testID="profile-password"
          title="Changer le mot de passe"
        />
        <ListRow onPress={confirmSignOut} testID="profile-logout" title="Se déconnecter" />
        <ListRow
          destructive
          onPress={() => navigation.navigate("DeleteAccount")}
          testID="profile-delete"
          title="Supprimer mon compte"
        />
      </View>
      <Text color="textMuted" style={styles.footer} variant="caption">
        {`${APP_NAME} est gratuit. Pas d'abonnement, pas de limite.`}
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  identity: { flexDirection: "row", alignItems: "center", gap: spacing.lg },
  identityText: { flex: 1, gap: spacing.xs },
  footer: { textAlign: "center" }
});
