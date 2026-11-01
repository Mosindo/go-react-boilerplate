import { Alert, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import { Avatar, Badge, ErrorState, Loading, Row, Screen, Text } from "../components/ui";
import { errorMessage } from "../lib/errors";
import { useAuth } from "../state/auth";
import { useMyProfile, useNotifications } from "../state/queries";
import { spacing } from "../theme";

export function ProfileScreen() {
  const navigation = useNavigation();
  const { signOut, user } = useAuth();
  const profile = useMyProfile();
  const notifications = useNotifications();

  if (profile.isLoading) {
    return (
      <Screen edges={["top"]}>
        <Loading />
      </Screen>
    );
  }
  if (profile.isError || !profile.data) {
    return (
      <Screen edges={["top"]}>
        <ErrorState message={errorMessage(profile.error)} onRetry={() => void profile.refetch()} />
      </Screen>
    );
  }
  const me = profile.data;

  return (
    <Screen edges={["top"]} scroll>
      <View style={{ alignItems: "center", marginVertical: spacing.lg }}>
        <Avatar name={me.firstName} photoId={me.photoIds[0] ?? null} size={112} />
        <Text style={{ marginTop: spacing.md }} variant="title">
          {me.firstName}, {me.age}
        </Text>
        <Text muted>{user?.email}</Text>
      </View>

      <Row
        icon="create-outline"
        onPress={() => navigation.navigate("EditProfile")}
        subtitle="Bio, localisation, centres d'intérêt"
        title="Modifier le profil"
      />
      <Row
        icon="images-outline"
        onPress={() => navigation.navigate("Photos")}
        subtitle={`${me.photoIds.length}/6 photos`}
        title="Mes photos"
      />
      <Row
        icon="options-outline"
        onPress={() => navigation.navigate("Settings")}
        subtitle="Qui voir, distance, confidentialité"
        title="Préférences"
      />
      <Row
        icon="notifications-outline"
        onPress={() => navigation.navigate("Notifications")}
        right={<Badge count={notifications.data?.unreadCount ?? 0} />}
        title="Notifications"
      />
      <Row icon="ban-outline" onPress={() => navigation.navigate("Blocked")} title="Profils bloqués" />
      <View style={{ height: spacing.lg }} />
      <Row
        icon="log-out-outline"
        onPress={() =>
          Alert.alert("Se déconnecter ?", undefined, [
            { text: "Annuler", style: "cancel" },
            { text: "Me déconnecter", onPress: () => void signOut() },
          ])
        }
        title="Se déconnecter"
      />
      <Row
        danger
        icon="trash-outline"
        onPress={() => navigation.navigate("DeleteAccount")}
        title="Supprimer mon compte"
      />
      <Text muted style={{ textAlign: "center", marginTop: spacing.lg }} variant="caption">
        Alma est 100 % gratuit : pas d'abonnement, pas de pub, pas de limite de likes.
      </Text>
    </Screen>
  );
}
