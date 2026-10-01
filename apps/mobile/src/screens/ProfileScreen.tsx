import React from "react";
import { ScrollView, StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { useQuery } from "@tanstack/react-query";
import { profileApi } from "../api/platform";
import { queryKeys } from "../api/queryClient";
import type { RootStackParamList } from "../navigation/types";
import { ErrorView, LoadingView } from "../shared/feedback";
import { SafeAreaLayout } from "../shared/layout";
import { PhotoImage } from "../shared/media/PhotoImage";
import { Button, Card, Chip, ListItem, Notice, Text, radii, spacing } from "../shared/ui";

type Nav = NativeStackNavigationProp<RootStackParamList>;

export default function ProfileScreen() {
  const navigation = useNavigation<Nav>();
  const { data: profile, isLoading, isError, refetch } = useQuery({ queryKey: queryKeys.profile, queryFn: profileApi.getOwn });

  if (isLoading) {
    return (
      <SafeAreaLayout edges={["top"]}>
        <LoadingView label="Chargement du profil…" />
      </SafeAreaLayout>
    );
  }
  if (isError || !profile) {
    return (
      <SafeAreaLayout edges={["top"]}>
        <ErrorView message="Impossible de charger votre profil." onAction={() => void refetch()} />
      </SafeAreaLayout>
    );
  }

  const tips: string[] = [];
  if (profile.bio.trim().length === 0) tips.push("Ajoutez une courte bio : elle fait la différence.");
  if (profile.interests.length < 3) tips.push("Choisissez au moins 3 centres d'intérêt pour de meilleures suggestions.");
  if (profile.photos.length < 3) tips.push("Les profils avec 3 photos ou plus reçoivent plus de réponses.");
  if (!profile.hasLocation) tips.push("Activez votre position pour voir des profils proches.");

  return (
    <SafeAreaLayout edges={["top"]}>
      <ScrollView contentContainerStyle={styles.content} testID="profile-screen">
        <Text accessibilityRole="header" variant="title">
          Mon profil
        </Text>

        <View style={styles.hero}>
          <PhotoImage
            accessibilityLabel="Votre photo principale"
            fallbackLabel={profile.firstName}
            path={profile.photos[0]?.url}
            style={styles.photo}
          />
          <Text variant="title">
            {profile.firstName}, {profile.age}
          </Text>
          {profile.city ? <Text tone="muted">{profile.city}</Text> : null}
          {!profile.discoverable ? (
            <Notice message="Votre profil est masqué : personne ne le voit dans la découverte." tone="warning" />
          ) : null}
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

        {tips.length > 0 ? (
          <Card>
            <Text weight="bold">Pour aller plus loin</Text>
            {tips.map((tip) => (
              <Text key={tip} tone="muted" variant="label">
                • {tip}
              </Text>
            ))}
          </Card>
        ) : null}

        <View style={styles.actions}>
          <Button label="Modifier mon profil" onPress={() => navigation.navigate("EditProfile")} />
          <Button label="Préférences et confidentialité" onPress={() => navigation.navigate("Settings")} variant="outline" />
        </View>
        <Card padded={false}>
          <ListItem
            onPress={() => navigation.navigate("BlockedUsers")}
            subtitle="Gérer les personnes que vous avez bloquées"
            title="Personnes bloquées"
          />
        </Card>
      </ScrollView>
    </SafeAreaLayout>
  );
}

const styles = StyleSheet.create({
  content: { padding: spacing.lg, gap: spacing.lg, paddingBottom: spacing.xxxl },
  hero: { alignItems: "center", gap: spacing.sm },
  photo: { width: 168, height: 224, borderRadius: radii.xl, overflow: "hidden" },
  chips: { flexDirection: "row", flexWrap: "wrap", gap: spacing.sm, justifyContent: "center" },
  actions: { gap: spacing.sm }
});
