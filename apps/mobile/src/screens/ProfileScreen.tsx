import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { StyleSheet, View } from "react-native";
import { useMyProfile } from "../hooks/queries";
import { GENDER_LABELS } from "../lib/labels";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Avatar, Button, Card, Chip, ListItem, Text } from "../shared/ui";
import { Ionicons } from "@expo/vector-icons";
import { spacing, useTheme } from "../theme";

/** "Profil" tab: preview of my profile and shortcuts to settings. */
export default function ProfileScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const { colors } = useTheme();
  const { data, isLoading, isError, error, refetch } = useMyProfile();

  if (isLoading) return <LoadingView />;
  if (isError || !data?.profile) return <ErrorView error={error} onRetry={() => void refetch()} />;
  const profile = data.profile;
  const chevron = <Ionicons name="chevron-forward" size={20} color={colors.textMuted} />;

  return (
    <ScreenContainer scroll edges={["top"]} testID="me-screen">
      <View style={styles.top}>
        <Avatar name={profile.firstName} path={profile.photos[0]?.url} size={112} />
        <Text variant="title" testID="me-name">
          {profile.firstName}, {profile.age}
        </Text>
        <Text tone="muted">{[GENDER_LABELS[profile.gender], profile.city].filter(Boolean).join(" · ")}</Text>
        <Button testID="me-edit" label="Modifier mon profil" icon="create-outline" compact onPress={() => navigation.navigate("EditProfile")} />
      </View>

      {profile.bio ? (
        <Card>
          <Text>{profile.bio}</Text>
        </Card>
      ) : null}
      {profile.interests.length > 0 ? (
        <View style={styles.chips}>
          {profile.interests.map((i) => (
            <Chip key={i.slug} label={i.label} />
          ))}
        </View>
      ) : null}
      {!profile.isDiscoverable ? (
        <Card>
          <Text variant="bodyStrong">Votre profil est en pause</Text>
          <Text variant="caption" tone="muted">
            Personne ne peut vous découvrir. Vous pouvez le réactiver dans les réglages.
          </Text>
        </Card>
      ) : null}

      <View>
        <ListItem testID="me-preferences" title="Mes préférences de rencontre" subtitle="Genre, âge, distance" trailing={chevron} onPress={() => navigation.navigate("Preferences")} />
        <ListItem testID="me-settings" title="Réglages et confidentialité" subtitle="Pause, distance visible, mot de passe, compte" trailing={chevron} onPress={() => navigation.navigate("Settings")} />
        <ListItem testID="me-blocked" title="Personnes bloquées" trailing={chevron} onPress={() => navigation.navigate("BlockedUsers")} />
      </View>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  top: { alignItems: "center", gap: spacing.sm, paddingTop: spacing.lg },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm }
});
