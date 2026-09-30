import React from "react";
import { StyleSheet, View } from "react-native";
import { useNavigation } from "@react-navigation/native";
import type { NativeStackNavigationProp } from "@react-navigation/native-stack";
import { Avatar, Card, ErrorState, ListRow, LoadingState, Screen, Text } from "../../design/components";
import { errorMessage } from "../../lib/api/client";
import { genderLabels } from "../../lib/format";
import type { AppStackParamList } from "../../navigation/types";
import { useProfile } from "./hooks";

export default function MyProfileScreen() {
  const navigation = useNavigation<NativeStackNavigationProp<AppStackParamList>>();
  const { data: profile, isLoading, error, refetch } = useProfile();
  if (isLoading) return <LoadingState />;
  if (error || !profile) return <ErrorState message={errorMessage(error)} onRetry={() => void refetch()} />;

  return (
    <Screen scroll testID="me-screen">
      <View style={styles.hero}>
        <Avatar name={profile.firstName} ring size={112} uri={profile.photos[0]?.url} />
        <Text variant="title">
          {profile.firstName}
          {profile.age ? `, ${profile.age}` : ""}
        </Text>
        <Text tone="muted">
          {[profile.gender ? genderLabels[profile.gender] : null, profile.city || null].filter(Boolean).join(" · ")}
        </Text>
        {!profile.discoverable ? (
          <Text tone="primary" variant="label">
            Profil en pause : vous n’apparaissez pas dans la découverte.
          </Text>
        ) : null}
      </View>
      <Card>
        <ListRow icon="create-outline" onPress={() => navigation.navigate("EditProfile")} subtitle="Photos, bio, centres d'intérêt, localisation" testID="me-edit" title="Modifier mon profil" />
        <ListRow icon="options-outline" onPress={() => navigation.navigate("Preferences")} subtitle="Âge, distance, genres" title="Préférences de rencontre" />
        <ListRow icon="shield-checkmark-outline" onPress={() => navigation.navigate("Settings")} subtitle="Confidentialité, blocages, compte" testID="me-settings" title="Confidentialité et compte" />
      </Card>
      <Text align="center" tone="subtle" variant="caption">
        Lueur est entièrement gratuite : aucun abonnement, aucune fonctionnalité payante.
      </Text>
    </Screen>
  );
}

const styles = StyleSheet.create({
  hero: { alignItems: "center", gap: 8, paddingVertical: 16 }
});
