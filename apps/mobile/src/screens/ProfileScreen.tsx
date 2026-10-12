import { Ionicons } from "@expo/vector-icons";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import React from "react";
import { StyleSheet, View } from "react-native";
import { errorMessage } from "../api/client";
import type { Card, MyProfile } from "../api/types";
import { PhotoCarousel } from "../components/PhotoCarousel";
import { ProfileDetails } from "../components/ProfileCard";
import { useMyProfile } from "../hooks/queries";
import { useAuth } from "../hooks/useAuth";
import type { MainStackParamList } from "../navigation/types";
import { radii, spacing } from "../theme";
import { Button, ErrorState, LoadingState, Screen, Text } from "../ui";

/** The owner's profile as other people see it (minus distance, which is relative to the viewer). */
function toCard(userId: string, data: MyProfile): Card | null {
  if (!data.profile) {
    return null;
  }
  return {
    userId,
    firstName: data.profile.firstName,
    age: data.profile.age,
    gender: data.profile.gender,
    bio: data.profile.bio,
    occupation: data.profile.occupation,
    city: data.profile.city,
    distanceKm: null,
    interests: data.interests,
    photos: data.photos
  };
}

export default function ProfileScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<MainStackParamList>>();
  const { user } = useAuth();
  const { data, isLoading, isError, error, refetch } = useMyProfile();

  if (isLoading) {
    return (
      <Screen>
        <LoadingState />
      </Screen>
    );
  }
  const card = data && user ? toCard(user.id, data) : null;
  if (isError || !card || !data) {
    return (
      <Screen>
        <ErrorState message={errorMessage(error, "Profil indisponible.")} onRetry={() => void refetch()} />
      </Screen>
    );
  }

  return (
    <Screen scroll>
      <View style={styles.header}>
        <Text variant="title" style={styles.flex}>
          Mon profil
        </Text>
        <Button label="Réglages" variant="secondary" onPress={() => navigation.navigate("Settings")} style={styles.small} />
      </View>
      <View style={styles.photo}>
        <PhotoCarousel photos={card.photos} name={card.firstName} />
      </View>
      <ProfileDetails card={card} />
      {!data.profile?.isVisible ? (
        <View style={styles.notice}>
          <Ionicons name="eye-off-outline" size={18} />
          <Text variant="caption" style={styles.flex}>
            Votre profil est masqué : personne ne peut le découvrir. Vos matchs peuvent toujours vous écrire.
          </Text>
        </View>
      ) : null}
      <Button label="Modifier mon profil et mes photos" onPress={() => navigation.navigate("EditProfile")} />
    </Screen>
  );
}

const styles = StyleSheet.create({
  flex: { flex: 1 },
  header: { flexDirection: "row", alignItems: "center", gap: spacing.md },
  small: { minHeight: 40, paddingHorizontal: spacing.lg },
  photo: { height: 440, borderRadius: radii.xl, overflow: "hidden" },
  notice: { flexDirection: "row", gap: spacing.sm, alignItems: "center" }
});
