import React from "react";
import { View } from "react-native";
import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { mediaUrl } from "../api/client";
import { ProfileBody } from "../components/ProfileBody";
import { useMyProfile } from "../hooks/queries";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { ScreenContainer } from "../shared/layout";
import { Avatar, Button, Card, ListItem, Notice, Text, spacing } from "../shared/ui";
import { useTheme } from "../theme";

/** "Me" tab: how others see me, plus shortcuts to everything editable. */
export default function ProfileScreen() {
  const palette = useTheme();
  const navigation = useNavigation<NativeStackNavigationProp<RootStackParamList>>();
  const query = useMyProfile();
  const profile = query.data?.profile;

  const row = (
    title: string,
    subtitle: string,
    icon: React.ComponentProps<typeof Ionicons>["name"],
    screen: keyof RootStackParamList,
    testID: string
  ) => (
    <ListItem
      bordered
      leading={<Ionicons color={palette.brand} name={icon} size={24} />}
      onPress={() => navigation.navigate(screen as never)}
      subtitle={subtitle}
      testID={testID}
      title={title}
      trailing={<Ionicons color={palette.textSubtle} name="chevron-forward" size={20} />}
    />
  );

  return (
    <ScreenContainer onRefresh={() => void query.refetch()} refreshing={query.isRefetching} scroll testID="profile-screen">
      <View style={{ flexDirection: "row", alignItems: "center", justifyContent: "space-between", minHeight: 56 }}>
        <Text accessibilityRole="header" variant="heading" weight="bold">
          Mon profil
        </Text>
        <Button label="Réglages" onPress={() => navigation.navigate("Settings")} size="sm" testID="open-settings" variant="ghost" />
      </View>

      {query.isLoading ? <LoadingView fullScreen /> : null}
      {query.isError ? <ErrorView error={query.error} onRetry={() => void query.refetch()} /> : null}

      {profile ? (
        <>
          <Card padded={false}>
            <View style={{ flexDirection: "row", alignItems: "center", gap: spacing.md, padding: spacing.lg }}>
              <Avatar name={profile.firstName} size="lg" uri={mediaUrl(profile.photos[0]?.thumbUrl)} />
              <View style={{ flex: 1, gap: 2 }}>
                <Text variant="subheading" weight="bold">
                  {profile.firstName}, {profile.age}
                </Text>
                <Text numberOfLines={1} tone="muted" variant="label">
                  {profile.city ?? (profile.hasLocation ? "Position enregistrée" : "Position non renseignée")}
                </Text>
              </View>
              <Button
                label="Modifier"
                onPress={() => navigation.navigate("EditProfile")}
                size="sm"
                testID="edit-profile"
                variant="secondary"
              />
            </View>
          </Card>

          {!profile.isVisible ? (
            <Notice
              message="Votre profil est masqué : personne ne le voit dans la découverte. Vos matchs et conversations restent actifs."
              title="Mode discret activé"
              tone="warning"
            />
          ) : null}

          <View>
            {row("Photos", `${profile.photos.length} sur 6`, "images-outline", "Photos", "open-photos")}
            {row(
              "Mes préférences",
              profile.preferences
                ? `${profile.preferences.minAge}–${profile.preferences.maxAge} ans · ${profile.preferences.maxDistanceKm} km`
                : "À renseigner",
              "options-outline",
              "Preferences",
              "open-preferences"
            )}
            {row(
              "Confidentialité, sécurité et compte",
              "Visibilité, utilisateurs bloqués, mot de passe",
              "shield-checkmark-outline",
              "Settings",
              "open-privacy"
            )}
          </View>

          <View style={{ gap: spacing.sm }}>
            <Text tone="muted" variant="eyebrow" weight="semibold">
              Aperçu de votre profil
            </Text>
            <ProfileBody
              age={profile.age}
              bio={profile.bio}
              city={profile.city}
              firstName={profile.firstName}
              gender={profile.gender}
              interests={profile.interests}
              photos={profile.photos}
            />
          </View>
        </>
      ) : null}
    </ScreenContainer>
  );
}
