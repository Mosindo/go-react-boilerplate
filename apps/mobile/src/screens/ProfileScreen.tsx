import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import type { MyProfile, PublicProfile } from "../api/types";
import { ProfileCard } from "../components/ProfileCard";
import { Screen } from "../components/Screen";
import { useMyProfile } from "../hooks/queries";
import type { MainStackParamList } from "../navigation/types";
import { ErrorView } from "../shared/feedback";
import { Button, Loader, Text, colors, spacing } from "../shared/ui";

type Nav = NativeStackNavigationProp<MainStackParamList>;

/** How other people see this profile (minus the distance, which depends on the viewer). */
export function toPublicProfile(profile: MyProfile): PublicProfile {
  return {
    userId: profile.userId,
    firstName: profile.firstName,
    age: profile.age,
    gender: profile.gender,
    bio: profile.bio,
    city: profile.city,
    distanceKm: null,
    interests: profile.interests,
    photos: profile.photos
  };
}

export default function ProfileScreen() {
  const navigation = useNavigation<Nav>();
  const { data: profile, isLoading, isError, refetch } = useMyProfile();

  return (
    <Screen edges={["top", "left", "right"]} contentStyle={styles.content} scroll testID="profile-screen">
      <Text variant="title" weight="bold">
        Mon profil
      </Text>
      {isLoading ? <Loader label="Chargement…" /> : null}
      {isError ? (
        <ErrorView
          actionLabel="Réessayer"
          message="Impossible de charger votre profil."
          onAction={() => void refetch()}
          title="Oups"
        />
      ) : null}
      {profile ? (
        <>
          {!profile.isVisible ? (
            <View style={styles.paused} testID="profile-paused">
              <Text weight="semibold">
                Votre profil est en pause : personne ne le découvre. Réactivez-le dans les paramètres.
              </Text>
            </View>
          ) : null}
          <View style={styles.preview}>
            <ProfileCard profile={toPublicProfile(profile)} variant="full" />
          </View>
          <Button
            label="Modifier mon profil"
            onPress={() => navigation.navigate("EditProfile")}
            size="lg"
            testID="edit-profile"
          />
          <Button
            label="Paramètres et confidentialité"
            onPress={() => navigation.navigate("Settings")}
            testID="open-settings"
            variant="outline"
          />
        </>
      ) : null}
    </Screen>
  );
}

const styles = StyleSheet.create({
  content: { paddingBottom: 110 },
  preview: { height: 520 },
  paused: { padding: spacing.md, borderRadius: 14, backgroundColor: colors.warningSoft }
});
