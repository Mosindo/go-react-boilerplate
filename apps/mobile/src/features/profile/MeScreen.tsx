import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMyProfile } from "../../hooks/queries";
import type { MainStackParamList } from "../../navigation/types";
import { useTheme } from "../../theme/ThemeProvider";
import { Avatar, ErrorState, Header, LoadingState, Row, Screen, Section, Text } from "../../ui";

export function MeScreen() {
  const theme = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const { data, isLoading, error, refetch } = useMyProfile();
  if (isLoading)
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  if (error || !data?.profile)
    return (
      <Screen>
        <ErrorState onRetry={() => refetch()} />
      </Screen>
    );
  const p = data.profile;

  return (
    <Screen scroll>
      <Header large title="Profil" />
      <View style={styles.hero}>
        <Avatar name={p.firstName} photo={data.photos[0]} size={112} />
        <Text variant="title">
          {p.firstName}, {p.age}
        </Text>
        {p.moderationStatus !== "active" ? (
          <View style={[styles.notice, { backgroundColor: theme.colors.dangerSoft }]}>
            <Text tone="danger" variant="small">
              Votre profil est temporairement masqué pendant l’examen d’un signalement.
            </Text>
          </View>
        ) : null}
      </View>
      <Section title="Mon profil">
        <Row icon="create-outline" label="Modifier mon profil et mes photos" onPress={() => navigation.navigate("EditProfile")} testID="me-edit" />
        <Row icon="options-outline" label="Préférences de rencontre" last onPress={() => navigation.navigate("Preferences")} testID="me-preferences" />
      </Section>
      <Section title="Sécurité">
        <Row icon="shield-checkmark-outline" label="Confidentialité et blocages" onPress={() => navigation.navigate("Privacy")} testID="me-privacy" />
        <Row icon="person-circle-outline" label="Compte" last onPress={() => navigation.navigate("Account")} testID="me-account" />
      </Section>
      <Text align="center" tone="subtle" variant="caption">
        Lueur est 100 % gratuite : pas d’abonnement, pas de publicité, pas de likes limités.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: 10, paddingVertical: 16 },
  notice: { borderRadius: 14, padding: 12 }
});
