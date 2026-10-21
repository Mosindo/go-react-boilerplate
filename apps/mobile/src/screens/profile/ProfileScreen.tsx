import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import React from "react";
import { StyleSheet, View } from "react-native";
import { fetchProfile } from "../../api/profile";
import { keys } from "../../hooks/queries";
import { useAuth } from "../../hooks/useAuth";
import { confirmAction } from "../../lib/platform";
import type { RootStackParamList } from "../../navigation/types";
import { ErrorView, LoadingView } from "../../shared/feedback";
import { ScreenContainer, Section } from "../../shared/layout";
import { Avatar, ListItem, Text } from "../../shared/ui";
import { spacing, useTheme } from "../../theme";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export function ProfileScreen() {
  const theme = useTheme();
  const navigation = useNavigation<Nav>();
  const { user, signOut } = useAuth();
  const query = useQuery({ queryKey: keys.profile, queryFn: fetchProfile });
  const profile = query.data;

  const chevron = <Ionicons name="chevron-forward" size={18} color={theme.textMuted} />;
  const go = (screen: keyof RootStackParamList) => () => navigation.navigate(screen as never);

  const logout = async () => {
    if (await confirmAction("Se déconnecter ?", "Vous devrez saisir à nouveau vos identifiants.", "Se déconnecter", false)) await signOut();
  };

  if (query.isPending) {
    return (
      <ScreenContainer edges={["top", "left", "right"]}>
        <LoadingView />
      </ScreenContainer>
    );
  }
  if (query.isError || !profile) {
    return (
      <ScreenContainer edges={["top", "left", "right"]}>
        <ErrorView message={query.error?.message} onRetry={() => void query.refetch()} />
      </ScreenContainer>
    );
  }

  return (
    <ScreenContainer scroll edges={["top", "left", "right"]} testID="profile-screen" contentStyle={styles.content}>
      <View style={styles.identity}>
        <Avatar name={profile.firstName} uri={profile.photos[0]?.thumbUrl} size={104} />
        <Text variant="heading" testID="profile-name">
          {profile.firstName}, {profile.age}
        </Text>
        <Text tone="muted">{user?.email}</Text>
      </View>

      <Section title="Mon profil">
        <ListItem
          testID="menu-edit-profile"
          title="Modifier mon profil"
          subtitle="Bio, centres d'intérêt, ville"
          chevron
          onPress={go("EditProfile")}
        />
        <ListItem testID="menu-photos" title="Mes photos" subtitle={`${profile.photos.length}/6`} chevron onPress={go("Photos")} />
        <ListItem
          testID="menu-preferences"
          title="Préférences de rencontre"
          subtitle="Qui, âge, distance"
          chevron
          onPress={go("Preferences")}
        />
      </Section>

      <Section title="Confidentialité et sécurité">
        <ListItem testID="menu-privacy" title="Visibilité et position" chevron onPress={go("Privacy")} />
        <ListItem testID="menu-blocked" title="Personnes bloquées" chevron onPress={go("BlockedUsers")} />
        <ListItem testID="menu-password" title="Mot de passe" chevron onPress={go("ChangePassword")} />
      </Section>

      <Section title="Compte">
        <ListItem testID="menu-logout" title="Se déconnecter" right={chevron} onPress={() => void logout()} />
        <ListItem testID="menu-delete" title="Supprimer mon compte" tone="danger" chevron onPress={go("DeleteAccount")} />
      </Section>
    </ScreenContainer>
  );
}

const styles = StyleSheet.create({
  content: { gap: spacing.xl, paddingBottom: spacing.xxl },
  identity: { alignItems: "center", gap: spacing.xs }
});
