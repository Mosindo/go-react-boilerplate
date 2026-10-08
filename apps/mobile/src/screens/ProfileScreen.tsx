import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useMe } from "../hooks/data";
import type { MainStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { GENDER_LABELS } from "../shared/format";
import { SafeAreaLayout, ScreenContainer } from "../shared/layout";
import { Avatar, Card, ListItem, Notice, Text, spacing } from "../shared/ui";

/** "Moi" tab: how others see me, plus entry points to editing, preferences and settings. */
export default function ProfileScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const { data: me, isLoading, isError, refetch } = useMe();

  if (isLoading) return <LoadingView fullScreen />;
  if (isError || !me?.profile) return <ErrorView message="Impossible de charger votre profil." onAction={() => void refetch()} />;
  const { profile, preferences } = me;

  return (
    <SafeAreaLayout edges={["top"]}>
      <ScreenContainer>
        <View style={styles.hero}>
          <Avatar name={profile.firstName} path={me.photos[0]?.thumbUrl} size="lg" />
          <View style={styles.heroText}>
            <Text accessibilityRole="header" variant="title">
              {profile.firstName}, {profile.age}
            </Text>
            <Text tone="muted">
              {GENDER_LABELS[profile.gender]}
              {profile.city ? ` · ${profile.city}` : ""}
            </Text>
          </View>
        </View>

        {!preferences.isVisible ? <Notice message="Votre profil est en pause : personne ne peut vous voir dans la découverte." /> : null}

        <Card>
          <ListItem
            icon="create-outline"
            onPress={() => navigation.navigate("EditProfile")}
            subtitle="Photos, bio, centres d'intérêt"
            title="Modifier mon profil"
          />
          <ListItem
            icon="options-outline"
            onPress={() => navigation.navigate("Preferences")}
            subtitle="Qui, quel âge, à quelle distance"
            title="Préférences de rencontre"
          />
          <ListItem icon="notifications-outline" onPress={() => navigation.navigate("Notifications")} title="Notifications" />
          <ListItem icon="settings-outline" onPress={() => navigation.navigate("Settings")} subtitle="Confidentialité, compte, sécurité" title="Réglages" />
        </Card>
        <Text align="center" tone="subtle" variant="caption">
          Lien est entièrement gratuit : aucune fonctionnalité n'est payante.
        </Text>
      </ScreenContainer>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  hero: { flexDirection: "row", alignItems: "center", gap: spacing.lg },
  heroText: { flex: 1, gap: 2 }
});
