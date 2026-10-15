import React from "react";
import { Linking, View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { savePrivacy } from "../api/platform";
import { queryKeys, useMyProfile } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import { errorMessage } from "../i18n/errors";
import { APP_NAME } from "../i18n/fr";
import { confirm } from "../lib/confirm";
import type { RootStackParamList } from "../navigation/types";
import { showError } from "../shared/feedback";
import { Header, ScreenContainer, Section } from "../shared/layout";
import { Button, ListItem, Text, Toggle, spacing } from "../shared/ui";
import { useTheme } from "../theme";

export default function SettingsScreen() {
  const palette = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const queryClient = useQueryClient();
  const { signOut, user } = useAuth();
  const query = useMyProfile();
  const profile = query.data?.profile;

  const privacy = useMutation({
    mutationFn: savePrivacy,
    onSuccess: (response) => queryClient.setQueryData(queryKeys.myProfile, response),
    onError: (error) => showError(errorMessage(error))
  });

  const chevron = <Ionicons color={palette.textSubtle} name="chevron-forward" size={20} />;

  return (
    <ScreenContainer scroll testID="settings-screen">
      <Header onBack={() => navigation.goBack()} title="Réglages" />

      <Section title="CONFIDENTIALITÉ">
        <Toggle
          description="Désactivez pour mettre votre profil en pause : vos matchs et conversations restent actifs."
          disabled={!profile || privacy.isPending}
          label="Visible dans la découverte"
          onValueChange={(value) => privacy.mutate({ isVisible: value })}
          testID="toggle-visible"
          value={profile?.isVisible ?? true}
        />
        <Toggle
          description="Les autres voient une distance approximative (par tranches de 5 km). Jamais votre position."
          disabled={!profile || privacy.isPending}
          label="Afficher ma distance"
          onValueChange={(value) => privacy.mutate({ showDistance: value })}
          testID="toggle-distance"
          value={profile?.showDistance ?? true}
        />
      </Section>

      <Section title="SÉCURITÉ">
        <ListItem
          bordered
          leading={<Ionicons color={palette.brand} name="ban-outline" size={24} />}
          onPress={() => navigation.navigate("Blocked")}
          testID="open-blocked"
          title="Utilisateurs bloqués"
          trailing={chevron}
        />
        <ListItem
          bordered
          leading={<Ionicons color={palette.brand} name="key-outline" size={24} />}
          onPress={() => navigation.navigate("Security")}
          testID="open-security"
          title="Mot de passe et suppression du compte"
          trailing={chevron}
        />
      </Section>

      <Section title="COMPTE">
        <Text tone="muted">{user?.email}</Text>
        <Button
          fullWidth
          label="Se déconnecter"
          onPress={() => confirm({ title: "Se déconnecter ?", confirmLabel: "Se déconnecter", onConfirm: () => void signOut() })}
          testID="sign-out"
          variant="outline"
        />
      </Section>

      <View style={{ gap: spacing.xs, paddingTop: spacing.md }}>
        <Text style={{ textAlign: "center" }} tone="subtle" variant="caption">
          {APP_NAME} est gratuite : pas d&apos;abonnement, pas de boost payant, pas de limite de likes.
        </Text>
        <Button
          label="Besoin d'aide ? Nous contacter"
          onPress={() => void Linking.openURL("mailto:support@alma.invalid")}
          size="sm"
          variant="ghost"
        />
      </View>
    </ScreenContainer>
  );
}
